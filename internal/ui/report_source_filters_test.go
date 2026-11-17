package ui

import (
	"bytes"
	"context"
	"fmt"
	"github.com/google/uuid"
	"github.com/ivantit66/onebase/internal/auth"
	"github.com/ivantit66/onebase/internal/dbtest"
	"github.com/ivantit66/onebase/internal/dsl/ast"
	"github.com/ivantit66/onebase/internal/dsl/interpreter"
	"github.com/ivantit66/onebase/internal/metadata"
	reportpkg "github.com/ivantit66/onebase/internal/report"
	"github.com/ivantit66/onebase/internal/runtime"
	"github.com/ivantit66/onebase/internal/storage"
	"github.com/xuri/excelize/v2"
	"net/http"
	"net/url"
	"strings"
	"testing"
)

func TestReportResultSourceFiltersHTTPMatrix(t *testing.T) {
	dbtest.ForEachDialect(t, func(t *testing.T, db *storage.DB) {
		ctx := context.Background()
		ref := &metadata.Entity{Name: "SourceFilterRef", Kind: metadata.KindCatalog, Fields: []metadata.Field{{Name: "Наименование", Type: metadata.FieldTypeString}}}
		entity := &metadata.Entity{Name: "SourceFilterRow", Kind: metadata.KindCatalog, Fields: []metadata.Field{{Name: "Метка", Type: metadata.FieldTypeString}, {Name: "Amount", Type: metadata.FieldTypeNumber}, {Name: "RefValue", Type: "reference:SourceFilterRef", RefEntity: ref.Name}, {Name: "Owner", Type: metadata.FieldTypeString}, {Name: "Secret", Type: metadata.FieldTypeString}}}
		if err := db.Migrate(ctx, []*metadata.Entity{ref, entity}); err != nil {
			t.Fatal(err)
		}
		if err := db.EnsureSettingsSchema(ctx); err != nil {
			t.Fatal(err)
		}
		a, b := uuid.New(), uuid.New()
		for _, id := range []uuid.UUID{a, b} {
			if err := db.Upsert(ctx, ref.Name, id, map[string]any{"Наименование": "Display label"}, ref); err != nil {
				t.Fatal(err)
			}
		}
		for i, label := range []string{"ALPHA", "BETA", "FOREIGN"} {
			id := a
			if i == 1 {
				id = b
			}
			owner := "source-runner"
			if i == 2 {
				owner = "other"
			}
			if err := db.Upsert(ctx, entity.Name, uuid.New(), map[string]any{"Метка": label, "Amount": i + 1, "RefValue": id.String(), "Owner": owner, "Secret": "RAW_SECRET"}, entity); err != nil {
				t.Fatal(err)
			}
		}
		cases := []string{"unfiltered", "get-alpha", "post-alpha", "case-field", "numeric", "reference", "masked", "empty", "variant-filter", "variant-conflict", "variant-no-user-filter", "preset-alpha", "default-preset-alpha", "unknown-field", "unsupported-op", "reference-label", "queryless-filter", "queryless-no-filter", "malformed-settings", "query-budget", "variant-budget", "no-composition", "settings-too-large", "source-read-denied"}
		for index, name := range cases {
			t.Run(name, func(t *testing.T) {
				rep := &reportpkg.Report{Name: fmt.Sprintf("SourceFilters%d", index), Query: `ВЫБРАТЬ П.Метка, П.Amount, П.RefValue.Ссылка КАК RefValue, П.Secret ИЗ Справочник.SourceFilterRow КАК П`, ResultProc: "Build", Composition: &reportpkg.Composition{Detail: true, Measures: []reportpkg.Measure{{Field: "Метка", Agg: "count"}}}, Variants: []reportpkg.ReportVariant{{Name: "beta", Query: `ВЫБРАТЬ П.Метка, П.Amount, П.RefValue.Ссылка КАК RefValue ИЗ Справочник.SourceFilterRow КАК П`, Filters: []reportpkg.VariantFilter{{Field: "Метка", Op: "eq", Value: "BETA"}}}}}
				prog := mustParse(t, `Функция Build() Д=Новый ТабличныйДокумент;Д.Ячейка(1,1).Текст="COUNT:"+Строка(Результат.Количество());Н=2;Для Каждого Стр Из Результат Цикл Д.Ячейка(Н,1).Текст=Стр.Метка;Д.Ячейка(Н,2).Текст=Строка(Стр.RefValue);Н=Н+1;КонецЦикла;Возврат Д;КонецФункции`)
				reg := runtime.NewRegistry()
				reg.Load(runtime.LoadOptions{Entities: []*metadata.Entity{ref, entity}, Reports: []*reportpkg.Report{rep}, Programs: map[string]*ast.Program{rep.Name: prog}})
				interp := interpreter.New()
				interp.LookupProc = reg.GetModuleProc
				s := &Server{reg: reg, store: db, interp: interp, messages: NewMessageStore(), lockMgr: runtime.NewLockManager(), ops: newOperationLimiter()}
				s.entitySvc = s.newEntityService(nil)
				t.Cleanup(s.Close)
				role := &auth.Role{Name: "SourceFilterRunner", Permissions: auth.Permission{Reports: map[string][]string{rep.Name: {"run"}}, Catalogs: map[string][]string{entity.Name: {"read"}, ref.Name: {"read"}}, RowAccess: auth.RowAccess{Catalogs: map[string]auth.RowPolicies{entity.Name: {"read": {Field: "Owner", Op: "eq", Value: auth.RowValue{User: "login"}}}}}, FieldAccess: auth.FieldAccess{Catalogs: map[string]auth.FieldPolicies{entity.Name: {"Secret": {Read: "mask_all"}}}}}}
				actor := &auth.User{ID: uuid.NewString(), Login: "source-runner", Roles: []*auth.Role{role}}
				settings := &reportpkg.UserReportSettings{Filters: []reportpkg.Filter{{Field: "Метка", Op: "eq", Value: "ALPHA"}}}
				want, status := 1, http.StatusSeeOther
				method := http.MethodPost
				switch name {
				case "unfiltered":
					settings = nil
					want = 2
				case "get-alpha":
					method = http.MethodGet
				case "case-field":
					settings.Filters[0].Field = "мЕТКА"
				case "numeric":
					settings.Filters = []reportpkg.Filter{{Field: "Amount", Op: "ge", Value: "2"}}
				case "reference":
					settings.Filters = []reportpkg.Filter{{Field: "RefValue", Op: "eq", Value: strings.ToUpper(a.String())}}
				case "masked":
					settings.Filters = []reportpkg.Filter{{Field: "Secret", Op: "eq", Value: "RAW_SECRET"}}
					status = http.StatusBadRequest // Protected fields cannot be used as a report oracle.
				case "empty":
					settings.Filters[0].Value = "NONE"
					want = 0
				case "variant-filter", "variant-budget":
					settings.Variant = "beta"
					settings.Filters[0].Value = "BETA"
					status = 400 // Native permanent filters wrap a derived SELECT: no lineage proof yet.
				case "variant-conflict":
					settings.Variant = "beta"
					want = 0
					status = 400
				case "variant-no-user-filter":
					settings.Variant = "beta"
					settings.Filters = nil
				case "unknown-field":
					settings.Filters[0].Field = "Unknown"
					status = 400
				case "unsupported-op":
					settings.Filters[0].Op = "bogus"
					status = 400
				case "reference-label":
					settings.Filters = []reportpkg.Filter{{Field: "RefValue", Op: "eq", Value: "Display label"}}
					status = 400
				case "queryless-filter":
					rep.Query = ""
					status = 422
				case "queryless-no-filter":
					rep.Query = ""
					settings = nil
					want = 0
				case "malformed-settings", "settings-too-large":
					status = 400
				case "query-budget":
					s.cfg.Limits.ReportMaxRows = 1
					want = 1 // The accepted ALPHA row fits; rejected rows do not consume the cap.
				case "no-composition":
					rep.Composition = nil
				case "source-read-denied":
					delete(role.Permissions.Catalogs, entity.Name)
					status = 403
				}
				if strings.HasPrefix(name, "variant-") {
					role.Permissions.FieldAccess = auth.FieldAccess{}
				}
				if name == "variant-budget" {
					s.cfg.Limits.ReportMaxRows = 1
				}
				form := url.Values{}
				if settings != nil {
					raw, err := settings.JSON()
					if err != nil {
						t.Fatal(err)
					}
					form.Set("__settings", raw)
				}
				if name == "malformed-settings" {
					form.Set("__settings", "{")
				}
				if name == "settings-too-large" {
					form.Set("__settings", strings.Repeat(" ", reportpkg.MaxSourceSettingsBytes+1))
				}
				if name == "preset-alpha" || name == "default-preset-alpha" {
					raw, _ := settings.JSON()
					id, err := db.SaveReportPreset(ctx, storage.ReportPreset{Report: rep.Name, User: actor.Login, Name: "Own filter", SettingsJSON: raw, IsDefault: name == "default-preset-alpha"})
					if err != nil {
						t.Fatal(err)
					}
					form.Del("__settings")
					if name == "preset-alpha" {
						form.Set("__preset", id)
					}
				}
				target := "/ui/report/" + rep.Name
				if method == http.MethodGet {
					target += "?__run=1&" + form.Encode()
					form = nil
				}
				before := len(s.tabularResults.items)
				w := tabularHTTP(s, actor, "own-source-filter", method, target, form)
				// Arbitrary DSL output needs a host producer proof even if its SQL
				// inputs can be filtered. Keep native validation failures above;
				// a previously successful filtered module must now stop explicitly.
				if status == http.StatusSeeOther && settings != nil && len(settings.Filters) > 0 {
					status = http.StatusUnprocessableEntity
				}
				if w.Code != status {
					t.Fatalf("HTTP%d want%d: %s", w.Code, status, w.Body.String())
				}
				if status != 303 {
					if len(s.tabularResults.items) != before || w.Header().Get("Location") != "" {
						t.Fatal("unsupported/denied/truncated input published a snapshot")
					}
					return
				}
				path := w.Header().Get("Location")
				result := s.tabularResults.items[strings.TrimPrefix(path, "/ui/tabular/")]
				if result == nil || result.doc.GetCell(0, 0).Text != fmt.Sprintf("COUNT:%d", want) {
					t.Fatal("result_proc got unfiltered rows")
				}
				for _, c := range result.doc.Cells {
					if strings.Contains(c.Text, "FOREIGN") || strings.Contains(c.Text, "RAW_SECRET") {
						t.Fatal("RLS/mask leak into module result")
					}
				}
				if name == "reference" && result.doc.GetCell(1, 1).Text != a.String() {
					t.Fatal("raw reference identity replaced by a display label")
				}
				view := tabularHTTP(s, actor, "own-source-filter", http.MethodGet, path, nil)
				if view.Code != 200 || !strings.Contains(view.Body.String(), fmt.Sprintf("COUNT:%d", want)) {
					t.Fatal("HTML does not show filtered snapshot")
				}
				xlsx := tabularHTTP(s, actor, "own-source-filter", http.MethodGet, path+"/xlsx", nil)
				f, err := excelize.OpenReader(bytes.NewReader(xlsx.Body.Bytes()))
				if err != nil {
					t.Fatal(err)
				}
				value, err := f.GetCellValue(f.GetSheetList()[0], "A1")
				f.Close()
				if err != nil || value != fmt.Sprintf("COUNT:%d", want) {
					t.Fatal("XLSX is not the filtered snapshot")
				}
				pdf := tabularHTTP(s, actor, "own-source-filter", http.MethodGet, path+"/pdf", nil)
				if pdf.Code != 200 || !bytes.HasPrefix(pdf.Body.Bytes(), []byte("%PDF-")) {
					t.Fatal("filtered snapshot PDF unavailable")
				}
				if s.tabularResults.items[strings.TrimPrefix(path, "/ui/tabular/")] != result || len(s.tabularResults.items) != before+1 {
					t.Fatal("exports reran report or replaced source snapshot")
				}
			})
		}
	})
}
