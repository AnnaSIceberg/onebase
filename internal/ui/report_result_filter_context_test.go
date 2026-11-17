package ui

import (
	"fmt"
	"net/http"
	"net/url"
	"strings"
	"testing"
	"time"

	"github.com/google/uuid"
	"github.com/ivantit66/onebase/internal/auth"
	"github.com/ivantit66/onebase/internal/dsl/ast"
	"github.com/ivantit66/onebase/internal/dsl/interpreter"
	"github.com/ivantit66/onebase/internal/metadata"
	reportpkg "github.com/ivantit66/onebase/internal/report"
	"github.com/ivantit66/onebase/internal/runtime"
	"github.com/ivantit66/onebase/internal/storage"
)

func TestReportResultFilterInputQueriesAndUnprovenOutputHTTP(t *testing.T) {
	for _, tc := range []struct {
		name, prefix, source string
		denied               bool
	}{
		{name: "two-sql-aliases", source: "FilterRow"},
		{name: "readonly-filter", prefix: `Попытка ЭффективныеОтборы.Получить(0).Значение="DROP"; Исключение Caught=Истина; КонецПопытки; Если НЕ Caught Тогда ВызватьИсключение "readonly setter did not reject"; КонецЕсли;`, source: "FilterRow"},
		{name: "readonly-uncaught", prefix: `ЭффективныеОтборы.Получить(0).Значение="DROP";`, source: "FilterRow", denied: true},
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
			actor := &auth.User{ID: uuid.NewString(), Login: "filter-runner", IsAdmin: true}
			response := tabularHTTP(s, actor, "filter-context-session", http.MethodPost, "/ui/report/"+rep.Name, form)
			if response.Code != http.StatusUnprocessableEntity || len(s.tabularResults.items) != 0 || !strings.Contains(response.Body.String(), errReportFilterProducerUnproven.Error()) {
				t.Fatal("unproven module output admitted", response.Code, response.Body.String())
			}

			// Exercise only the input capabilities, without publishing a snapshot
			// or claiming that a successful DSL run establishes output lineage.
			ctx = auth.ContextWithUser(ctx, actor)
			compiled, err := s.compileReportVariantQuery(ctx, rep, "", nil)
			if err != nil {
				t.Fatal(err)
			}
			settings := &reportpkg.UserReportSettings{Filters: []reportpkg.Filter{{Field: "bucket", Op: "eq", Value: "KEEP"}}}
			rows, _, truncated, err := s.runReportResultRows(ctx, &compiled, settings, 1)
			if err != nil || truncated {
				t.Fatal("initial typed input", truncated, err)
			}
			ctx = storage.ReadOnlyContext(s.withReportSourceFilters(ctx, compiled, settings))
			tx := interpreter.NewTxState(ctx)
			defer rollbackDSLExecution(tx)
			vars, ok := s.reportFilteredVars(ctx, tx)
			if !ok {
				t.Fatal("missing guarded input profile")
			}
			for name, fn := range interpreter.NewSpreadsheetFunctions() {
				vars[name] = fn
			}
			input := &interpreter.Array{}
			for _, row := range rows {
				input.CallMethod("добавить", []any{interpreter.NewStructFromMap(row)})
			}
			vars["Результат"] = input
			var output any
			err = s.interp.RunSandboxed(s.reg.GetProcedure(rep.Name, rep.ResultProc), &interpreter.MapThis{}, interpreter.SandboxProfile{Context: ctx, MaxWallClock: time.Second, DenyNet: true, DenyFile: true, DenyExec: true, DenySecrets: true}, &output, vars)
			if tc.denied {
				if err == nil {
					t.Fatal("unproven input capability admitted")
				}
				return
			}
			if err != nil {
				t.Fatal(err)
			}
			result, ok := output.(*interpreter.SpreadsheetDocument)
			if !ok || result == nil || result.Doc.GetCell(0, 0).Text != "COUNTS:1:1:1" || result.Doc.GetCell(1, 0).Text != "KEEP:KEEP:KEEP" {
				t.Fatal("effective filters did not cover all successful SQL inputs")
			}
		})
	}
}
