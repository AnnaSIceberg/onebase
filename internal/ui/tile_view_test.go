package ui

import (
	"net/http"
	"net/http/httptest"
	"strings"
	"testing"

	"github.com/go-chi/chi/v5"
	"github.com/google/uuid"
	"github.com/ivantit66/onebase/internal/auth"

	"github.com/ivantit66/onebase/internal/metadata"
	"github.com/ivantit66/onebase/internal/storage"
)

func TestPageList_TileViewUsesConfiguredFields(t *testing.T) {
	ent := &metadata.Entity{
		Name: "Товар",
		Kind: metadata.KindCatalog,
		Fields: []metadata.Field{
			{Name: "Наименование", Type: metadata.FieldTypeString},
			{Name: "Артикул", Type: metadata.FieldTypeString},
			{Name: "Фото", Type: metadata.FieldTypeImage},
			{Name: "Цена", Type: metadata.FieldTypeNumber},
			{Name: "Остаток", Type: metadata.FieldTypeNumber},
			{Name: "Скрыто", Type: metadata.FieldTypeString},
		},
		TileView: &metadata.TileView{
			Image:    "Фото",
			Title:    "Артикул",
			Subtitle: "Наименование",
			Fields:   []string{"Цена", "Остаток"},
		},
	}
	html := renderPageList(t, map[string]any{
		"Entity":           ent,
		"Rows":             []map[string]any{{"id": "1", "Наименование": "Кофе", "Артикул": "A-1", "Фото": "pic-1", "Цена": 0, "Остаток": 12, "Скрыто": "secret"}},
		"Params":           storage.ListParams{},
		"RefFilterOptions": map[string]any{},
		"Lang":             "ru",
		"TilesView":        true,
		"Total":            1, "Page": 1, "TotalPages": 1,
	})

	for _, want := range []string{
		`background-image:url('/ui/_image/pic-1')`,
		`class="tile-title">A-1`,
		`class="tile-subtitle">Кофе`,
		`<span class="tile-label">Цена:</span>`,
		`<span class="tile-val">0</span>`,
		`<span class="tile-label">Остаток:</span>`,
		`<span class="tile-val">12</span>`,
	} {
		if !strings.Contains(html, want) {
			t.Errorf("плитка не содержит %q", want)
		}
	}
	visibleTile := html
	for _, unwanted := range []string{"Скрыто:", "secret"} {
		if strings.Contains(visibleTile, unwanted) {
			t.Errorf("плитка содержит скрытое поле %q", unwanted)
		}
	}
}

func TestResolveTileViewExplicitEmptyFields(t *testing.T) {
	ent := &metadata.Entity{
		Name: "Товар",
		Kind: metadata.KindCatalog,
		Fields: []metadata.Field{
			{Name: "Наименование", Type: metadata.FieldTypeString},
			{Name: "Цена", Type: metadata.FieldTypeNumber},
		},
		TileView: &metadata.TileView{
			Title:     "Наименование",
			FieldsSet: true,
		},
	}
	view := resolveTileView(ent)
	if view.TitleField == nil || view.TitleField.Name != "Наименование" {
		t.Fatalf("TitleField = %+v, ожидалось Наименование", view.TitleField)
	}
	if len(view.Fields) != 0 {
		t.Fatalf("явный пустой fields не должен подставлять автополя, got %+v", view.Fields)
	}
}

// resolveListColumns: без tile_view — все поля; с набором — Заголовок,
// Подзаголовок и выбранные поля (без картинки и невыбранных) (#216).
func TestResolveListColumns(t *testing.T) {
	ent := &metadata.Entity{
		Name: "Товар",
		Kind: metadata.KindCatalog,
		Fields: []metadata.Field{
			{Name: "Наименование", Type: metadata.FieldTypeString},
			{Name: "Артикул", Type: metadata.FieldTypeString},
			{Name: "Фото", Type: metadata.FieldTypeImage},
			{Name: "Цена", Type: metadata.FieldTypeNumber},
			{Name: "Остаток", Type: metadata.FieldTypeNumber},
			{Name: "Скрыто", Type: metadata.FieldTypeString},
		},
	}
	if got := resolveListColumns(ent); len(got) != len(ent.Fields) {
		t.Fatalf("без конфигурации ожидались все поля (%d), got %d", len(ent.Fields), len(got))
	}

	ent.TileView = &metadata.TileView{
		Image:    "Фото",
		Title:    "Артикул",
		Subtitle: "Наименование",
		Fields:   []string{"Цена", "Остаток"},
	}
	var names []string
	for _, f := range resolveListColumns(ent) {
		names = append(names, f.Name)
	}
	want := "Артикул,Наименование,Цена,Остаток"
	if strings.Join(names, ",") != want {
		t.Fatalf("колонки списка = %v, ожидалось %q", names, want)
	}
}

func TestResolveListColumnsEntityListForm(t *testing.T) {
	ent := &metadata.Entity{
		Name: "Показания",
		Kind: metadata.KindCatalog,
		Fields: []metadata.Field{
			{Name: "Наименование", Type: metadata.FieldTypeString},
			{Name: "ДатаНачала", Type: metadata.FieldTypeDate},
			{Name: "ДатаОкончания", Type: metadata.FieldTypeDate},
			{Name: "ПоказанияНачала", Type: metadata.FieldTypeNumber},
		},
		ListForm: []string{"ДатаНачала", "Наименование"},
		TileView: &metadata.TileView{Title: "ПоказанияНачала"},
	}

	var names []string
	for _, field := range resolveListColumns(ent) {
		names = append(names, field.Name)
	}
	if got, want := strings.Join(names, ","), "ДатаНачала,Наименование"; got != want {
		t.Fatalf("колонки list_form = %q, ожидалось %q", got, want)
	}
}

func TestResolveListColumnsManagedListForm(t *testing.T) {
	form := &metadata.FormModule{
		Name:       "ФормаСписка",
		Kind:       "list",
		LayoutKind: metadata.FormLayoutManaged,
		Elements: []*metadata.FormElement{
			{
				Kind:     metadata.FormElementTable,
				Name:     "Список",
				DataPath: "Список",
				Children: []*metadata.FormElement{
					{Kind: metadata.FormElementColumn, Name: "КолПоказания", DataPath: "Список.ПоказанияНачала", TitleMap: map[string]string{"ru": "Показание формы"}},
					{Kind: metadata.FormElementColumn, Name: "КолНаименование", DataPath: "Список.Наименование"},
				},
			},
		},
	}
	ent := &metadata.Entity{
		Name: "Показания",
		Kind: metadata.KindCatalog,
		Fields: []metadata.Field{
			{Name: "Наименование", Type: metadata.FieldTypeString},
			{Name: "ДатаНачала", Type: metadata.FieldTypeDate},
			{Name: "ПоказанияНачала", Type: metadata.FieldTypeNumber},
		},
		ListForm: []string{"ДатаНачала"},
		Forms:    []*metadata.FormModule{form},
	}

	cols := resolveListColumns(ent)
	var names []string
	for _, field := range cols {
		names = append(names, field.Name)
	}
	if got, want := strings.Join(names, ","), "ПоказанияНачала,Наименование"; got != want {
		t.Fatalf("колонки управляемой формы списка = %q, ожидалось %q", got, want)
	}
	if got := cols[0].DisplayName("ru"); got != "Показание формы" {
		t.Fatalf("заголовок колонки управляемой формы списка = %q", got)
	}
}

func TestPageList_ManagedListFormUsesElementColumnTitle(t *testing.T) {
	form := &metadata.FormModule{
		Name: "ФормаСписка", Kind: "list", LayoutKind: metadata.FormLayoutManaged,
		Elements: []*metadata.FormElement{{
			Kind: metadata.FormElementTable, Name: "Список", DataPath: "Список",
			Children: []*metadata.FormElement{{
				Kind: metadata.FormElementColumn, Name: "КолУровень",
				DataPath: "Список.Уровень", TitleMap: map[string]string{"ru": "ТУН"},
			}},
		}},
	}
	ent := &metadata.Entity{
		Name: "Регион", Kind: metadata.KindCatalog,
		Fields: []metadata.Field{{
			Name: "Уровень", Type: metadata.FieldTypeString, Title: "Заголовок реквизита",
		}},
		Forms: []*metadata.FormModule{form},
	}
	s, ctx := newSubmitTestServer(t, []*metadata.Entity{ent})
	if err := s.store.Upsert(ctx, ent.Name, uuid.New(), map[string]any{"Уровень": "ВН"}, ent); err != nil {
		t.Fatal(err)
	}

	req := reqWithChi(http.MethodGet, "/ui/catalog/регион", nil, map[string]string{"entity": "регион"})
	rec := httptest.NewRecorder()
	s.list(rec, req)
	if rec.Code != http.StatusOK {
		t.Fatalf("GET списка: code = %d, body: %s", rec.Code, rec.Body.String())
	}
	body := rec.Body.String()
	headerStart := strings.Index(body, "data-ob-list-table") // шапка самой таблицы списка, не окна настройки
	if headerStart < 0 {
		t.Fatalf("страница списка не содержит заголовок таблицы: %s", body)
	}
	headerEnd := strings.Index(body[headerStart:], "</thead>")
	if headerEnd < 0 {
		t.Fatalf("заголовок таблицы списка не закрыт: %s", body[headerStart:])
	}
	header := body[headerStart : headerStart+headerEnd]
	if !strings.Contains(header, "ТУН") {
		t.Fatalf("страница списка не использовала title элемента колонки: %s", header)
	}
	if strings.Contains(header, "Заголовок реквизита") {
		t.Fatal("страница списка оставила заголовок реквизита старше title элемента")
	}
}

func TestPageList_HonorsEntityListForm(t *testing.T) {
	ent := &metadata.Entity{
		Name: "Показания",
		Kind: metadata.KindCatalog,
		Fields: []metadata.Field{
			{Name: "Наименование", Type: metadata.FieldTypeString},
			{Name: "ДатаНачала", Type: metadata.FieldTypeDate},
			{Name: "ДатаОкончания", Type: metadata.FieldTypeDate},
			{Name: "ПоказанияНачала", Type: metadata.FieldTypeNumber},
		},
		ListForm: []string{"Наименование", "ДатаНачала"},
	}
	html := renderPageList(t, map[string]any{
		"Entity": ent,
		"Rows": []map[string]any{{
			"id": "1", "Наименование": "Счётчик 1", "ДатаНачала": "2026-07-01T12:00:00Z",
			"ДатаОкончания": "2099-12-31T12:00:00Z", "ПоказанияНачала": "987654.321",
		}},
		"Params": storage.ListParams{}, "RefFilterOptions": map[string]any{},
		"Lang": "ru", "Total": 1, "Page": 1, "TotalPages": 1,
	})
	for _, want := range []string{"Счётчик 1", "01.07.2026"} {
		if !strings.Contains(html, want) {
			t.Errorf("список не содержит выбранное значение %q", want)
		}
	}
	cells := html
	for _, unwanted := range []string{"2099", "987654.321"} {
		if strings.Contains(cells, unwanted) {
			t.Errorf("список показал значение невыбранной колонки %q", unwanted)
		}
	}
}

// Табличный режим списка (страницы/лента) теперь уважает tile_view.fields:
// выбранные колонки показываются, невыбранные — нет (#216).
func TestPageList_ListViewHonorsTileFields(t *testing.T) {
	ent := &metadata.Entity{
		Name: "Товар",
		Kind: metadata.KindCatalog,
		Fields: []metadata.Field{
			{Name: "Наименование", Type: metadata.FieldTypeString},
			{Name: "Артикул", Type: metadata.FieldTypeString},
			{Name: "Цена", Type: metadata.FieldTypeNumber},
			{Name: "Скрыто", Type: metadata.FieldTypeString},
		},
		TileView: &metadata.TileView{Title: "Артикул", Fields: []string{"Цена"}},
	}
	html := renderPageList(t, map[string]any{
		"Entity":           ent,
		"Rows":             []map[string]any{{"id": "1", "Наименование": "Кофе", "Артикул": "A-1", "Цена": 100, "Скрыто": "secret"}},
		"Params":           storage.ListParams{},
		"RefFilterOptions": map[string]any{},
		"Lang":             "ru",
		"Total":            1, "Page": 1, "TotalPages": 1,
	})
	for _, want := range []string{"Артикул", "Цена", "A-1"} {
		if !strings.Contains(html, want) {
			t.Errorf("список не содержит выбранную колонку/значение %q", want)
		}
	}
	cells := html
	for _, unwanted := range []string{"Кофе", "secret"} {
		if strings.Contains(cells, unwanted) {
			t.Errorf("список показал значение невыбранной колонки: %q", unwanted)
		}
	}
}

func TestPageList_TreeViewKeepsToggleWhenNameHiddenByTileFields(t *testing.T) {
	ent := &metadata.Entity{
		Name:         "Товар",
		Kind:         metadata.KindCatalog,
		Hierarchical: true,
		Fields: []metadata.Field{
			{Name: "Наименование", Type: metadata.FieldTypeString},
			{Name: "Артикул", Type: metadata.FieldTypeString},
			{Name: "Цена", Type: metadata.FieldTypeNumber},
			{Name: "Скрыто", Type: metadata.FieldTypeString},
		},
		TileView: &metadata.TileView{Title: "Артикул", Fields: []string{"Цена"}},
	}
	html := renderPageList(t, map[string]any{
		"Entity":           ent,
		"TreeView":         true,
		"TreeRows":         []map[string]any{{"id": "1", "is_folder": true, "_depth": 0, "Наименование": "Кофе", "Артикул": "A-1", "Цена": 100, "Скрыто": "secret"}},
		"Params":           storage.ListParams{},
		"RefFilterOptions": map[string]any{},
		"Lang":             "ru",
		"Total":            1, "Page": 1, "TotalPages": 1,
	})
	for _, want := range []string{`class="tree-toggle"`, "Артикул", "Цена", "A-1", "100"} {
		if !strings.Contains(html, want) {
			t.Errorf("дерево не содержит %q", want)
		}
	}
	visibleTree := html
	for _, unwanted := range []string{"Кофе", "secret"} {
		if strings.Contains(visibleTree, unwanted) {
			t.Errorf("дерево показало значение невыбранной колонки: %q", unwanted)
		}
	}
}

func TestListCardsShowOnlyFilledAndDisclosedFieldsHTTP(t *testing.T) {
	entity := &metadata.Entity{
		Name: "Карточка", Kind: metadata.KindCatalog,
		Fields: []metadata.Field{
			{Name: "Наименование", Type: metadata.FieldTypeString},
			{Name: "АдресКвартира", Type: metadata.FieldTypeString},
			{Name: "ГородАдресныйКлассификатор", Type: metadata.FieldTypeString, Title: "Город"},
			{Name: "КонтактДляПроверки", Type: metadata.FieldTypeString, PII: true},
			{Name: "ПустаяСтрока", Type: metadata.FieldTypeString},
			{Name: "ЗаглушкаТире", Type: metadata.FieldTypeString},
			{Name: "ЗаглушкаСимвол", Type: metadata.FieldTypeString},
			{Name: "ЛожныйФлажок", Type: metadata.FieldTypeBool},
			{Name: "ЗаполненныйФлажок", Type: metadata.FieldTypeBool, Title: "Подтверждено"},
			{Name: "ПустаяДата", Type: metadata.FieldTypeDate},
		},
		Forms: []*metadata.FormModule{{Name: "ФормаКарточки", Kind: "object", LayoutKind: metadata.FormLayoutManaged,
			Elements: []*metadata.FormElement{{Kind: metadata.FormElementField, Name: "ПолеКвартиры",
				DataPath: "Объект.АдресКвартира", Title: "Квартира"}}}},
	}
	s, ctx := newSubmitTestServer(t, []*metadata.Entity{entity})
	if err := s.store.Upsert(ctx, entity.Name, uuid.New(), map[string]any{
		"Наименование":               "Строка проверки",
		"АдресКвартира":              "Квартира22",
		"ГородАдресныйКлассификатор": "Тестовый город",
		"КонтактДляПроверки":         "Контакт-Синтетика-918",
		"ПустаяСтрока":               "   ",
		"ЗаглушкаТире":               "—",
		"ЗаглушкаСимвол":             "_",
		"ЛожныйФлажок":               false,
		"ЗаполненныйФлажок":          true,
		"ПустаяДата":                 "0001-01-01",
	}, entity); err != nil {
		t.Fatal(err)
	}
	router := chi.NewRouter()
	s.Mount(router)
	user := &auth.User{Login: "reader", Roles: []*auth.Role{{Name: "Reader", Permissions: auth.Permission{
		Catalogs: map[string][]string{entity.Name: {"read"}},
	}}}}
	body := func(u *auth.User) string {
		req := httptest.NewRequest(http.MethodGet, "/ui/catalog/карточка?view=tiles", nil)
		req = req.WithContext(auth.ContextWithUser(req.Context(), u))
		rec := httptest.NewRecorder()
		router.ServeHTTP(rec, req)
		if rec.Code != http.StatusOK {
			t.Fatalf("status %d: %s", rec.Code, rec.Body.String())
		}
		_, tail, ok := strings.Cut(rec.Body.String(), `class="tile-card`)
		if !ok {
			t.Fatal("missing card")
		}
		card, _, _ := strings.Cut(tail, `class="tile-foot`)
		return card
	}
	card := body(user)
	for _, want := range []string{"Квартира:", "Город:", "Подтверждено:", "Квартира22"} {
		if !strings.Contains(card, want) {
			t.Errorf("missing %q", want)
		}
	}
	for _, unwanted := range []string{"АдресКвартира", "ГородАдресныйКлассификатор", "Контакт", "•••", "Пустая строка:", "Заглушка тире:", "Заглушка символ:", "Ложный флажок:", "Пустая дата:"} {
		if strings.Contains(card, unwanted) {
			t.Errorf("unexpected %q", unwanted)
		}
	}
	if shown := body(&auth.User{Login: "admin", IsAdmin: true}); !strings.Contains(shown, "Контакт-Синтетика-918") {
		t.Error("disclosed field was removed from the privileged card")
	}
}
