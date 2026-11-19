package ui

import (
	"net/http"
	"net/http/httptest"
	"net/url"
	"regexp"
	"strings"
	"testing"
	"time"

	"github.com/go-chi/chi/v5"
	"github.com/ivantit66/onebase/internal/auth"
	"github.com/ivantit66/onebase/internal/metadata"
	"github.com/ivantit66/onebase/internal/runtime"
)

func TestManagedFormInstanceThroughHTTP(t *testing.T) {
	srv, fixture := setupManagedEventsServer(t, `
Процедура ЗапомнитьЭкземпляр()
	Объект.Наименование = ИдЭкземпляраФормы();
КонецПроцедуры
`, map[metadata.FormEventType]string{metadata.FormEventOnOpen: "ЗапомнитьЭкземпляр"}, []*metadata.FormElement{{
		Kind: metadata.FormElementGroupBox, Name: "История",
		Handlers: map[metadata.FormEventType]string{metadata.FormEventAfterOpen: "ЗапомнитьЭкземпляр"},
	}})
	form := fixture.Forms[0]
	form.InstanceContext = true
	ent := &metadata.Entity{Name: "CallInstance", Kind: metadata.KindDocument, Fields: fixture.Fields, Forms: fixture.Forms}
	form.EntityName = ent.Name
	srv.reg.Load(runtime.LoadOptions{Entities: []*metadata.Entity{ent}})
	if err := srv.store.Migrate(t.Context(), []*metadata.Entity{ent}); err != nil {
		t.Fatal(err)
	}
	user := &auth.User{ID: "test-user", Login: "test-user", IsAdmin: true}
	router := chi.NewRouter()
	srv.Mount(router)
	path := "/ui/document/" + ent.Name

	request := func(method, suffix, session, token string, values url.Values) *http.Request {
		r := httptest.NewRequest(method, path+suffix, strings.NewReader(values.Encode()))
		r.Header.Set("Content-Type", "application/x-www-form-urlencoded")
		r.AddCookie(&http.Cookie{Name: "onebase_session", Value: session})
		if token != "" {
			r.Header.Set(managedFormInstanceHeader, token)
		}
		return r.WithContext(auth.ContextWithUser(r.Context(), user))
	}
	open := func() string {
		t.Helper()
		rec := httptest.NewRecorder()
		router.ServeHTTP(rec, request("GET", "/new", "session-1", "", nil))
		match := regexp.MustCompile(`data-ob-form-instance="([^"]+)"`).FindStringSubmatch(rec.Body.String())
		if rec.Code != http.StatusOK || len(match) != 2 {
			t.Fatalf("open: HTTP %d, found instance %v", rec.Code, len(match) == 2)
		}
		return match[1]
	}
	event := func(kind metadata.FormEventType, target, session, token string) string {
		t.Helper()
		values := url.Values{"_event": {string(kind)}, "_element": {target}, "_kind": {"object"},
			"Наименование": {"initial"}, "ИдЭкземпляраФормы": {"client-value"}}
		rec := httptest.NewRecorder()
		router.ServeHTTP(rec, request("POST", "/form-event", session, token, values))
		response := decodeFormEventResponse(t, rec.Body.Bytes())
		if rec.Code != http.StatusOK || !response.OK {
			t.Fatalf("event %s: HTTP %d: %s", kind, rec.Code, response.Error)
		}
		return anyString(response.Values["Наименование"])
	}

	first, second := open(), open()
	id1 := event(metadata.FormEventOnOpen, "", "session-1", first)
	id2 := event(metadata.FormEventOnOpen, "", "session-1", second)
	if id1 == "" || id2 == "" || id1 == id2 || id1 == first || id2 == second {
		t.Fatal("opening must produce independent identities distinct from the bearer")
	}
	if id1 != event(metadata.FormEventAfterOpen, "История", "session-1", first) ||
		id2 != event(metadata.FormEventAfterOpen, "История", "session-1", second) {
		t.Fatal("deferred events must use the identities of their own instances")
	}
	expired := srv.newManagedFormInstance(request("GET", "/new", "session-1", "", nil),
		entityFormCloseSchema(ent, form), "", time.Now().Add(-managedFormInstanceTTL-time.Minute))
	if expired == "" {
		t.Fatal("expired context fixture was not issued")
	}
	for _, tc := range []struct{ name, session, token string }{
		{"missing", "session-1", ""},
		{"non-context", "session-1", id1},
		{"other session", "session-2", first},
		{"expired", "session-1", expired},
	} {
		t.Run(tc.name, func(t *testing.T) {
			if got := event(metadata.FormEventAfterOpen, "История", tc.session, tc.token); got != "" {
				t.Fatal("unavailable context produced a form identity")
			}
		})
	}
}
