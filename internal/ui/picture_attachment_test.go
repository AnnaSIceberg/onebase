package ui

import (
	"context"
	"net/http"
	"net/http/httptest"
	"strings"
	"testing"

	"github.com/go-chi/chi/v5"
	"github.com/google/uuid"

	"github.com/ivantit66/onebase/internal/metadata"
)

// ПолеКартинки с data_path показывает вложение, идентификатор которого лежит в
// реквизите (фото мастера в заявке), и открывает его в полном размере по
// щелчку — как «Поле картинки … Нажатие» в 1С. Публичный путь — GET формы
// объекта. Значение, не являющееся UUID, в src/href не попадает.
func TestManagedFormPictureFieldShowsAttachment(t *testing.T) {
	srv, ent := setupManagedEventsServer(t, ``, nil, []*metadata.FormElement{
		{Kind: metadata.FormElementPicture, Name: "ПолеФото", DataPath: "Объект.Наименование", Width: 240, Height: 180},
	})
	open := func(t *testing.T, value string) string {
		t.Helper()
		id := uuid.New()
		if err := srv.store.Upsert(context.Background(), ent.Name, id,
			map[string]any{"Наименование": value}, ent); err != nil {
			t.Fatal(err)
		}
		req := httptest.NewRequest(http.MethodGet, "/ui/catalog/"+ent.Name+"/"+id.String(), nil)
		rctx := chi.NewRouteContext()
		rctx.URLParams.Add("kind", "catalog")
		rctx.URLParams.Add("entity", ent.Name)
		rctx.URLParams.Add("id", id.String())
		req = req.WithContext(context.WithValue(req.Context(), chi.RouteCtxKey, rctx))
		rec := httptest.NewRecorder()
		srv.formEdit(rec, req)
		if rec.Code != http.StatusOK {
			t.Fatalf("GET формы: %d %s", rec.Code, rec.Body.String())
		}
		return rec.Body.String()
	}

	t.Run("идентификатор вложения — картинка и ссылка", func(t *testing.T) {
		aid := uuid.New().String()
		body := open(t, strings.ToUpper(aid))
		url := "/ui/attachments/" + aid + "/download"
		for _, want := range []string{
			`<img src="` + url + `"`,
			`data-ob-picture-link="ПолеФото" href="` + url + `" target="_blank" rel="noopener"`,
			`max-width:240px;max-height:180px`,
			`name="Наименование" value="` + aid + `" data-ob-picture-for="ПолеФото"`,
			`data-ob-picture-empty="ПолеФото" style="color:#94a3b8" hidden>`,
		} {
			if !strings.Contains(body, want) {
				t.Errorf("в разметке нет %s", want)
			}
		}
	})

	t.Run("не идентификатор — пусто, адрес не подставлен", func(t *testing.T) {
		body := open(t, "javascript:alert(1)")
		// Наименование записи есть и в заголовке формы — смотрим только блок картинки.
		start := strings.Index(body, `data-ob-picture-for="ПолеФото"`)
		end := strings.Index(body, `data-ob-picture-empty="ПолеФото"`)
		if start < 0 || end < start {
			t.Fatalf("нет блока картинки")
		}
		if strings.Contains(body[start:end], "javascript:alert") || strings.Contains(body[start-80:start], "javascript:alert") {
			t.Fatalf("значение реквизита попало в разметку картинки")
		}
		for _, want := range []string{
			`data-ob-picture-link="ПолеФото" href="" target="_blank" rel="noopener" title="Открыть в полном размере" hidden>`,
			`data-ob-picture-empty="ПолеФото" style="color:#94a3b8">Нет изображения`,
		} {
			if !strings.Contains(body, want) {
				t.Errorf("в разметке нет %s", want)
			}
		}
	})
}
