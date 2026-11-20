package ui

import (
	"net/http"
	"net/http/httptest"
	"net/url"
	"regexp"
	"strings"
	"testing"

	"github.com/go-chi/chi/v5"
	"github.com/google/uuid"
	"github.com/ivantit66/onebase/internal/auth"
	"github.com/ivantit66/onebase/internal/dbtest"
	"github.com/ivantit66/onebase/internal/metadata"
	"github.com/ivantit66/onebase/internal/storage"
)

func TestSavedPhoneMatchThroughManagedForm(t *testing.T) {
	dbtest.ForEachDialect(t, testSavedPhoneMatchThroughManagedForm)
}

func testSavedPhoneMatchThroughManagedForm(t *testing.T, db *storage.DB) {
	form := &metadata.FormModule{
		Name: "Main", Kind: "object", EntityName: "TaskMatch", LayoutKind: metadata.FormLayoutManaged,
		SavedPhoneMatch: &metadata.SavedPhoneMatchSpec{
			SourceFields: []string{"Phone", "Alternate"}, TargetDocument: "ClaimMatch", TargetFields: []string{"Mobile", "Home", "Work"},
		},
		Handlers: map[metadata.FormEventType]string{
			metadata.FormEventOnCreateAtServer: "Подобрать", metadata.FormEventOnOpen: "Подобрать",
			metadata.FormEventOnReadAtServer: "ПроверитьЧтение",
		},
		Attributes: []*metadata.FormAttribute{{Name: "Результат", TypeRef: "string"}},
		Elements: []*metadata.FormElement{
			{Kind: metadata.FormElementField, Name: "Результат", DataPath: "Объект.Результат"},
			{Kind: metadata.FormElementButton, Name: "Проверка", Handlers: map[metadata.FormEventType]string{metadata.FormEventOnClick: "Подобрать"}},
		},
		ProgramAST: mustParse(t, `
Процедура Подобрать()
	Если НЕ ЗначениеЗаполнено(Объект.Ссылка) Тогда
		Возврат;
	КонецЕсли;
	З = Новый Запрос;
	З.Текст = "ВЫБРАТЬ З.Ссылка КАК Ссылка, З.Label КАК Метка ИЗ Документ.ClaimMatch КАК З
		|ГДЕ СОХРАНЕННЫЙТЕЛЕФОНСОВПАДАЕТ(З) И З.Closed = ЛОЖЬ
		|УПОРЯДОЧИТЬ ПО Метка";
	Объект.Результат = "";
	Для Каждого Строка Из З.Выполнить() Цикл
		Объект.Результат = Объект.Результат + Строка.Метка + "|";
	КонецЦикла;
КонецПроцедуры

Процедура ПроверитьЧтение()
	Если Объект.Label = "revoked" Тогда
		ВызватьИсключение("Задача закрыта для чтения");
	КонецЕсли;
КонецПроцедуры
`),
	}
	task := &metadata.Entity{Name: "TaskMatch", Kind: metadata.KindDocument, Fields: []metadata.Field{
		{Name: "Label", Type: metadata.FieldTypeString}, {Name: "Owner", Type: metadata.FieldTypeString},
		{Name: "Phone", Type: metadata.FieldTypeString, PII: true}, {Name: "Alternate", Type: metadata.FieldTypeString, PII: true},
	}, Forms: []*metadata.FormModule{form}}
	claim := &metadata.Entity{Name: "ClaimMatch", Kind: metadata.KindDocument, Fields: []metadata.Field{
		{Name: "Label", Type: metadata.FieldTypeString}, {Name: "Owner", Type: metadata.FieldTypeString}, {Name: "Closed", Type: metadata.FieldTypeBool},
		{Name: "Mobile", Type: metadata.FieldTypeString, PII: true}, {Name: "Home", Type: metadata.FieldTypeString, PII: true},
		{Name: "Work", Type: metadata.FieldTypeString, PII: true},
	}}
	for _, name := range []string{"Mobile", "Home", "Work"} {
		claim.Indexes = append(claim.Indexes, metadata.IndexSpec{Fields: []string{name}, Normalizer: metadata.IndexNormalizerPhoneLast10})
	}
	server := newSubmitTestServerOn(t, db, []*metadata.Entity{task, claim})
	server.ops = newOperationLimiter()
	save := func(entity *metadata.Entity, id uuid.UUID, fields map[string]any) {
		t.Helper()
		if err := db.Upsert(t.Context(), entity.Name, id, fields, entity); err != nil {
			t.Fatal(err)
		}
	}
	first, other, denied, invalid := uuid.New(), uuid.New(), uuid.New(), uuid.New()
	firstFields := map[string]any{"Label": "state", "Owner": "own", "Phone": "+7 (901) 222-01-00", "Alternate": "8 (902) 333-44-00"}
	save(task, first, firstFields)
	save(task, other, map[string]any{"Owner": "own", "Phone": "9035556600"})
	save(task, denied, map[string]any{"Owner": "other", "Phone": "9035556600"})
	save(task, invalid, map[string]any{"Owner": "own", "Phone": "abc9035556600"})

	for _, c := range []struct {
		label, owner, mobile, home, work string
		closed                           bool
	}{
		{"first-match", "own", "89012220100", "", "", false},
		{"second-match", "own", "", "", "9023334400", false},
		{"hidden-claim", "other", "", "9012220100", "", false},
		{"closed-claim", "own", "", "9023334400", "", true},
		{"other-claim", "own", "9035556600", "", "", false},
	} {
		save(claim, uuid.New(), map[string]any{
			"Label": c.label, "Owner": c.owner, "Closed": c.closed, "Mobile": c.mobile, "Home": c.home, "Work": c.work,
		})
	}

	router := chi.NewRouter()
	server.Mount(router)
	send := func(user *auth.User, method, path, token string, fields url.Values) *httptest.ResponseRecorder {
		t.Helper()
		req := httptest.NewRequest(method, path, strings.NewReader(fields.Encode()))
		req.Header.Set("Content-Type", "application/x-www-form-urlencoded")
		req.AddCookie(&http.Cookie{Name: "onebase_session", Value: "saved-phone-test"})
		if token != "" {
			req.Header.Set(managedFormInstanceHeader, token)
		}
		out := httptest.NewRecorder()
		router.ServeHTTP(out, req.WithContext(auth.ContextWithUser(req.Context(), user)))
		if strings.Contains(out.Body.String(), "9012220100") || strings.Contains(out.Body.String(), "9023334400") ||
			strings.Contains(out.Body.String(), "901) 222-01-00") {
			t.Fatal("phone appeared in a form response")
		}
		return out
	}
	formInstance := regexp.MustCompile(`data-ob-form-instance="([^"]+)"`)
	formURL := "/ui/document/TaskMatch/"
	event := func(user *auth.User, id, token, kind, element string) formEventResponse {
		t.Helper()
		res := send(user, http.MethodPost, formURL+"form-event", token, url.Values{
			"_id": {id}, "_kind": {"object"}, "_event": {kind}, "_element": {element}, "_version": {"1"},
			"Phone": {"9035556600"},
		})
		return decodeFormEventResponse(t, res.Body.Bytes())
	}

	for _, role := range []string{"ОператорКЦ", "ОператорМессенджера"} {
		t.Run(role, func(t *testing.T) {
			policy := auth.RowPolicy{Field: "Owner", Op: "eq", Value: auth.RowValue{Literal: "own"}}
			taskRights, claimRights := []string{"read"}, []string{"read"}
			if role == "ОператорКЦ" {
				taskRights, claimRights = append(taskRights, "disclose"), append(claimRights, "disclose")
			}
			user := &auth.User{ID: "operator-" + role, Login: "operator", Roles: []*auth.Role{{
				Name: role, Permissions: auth.Permission{
					Documents: map[string][]string{task.Name: taskRights, claim.Name: claimRights},
					RowAccess: auth.RowAccess{Documents: map[string]auth.RowPolicies{task.Name: {"read": policy}, claim.Name: {"read": policy}}},
				},
			}}}
			page := send(user, http.MethodGet, formURL+first.String(), "", nil)
			tokens := formInstance.FindStringSubmatch(page.Body.String())
			if page.Code != http.StatusOK || len(tokens) != 2 || !strings.Contains(page.Body.String(), "first-match|second-match|") {
				t.Fatalf("saved GET: HTTP %d, token=%t, result=%t", page.Code, len(tokens) == 2, strings.Contains(page.Body.String(), "first-match|second-match|"))
			}
			if value := event(user, first.String(), tokens[1], string(metadata.FormEventOnOpen), ""); !value.OK || value.Values["Результат"] != "first-match|second-match|" {
				t.Fatalf("saved opening event: result %q, error %q", value.Values["Результат"], value.Error)
			}
			for _, invalidEvent := range []struct{ id, token, kind, element string }{
				{other.String(), tokens[1], string(metadata.FormEventOnOpen), ""},
				{first.String(), "", string(metadata.FormEventOnOpen), ""},
				{first.String(), tokens[1], string(metadata.FormEventOnClick), "Проверка"},
			} {
				if result := event(user, invalidEvent.id, invalidEvent.token, invalidEvent.kind, invalidEvent.element); result.OK {
					t.Fatal("lookup escaped its saved opening scope")
				}
			}
			if unreadable := send(user, http.MethodGet, formURL+denied.String(), "", nil); unreadable.Code != http.StatusForbidden {
				t.Fatalf("foreign source: HTTP %d", unreadable.Code)
			}
			if empty := send(user, http.MethodGet, formURL+invalid.String(), "", nil); empty.Code != http.StatusOK ||
				strings.Contains(empty.Body.String(), "other-claim|") || strings.Contains(empty.Body.String(), "first-match|") {
				t.Fatalf("invalid saved phone: HTTP %d", empty.Code)
			}
			revoked := cloneRecord(firstFields)
			revoked["Label"] = "revoked"
			save(task, first, revoked)
			if result := event(user, first.String(), tokens[1], string(metadata.FormEventOnOpen), ""); result.OK {
				t.Fatal("lookup survived the saved task read hook denial")
			}
			save(task, first, firstFields)
		})
	}
}
