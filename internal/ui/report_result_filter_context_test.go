package ui

import (
	"fmt"
	"net/http"
	"net/url"
	"strings"
	"testing"

	"github.com/google/uuid"
	"github.com/ivantit66/onebase/internal/auth"
	"github.com/ivantit66/onebase/internal/dsl/ast"
	"github.com/ivantit66/onebase/internal/metadata"
	reportpkg "github.com/ivantit66/onebase/internal/report"
	"github.com/ivantit66/onebase/internal/runtime"
)

func TestReportResultFilterResultProcMultipleQueriesHTTP(t *testing.T) {
	for _, tc := range []struct {
		name, prefix, source string
		denied               bool
	}{
		{name: "two-sql-aliases", source: "FilterRow"},
		{name: "readonly-filter", prefix: `Попытка ЭффективныеОтборы.Получить(0).Значение="DROP"; Исключение КонецПопытки;`, source: "FilterRow"},
		{name: "alias-does-not-prove-source", source: "OtherRow", denied: true},
		{name: "untraced-manager", prefix: `X=Справочники.FilterRow.Выбрать();`, source: "FilterRow", denied: true},
	} {
		t.Run(tc.name, func(t *testing.T) {
			entity := &metadata.Entity{Name: "FilterRow", Kind: metadata.KindCatalog, Fields: []metadata.Field{{Name: "Bucket", Type: "string"}}}
			other := &metadata.Entity{Name: "OtherRow", Kind: metadata.KindCatalog, Fields: []metadata.Field{{Name: "Bucket", Type: "string"}}}
			s, ctx := newSubmitTestServer(t, []*metadata.Entity{entity, other})
			t.Cleanup(s.Close)
			if err := s.store.EnsureServiceSchema(ctx); err != nil {
				t.Fatal(err)
			}
			for _, meta := range []*metadata.Entity{entity, other} {
				for _, bucket := range []string{"DROP", "KEEP"} {
					if err := s.store.Upsert(ctx, meta.Name, uuid.New(), map[string]any{"Bucket": bucket}, meta); err != nil {
						t.Fatal(err)
					}
				}
			}
			rep := &reportpkg.Report{Name: "ResultFilterContext", Query: `ВЫБРАТЬ Bucket КАК bucket ИЗ Справочник.FilterRow`, ResultProc: "Build"}
			program := mustParse(t, fmt.Sprintf(`
Функция Build()
 %s
 Q=Новый Запрос; Q.Текст="ВЫБРАТЬ Bucket КАК first ИЗ Справочник.%s";
 A=Q.Выполнить();
 Q=Новый Запрос; Q.Текст="ВЫБРАТЬ Bucket КАК second ИЗ Справочник.FilterRow";
 B=Q.Выполнить();
 D=Новый ТабличныйДокумент;
 D.Ячейка(1,1).Текст="COUNTS:"+Строка(Результат.Количество())+":"+Строка(A.Количество())+":"+Строка(B.Количество());
 D.Ячейка(2,1).Текст=A[0].first+":"+B[0].second+":"+ЭффективныеОтборы.Получить(0).Значение;
 Возврат D;
КонецФункции`, tc.prefix, tc.source))
			s.reg.Load(runtime.LoadOptions{Entities: []*metadata.Entity{entity, other}, Reports: []*reportpkg.Report{rep}, Programs: map[string]*ast.Program{rep.Name: program}})
			s.interp.LookupProc = s.reg.GetModuleProc
			s.cfg.Limits.ReportMaxRows = 1
			form := url.Values{"__settings": {`{"filters":[{"field":"bucket","op":"eq","value":"KEEP"}]}`}}
			response := tabularHTTP(s, &auth.User{ID: uuid.NewString(), Login: "filter-runner", IsAdmin: true}, "filter-context-session", http.MethodPost, "/ui/report/"+rep.Name, form)
			if tc.denied {
				if response.Code == http.StatusSeeOther || response.Header().Get("Location") != "" || len(s.tabularResults.items) != 0 {
					t.Fatal("unproven source published", response.Code, response.Body.String())
				}
				return
			}
			if response.Code != http.StatusSeeOther {
				t.Fatalf("HTTP %d: %s", response.Code, response.Body.String())
			}
			result := s.tabularResults.items[strings.TrimPrefix(response.Header().Get("Location"), "/ui/tabular/")]
			if result == nil || result.doc.GetCell(0, 0).Text != "COUNTS:1:1:1" || result.doc.GetCell(1, 0).Text != "KEEP:KEEP:KEEP" {
				t.Fatal("effective filters did not cover all successful SQL inputs")
			}
		})
	}
}
