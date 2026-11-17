package ui

import (
	"archive/zip"
	"bytes"
	"compress/zlib"
	"context"
	"fmt"
	"io"
	"net/http/httptest"
	"regexp"
	"strings"
	"testing"

	"github.com/google/uuid"
	"github.com/ivantit66/onebase/internal/auth"
	"github.com/ivantit66/onebase/internal/dbtest"
	"github.com/ivantit66/onebase/internal/dsl/ast"
	"github.com/ivantit66/onebase/internal/dsl/interpreter"
	"github.com/ivantit66/onebase/internal/metadata"
	reportpkg "github.com/ivantit66/onebase/internal/report"
	"github.com/ivantit66/onebase/internal/runtime"
	"github.com/ivantit66/onebase/internal/storage"
)

// This tests output noninterference, not a search over compressed binary data:
// changing only PII and the arbitrary stored label must not change any XLSX XML
// entry or PDF drawing instruction, including text wrapped across PDF lines.
func TestReportOutputPrivacyHTTPMatrix(t *testing.T) {
	dbtest.ForEachDialect(t, func(t *testing.T, db *storage.DB) {
		ctx := context.Background()
		entity := &metadata.Entity{Name: "SensitiveDoc", Kind: metadata.KindDocument, Presentation: []string{"Label"}, Fields: []metadata.Field{
			{Name: "Label", Type: metadata.FieldTypeString}, {Name: "Phone", Type: metadata.FieldTypeString, PII: true}, {Name: "Address", Type: metadata.FieldTypeString, ReportPII: true}, {Name: "Amount", Type: metadata.FieldTypeNumber},
		}}
		if err := db.Migrate(ctx, []*metadata.Entity{entity}); err != nil {
			t.Fatal(err)
		}
		rep := &reportpkg.Report{Name: "Privacy", Query: "ВЫБРАТЬ Ссылка КАК Doc, Amount ИЗ Документ.SensitiveDoc"}
		reg := runtime.NewRegistry()
		reg.Load(runtime.LoadOptions{Entities: []*metadata.Entity{entity}, Reports: []*reportpkg.Report{rep}})
		s := &Server{store: db, reg: reg, interp: interpreter.New(), lockMgr: runtime.NewLockManager(), messages: NewMessageStore(), ops: newOperationLimiter()}
		id := uuid.New()
		write := func(label, phone, address string) {
			t.Helper()
			if err := db.Upsert(ctx, entity.Name, id, map[string]any{"Label": label, "Phone": phone, "Address": address, "Amount": 12345}, entity); err != nil {
				t.Fatal(err)
			}
		}
		request := func(kind string, user *auth.User) []byte {
			t.Helper()
			r := reqWithChi("GET", "/ui/report/Privacy?__run=1", nil, map[string]string{"name": rep.Name})
			if user != nil {
				r = r.WithContext(auth.ContextWithUser(r.Context(), user))
			}
			w := httptest.NewRecorder()
			switch kind {
			case "html":
				s.reportForm(w, r)
			case "xlsx":
				s.reportExcel(w, r)
			case "pdf":
				s.reportPDF(w, r)
			}
			if w.Code != 200 {
				t.Fatalf("%s: %d %s", kind, w.Code, w.Body.String())
			}
			return w.Body.Bytes()
		}
		xml := func(data []byte) string {
			t.Helper()
			z, err := zip.NewReader(bytes.NewReader(data), int64(len(data)))
			if err != nil {
				t.Fatal(err)
			}
			var all strings.Builder
			for _, f := range z.File {
				if !strings.HasPrefix(f.Name, "xl/") {
					continue
				}
				r, err := f.Open()
				if err != nil {
					t.Fatal(err)
				}
				b, err := io.ReadAll(r)
				_ = r.Close()
				if err != nil {
					t.Fatal(err)
				}
				all.WriteString(f.Name)
				all.Write(b)
			}
			return all.String()
		}
		pdfPages := func(data []byte) string {
			t.Helper()
			var all strings.Builder
			for _, stream := range regexp.MustCompile(`(?s)stream\r?\n(.*?)\r?\nendstream`).FindAllSubmatch(data, -1) {
				r, err := zlib.NewReader(bytes.NewReader(stream[1]))
				if err != nil {
					continue
				}
				b, err := io.ReadAll(r)
				_ = r.Close()
				if err != nil {
					t.Fatal(err)
				}
				if bytes.Contains(b, []byte("BT ")) {
					all.Write(b)
				}
			}
			if all.Len() == 0 {
				t.Fatal("no PDF page text found")
			}
			return all.String()
		}
		otkPermission := auth.Permission{
			Reports:   map[string][]string{rep.Name: {"run"}},
			Documents: map[string][]string{entity.Name: {"read", "disclose"}},
			FieldAccess: auth.FieldAccess{Documents: map[string]auth.FieldPolicies{
				entity.Name: {"Phone": {Read: "full"}, "Address": {Read: "full"}},
			}},
		}
		users := []*auth.User{{IsAdmin: true}, {Login: "otk", Roles: []*auth.Role{{Permissions: otkPermission}}}}
		for _, user := range users {
			for _, composed := range []bool{false, true} {
				if composed {
					rep.Composition = &reportpkg.Composition{Groupings: []string{"Doc"}, Measures: []reportpkg.Measure{{Field: "Amount", Agg: "sum"}}, Detail: true, DetailLink: "Doc", DetailEntity: entity.Name}
				} else {
					rep.Composition = nil
				}
				write("Baseline", "BASE_PHONE", "BASE_ADDRESS")
				baselineHTML := string(request("html", user))
				baselineXLSX := xml(request("xlsx", user))
				baselinePDF := pdfPages(request("pdf", user))
				if !strings.Contains(baselineHTML, id.String()) || !strings.Contains(baselineXLSX, "12345") {
					t.Fatal("lost UUID or ordinary measure")
				}
				write("Request 79990009987 HOUSE_987 FLAT_654", "79990009987", "HOUSE_987 FLAT_654")
				changedHTML := string(request("html", user))
				if strings.Contains(changedHTML, "79990009987") || strings.Contains(changedHTML, "HOUSE_987") || strings.Contains(changedHTML, "FLAT_654") {
					t.Fatal("PII in report HTML")
				}
				if changedHTML != baselineHTML {
					t.Fatal("changing only PII/label changed report HTML")
				}
				if xml(request("xlsx", user)) != baselineXLSX {
					t.Fatal("changing only PII/label changed XLSX XML/relationships")
				}
				if pdfPages(request("pdf", user)) != baselinePDF {
					t.Fatal("changing only PII/label changed PDF drawing instructions")
				}
			}
		}
		// Stored data and ordinary privileged cards retain their previous policy.
		row, err := db.GetByID(ctx, entity.Name, id, entity)
		if err != nil {
			t.Fatal(err)
		}
		if got := s.maskedRecordLabel(auth.ContextWithUser(ctx, users[0]), entity, row); !strings.Contains(got, "79990009987") {
			t.Fatal("ordinary card label changed")
		}
	})
}

func TestReportUnknownUUIDDoesNotResolveUnrelatedLabelHTTP(t *testing.T) {
	id := uuid.New()
	entity := &metadata.Entity{Name: "Unrelated", Kind: metadata.KindCatalog, Presentation: []string{"Label"}, Fields: []metadata.Field{{Name: "Label", Type: metadata.FieldTypeString}}}
	rep := &reportpkg.Report{Name: "LiteralUUID", Query: fmt.Sprintf(`ВЫБРАТЬ "%s" КАК Value`, id)}
	s, ctx := newSubmitTestServer(t, []*metadata.Entity{entity})
	s.reg.Load(runtime.LoadOptions{Entities: []*metadata.Entity{entity}, Reports: []*reportpkg.Report{rep}})
	request := func() string {
		t.Helper()
		r := reqWithChi("GET", "/ui/report/LiteralUUID?__run=1", nil, map[string]string{"name": rep.Name})
		r = r.WithContext(auth.ContextWithUser(r.Context(), &auth.User{IsAdmin: true}))
		w := httptest.NewRecorder()
		s.reportForm(w, r)
		if w.Code != 200 {
			t.Fatal("report failed", w.Code)
		}
		return w.Body.String()
	}
	if err := s.store.Upsert(ctx, entity.Name, id, map[string]any{"Label": "BASE_UNRELATED_LABEL"}, entity); err != nil {
		t.Fatal(err)
	}
	before := request()
	if err := s.store.Upsert(ctx, entity.Name, id, map[string]any{"Label": "79990009987 HOUSE_987 FLAT_654"}, entity); err != nil {
		t.Fatal(err)
	}
	after := request()
	if !strings.Contains(after, id.String()) || strings.Contains(after, "79990009987") || before != after {
		t.Fatal("an unproven UUID string resolved an unrelated object's presentation")
	}
}

func TestReportResultProcDiscloseCannotReadPrivateReferenceHTTP(t *testing.T) {
	entity := &metadata.Entity{Name: "RefPrivacy", Kind: metadata.KindCatalog, Presentation: []string{"Label"}, Fields: []metadata.Field{
		{Name: "Label", Type: metadata.FieldTypeString}, {Name: "Phone", Type: metadata.FieldTypeString, PII: true}, {Name: "House", Type: metadata.FieldTypeString, ReportPII: true},
	}}
	rep := &reportpkg.Report{Name: "RefPrivacyModule", Query: "ВЫБРАТЬ Ссылка ИЗ Справочник.RefPrivacy", ResultProc: "Build"}
	s, ctx := newSubmitTestServer(t, []*metadata.Entity{entity})
	program := mustParse(t, `Функция Build()
Д = Новый ТабличныйДокумент;
Д.Ячейка(1, 1).Текст = "SAFE_REPORT_MARKER";
З = Новый Запрос;
З.Текст = "ВЫБРАТЬ Ссылка КАК Ключ ИЗ Справочник.RefPrivacy";
Для Каждого Стр Из З.Выполнить() Цикл
    Об = Справочники.RefPrivacy.НайтиПоИдентификатору(Стр.Ключ.УникальныйИдентификатор()).ПолучитьОбъект();
    Д.Ячейка(2, 1).Текст = Строка(Об.Label) + Строка(Об.Phone) + Строка(Об.House);
    Д.Ячейка(3, 1).Текст = "REFERENCE_READ_MARKER";
КонецЦикла;
Возврат Д;
КонецФункции`)
	s.reg.Load(runtime.LoadOptions{Entities: []*metadata.Entity{entity}, Reports: []*reportpkg.Report{rep}, Programs: map[string]*ast.Program{rep.Name: program}})
	s.interp.LookupProc = s.reg.GetModuleProc
	id := uuid.New()
	if err := s.store.Upsert(ctx, entity.Name, id, map[string]any{"Label": "HOUSE_987 FLAT_654", "Phone": "79990009987", "House": "HOUSE_987"}, entity); err != nil {
		t.Fatal(err)
	}
	users := []*auth.User{
		{ID: uuid.NewString(), Login: "privacy-admin", IsAdmin: true},
		{ID: uuid.NewString(), Login: "privacy-reader", Roles: []*auth.Role{{Permissions: auth.Permission{
			Reports: map[string][]string{rep.Name: {"run"}}, Catalogs: map[string][]string{entity.Name: {"read", "disclose"}},
		}}}},
	}
	for _, actor := range users {
		response := tabularHTTP(s, actor, "privacy-session", "POST", "/ui/report/"+rep.Name, nil)
		if response.Code != 303 {
			t.Fatalf("report module: HTTP%d %s", response.Code, response.Body.String())
		}
		view := tabularHTTP(s, actor, "privacy-session", "GET", response.Header().Get("Location"), nil)
		if view.Code != 200 || !strings.Contains(view.Body.String(), "SAFE_REPORT_MARKER") || !strings.Contains(view.Body.String(), "REFERENCE_READ_MARKER") {
			t.Fatal("report result missing", view.Code)
		}
		for _, secret := range []string{"79990009987", "HOUSE_987", "FLAT_654"} {
			if strings.Contains(view.Body.String(), secret) {
				t.Fatal("report module used ordinary disclose to reveal a protected reference")
			}
		}
	}
}

func TestReportOutputPrivacyUsesSourceKindHTTP(t *testing.T) {
	cat := &metadata.Entity{Name: "Shared", Kind: metadata.KindCatalog, Fields: []metadata.Field{{Name: "Phone", Type: metadata.FieldTypeString}}}
	ir := &metadata.InfoRegister{Name: "Shared", Dimensions: []metadata.Field{{Name: "Phone", Type: metadata.FieldTypeString, PII: true}}}
	rep := &reportpkg.Report{Name: "KindPrivacy", Query: "ВЫБРАТЬ Phone ИЗ РегистрСведений.Shared ГДЕ Phone = &Value", Params: []reportpkg.Param{{Name: "Value", Type: "string"}}}
	s, _ := newSubmitTestServer(t, []*metadata.Entity{cat})
	s.reg.Load(runtime.LoadOptions{Entities: []*metadata.Entity{cat}, InfoRegs: []*metadata.InfoRegister{ir}, Reports: []*reportpkg.Report{rep}})
	w := httptest.NewRecorder()
	r := reqWithChi("GET", "/ui/report/KindPrivacy?__run=1&Value=synthetic", nil, map[string]string{"name": rep.Name})
	r = r.WithContext(auth.ContextWithUser(r.Context(), &auth.User{IsAdmin: true}))
	s.reportForm(w, r)
	if w.Code != 200 || !strings.Contains(w.Body.String(), "защищённое поле") {
		t.Fatalf("mandatory source-kind gate missing: status %d", w.Code)
	}
}
