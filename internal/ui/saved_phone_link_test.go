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
	"github.com/google/uuid"
	"github.com/ivantit66/onebase/internal/auth"
	"github.com/ivantit66/onebase/internal/dbtest"
	"github.com/ivantit66/onebase/internal/metadata"
	"github.com/ivantit66/onebase/internal/storage"
	"github.com/stretchr/testify/require"
)

func TestSavedPhoneLinkThroughManagedForm(t *testing.T) {
	dbtest.ForEachDialect(t, testSavedPhoneLinkThroughManagedForm)
}

func testSavedPhoneLinkThroughManagedForm(t *testing.T, db *storage.DB) {
	elementEvents := func(event metadata.FormEventType, proc string) map[metadata.FormEventType]string {
		return map[metadata.FormEventType]string{event: proc}
	}
	sourceForm := &metadata.FormModule{
		Name: "TaskForm", EntityName: "TaskLink", Kind: "object", HeaderInline: true, LayoutKind: metadata.FormLayoutManaged,
		SavedPhoneMatch: &metadata.SavedPhoneMatchSpec{
			SourceFields: []string{"Phone"}, TargetDocument: "ClaimLink", TargetFields: []string{"Phone"}, LinkCommand: "Claim",
		},
		Elements: []*metadata.FormElement{
			{Kind: metadata.FormElementButton, Name: "Claim", Placement: "header_end", Handlers: elementEvents(metadata.FormEventOnClick, "Перейти")},
			{Kind: metadata.FormElementButton, Name: "Other", Handlers: elementEvents(metadata.FormEventOnClick, "Перейти")},
			{Kind: metadata.FormElementGroupBox, Name: "LoadLink", Handlers: elementEvents(metadata.FormEventAfterOpen, "ПоказатьСсылку")},
		},
		ProgramAST: mustParse(t, `
Функция Найти()
	З = Новый Запрос;
	З.Текст = "ВЫБРАТЬ З.Ссылка КАК Ссылка, З.Label КАК Метка ИЗ Документ.ClaimLink КАК З
		|ГДЕ СОХРАНЕННЫЙТЕЛЕФОНСОВПАДАЕТ(З) И З.Closed = ЛОЖЬ
		|УПОРЯДОЧИТЬ ПО Метка";
	Для Каждого Строка Из З.Выполнить() Цикл
		Возврат Строка;
	КонецЦикла;
	Возврат Неопределено;
КонецФункции
Процедура ПоказатьСсылку()
	Результат = Найти();
	Если Результат <> Неопределено Тогда
		ПоказатьСсылкуНайденнойЗаявки(Результат.Ссылка, Результат.Метка);
	КонецЕсли;
КонецПроцедуры
Процедура Перейти()
	Результат = Найти();
	Если Результат = Неопределено Тогда
		ОткрытьНайденнуюЗаявку(Неопределено);
	Иначе
		ОткрытьНайденнуюЗаявку(Результат.Ссылка);
	КонецЕсли;
КонецПроцедуры
`),
	}
	compact := *sourceForm
	compact.Name = "CompactTaskForm"
	compact.Density = metadata.FormDensityCompact
	compact.Roles = []string{"ОператорМессенджера"}
	targetForm := &metadata.FormModule{
		Name: "ClaimForm", EntityName: "ClaimLink", Kind: "object", LayoutKind: metadata.FormLayoutManaged, InstanceContext: true,
		Handlers: elementEvents(metadata.FormEventOnOpen, "ПриОткрытии"),
		Elements: []*metadata.FormElement{
			{Kind: metadata.FormElementField, Name: "Phone", DataPath: "Объект.Phone"},
			{Kind: metadata.FormElementField, Name: "Label", DataPath: "Объект.Label"},
		},
		ProgramAST: mustParse(t, `
Процедура ПриОткрытии()
	Сообщить("объект:" + Объект.Phone);
	Сообщить("ссылка:" + Объект.Ссылка.Phone);
	СохраненнаяЗаявка = Объект.Ссылка.ПолучитьОбъект();
	Сообщить("сохранено:" + СохраненнаяЗаявка.Phone);
	З = Новый Запрос;
	З.Текст = "ВЫБРАТЬ З.Phone КАК Телефон ИЗ Документ.ClaimLink КАК З ГДЕ З.Ссылка = &Ид";
	З.УстановитьПараметр("Ид", Объект.Ссылка);
	Для Каждого Строка Из З.Выполнить() Цикл
		Сообщить("запрос:" + Строка.Телефон);
	КонецЦикла;
КонецПроцедуры
`),
	}
	task := &metadata.Entity{Name: "TaskLink", Kind: metadata.KindDocument, Fields: []metadata.Field{
		{Name: "Phone", Type: metadata.FieldTypeString, PII: true},
	}, Forms: []*metadata.FormModule{&compact, sourceForm}}
	claim := &metadata.Entity{Name: "ClaimLink", Kind: metadata.KindDocument, Fields: []metadata.Field{
		{Name: "Phone", Type: metadata.FieldTypeString, PII: true, PIIOpenWhen: "НЕ Closed"},
		{Name: "Label", Type: metadata.FieldTypeString},
		{Name: "Owner", Type: metadata.FieldTypeString},
		{Name: "Closed", Type: metadata.FieldTypeBool},
	}, Forms: []*metadata.FormModule{targetForm}, Indexes: []metadata.IndexSpec{{Fields: []string{"Phone"}, Normalizer: metadata.IndexNormalizerPhoneLast10}}}
	server := newSubmitTestServerOn(t, db, []*metadata.Entity{task, claim})
	server.ops = newOperationLimiter()
	router := chi.NewRouter()
	server.Mount(router)
	taskID, claimID, otherID := uuid.New(), uuid.New(), uuid.New()
	const number = "79991234567"
	const label = "У клиента есть незакрытая заявка № 0001 — бренд <Бренд> & ещё 1"
	require.NoError(t, db.Upsert(t.Context(), task.Name, taskID, map[string]any{"Phone": number}, task))
	require.NoError(t, db.Upsert(t.Context(), task.Name, otherID, map[string]any{"Phone": "79993334455"}, task))
	saveClaim := func(owner, caption string) {
		t.Helper()
		require.NoError(t, db.UpsertVersioned(t.Context(), claim.Name, claimID,
			map[string]any{"Phone": number, "Label": caption, "Owner": owner, "Closed": false}, claim, nil))
	}

	tokenPattern := regexp.MustCompile(`data-ob-form-instance="([^"]+)"`)
	for _, role := range []string{"ОператорКЦ", "ОператорМессенджера"} {
		t.Run(role, func(t *testing.T) {
			saveClaim("own", label)
			policy := auth.RowPolicy{Field: "Owner", Op: "eq", Value: auth.RowValue{Literal: "own"}}
			claimRights := []string{"read", "write"}
			if role == "ОператорКЦ" {
				claimRights = append(claimRights, "disclose")
			}
			user := &auth.User{ID: role, Login: role, Roles: []*auth.Role{{
				Name: role, Permissions: auth.Permission{
					Documents: map[string][]string{task.Name: {"read"}, claim.Name: claimRights},
					RowAccess: auth.RowAccess{Documents: map[string]auth.RowPolicies{claim.Name: {"read": policy}}},
				},
			}}}
			send := func(method, path, token, proof string, fields url.Values) *httptest.ResponseRecorder {
				t.Helper()
				req := httptest.NewRequest(method, path, strings.NewReader(fields.Encode()))
				req.Header.Set("Content-Type", "application/x-www-form-urlencoded")
				req.Header.Set(managedFormInstanceHeader, token)
				req.Header.Set(savedPhoneLinkHeader, proof)
				req.AddCookie(&http.Cookie{Name: "onebase_session", Value: "saved-phone-link-test"})
				out := httptest.NewRecorder()
				router.ServeHTTP(out, req.WithContext(auth.ContextWithUser(req.Context(), user)))
				return out
			}
			run := func(entity *metadata.Entity, id uuid.UUID, event metadata.FormEventType, element, token, proof string) formEventResponse {
				t.Helper()
				fields := url.Values{
					"_id": {id.String()}, "_kind": {"object"}, "_event": {string(event)}, "_element": {element},
					"ИдНезакрытойЗаявки": {otherID.String()},
				}
				if entity == task {
					fields.Set("Phone", "79993334455")
				}
				res := send(http.MethodPost, "/ui/document/"+entity.Name+"/form-event", token, proof, fields)
				require.NotContains(t, res.Body.String(), number)
				return decodeFormEventResponse(t, res.Body.Bytes())
			}
			taskPage := func(id uuid.UUID) string {
				t.Helper()
				page := send(http.MethodGet, "/ui/document/"+task.Name+"/"+id.String(), "", "", nil)
				require.Equal(t, http.StatusOK, page.Code)
				body := page.Body.String()
				require.Contains(t, body, `data-ob-saved-phone-link data-ob-fire-click="Claim" hidden`)
				require.NotContains(t, body, "У клиента есть незакрытая заявка")
				if role == "ОператорМессенджера" {
					require.Contains(t, body, "ob-density-compact")
				}
				match := tokenPattern.FindStringSubmatch(body)
				require.Len(t, match, 2)
				return match[1]
			}
			token := taskPage(taskID)
			open := func(token string, id uuid.UUID) formEventResponse {
				return run(task, id, metadata.FormEventAfterOpen, "LoadLink", token, "")
			}
			click := func(token, proof, element string) formEventResponse {
				return run(task, taskID, metadata.FormEventOnClick, element, token, proof)
			}
			opened := open(token, taskID)
			require.True(t, opened.OK, opened.Error)
			require.NotNil(t, opened.SavedPhoneLink)
			require.Equal(t, label, opened.SavedPhoneLink.Label)
			proof := opened.SavedPhoneLink.Proof
			require.NotEmpty(t, proof)
			require.Nil(t, open(taskPage(otherID), otherID).SavedPhoneLink)
			require.Nil(t, click(token, "", "Claim").Navigation)
			require.Nil(t, click(token, proof+"x", "Claim").Navigation)
			require.Nil(t, click(taskPage(taskID), proof, "Claim").Navigation)
			require.Nil(t, click(token, proof, "Other").Navigation)

			navigate := click(token, proof, "Claim")
			require.True(t, navigate.OK, navigate.Error)
			require.NotNil(t, navigate.Navigation)
			require.True(t, navigate.Navigation.OpenTab)
			plainURL := "/ui/document/" + claim.Name + "/" + claimID.String()
			require.True(t, strings.HasPrefix(navigate.Navigation.URL, plainURL+"?"+formOpenSavedPhoneQuery+"="))
			protected := send(http.MethodGet, navigate.Navigation.URL, "", "", nil)
			require.Equal(t, http.StatusOK, protected.Code, protected.Body.String())
			require.NotContains(t, protected.Body.String(), number)
			protectedToken := tokenPattern.FindStringSubmatch(protected.Body.String())
			require.Len(t, protectedToken, 2)
			result := run(claim, claimID, metadata.FormEventOnOpen, "", protectedToken[1], "")
			require.True(t, result.OK, result.Error)
			require.NotEqual(t, number, result.Values["Phone"])
			require.Len(t, result.Messages, 4, result.Messages)
			for i, prefix := range []string{"объект:", "ссылка:", "сохранено:", "запрос:"} {
				require.True(t, strings.HasPrefix(result.Messages[i], prefix), result.Messages[i])
				require.NotEqual(t, prefix, result.Messages[i], "expected masked value")
			}
			require.Equal(t, http.StatusForbidden, send(http.MethodGet, navigate.Navigation.URL, "", "", nil).Code)
			direct := send(http.MethodGet, plainURL, "", "", nil)
			require.Equal(t, http.StatusOK, direct.Code)
			require.Contains(t, direct.Body.String(), number)

			now := time.Now()
			server.concealClock = func() time.Time { return now.Add(managedFormInstanceTTL + time.Minute) }
			expired := run(claim, claimID, metadata.FormEventOnOpen, "", protectedToken[1], "")
			require.False(t, expired.OK)
			require.Empty(t, expired.Values)
			server.concealClock = nil

			late := click(token, proof, "Claim")
			require.NotNil(t, late.Navigation)
			saveClaim("own", "Новый бренд")
			require.Equal(t, http.StatusForbidden, send(http.MethodGet, late.Navigation.URL, "", "", nil).Code)
			require.Nil(t, click(token, proof, "Claim").Navigation)
			updated := open(token, taskID)
			require.NotNil(t, updated.SavedPhoneLink)
			saveClaim("other", "Недоступная заявка")
			require.Nil(t, click(token, updated.SavedPhoneLink.Proof, "Claim").Navigation)
		})
	}
}
