package ui

import (
	"context"
	"net/http"
	"path/filepath"
	"strings"
	"testing"

	"github.com/ivantit66/onebase/internal/dsl/ast"
	reportpkg "github.com/ivantit66/onebase/internal/report"
	"github.com/ivantit66/onebase/internal/runtime"
	"github.com/ivantit66/onebase/internal/storage"
)

// Negative capability observation on exact B30. Passing this diagnostic only
// confirms the current unsupported-row-return response; it does NOT implement
// or accept ENG12. The SpreadsheetDocument controls prove the public route ran.
func TestENG12B30TypedRowsCapabilityProbe(t *testing.T) {
	tableSource := `Функция Build()
  Т=Новый ТаблицаЗначений;
  Т.Колонки.Добавить("Amount");Т.Колонки.Добавить("Moment");Т.Колонки.Добавить("Works");
  С=Т.Добавить();С.Amount=125.5;С.Moment=Дата(2026,11,16);С.Works="Работа А; Работа Б; Заметка";
  Возврат Т;
 КонецФункции`
	arraySource := `Функция Build() Возврат Результат; КонецФункции`
	documentSource := `Функция Build() Д=Новый ТабличныйДокумент;Д.Ячейка(1,1).Текст="CONTROL";Возврат Д;КонецФункции`
	for _, tc := range []struct {
		name, source string
		query        bool
		want         int
	}{
		{"value-table-queryless", tableSource, false, 500}, {"value-table-query", tableSource, true, 500},
		{"row-array-queryless", arraySource, false, 500}, {"row-array-query", arraySource, true, 500},
		{"document-queryless-control", documentSource, false, 303}, {"document-query-control", documentSource, true, 303},
	} {
		t.Run(tc.name, func(t *testing.T) {
			db, err := storage.ConnectSQLite(context.Background(), filepath.Join(t.TempDir(), "eng12-probe.sqlite"))
			if err != nil {
				t.Fatal(err)
			}
			t.Cleanup(db.Close)
			if err := db.Migrate(context.Background(), nil); err != nil {
				t.Fatal(err)
			}
			if err := db.EnsureServiceSchema(context.Background()); err != nil {
				t.Fatal(err)
			}
			s, user, rep := tabularFixture(t, db)
			t.Cleanup(s.Close)
			if tc.query {
				rep.Query = "ВЫБРАТЬ 125.5 КАК Amount"
			}
			s.cfg.Limits.ReportMaxRows = 10
			prog := mustParse(t, tc.source)
			s.reg.Load(runtime.LoadOptions{Reports: []*reportpkg.Report{rep}, Programs: map[string]*ast.Program{rep.Name: prog}})
			w := tabularHTTP(s, user, "eng12-own-probe", http.MethodPost, "/ui/report/"+rep.Name, nil)
			if w.Code != tc.want {
				t.Fatalf("unexpected capability response HTTP%d: %s", w.Code, w.Body.String())
			}
			if tc.want == 500 {
				if !strings.Contains(w.Body.String(), "процедура должна возвращать ТабличныйДокумент") {
					t.Fatalf("negative result was caused by another error: %s", w.Body.String())
				}
				t.Logf("B30_CAPABILITY=UNSUPPORTED_ROW_RETURN; HTTP%d; native type gate=%s", w.Code, strings.TrimSpace(w.Body.String()))
			} else {
				if !strings.HasPrefix(w.Header().Get("Location"), "/ui/tabular/") {
					t.Fatal("document control was not published")
				}
				t.Log("B30_CAPABILITY=SPREADSHEET_DOCUMENT_SUPPORTED; HTTP303")
			}
		})
	}
}
