package ui

import (
	"fmt"
	"net/http"
	"net/http/httptest"
	"net/url"
	"regexp"
	"slices"
	"strings"
	"testing"
	"time"

	"github.com/go-chi/chi/v5"
	"github.com/google/uuid"
	"github.com/ivantit66/onebase/internal/auth"
	"github.com/ivantit66/onebase/internal/metadata"
	"github.com/ivantit66/onebase/internal/runtime"
)

func TestManagedFormOpeningContextThroughHTTP(t *testing.T) {
	taskID, orderID, otherID := uuid.New(), uuid.New(), uuid.New()
	server, fixture := setupManagedEventsServer(t, fmt.Sprintf(`
Процедура Перейти()
	ОткрытьДокументСКонтекстом("Order2K", "%s", Новый Структура("Задача2К", "%s"));
КонецПроцедуры
`, orderID, taskID), nil, []*metadata.FormElement{
		{Kind: metadata.FormElementButton, Name: "Перейти", Handlers: map[metadata.FormEventType]string{metadata.FormEventOnClick: "Перейти"}},
	})
	source := &metadata.Entity{Name: "Task2K", Kind: metadata.KindDocument, Fields: fixture.Fields, Forms: fixture.Forms}
	source.Forms[0].EntityName = source.Name
	targetForm := &metadata.FormModule{
		Name: "OrderForm", Kind: "object", EntityName: "Order2K",
		InstanceContext: true, LayoutKind: metadata.FormLayoutManaged,
		Handlers: map[metadata.FormEventType]string{
			metadata.FormEventOnOpen:           "Контекст",
			metadata.FormEventOnCreateAtServer: "Контекст",
			metadata.FormEventBeforeWrite:      "ПроверитьЗапись",
		},
		Attributes: []*metadata.FormAttribute{{Name: "ПоказатьКолонки", TypeRef: "bool"}},
		Elements: []*metadata.FormElement{
			{Kind: metadata.FormElementGroupBox, Name: "Колонки", Orientation: "horizontal", EqualColumnsWhen: "ПоказатьКолонки",
				Children: []*metadata.FormElement{{Kind: metadata.FormElementField, Name: "Имя", DataPath: "Объект.Наименование"}}},
			{Kind: metadata.FormElementButton, Name: "Проверка", Handlers: map[metadata.FormEventType]string{metadata.FormEventOnClick: "Контекст"}},
		},
		ProgramAST: mustParse(t, `
Процедура Контекст()
	ИдЗадачи = КонтекстОткрытияФормы("Задача2К");
	Объект.ПоказатьКолонки = ИдЗадачи <> "";
	Сообщить("context:" + ИдЗадачи);
КонецПроцедуры
Процедура ПроверитьЗапись()
	Если КонтекстОткрытияФормы("Задача2К") <> "" Тогда
		ВызватьИсключение("readonly-from-task");
	КонецЕсли;
КонецПроцедуры
`),
	}
	target := &metadata.Entity{Name: targetForm.EntityName, Kind: metadata.KindDocument, Fields: fixture.Fields, Forms: []*metadata.FormModule{targetForm}}
	entities := []*metadata.Entity{source, target}
	server.reg.Load(runtime.LoadOptions{Entities: entities})
	if err := server.store.Migrate(t.Context(), entities); err != nil {
		t.Fatal(err)
	}
	for _, row := range []struct {
		entity *metadata.Entity
		id     uuid.UUID
	}{{source, taskID}, {target, orderID}, {target, otherID}} {
		if err := server.store.Upsert(t.Context(), row.entity.Name, row.id, map[string]any{"Наименование": "Объект"}, row.entity); err != nil {
			t.Fatal(err)
		}
	}
	now := time.Now()
	server.concealClock = func() time.Time { return now }
	operator := &auth.User{ID: "operator", Login: "operator", Roles: []*auth.Role{{
		Name: "ОператорКЦ", Permissions: auth.Permission{Documents: map[string][]string{
			source.Name: {"read", "write"}, target.Name: {"read", "write"},
		}},
	}}}
	otherOperator := &auth.User{ID: "other", Login: "other", Roles: operator.Roles}
	withoutTarget := &auth.User{ID: "limited", Login: "limited", Roles: []*auth.Role{{
		Name: "ТолькоЗадача", Permissions: auth.Permission{Documents: map[string][]string{source.Name: {"read", "write"}}},
	}}}

	router := chi.NewRouter()
	server.Mount(router)
	send := func(method, path, session, instance string, user *auth.User, fields url.Values) *httptest.ResponseRecorder {
		t.Helper()
		req := httptest.NewRequest(method, path, strings.NewReader(fields.Encode()))
		req.Header.Set("Content-Type", "application/x-www-form-urlencoded")
		req.AddCookie(&http.Cookie{Name: "onebase_session", Value: session})
		if instance != "" {
			req.Header.Set(managedFormInstanceHeader, instance)
		}
		rec := httptest.NewRecorder()
		router.ServeHTTP(rec, req.WithContext(auth.ContextWithUser(req.Context(), user)))
		return rec
	}
	run := func(entity *metadata.Entity, id uuid.UUID, element string, event metadata.FormEventType, session, instance string, user *auth.User) formEventResponse {
		t.Helper()
		resp := send(http.MethodPost, "/ui/document/"+entity.Name+"/form-event", session, instance, user, url.Values{
			"_id": {id.String()}, "_kind": {"object"}, "_element": {element}, "_event": {string(event)},
			"_version": {"1"}, "Наименование": {"Объект"}, formOpenContextQuery: {taskID.String()},
		})
		if resp.Code != http.StatusOK {
			t.Fatalf("form-event: HTTP %d %s", resp.Code, resp.Body.String())
		}
		return decodeFormEventResponse(t, resp.Body.Bytes())
	}
	launch := func() string {
		t.Helper()
		resp := run(source, taskID, "Перейти", metadata.FormEventOnClick, "session-1", "", operator)
		if !resp.OK || resp.Navigation == nil || !resp.Navigation.OpenTab || strings.Contains(resp.Navigation.URL, taskID.String()) {
			t.Fatalf("expected an opaque navigation ticket, got ok=%t nav=%+v error=%s", resp.OK, resp.Navigation, resp.Error)
		}
		return resp.Navigation.URL
	}
	tokenPattern := regexp.MustCompile(`data-ob-form-instance="([^"]+)"`)
	open := func(path, session string, user *auth.User, wantColumns bool) string {
		t.Helper()
		rec := send(http.MethodGet, path, session, "", user, nil)
		match := tokenPattern.FindStringSubmatch(rec.Body.String())
		if rec.Code != http.StatusOK || len(match) != 2 || strings.Contains(rec.Body.String(), taskID.String()) {
			t.Fatalf("GET: HTTP %d, found instance %t", rec.Code, len(match) == 2)
		}
		gotColumns := strings.Contains(rec.Body.String(), "managed-group-horizontal ob-equal-columns ob-equal-columns-conditional")
		if gotColumns != wantColumns {
			t.Fatalf("conditional layout on the first GET = %t, want %t", gotColumns, wantColumns)
		}
		field := entityFormServiceFieldNames(target, targetForm)[formOpenInstanceField]
		gotNativeTransport := strings.Contains(rec.Body.String(), `name="`+field+`" value="`+match[1]+`"`)
		if gotNativeTransport != wantColumns {
			t.Fatalf("native proof input exists = %t, want %t", gotNativeTransport, wantColumns)
		}
		return match[1]
	}
	assertContext := func(id uuid.UUID, session, instance string, user *auth.User, want string, afterOpen bool) {
		t.Helper()
		event, element := metadata.FormEventOnOpen, ""
		if afterOpen {
			event, element = metadata.FormEventOnClick, "Проверка"
		}
		resp := run(target, id, element, event, session, instance, user)
		if !resp.OK || !slices.Contains(resp.Messages, "context:"+want) {
			t.Fatalf("unexpected context: messages=%v error=%s", resp.Messages, resp.Error)
		}
		if resp.ElementStates == nil || len(resp.ElementStates.EqualColumns) != 1 {
			t.Fatalf("missing conditional layout state: %+v", resp.ElementStates)
		}
		for _, equalColumns := range resp.ElementStates.EqualColumns {
			if equalColumns != (want != "") {
				t.Fatalf("unexpected equal columns=%t for context %q and values %v", equalColumns, want, resp.Values)
			}
		}
	}

	path := "/ui/document/" + target.Name + "/" + orderID.String()
	transfer := launch()
	deny := run(source, taskID, "Перейти", metadata.FormEventOnClick, "session-1", "", withoutTarget)
	if deny.OK || deny.Navigation != nil {
		t.Fatal("source access must not grant target navigation")
	}
	assertContext(orderID, "session-2", open(transfer, "session-2", operator, false), operator, "", false)
	assertContext(orderID, "session-1", open(transfer, "session-1", otherOperator, false), otherOperator, "", false)
	assertContext(otherID, "session-1", open(strings.Replace(transfer, orderID.String(), otherID.String(), 1), "session-1", operator, false), operator, "", false)
	token := open(transfer, "session-1", operator, true)
	assertContext(orderID, "session-1", token, operator, taskID.String(), false)
	assertContext(orderID, "session-1", token, operator, taskID.String(), true)
	assertContext(orderID, "session-1", open(path, "session-1", operator, false), operator, "", false)
	assertContext(orderID, "session-1", open(transfer, "session-1", operator, false), operator, "", false)
	assertContext(orderID, "session-2", token, operator, "", false)
	assertContext(otherID, "session-1", token, operator, "", false)

	field := entityFormServiceFieldNames(target, targetForm)[formOpenInstanceField]
	blocked := send(http.MethodPost, path, "session-1", "", operator, url.Values{
		"Наименование": {"attempted native write"}, "_action": {"save"}, field: {token},
	})
	if blocked.Code == http.StatusSeeOther || !strings.Contains(blocked.Body.String(), "readonly-from-task") ||
		!strings.Contains(blocked.Body.String(), `name="`+field+`" value="`+token+`"`) {
		t.Fatalf("native save must fail and preserve the verified instance, HTTP %d", blocked.Code)
	}

	closeBody := closeIntentBody(uuid.NewString(), "close", "attempted close write")
	closeBody.Set("_id", orderID.String())
	closeBody.Set("_version", "1")
	closeBody.Set("_close_mode", "save")
	closeBody.Set("_close_first_attempt", "1")
	ensureEntityCloseIntentSchema(target, closeBody)
	closeRequest := httptest.NewRequest(http.MethodPost, "/ui/document/"+target.Name+"/form-close-intent", strings.NewReader(closeBody.Encode()))
	closeRequest.Header.Set("Content-Type", "application/x-www-form-urlencoded")
	closeRequest.Header.Set(managedFormInstanceHeader, token)
	closeRequest.AddCookie(&http.Cookie{Name: "onebase_session", Value: "session-1"})
	setEntityCloseIntentHeaders(closeRequest, closeBody)
	closeRecorder := httptest.NewRecorder()
	router.ServeHTTP(closeRecorder, closeRequest.WithContext(auth.ContextWithUser(closeRequest.Context(), operator)))
	closed := decodeCloseIntentResponse(t, closeRecorder)
	if closed.OK || !strings.Contains(closed.Error, "readonly-from-task") || (closed.Close != nil && closed.Close.Saved) {
		t.Fatalf("save on close must fail in the contextual instance: HTTP %d %s", closeRecorder.Code, closed.Error)
	}

	expiredTicket := launch()
	now = now.Add(managedFormInstanceTTL + formOpenTicketTTL)
	assertContext(orderID, "session-1", open(expiredTicket, "session-1", operator, false), operator, "", false)
	assertContext(orderID, "session-1", token, operator, "", true)

	normalToken := open(path, "session-1", operator, false)
	for _, attempt := range []struct{ name, proof string }{
		{"ordinary instance", normalToken}, {"native without proof", ""},
	} {
		rec := send(http.MethodPost, path, "session-1", attempt.proof, operator,
			url.Values{"Наименование": {attempt.name}, "_action": {"save"}})
		row, err := server.store.GetByID(t.Context(), target.Name, orderID, target)
		if rec.Code != http.StatusSeeOther || err != nil || row["Наименование"] != attempt.name {
			t.Fatalf("ordinary save %q: HTTP %d, err=%v", attempt.name, rec.Code, err)
		}
	}
}
