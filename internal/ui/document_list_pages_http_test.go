package ui

import (
	"context"
	"encoding/json"
	"net/http"
	"net/http/httptest"
	"net/url"
	"path/filepath"
	"strings"
	"testing"
	"time"

	"github.com/go-chi/chi/v5"
	"github.com/ivantit66/onebase/internal/auth"
	"github.com/ivantit66/onebase/internal/dsl/interpreter"
	"github.com/ivantit66/onebase/internal/metadata"
	"github.com/ivantit66/onebase/internal/runtime"
	"github.com/ivantit66/onebase/internal/storage"
)

func TestDocumentListPages_PublicRoute(t *testing.T) {
	ctx := context.Background()
	db, err := storage.ConnectSQLite(ctx, filepath.Join(t.TempDir(), "pages.db"))
	if err != nil {
		t.Fatal(err)
	}
	t.Cleanup(db.Close)
	doc := &metadata.Entity{Name: "Звонок", Kind: metadata.KindDocument, Posting: true, Fields: []metadata.Field{
		{Name: "Перезвонить", Type: metadata.FieldTypeDate},
		{Name: "Дата", Type: metadata.FieldTypeDate},
		{Name: "Номер", Type: metadata.FieldTypeString},
	}}
	catalog := &metadata.Entity{Name: "ТестКаталог", Kind: metadata.KindCatalog, Fields: []metadata.Field{
		{Name: "Наименование", Type: metadata.FieldTypeString},
	}}
	entities := []*metadata.Entity{doc, catalog}
	if err := db.Migrate(ctx, entities); err != nil {
		t.Fatal(err)
	}
	_, err = db.Exec(ctx, `WITH RECURSIVE seq(n) AS (VALUES(1) UNION ALL SELECT n + 1 FROM seq WHERE n < 10075)
INSERT INTO звонок(id, перезвонить, дата, номер)
SELECT printf('00000000-0000-4000-a000-%012x', n),
       datetime('2026-11-01', printf('-%d minutes', n)),
       datetime('2026-11-01', printf('+%d minutes', n)),
       printf('Номер-%05d', n)
FROM seq`)
	if err != nil {
		t.Fatal(err)
	}
	_, err = db.Exec(ctx, `INSERT INTO тесткаталог(id, наименование) VALUES
('00000000-0000-4000-a000-000000000001', 'Яблоко'),
('00000000-0000-4000-a000-000000000002', 'Абрикос')`)
	if err != nil {
		t.Fatal(err)
	}
	reg := runtime.NewRegistry()
	reg.Load(runtime.LoadOptions{Entities: entities})
	s := New(reg, db, interpreter.New(), nil, Config{UIExceptRoles: []string{"ОператорКЦ", "ОператорМессенджера"}}, nil)
	router := chi.NewRouter()
	s.Mount(router)

	get := func(path, role string) string {
		t.Helper()
		r := httptest.NewRequest(http.MethodGet, path, nil)
		if role != "" {
			user := &auth.User{Login: "fixture", Roles: []*auth.Role{{Name: role, Permissions: auth.Permission{
				Documents: map[string][]string{doc.Name: {"read", "write", "post", "unpost"}},
			}}}}
			r = r.WithContext(auth.ContextWithUser(r.Context(), user))
		}
		rec := httptest.NewRecorder()
		start := time.Now()
		router.ServeHTTP(rec, r)
		t.Logf("GET %s (%s): %v", path, role, time.Since(start))
		if rec.Code != http.StatusOK {
			t.Fatalf("GET %s: %d: %s", path, rec.Code, rec.Body.String())
		}
		return rec.Body.String()
	}
	assertRows := func(body string, n int, first, last string) {
		t.Helper()
		count := strings.Count(body, "data-ob-list-row tabindex")
		a, b := strings.Index(body, first), strings.Index(body, last)
		if count != n || a < 0 || b < 0 || a > b {
			t.Fatalf("видимых строк %d, ожидалось %d; %q на %d, %q на %d", count, n, first, a, last, b)
		}
	}
	base := "/ui/document/" + doc.Name
	body := get(base, "")
	assertRows(body, 25, "Номер-10051", "Номер-10075")
	if !strings.Contains(body, `data-ob-select-last`) || !strings.Contains(body, `data-ob-page-indicator>Стр. 202 из 202`) {
		t.Fatal("на последней странице нужен курсор в конце и счётчик страниц")
	}
	for _, command := range []string{"first", "prev"} {
		if !strings.Contains(body, `data-ob-page="`+command+`"`) {
			t.Fatalf("не найдена кнопка перехода %s", command)
		}
	}
	body = get(base+"?page=1", "")
	assertRows(body, 50, "Номер-00001", "Номер-00050")
	if !strings.Contains(body, `data-ob-page="last" href="?page=last"`) || strings.Contains(body, `data-ob-select-last`) {
		t.Fatal("на первой странице нужен переход на последнюю и обычный курсор")
	}
	body = get(base+"?page=last&sort="+doc.Fields[1].Name+"&dir=asc", "")
	assertRows(body, 25, "Номер-10051", "Номер-10075")
	if !strings.Contains(body, `dir=asc`) {
		t.Fatal("переход между страницами потерял сортировку")
	}

	for _, role := range []string{"ОператорКЦ", "ОператорМессенджера"} {
		body := get(base, role)
		assertRows(body, 100, "Номер-00001", "Номер-00100")
		if strings.Contains(body, `data-ob-page-indicator`) || strings.Contains(body, `data-ob-select-last`) || strings.Contains(body, `data-ob-list-command="mark"`) {
			t.Fatalf("вид списка изменился у роли %s", role)
		}
		body = get(base+"/new", role)
		if !strings.Contains(body, `value="post_and_close"`) {
			t.Fatalf("форма оператора потеряла прежнюю кнопку проведения и закрытия: %s", role)
		}
	}
	assertRows(get("/ui/catalog/"+catalog.Name+"?limit=1", ""), 1, "Абрикос", "Абрикос")
	if err := db.SaveListPageSize(ctx, 61); err != nil {
		t.Fatal(err)
	}
	assertRows(get(base+"?page=1", ""), 61, "Номер-00001", "Номер-00061")

	if err := db.SaveListPageSize(ctx, 50); err != nil {
		t.Fatal(err)
	}
	s.cfg.UIEnhancements = true
	body = get(base, "")
	assertRows(body, 25, "Номер-10051", "Номер-10075")
	if !strings.Contains(body, `data-ob-stream="up"`) || !strings.Contains(body, "data-stream-select-last") || strings.Contains(body, "data-ob-page-indicator") {
		t.Fatal("continuous document list must open at the end without pagination")
	}
	body = get(base+"?_list_chunk=1&page=201", "")
	assertRows(body, 50, "Номер-10001", "Номер-10050")
	if strings.Contains(body, "data-stream-select-last") {
		t.Fatal("chunk must not select the last row")
	}
	body = get(base+"?sort="+url.QueryEscape("Номер")+"&dir=desc", "")
	assertRows(body, 50, "Номер-10075", "Номер-10026")
	if !strings.Contains(body, `data-ob-stream="down"`) {
		t.Fatal("custom order must continue in the selected direction")
	}
	body = get(base+"?f."+url.QueryEscape("Номер")+".eq="+url.QueryEscape("Номер-10075"), "")
	assertRows(body, 1, "Номер-10075", "Номер-10075")
	for _, role := range []string{"ОператорКЦ", "ОператорМессенджера"} {
		if strings.Contains(get(base, role), "data-stream-next=") {
			t.Fatalf("stream must not change operator %s", role)
		}
	}

	body = get("/ui/catalog/"+catalog.Name+"?limit=1", "")
	assertRows(body, 1, "Абрикос", "Абрикос")
	if !strings.Contains(body, `data-ob-stream="down"`) || !strings.Contains(body, "page=2") {
		t.Fatal("catalog must open at the beginning and load down")
	}
	assertRows(get("/ui/catalog/"+catalog.Name+"?limit=1&_list_chunk=1&page=2", ""), 1, "Яблоко", "Яблоко")
	reg.LoadJournals([]*metadata.Journal{{Name: "ИсторияЗвонков", Documents: []string{doc.Name}, Columns: []metadata.JournalColumn{{Field: "Дата"}, {Field: "Номер"}}}})
	journal := "/ui/journal/ИсторияЗвонков"
	assertRows(get(journal, ""), 25, "Номер-10051", "Номер-10075")
	assertRows(get(journal+"?_list_chunk=1&offset=10000", ""), 50, "Номер-10001", "Номер-10050")

	s.cfg.UIRoles = []string{"ОТКРуководитель"}
	reg.LoadWidgets([]*metadata.Widget{{
		Name: "Calls", Title: "Заявки", Type: metadata.WidgetTypeList, ScrollFromEnd: true, Limit: 50,
		Query:   "ВЫБРАТЬ Ссылка, Номер, Дата ИЗ Документ.Звонок ГДЕ (&Номер ЕСТЬ ПУСТО ИЛИ Номер = &Номер) УПОРЯДОЧИТЬ ПО Дата УБЫВ, Ссылка УБЫВ",
		Filters: []metadata.WidgetFilter{{Name: "Номер", Param: "Номер", Type: "string"}},
	}})
	widgetBody := func(q string) string {
		t.Helper()
		var result widgetPartialResponse
		if err := json.Unmarshal([]byte(get("/ui/_widget/Calls"+q, "Кассир")), &result); err != nil {
			t.Fatal(err)
		}
		return result.HTML
	}
	if legacy := get("/ui/_widget/Calls", "ОператорМессенджера"); strings.Contains(legacy, "data-stream-widget") {
		t.Fatal("the operator must keep the existing widget behavior")
	}
	body = widgetBody("")
	if !strings.Contains(body, "__older=50") || strings.Index(body, "Номер-10026") >= strings.Index(body, "Номер-10075") ||
		!strings.Contains(body, "Номер-10026") || strings.Contains(body, "Номер-10025") {
		t.Fatal("widget newest batch order")
	}
	body = widgetBody("?__older=50")
	if !strings.Contains(body, "__older=100") || !strings.Contains(body, "Номер-09976") || strings.Contains(body, "Номер-10026") {
		t.Fatal("widget previous batch order")
	}
	body = widgetBody("?" + url.QueryEscape(widgetFilterKey("Calls", "Номер")) + "=" + url.QueryEscape("Номер-00002"))
	if !strings.Contains(body, "Номер-00002") || strings.Contains(body, "__older=") {
		t.Fatal("widget filter must apply before pagination")
	}

}
