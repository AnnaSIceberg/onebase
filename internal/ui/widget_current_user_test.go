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
	"github.com/google/uuid"
	"github.com/ivantit66/onebase/internal/auth"
	"github.com/ivantit66/onebase/internal/metadata"
	"github.com/ivantit66/onebase/internal/runtime"
	"github.com/ivantit66/onebase/internal/storage"
	"github.com/ivantit66/onebase/internal/widget"
)

// Виджет-список «мои записи»: &ТекущийПользователь — идентификатор учётки
// смотрящего, &ТекущийЛогин — его логин. Один и тот же виджет на главной у
// двух пользователей показывает каждому своё — проверка через тот же адрес,
// которым браузер обновляет виджет.
func TestWidgetPartial_CurrentUserParams(t *testing.T) {
	ctx := context.Background()
	db, err := storage.ConnectSQLite(ctx, filepath.Join(t.TempDir(), "widget-user.db"))
	if err != nil {
		t.Fatalf("ConnectSQLite: %v", err)
	}
	t.Cleanup(func() { db.Close() })
	задача := &metadata.Entity{
		Name: "Задача", Kind: metadata.KindCatalog,
		Fields: []metadata.Field{
			{Name: "Наименование", Type: metadata.FieldTypeString},
			{Name: "Исполнитель", Type: metadata.FieldTypeString},
			{Name: "Логин", Type: metadata.FieldTypeString},
		},
	}
	if err := db.Migrate(ctx, []*metadata.Entity{задача}); err != nil {
		t.Fatalf("Migrate: %v", err)
	}
	for _, row := range []map[string]any{
		{"Наименование": "задача первого", "Исполнитель": "u-1", "Логин": "first"},
		{"Наименование": "задача второго", "Исполнитель": "u-2", "Логин": "second"},
	} {
		if err := db.Upsert(ctx, задача.Name, uuid.New(), row, задача); err != nil {
			t.Fatalf("Upsert: %v", err)
		}
	}
	reg := runtime.NewRegistry()
	reg.Load(runtime.LoadOptions{Entities: []*metadata.Entity{задача}})
	reg.LoadWidgets([]*metadata.Widget{
		{Name: "МоиЗадачи", Type: metadata.WidgetTypeList, Title: "Мои задачи",
			Query: "ВЫБРАТЬ Наименование ИЗ Справочник.Задача ГДЕ Исполнитель = &ТекущийПользователь"},
		{Name: "ПоЛогину", Type: metadata.WidgetTypeList, Title: "По логину",
			Query: "ВЫБРАТЬ Наименование ИЗ Справочник.Задача ГДЕ Логин = &ТекущийЛогин"},
	})
	reg.LoadHomePage(&metadata.HomePage{Layout: "rows", Rows: []metadata.HomePageRow{{Widgets: []string{"МоиЗадачи", "ПоЛогину"}}}})
	s := &Server{reg: reg, store: db, widgetCache: widget.NewCache(time.Minute), messages: NewMessageStore()}
	router := chi.NewRouter()
	s.Mount(router)

	html := func(name string, user *auth.User) string {
		t.Helper()
		req := httptest.NewRequest(http.MethodGet, "/ui/_widget/"+url.PathEscape(name), nil)
		req = req.WithContext(auth.ContextWithUser(req.Context(), user))
		rec := httptest.NewRecorder()
		router.ServeHTTP(rec, req)
		if rec.Code != http.StatusOK {
			t.Fatalf("%s: status=%d body=%q", name, rec.Code, rec.Body.String())
		}
		var envelope widgetPartialResponse
		if err := json.Unmarshal(rec.Body.Bytes(), &envelope); err != nil {
			t.Fatalf("decode: %v", err)
		}
		return envelope.HTML
	}
	первый := &auth.User{ID: "u-1", Login: "first", IsAdmin: true}
	второй := &auth.User{ID: "u-2", Login: "second", IsAdmin: true}
	for _, w := range []string{"МоиЗадачи", "ПоЛогину"} {
		if got := html(w, первый); !strings.Contains(got, "задача первого") || strings.Contains(got, "задача второго") {
			t.Fatalf("%s первому пользователю: %q", w, got)
		}
		// Кэш виджета — по пользователю: второй не получает чужой результат.
		if got := html(w, второй); !strings.Contains(got, "задача второго") || strings.Contains(got, "задача первого") {
			t.Fatalf("%s второму пользователю: %q", w, got)
		}
	}
}
