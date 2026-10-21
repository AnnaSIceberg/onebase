package ui

import (
	"net/http"
	"net/http/httptest"
	"strings"
	"testing"

	"github.com/ivantit66/onebase/internal/metadata"
)

// Высота набора СтраницыФормы: fit_window — страницы занимают остаток окна,
// equal_height — по самой высокой странице, height — постоянная высота с
// прокруткой страницы внутри. Карточка открыта штатным обработчиком формы.
func TestManagedFormPagesHeightKeys(t *testing.T) {
	page := func(name string) *metadata.FormElement {
		return &metadata.FormElement{Kind: "Страница", Name: name, Children: []*metadata.FormElement{fieldEl("Поле"+name, "Объект.Наименование")}}
	}
	for _, tc := range []struct {
		name string
		set  func(el *metadata.FormElement)
		want string
	}{
		{"fit_window", func(el *metadata.FormElement) { el.FitWindow = true }, `data-ob-fit-window="1"`},
		{"equal_height", func(el *metadata.FormElement) { el.EqualHeight = true }, `data-ob-equal-height="1"`},
		{"height", func(el *metadata.FormElement) { el.Height = 400 }, `class="managed-tabs managed-tabs-fill"`},
	} {
		t.Run(tc.name, func(t *testing.T) {
			pages := &metadata.FormElement{Kind: "СтраницыФормы", Name: "Закладки", Children: []*metadata.FormElement{page("Первая"), page("Вторая")}}
			tc.set(pages)
			ent := layoutTestEntity(pages)
			srv, _ := newSubmitTestServer(t, []*metadata.Entity{ent})
			req := reqWithChi(http.MethodGet, "/ui/catalog/"+ent.Name+"/new", nil, map[string]string{"entity": ent.Name})
			rec := httptest.NewRecorder()
			srv.form(rec, req)
			if rec.Code != http.StatusOK {
				t.Fatalf("form: %d %s", rec.Code, rec.Body.String())
			}
			if !strings.Contains(rec.Body.String(), tc.want) {
				t.Fatalf("нет %q в наборе страниц:\n%s", tc.want, rec.Body.String())
			}
		})
	}
}
