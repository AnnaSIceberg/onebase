package ui

import (
	"bytes"
	"context"
	"log/slog"
	"net/http"
	"net/http/httptest"
	"net/url"
	"strings"
	"testing"

	"github.com/go-chi/chi/v5"
	"github.com/google/uuid"
	"github.com/ivantit66/onebase/internal/incident"
	"github.com/ivantit66/onebase/internal/metadata"
	"github.com/ivantit66/onebase/internal/runtime"
)

// Путь из журнала стенда КЦ: клиент закрыл карточку (iframe удалён при
// открытии формы результата), пока шёл form-event. После обработчика форма
// с title_proc строит объект для заголовка, предзагрузка ссылок получает
// context canceled и паникует вне интерпретатора. Под тем же
// incident.Recoverer, что у сервера, это должно быть тихой отменой: без
// инцидента и без ERROR.
func TestFormEventClientCancelDuringTitleIsNotIncident(t *testing.T) {
	src := `
Процедура ПолеСкладПриИзменении()
	Сообщить("изменено");
КонецПроцедуры

Функция Заголовок()
	Возврат "Карточка";
КонецФункции
`
	s, ent := setupManagedEventsServer(t, src, nil, []*metadata.FormElement{
		{Kind: metadata.FormElementField, Name: "ПолеСклад", DataPath: "Объект.Склад",
			Handlers: map[metadata.FormEventType]string{metadata.FormEventOnChange: "ПолеСкладПриИзменении"}},
	})
	ent.Forms[0].TitleProc = "Заголовок"
	ctx := context.Background()
	sklad := &metadata.Entity{Name: "Склад", Kind: metadata.KindCatalog,
		Fields: []metadata.Field{{Name: "Наименование", Type: metadata.FieldTypeString}}}
	ent.Fields = append(ent.Fields, metadata.Field{Name: "Склад", Type: metadata.FieldType("reference:Склад"), RefEntity: "Склад"})
	if err := s.store.Migrate(ctx, []*metadata.Entity{sklad, ent}); err != nil {
		t.Fatal(err)
	}
	skladID := uuid.New()
	if err := s.store.Upsert(ctx, sklad.Name, skladID, map[string]any{"Наименование": "Основной"}, sklad); err != nil {
		t.Fatal(err)
	}
	reg := runtime.NewRegistry()
	reg.Load(runtime.LoadOptions{Entities: []*metadata.Entity{ent, sklad}})
	s.reg = reg
	s.interp.LookupProc = reg.GetModuleProc
	s.incidents = incident.NewStore(16)

	var logs bytes.Buffer
	prev := slog.Default()
	slog.SetDefault(slog.New(slog.NewTextHandler(&logs, &slog.HandlerOptions{Level: slog.LevelDebug})))
	t.Cleanup(func() { slog.SetDefault(prev) })

	router := chi.NewRouter()
	router.Use(incident.Recoverer(s.Incidents(), nil))
	s.Mount(router)
	post := func(ctx context.Context) *httptest.ResponseRecorder {
		body := url.Values{"_element": {"ПолеСклад"}, "_event": {"ПриИзменении"}, "Склад": {skladID.String()}}
		req := httptest.NewRequest(http.MethodPost, "/ui/catalog/"+url.PathEscape(ent.Name)+"/form-event", strings.NewReader(body.Encode()))
		req.Header.Set("Content-Type", "application/x-www-form-urlencoded; charset=utf-8")
		rec := httptest.NewRecorder()
		router.ServeHTTP(rec, req.WithContext(ctx))
		return rec
	}

	// The connection drops while the handler runs: every DSL expression of
	// the handler passes this guard, so the context is canceled mid-request.
	reqCtx, cancel := context.WithCancel(context.Background())
	defer cancel()
	s.interp.ValidateObjectAccess = func(any) { cancel() }
	rec := post(reqCtx)
	s.interp.ValidateObjectAccess = nil
	if rec.Code == http.StatusInternalServerError {
		t.Fatalf("client cancel answered 500: %s", rec.Body.String())
	}
	if got := s.Incidents().Recent("", 16); len(got) != 0 {
		t.Fatalf("client cancel registered incidents: %+v", got)
	}
	if strings.Contains(logs.String(), "level=ERROR") {
		t.Fatalf("client cancel logged ERROR:\n%s", logs.String())
	}

	if rec := post(context.Background()); rec.Code != http.StatusOK {
		t.Fatalf("live request: %d %s", rec.Code, rec.Body.String())
	}
}
