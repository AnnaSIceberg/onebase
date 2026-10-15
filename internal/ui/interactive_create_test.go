package ui

import (
	"context"
	"net/http"
	"net/http/httptest"
	"net/url"
	"os"
	"path/filepath"
	"strings"
	"testing"

	"github.com/go-chi/chi/v5"
	"github.com/google/uuid"

	"github.com/ivantit66/onebase/internal/auth"
	"github.com/ivantit66/onebase/internal/metadata"
	"github.com/ivantit66/onebase/internal/storage"
)

// `interactive_create: false` (аналог снятого права «Интерактивное
// добавление» 1С): документ, который рождается только из другого (заявка из
// звонка), нельзя создать из интерфейса — ни кнопкой, ни копированием, ни из
// поля ссылки, ни адресом /new. Сущность грузится из YAML, запросы идут через
// HTTP-обработчики — тем же путём, что у пользователя.

func loadNoCreateDoc(t *testing.T) *metadata.Entity {
	t.Helper()
	p := filepath.Join(t.TempDir(), "Заявка.yaml")
	if err := os.WriteFile(p, []byte(`name: Заявка
interactive_create: false
fields:
  - name: Описание
    type: string
`), 0o600); err != nil {
		t.Fatal(err)
	}
	ent, err := metadata.LoadFile(p, metadata.KindDocument)
	if err != nil {
		t.Fatal(err)
	}
	if !ent.NoInteractiveCreate {
		t.Fatal("interactive_create: false не прочитан из YAML")
	}
	return ent
}

func noCreateRequest(ctx context.Context, method, target string, params map[string]string, body url.Values) *http.Request {
	var req *http.Request
	if body != nil {
		req = httptest.NewRequest(method, target, strings.NewReader(body.Encode()))
		req.Header.Set("Content-Type", "application/x-www-form-urlencoded")
	} else {
		req = httptest.NewRequest(method, target, nil)
	}
	rctx := chi.NewRouteContext()
	for k, v := range params {
		rctx.URLParams.Add(k, v)
	}
	return req.WithContext(context.WithValue(ctx, chi.RouteCtxKey, rctx))
}

func TestInteractiveCreateFalse_HidesCreateAndCopy(t *testing.T) {
	doc := loadNoCreateDoc(t)
	s, ctx := newSubmitTestServer(t, []*metadata.Entity{doc})

	list := func() string {
		t.Helper()
		rec := httptest.NewRecorder()
		s.list(rec, noCreateRequest(ctx, "GET", "/ui/document/заявка", map[string]string{"kind": "document", "entity": "заявка"}, nil))
		if rec.Code != http.StatusOK {
			t.Fatalf("список: code = %d, body: %s", rec.Code, rec.Body.String())
		}
		return rec.Body.String()
	}

	empty := list()
	if strings.Contains(empty, "data-ob-list-create") || strings.Contains(empty, "создать первую") {
		t.Error("пустой список предлагает создание («+ Создать» / «создать первую»)")
	}

	id := uuid.New()
	if err := s.store.Upsert(ctx, doc.Name, id, map[string]any{"Описание": "из звонка"}, doc); err != nil {
		t.Fatal(err)
	}
	full := list()
	if strings.Contains(full, "data-ob-list-create") {
		t.Error("в списке осталась кнопка «+ Создать» (и клавиша Insert)")
	}
	if !strings.Contains(full, `data-ob-row-can-copy=""`) {
		t.Error("копирование строки списка не выключено")
	}

	rec := httptest.NewRecorder()
	s.formEdit(rec, noCreateRequest(ctx, "GET", "/ui/document/заявка/"+id.String(),
		map[string]string{"kind": "document", "entity": "заявка", "id": id.String()}, nil))
	if rec.Code != http.StatusOK {
		t.Fatalf("карточка: code = %d, body: %s", rec.Code, rec.Body.String())
	}
	if strings.Contains(rec.Body.String(), "/new?copy=") {
		t.Error("в карточке осталась кнопка «Скопировать»")
	}
}

// Кнопки скрыты, но адрес создания можно набрать руками — сервер отказывает,
// и POST ничего не записывает.
func TestInteractiveCreateFalse_RefusesNewURL(t *testing.T) {
	doc := loadNoCreateDoc(t)
	s, ctx := newSubmitTestServer(t, []*metadata.Entity{doc})
	params := map[string]string{"kind": "document", "entity": "заявка"}

	get := httptest.NewRecorder()
	s.form(get, noCreateRequest(ctx, "GET", "/ui/document/заявка/new", params, nil))
	if get.Code != http.StatusForbidden || !strings.Contains(get.Body.String(), "создаётся только из других объектов") {
		t.Fatalf("GET /new: code = %d, ожидался 403 с причиной; body: %s", get.Code, get.Body.String())
	}

	post := httptest.NewRecorder()
	s.submit(post, noCreateRequest(ctx, "POST", "/ui/document/заявка/new", params, url.Values{"Описание": {"в обход"}}))
	if post.Code != http.StatusForbidden {
		t.Fatalf("POST /new: code = %d, ожидался 403", post.Code)
	}
	rows, err := s.store.List(ctx, doc.Name, doc, storage.ListParams{})
	if err != nil {
		t.Fatal(err)
	}
	if len(rows) != 0 {
		t.Fatalf("POST /new записал документ: %d строк", len(rows))
	}

	ref := httptest.NewRecorder()
	s.refCreateRedirect(ref, noCreateRequest(ctx, "GET", "/ui/_ref-create/Заявка", map[string]string{"entity": doc.Name}, nil))
	if ref.Code != http.StatusForbidden {
		t.Fatalf("создание из поля ссылки: code = %d, ожидался 403", ref.Code)
	}
}

// Поле ссылки на такой документ не предлагает «+ Создать», даже при праве
// записи на него.
func TestInteractiveCreateFalse_NoInlineCreateFromRefField(t *testing.T) {
	doc := loadNoCreateDoc(t)
	call := &metadata.Entity{
		Name: "Звонок", Kind: metadata.KindDocument,
		Fields: []metadata.Field{{Name: "Заявка", Type: "reference:Заявка", RefEntity: doc.Name}},
	}
	s, _ := newSubmitTestServer(t, []*metadata.Entity{doc, call})
	user := &auth.User{Login: "operator", Roles: []*auth.Role{{Permissions: auth.Permission{
		Documents: map[string][]string{doc.Name: {"read", "write"}, call.Name: {"read", "write"}},
	}}}}
	req := reqWithChi(http.MethodGet, "/ui/document/Звонок/new", nil, map[string]string{"kind": "document", "entity": call.Name})
	req = req.WithContext(auth.ContextWithUser(req.Context(), user))
	rec := httptest.NewRecorder()
	s.form(rec, req)
	if rec.Code != http.StatusOK {
		t.Fatalf("форма звонка: code = %d, body: %s", rec.Code, rec.Body.String())
	}
	body := rec.Body.String()
	if !strings.Contains(body, `data-ref-entity="Заявка"`) {
		t.Fatal("поле ссылки на заявку не отрисовано")
	}
	if strings.Contains(body, `data-ref-allow-create="1"`) {
		t.Error("поле ссылки предлагает создать заявку из звонка вручную")
	}
}
