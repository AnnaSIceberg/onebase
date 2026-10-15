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

// `interactive_create: false` — аналог права «Интерактивное добавление» 1С:
// документ, который рождается из другого (заявка из звонка), из интерфейса
// создаёт только роль с явным `create` (и администратор); одного `write` мало.
// Сущность грузится из YAML, запросы идут через HTTP-обработчики — тем же
// путём, что у пользователя.

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

func noCreateUser(login string, admin bool, ops ...string) *auth.User {
	return &auth.User{Login: login, IsAdmin: admin, Roles: []*auth.Role{{Permissions: auth.Permission{
		Documents: map[string][]string{"Заявка": ops, "Звонок": {"read", "write"}},
	}}}}
}

var (
	operatorNoCreate = noCreateUser("operator", false, "read", "write")
	managerCreate    = noCreateUser("manager", false, "read", "write", "create")
	adminCreate      = noCreateUser("admin", true)
)

func noCreateRequest(ctx context.Context, user *auth.User, method, target string, params map[string]string, body url.Values) *http.Request {
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
	ctx = context.WithValue(ctx, chi.RouteCtxKey, rctx)
	return req.WithContext(auth.ContextWithUser(ctx, user))
}

func TestInteractiveCreateFalse_ListAndCardByRole(t *testing.T) {
	doc := loadNoCreateDoc(t)
	s, ctx := newSubmitTestServer(t, []*metadata.Entity{doc})
	params := map[string]string{"kind": "document", "entity": "заявка"}

	list := func(u *auth.User) string {
		t.Helper()
		rec := httptest.NewRecorder()
		s.list(rec, noCreateRequest(ctx, u, "GET", "/ui/document/заявка", params, nil))
		if rec.Code != http.StatusOK {
			t.Fatalf("список (%s): code = %d, body: %s", u.Login, rec.Code, rec.Body.String())
		}
		return rec.Body.String()
	}

	if empty := list(operatorNoCreate); strings.Contains(empty, "data-ob-list-create") || strings.Contains(empty, "создать первую") {
		t.Error("пустой список предлагает оператору создание («+ Создать» / «создать первую»)")
	}

	id := uuid.New()
	if err := s.store.Upsert(ctx, doc.Name, id, map[string]any{"Описание": "из звонка"}, doc); err != nil {
		t.Fatal(err)
	}
	full := list(operatorNoCreate)
	if strings.Contains(full, "data-ob-list-create") {
		t.Error("у оператора без create осталась кнопка «+ Создать» (и клавиша Insert)")
	}
	if !strings.Contains(full, `data-ob-row-can-copy=""`) {
		t.Error("у оператора без create не выключено копирование строки")
	}
	for _, u := range []*auth.User{managerCreate, adminCreate} {
		page := list(u)
		if !strings.Contains(page, "data-ob-list-create") || !strings.Contains(page, `data-ob-row-can-copy="1"`) {
			t.Errorf("%s (право create) не видит «+ Создать» / копирования", u.Login)
		}
	}

	card := func(u *auth.User) string {
		t.Helper()
		rec := httptest.NewRecorder()
		s.formEdit(rec, noCreateRequest(ctx, u, "GET", "/ui/document/заявка/"+id.String(),
			map[string]string{"kind": "document", "entity": "заявка", "id": id.String()}, nil))
		if rec.Code != http.StatusOK {
			t.Fatalf("карточка (%s): code = %d, body: %s", u.Login, rec.Code, rec.Body.String())
		}
		return rec.Body.String()
	}
	if strings.Contains(card(operatorNoCreate), "/new?copy=") {
		t.Error("в карточке у оператора без create осталась кнопка «Скопировать»")
	}
	if !strings.Contains(card(managerCreate), "/new?copy=") {
		t.Error("в карточке у роли с create нет «Скопировать»")
	}
}

// Кнопки скрыты, но адрес создания можно набрать руками — сервер отказывает
// без права create, и POST ничего не записывает.
func TestInteractiveCreateFalse_NewURLByRole(t *testing.T) {
	doc := loadNoCreateDoc(t)
	s, ctx := newSubmitTestServer(t, []*metadata.Entity{doc})
	params := map[string]string{"kind": "document", "entity": "заявка"}

	get := httptest.NewRecorder()
	s.form(get, noCreateRequest(ctx, operatorNoCreate, "GET", "/ui/document/заявка/new", params, nil))
	if get.Code != http.StatusForbidden || !strings.Contains(get.Body.String(), "создаётся только из других объектов") {
		t.Fatalf("GET /new без create: code = %d, ожидался 403 с причиной; body: %s", get.Code, get.Body.String())
	}

	post := httptest.NewRecorder()
	s.submit(post, noCreateRequest(ctx, operatorNoCreate, "POST", "/ui/document/заявка/new", params, url.Values{"Описание": {"в обход"}}))
	if post.Code != http.StatusForbidden {
		t.Fatalf("POST /new без create: code = %d, ожидался 403", post.Code)
	}
	rows, err := s.store.List(ctx, doc.Name, doc, storage.ListParams{})
	if err != nil {
		t.Fatal(err)
	}
	if len(rows) != 0 {
		t.Fatalf("POST /new без create записал документ: %d строк", len(rows))
	}

	ref := httptest.NewRecorder()
	s.refCreateRedirect(ref, noCreateRequest(ctx, operatorNoCreate, "GET", "/ui/_ref-create/Заявка", map[string]string{"entity": doc.Name}, nil))
	if ref.Code != http.StatusForbidden {
		t.Fatalf("создание из поля ссылки без create: code = %d, ожидался 403", ref.Code)
	}

	for _, u := range []*auth.User{managerCreate, adminCreate} {
		rec := httptest.NewRecorder()
		s.form(rec, noCreateRequest(ctx, u, "GET", "/ui/document/заявка/new", params, nil))
		if rec.Code != http.StatusOK {
			t.Errorf("GET /new у %s (право create): code = %d", u.Login, rec.Code)
		}
	}
}

// Поле ссылки на такой документ не предлагает «+» без права create.
func TestInteractiveCreateFalse_InlineCreateFromRefFieldByRole(t *testing.T) {
	doc := loadNoCreateDoc(t)
	call := &metadata.Entity{
		Name: "Звонок", Kind: metadata.KindDocument,
		Fields: []metadata.Field{{Name: "Заявка", Type: "reference:Заявка", RefEntity: doc.Name}},
	}
	s, _ := newSubmitTestServer(t, []*metadata.Entity{doc, call})
	render := func(u *auth.User) string {
		t.Helper()
		req := reqWithChi(http.MethodGet, "/ui/document/Звонок/new", nil, map[string]string{"kind": "document", "entity": call.Name})
		req = req.WithContext(auth.ContextWithUser(req.Context(), u))
		rec := httptest.NewRecorder()
		s.form(rec, req)
		if rec.Code != http.StatusOK {
			t.Fatalf("форма звонка (%s): code = %d, body: %s", u.Login, rec.Code, rec.Body.String())
		}
		body := rec.Body.String()
		if !strings.Contains(body, `data-ref-entity="Заявка"`) {
			t.Fatal("поле ссылки на заявку не отрисовано")
		}
		return body
	}
	if strings.Contains(render(operatorNoCreate), `data-ref-allow-create="1"`) {
		t.Error("поле ссылки предлагает оператору без create создать заявку вручную")
	}
	if !strings.Contains(render(managerCreate), `data-ref-allow-create="1"`) {
		t.Error("роль с create не может создать заявку из поля ссылки")
	}
}
