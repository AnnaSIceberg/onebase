package ui

import (
	"net/http"
	"net/http/httptest"
	"net/url"
	"regexp"
	"strconv"
	"strings"
	"testing"
	"time"

	"github.com/go-chi/chi/v5"
	"github.com/google/uuid"
	"github.com/ivantit66/onebase/internal/auth"
	"github.com/ivantit66/onebase/internal/metadata"
	"github.com/ivantit66/onebase/internal/runtime"
)

func TestManagedFormInstanceThroughHTTP(t *testing.T) {
	srv, fixture := setupManagedEventsServer(t, `
Процедура ЗапомнитьЭкземпляр()
	Объект.Наименование = ИдЭкземпляраФормы();
	ПодключитьОбработчикОжидания("Пульс", 60);
КонецПроцедуры

Процедура Пульс()
	Объект.Наименование = ИдЭкземпляраФормы();
КонецПроцедуры

Процедура ЗаписатьДокумент()
	Объект.Записать();
КонецПроцедуры

Процедура ОтменитьЗакрытие(Отказ)
	Сообщить("instance:" + ИдЭкземпляраФормы());
	Отказ = Истина;
КонецПроцедуры
`, map[metadata.FormEventType]string{
		metadata.FormEventOnOpen: "ЗапомнитьЭкземпляр", metadata.FormEventBeforeClose: "ОтменитьЗакрытие",
	}, []*metadata.FormElement{
		{Kind: metadata.FormElementGroupBox, Name: "История", Handlers: map[metadata.FormEventType]string{metadata.FormEventAfterOpen: "ЗапомнитьЭкземпляр"}},
		{Kind: metadata.FormElementButton, Name: "Запись", Handlers: map[metadata.FormEventType]string{metadata.FormEventOnClick: "ЗаписатьДокумент"}},
	})
	form := fixture.Forms[0]
	form.InstanceContext = true
	form.RetainInstanceOnReload = true
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
	postEvent := func(kind metadata.FormEventType, target, session, token, id string, version int64) formEventResponse {
		t.Helper()
		values := url.Values{"_event": {string(kind)}, "_element": {target}, "_kind": {"object"},
			"Наименование": {"initial"}, "ИдЭкземпляраФормы": {"client-value"}}
		if id != "" {
			values.Set("_id", id)
			values.Set("_version", strconv.FormatInt(version, 10))
		}
		rec := httptest.NewRecorder()
		router.ServeHTTP(rec, request("POST", "/form-event", session, token, values))
		response := decodeFormEventResponse(t, rec.Body.Bytes())
		if rec.Code != http.StatusOK || !response.OK {
			t.Fatalf("event %s: HTTP %d: %s", kind, rec.Code, response.Error)
		}
		return response
	}
	event := func(kind metadata.FormEventType, target, session, token string) string {
		t.Helper()
		return anyString(postEvent(kind, target, session, token, "", 0).Values["Наименование"])
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

	save := postEvent(metadata.FormEventOnClick, "Запись", "session-1", first, "", 0)
	if save.SavedID == "" || save.FormInstance == "" || save.FormInstance == first {
		t.Fatal("confirmed insert did not return its newly scoped form context")
	}
	checkSaved := func(saved formEventResponse, expectedID, previousToken string) {
		t.Helper()
		after := postEvent(metadata.FormEventAfterOpen, "История", "session-1", saved.FormInstance, saved.SavedID, saved.Version)
		if got := anyString(after.Values["Наименование"]); got != expectedID {
			t.Fatalf("saved form lost its original identity: %q, want %q", got, expectedID)
		}
		old := postEvent(metadata.FormEventAfterOpen, "История", "session-1", previousToken, saved.SavedID, saved.Version)
		if anyString(old.Values["Наименование"]) != "" {
			t.Fatal("new-record context was accepted for a saved document")
		}
	}
	checkSaved(save, id1, first)

	t.Run("valid idle tick renews the same scoped instance", func(t *testing.T) {
		now := time.Now()
		schema := entityFormCloseSchema(ent, form)
		expiry := now.Add(time.Minute)
		get := request("GET", "/"+save.SavedID, "session-1", "", nil)
		ticket := srv.signManagedFormInstance(get, schema, save.SavedID, id1, strconv.FormatInt(expiry.Unix(), 10))
		open := postEvent(metadata.FormEventOnOpen, "", "session-1", ticket, save.SavedID, save.Version)
		if len(open.Idle) != 1 || open.FormInstance != "" {
			t.Fatal("only an accepted idle tick should return a renewal")
		}

		tick := func(session, instance, idleProof string) formEventResponse {
			values := url.Values{
				"_event": {formIdleEvent}, "_kind": {"object"},
				"_id": {save.SavedID}, "_version": {strconv.FormatInt(save.Version, 10)},
				"_idle_proc": {open.Idle[0].Proc}, "_idle_token": {idleProof},
			}
			rec := httptest.NewRecorder()
			router.ServeHTTP(rec, request("POST", "/form-event", session, instance, values))
			return decodeFormEventResponse(t, rec.Body.Bytes())
		}

		next := tick("session-1", ticket, open.Idle[0].Token)
		if !next.OK || anyString(next.Values["Наименование"]) != id1 || next.FormInstance == "" {
			t.Fatal("tick must use and renew the same identity")
		}
		if got, _ := srv.validManagedFormInstanceAt(request("POST", "/form-event", "session-1", next.FormInstance, nil),
			save.SavedID, schema, expiry.Add(time.Second)); got != id1 {
			t.Fatal("renewal must still validate after the original expiry")
		}
		for _, tc := range []struct {
			name, session, instance, proof string
		}{
			{"fresh", "session-1", save.FormInstance, open.Idle[0].Token},
			{"wrong session", "session-2", ticket, open.Idle[0].Token},
			{"wrong record", "session-1", srv.signManagedFormInstance(get, schema, uuid.NewString(), id1, strconv.FormatInt(expiry.Unix(), 10)), open.Idle[0].Token},
			{"tampered", "session-1", ticket + "a", open.Idle[0].Token},
			{"expired", "session-1", srv.signManagedFormInstance(get, schema, save.SavedID, id1, strconv.FormatInt(now.Add(-time.Second).Unix(), 10)), open.Idle[0].Token},
			{"saved-phone context", "session-1", srv.signManagedFormInstance(get, schema, save.SavedID, savedPhoneOpeningInstancePrefix+id1, strconv.FormatInt(expiry.Unix(), 10)), open.Idle[0].Token},
			{"wrong idle proof", "session-1", ticket, "forged"},
		} {
			t.Run(tc.name, func(t *testing.T) {
				if got := tick(tc.session, tc.instance, tc.proof); got.FormInstance != "" {
					t.Fatal("unexpected renewal")
				}
			})
		}
	})

	// A close request can save and keep the card open when BeforeClose cancels.
	// Exact transport replay still returns the existing reload-only tombstone.
	body := closeIntentBody(uuid.NewString(), "close", "before-close")
	body.Set("_close_mode", "save")
	body.Set("_close_first_attempt", "1")
	ensureEntityCloseIntentSchema(ent, body)
	closeForm := func(firstAttempt bool) formEventResponse {
		t.Helper()
		req := request("POST", "/form-close-intent", "session-1", second, body)
		setEntityCloseIntentHeaders(req, body)
		if !firstAttempt {
			req.Header.Del("X-OneBase-Close-First-Attempt")
		}
		rec := httptest.NewRecorder()
		router.ServeHTTP(rec, req)
		response := decodeCloseIntentResponse(t, rec)
		if rec.Code != http.StatusOK || response.Close == nil || response.Close.Allowed ||
			!response.Close.Saved || response.SavedID == "" {
			t.Fatalf("expected a saved card kept open by BeforeClose: HTTP %d, close=%+v, error=%q", rec.Code, response.Close, response.Error)
		}
		if firstAttempt && (response.FormInstance == "" || !strings.Contains(strings.Join(response.Messages, "\n"), "instance:"+id2)) {
			t.Fatal("BeforeClose did not receive and preserve its instance identity")
		}
		return response
	}
	closed := closeForm(true)
	replayed := closeForm(false)
	if replayed.SavedID != closed.SavedID || !replayed.Close.Reload || replayed.FormInstance != "" {
		t.Fatal("exact close retry must recover the confirmed record without cached form state")
	}
	checkSaved(closed, id2, second)
}
