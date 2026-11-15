package api

import (
	"context"
	"encoding/json"
	"github.com/go-chi/chi/v5"
	"github.com/google/uuid"
	"github.com/ivantit66/onebase/internal/auth"
	"github.com/ivantit66/onebase/internal/dbtest"
	"github.com/ivantit66/onebase/internal/dsl/interpreter"
	"github.com/ivantit66/onebase/internal/metadata"
	reportpkg "github.com/ivantit66/onebase/internal/report"
	"github.com/ivantit66/onebase/internal/runtime"
	"github.com/ivantit66/onebase/internal/storage"
	"net/http"
	"net/http/httptest"
	"net/url"
	"strings"
	"testing"
)

func TestAPIReportSourceFiltersMatrix(t *testing.T) {
	dbtest.ForEachDialect(t, func(t *testing.T, db *storage.DB) {
		ctx := context.Background()
		e := &metadata.Entity{Name: "APISourceFilter", Kind: metadata.KindCatalog, Fields: []metadata.Field{{Name: "Метка", Type: metadata.FieldTypeString}, {Name: "Amount", Type: metadata.FieldTypeNumber}, {Name: "Owner", Type: metadata.FieldTypeString}, {Name: "Secret", Type: metadata.FieldTypeString}}}
		if err := db.Migrate(ctx, []*metadata.Entity{e}); err != nil {
			t.Fatal(err)
		}
		for i, label := range []string{"ALPHA", "BETA", "FOREIGN"} {
			owner := "runner"
			if i == 2 {
				owner = "other"
			}
			if err := db.Upsert(ctx, e.Name, uuid.New(), map[string]any{"Метка": label, "Amount": i + 1, "Owner": owner, "Secret": "RAW_SECRET"}, e); err != nil {
				t.Fatal(err)
			}
		}
		rep := &reportpkg.Report{Name: "APISourceFilters", Query: `ВЫБРАТЬ Метка, Amount, Secret ИЗ Справочник.APISourceFilter`, Composition: &reportpkg.Composition{Groupings: []string{"Метка"}, Detail: true, Measures: []reportpkg.Measure{{Field: "Amount", Agg: "sum"}}}, Variants: []reportpkg.ReportVariant{{Name: "beta", Query: `ВЫБРАТЬ Метка, Amount ИЗ Справочник.APISourceFilter`, Filters: []reportpkg.VariantFilter{{Field: "Метка", Op: "eq", Value: "BETA"}}}}}
		reg := runtime.NewRegistry()
		reg.Load(runtime.LoadOptions{Entities: []*metadata.Entity{e}, Reports: []*reportpkg.Report{rep}})
		h := &handler{reg: reg, store: db, interp: interpreter.New()}
		router := chi.NewRouter()
		h.mountV2(router)
		perm := auth.Permission{Reports: map[string][]string{rep.Name: {"run"}}, Catalogs: map[string][]string{e.Name: {"read"}}, RowAccess: auth.RowAccess{Catalogs: map[string]auth.RowPolicies{e.Name: {"read": {Field: "Owner", Op: "eq", Value: auth.RowValue{User: "login"}}}}}, FieldAccess: auth.FieldAccess{Catalogs: map[string]auth.FieldPolicies{e.Name: {"Secret": {Read: "mask_all"}}}}}
		for _, name := range []string{"alpha", "page-after-filter", "numeric", "case-field", "empty", "masked", "variant", "variant-conflict", "composed", "unknown-field", "invalid-op", "invalid-json", "oversize", "source-denied"} {
			t.Run(name, func(t *testing.T) {
				settings := &reportpkg.UserReportSettings{Filters: []reportpkg.Filter{{Field: "Метка", Op: "eq", Value: "ALPHA"}}}
				q := url.Values{}
				want, status := 1, 200
				p := perm
				switch name {
				case "page-after-filter":
					q.Set("limit", "1")
					settings.Filters[0].Value = "BETA"
				case "numeric":
					settings.Filters = []reportpkg.Filter{{Field: "Amount", Op: "ge", Value: "2"}}
				case "case-field":
					settings.Filters[0].Field = "мЕТКА"
				case "empty":
					settings.Filters[0].Value = "NONE"
					want = 0
				case "masked":
					settings.Filters = []reportpkg.Filter{{Field: "Secret", Op: "eq", Value: "RAW_SECRET"}}
					want = 0
				case "variant":
					settings.Variant = "beta"
					settings.Filters[0].Value = "BETA"
				case "variant-conflict":
					settings.Variant = "beta"
					want = 0
				case "composed":
					q.Set("compose", "1")
				case "unknown-field":
					settings.Filters[0].Field = "Unknown"
					status = 400
				case "invalid-op":
					settings.Filters[0].Op = "bogus"
					status = 400
				case "invalid-json", "oversize":
					status = 400
				case "source-denied":
					p.Catalogs = map[string][]string{}
					status = 403
				}
				if strings.HasPrefix(name, "variant") {
					p.FieldAccess = auth.FieldAccess{}
				}
				raw, _ := settings.JSON()
				q.Set("__settings", raw)
				if name == "invalid-json" {
					q.Set("__settings", "{")
				}
				if name == "oversize" {
					q.Set("__settings", strings.Repeat(" ", reportpkg.MaxSourceSettingsBytes+1))
				}
				r := withUser(httptest.NewRequest("GET", "/api/v2/report/"+rep.Name+"?"+q.Encode(), nil), apiUser("runner", p))
				w := httptest.NewRecorder()
				router.ServeHTTP(w, r)
				if w.Code != status {
					t.Fatalf("HTTP%d want%d:%s", w.Code, status, w.Body.String())
				}
				if status != 200 {
					return
				}
				var body struct {
					Data json.RawMessage `json:"data"`
					Meta restV2Meta      `json:"meta"`
				}
				if err := json.Unmarshal(w.Body.Bytes(), &body); err != nil {
					t.Fatal(err)
				}
				if body.Meta.Total != want {
					t.Fatalf("filtered total%d want%d", body.Meta.Total, want)
				}
				if strings.Contains(string(body.Data), "FOREIGN") || strings.Contains(string(body.Data), "RAW_SECRET") {
					t.Fatal("RLS/mask leak")
				}
				if name != "composed" {
					var rows []map[string]any
					if err := json.Unmarshal(body.Data, &rows); err != nil {
						t.Fatal(err)
					}
					if len(rows) != want {
						t.Fatal("filter applied after pagination or ignored")
					}
				}
			})
		}
		t.Run("source-budget-refusal", func(t *testing.T) {
			for i := 0; i <= restMaxLimit; i++ {
				if err := db.Upsert(ctx, e.Name, uuid.New(), map[string]any{"Метка": "BETA", "Amount": 1, "Owner": "runner"}, e); err != nil {
					t.Fatal(err)
				}
			}
			raw, _ := (&reportpkg.UserReportSettings{Filters: []reportpkg.Filter{{Field: "Метка", Op: "eq", Value: "ALPHA"}}}).JSON()
			w := httptest.NewRecorder()
			router.ServeHTTP(w, withUser(httptest.NewRequest(http.MethodGet, "/api/v2/report/"+rep.Name+"?__settings="+url.QueryEscape(raw), nil), apiUser("runner", perm)))
			if w.Code != 422 {
				t.Fatalf("cropped filter source HTTP%d", w.Code)
			}
		})
	})
}
