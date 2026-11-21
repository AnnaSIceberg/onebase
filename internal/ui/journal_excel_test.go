package ui

import (
	"bytes"
	"net/http"
	"net/http/httptest"
	"net/url"
	"os"
	"path/filepath"
	"reflect"
	"strings"
	"testing"
	"time"

	"github.com/google/uuid"
	"github.com/ivantit66/onebase/internal/auth"
	"github.com/ivantit66/onebase/internal/configcheck"
	"github.com/ivantit66/onebase/internal/dbtest"
	"github.com/ivantit66/onebase/internal/metadata"
	"github.com/ivantit66/onebase/internal/project"
	"github.com/ivantit66/onebase/internal/storage"
	"github.com/xuri/excelize/v2"
)

// Выгрузка журнала в Excel объявляла колонки «Дата» и «Вид» и заполняла их из
// ключей `date`/`doc_type`, которых в строках журнала нет: JournalQuery отдаёт
// `_doc_kind`, `id` и поля колонок журнала. Пользователь получал файл с двумя
// мёртвыми колонками (#886).
//
// Проверка идёт через публичный вход `/ui/journal/{name}/excel` и читает
// получившуюся книгу: сверять внутренние ключи между собой бессмысленно —
// именно расхождение ключа с фактической строкой и было дефектом.
func TestJournalExcel_ЛевыеКолонкиНеПустые(t *testing.T) {
	doc := &metadata.Entity{
		Name: "РасходныйОрдер",
		Kind: metadata.KindDocument,
		Fields: []metadata.Field{
			{Name: "Дата", Type: metadata.FieldTypeDate},
			{Name: "Номер", Type: metadata.FieldTypeString},
			{Name: "Сумма", Type: metadata.FieldTypeNumber},
		},
	}
	j := &metadata.Journal{
		Name:      "ЖурналОрдеров",
		Documents: []string{doc.Name},
		Columns: []metadata.JournalColumn{
			{Field: "Дата", Label: "Дата"},
			// Пустой label должен использовать имя поля, как и таблица на экране,
			// а не создавать ещё одну пустую шапку в Excel.
			{Field: "Номер"},
			{Field: "Сумма", Label: "Сумма"},
		},
	}

	s, ctx := newSubmitTestServer(t, []*metadata.Entity{doc})
	s.reg.LoadJournals([]*metadata.Journal{j})
	if err := s.store.Upsert(ctx, doc.Name, uuid.New(), map[string]any{
		"Дата":  time.Date(2026, 8, 14, 0, 0, 0, 0, time.UTC),
		"Номер": "РО-000042",
		"Сумма": 150.5,
	}, doc); err != nil {
		t.Fatal(err)
	}

	req := reqWithChi(http.MethodGet, "/ui/journal/"+j.Name+"/excel", nil, map[string]string{"name": j.Name})
	w := httptest.NewRecorder()
	s.journalExcel(w, req)
	if w.Code != http.StatusOK {
		t.Fatalf("journalExcel: %d %s", w.Code, w.Body.String())
	}

	book, err := excelize.OpenReader(bytes.NewReader(w.Body.Bytes()))
	if err != nil {
		t.Fatal(err)
	}
	defer func() {
		if err := book.Close(); err != nil {
			t.Errorf("закрытие книги: %v", err)
		}
	}()
	sheets := book.GetSheetList()
	if len(sheets) == 0 {
		t.Fatal("в книге нет листов")
	}
	rows, err := book.GetRows(sheets[0])
	if err != nil {
		t.Fatal(err)
	}
	if len(rows) < 2 {
		t.Fatalf("в книге нет строк данных: %#v", rows)
	}

	// Шапка повторяет таблицу на экране: вид документа + видимые колонки.
	wantHeader := []string{"Документ", "Дата", "Номер", "Сумма"}
	if len(rows[0]) != len(wantHeader) {
		t.Fatalf("шапка %#v, ожидалась %#v", rows[0], wantHeader)
	}
	for i, want := range wantHeader {
		if rows[0][i] != want {
			t.Errorf("колонка %d: %q, ожидалось %q", i, rows[0][i], want)
		}
	}

	// Главное: ни одна объявленная колонка не приезжает пустой.
	data := rows[1]
	for i, head := range wantHeader {
		if i >= len(data) || data[i] == "" {
			t.Errorf("колонка «%s» пуста — объявлена в шапке, но не заполнена", head)
		}
	}
	if len(data) > 0 && data[0] != doc.Name {
		t.Errorf("вид документа %q, ожидался %q", data[0], doc.Name)
	}
	if len(data) > 2 && data[2] != "РО-000042" {
		t.Errorf("номер %q, ожидался «РО-000042»", data[2])
	}
}

func TestJournalDocumentKindColumnHTTP(t *testing.T) {
	path := filepath.Join(t.TempDir(), "fines.yaml")
	fixture := `name: Fines
documents: [FineOrder, RestrictedOrder]
columns:
  - {field: Дата, label: Дата}
  - {field: ФилиалАйсберг, label: Филиал}
  - {field: _doc_kind, label: Вид документа}
  - {field: Номер, label: Номер}
filters:
  - {field: Номер, type: string}
`
	if err := os.WriteFile(path, []byte(fixture), 0o600); err != nil {
		t.Fatal(err)
	}
	j, err := metadata.LoadJournalFile(path)
	if err != nil {
		t.Fatal(err)
	}
	fields := []metadata.Field{
		{Name: "Дата", Type: metadata.FieldTypeDate},
		{Name: "ФилиалАйсберг", Type: metadata.FieldTypeString},
		{Name: "Номер", Type: metadata.FieldTypeString},
	}
	doc := &metadata.Entity{Name: "FineOrder", Title: "Штраф мастеру", Kind: metadata.KindDocument, Fields: fields}
	restricted := &metadata.Entity{Name: "RestrictedOrder", Title: "Закрытый вид", Kind: metadata.KindDocument, Fields: fields}
	entities := []*metadata.Entity{doc, restricted}
	proj := &project.Project{Entities: entities, Journals: []*metadata.Journal{j}}
	if issues := configcheck.CheckCrossRefs(proj, nil); len(issues) != 0 {
		t.Fatalf("journal column check: %+v", issues)
	}
	dbtest.ForEachDialect(t, func(t *testing.T, db *storage.DB) {
		s := newSubmitTestServerOn(t, db, entities)
		s.reg.LoadJournals([]*metadata.Journal{j})
		id := uuid.New()
		insert := func(entity *metadata.Entity, recordID uuid.UUID, branch, number string) {
			t.Helper()
			if err := db.Upsert(t.Context(), entity.Name, recordID, map[string]any{
				"Дата":          time.Date(2026, 11, 21, 0, 0, 0, 0, time.UTC),
				"ФилиалАйсберг": branch,
				"Номер":         number,
			}, entity); err != nil {
				t.Fatal(err)
			}
		}
		insert(doc, id, "SampleBranch", "MatchFine")
		insert(doc, uuid.New(), "SampleBranch", "OtherNumber")
		insert(doc, uuid.New(), "OtherBranch", "MatchOtherBranch")
		insert(restricted, uuid.New(), "SampleBranch", "MatchRestricted")
		role := &auth.Role{Name: "Reader", Permissions: auth.Permission{
			Documents: map[string][]string{doc.Name: {"read"}},
			RowAccess: auth.RowAccess{Documents: map[string]auth.RowPolicies{
				doc.Name: {"read": {
					Field: "ФилиалАйсберг", Op: "eq", Value: auth.RowValue{Literal: "SampleBranch", LiteralSet: true},
				}},
			}},
		}}
		user := &auth.User{Login: "journal-reader", Roles: []*auth.Role{role}}
		request := func(handler http.HandlerFunc, method, path string, form url.Values, status int) *httptest.ResponseRecorder {
			t.Helper()
			req := reqWithChi(method, path, form, map[string]string{"name": j.Name})
			rec := httptest.NewRecorder()
			handler.ServeHTTP(rec, req.WithContext(auth.ContextWithUser(req.Context(), user)))
			if rec.Code != status {
				t.Fatalf("%s %s: %d %s", method, path, rec.Code, rec.Body.String())
			}
			return rec
		}
		check := func(headers, values []string) {
			t.Helper()
			query := "?" + url.Values{"f.Номер": {"Match"}}.Encode()
			body := request(s.journalList, http.MethodGet, "/ui/journal/fines"+query, nil, http.StatusOK).Body.String()
			headAt := strings.Index(body, "<table><thead><tr>")
			if headAt < 0 {
				t.Fatal("journal data table is absent")
			}
			dataTable := body[headAt:]
			gotHead, _, _ := strings.Cut(dataTable, "</tr>")
			if strings.Count(gotHead, "<th>") != len(headers) {
				t.Fatalf("unexpected journal headings: %s", gotHead)
			}
			for _, column := range headers {
				var found bool
				_, gotHead, found = strings.Cut(gotHead, "<th>"+column+"</th>")
				if !found {
					t.Fatalf("headings %v missing or out of order near %q", headers, gotHead)
				}
			}
			if strings.Count(dataTable, "data-ob-list-row") != 1 ||
				!strings.Contains(dataTable, "data-open-url=\"/ui/document/fineorder/"+id.String()+"\"") ||
				strings.Contains(dataTable, "MatchRestricted") || strings.Contains(dataTable, "MatchOtherBranch") ||
				strings.Contains(dataTable, "OtherNumber") {
				t.Fatal("journal list changed navigation, ACL or filter behavior")
			}
			for _, value := range values {
				if !strings.Contains(dataTable, ">"+value+"</td>") {
					t.Fatalf("missing journal cell %q", value)
				}
			}
			resp := request(s.journalExcel, http.MethodGet, "/ui/journal/fines/excel"+query, nil, http.StatusOK)
			book, err := excelize.OpenReader(bytes.NewReader(resp.Body.Bytes()))
			if err != nil {
				t.Fatal(err)
			}
			defer book.Close()
			rows, err := book.GetRows(book.GetSheetName(0))
			if err != nil || len(rows) != 2 || !reflect.DeepEqual(rows[0], headers) ||
				len(rows[1]) != len(headers) || !reflect.DeepEqual(rows[1][1:], values) {
				t.Fatalf("journal XLSX = %#v, err=%v", rows, err)
			}
		}
		headers := []string{"Дата", "Филиал", "Вид документа", "Номер"}
		check(headers, []string{"SampleBranch", "Штраф мастеру", "MatchFine"})
		request(s.journalSettingsSave, http.MethodPost, "/ui/journal/fines/settings/save", url.Values{
			"__journal_settings": {`{"columns":[{"field":"_doc_kind","visible":false}]}`},
		}, http.StatusSeeOther)
		check([]string{"Дата", "Филиал", "Номер"}, []string{"SampleBranch", "MatchFine"})
		request(s.journalSettingsReset, http.MethodPost, "/ui/journal/fines/settings/reset", nil, http.StatusSeeOther)
		check(headers, []string{"SampleBranch", "Штраф мастеру", "MatchFine"})
	})
}
