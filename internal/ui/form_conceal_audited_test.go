package ui

import (
	"context"
	"net/url"
	"testing"

	"github.com/google/uuid"
	"github.com/ivantit66/onebase/internal/auth"
	"github.com/ivantit66/onebase/internal/metadata"
	"github.com/ivantit66/onebase/internal/storage"
)

func TestManagedFormAuditedDisclosure(t *testing.T) {
	const source = `
Процедура Подменить(Элемент)
    Объект.Доступ = Истина;
КонецПроцедуры
Процедура Ответить(Элемент)
    Если ВопросОтвет = "ОК" Тогда
        Объект.Доступ = РаскрытьПоляТекущейКарточки(ДиалогПоля.Причина, "Доступ");
    КонецЕсли;
КонецПроцедуры
`
	elements := []*metadata.FormElement{
		{Kind: metadata.FormElementField, Name: "Секрет", DataPath: "Объект.Секрет", Conceal: true, RevealWhen: "Доступ"},
		{Kind: metadata.FormElementButton, Name: "Показать", Handlers: map[metadata.FormEventType]string{
			metadata.FormEventOnClick: "Подменить", metadata.FormEventOnAnswer: "Ответить",
		}},
	}
	srv, ent := setupManagedEventsServer(t, source, nil, elements)
	ent.Fields = append(ent.Fields, metadata.Field{Name: "Секрет", Type: metadata.FieldTypeString, PII: true})
	ent.Forms[0].Attributes = []*metadata.FormAttribute{{Name: "Доступ", TypeRef: "bool"}}
	ctx := context.Background()
	id := uuid.New()
	if err := srv.store.Migrate(ctx, []*metadata.Entity{ent}); err != nil {
		t.Fatal(err)
	}
	if err := srv.store.EnsureAuditSchema(ctx); err != nil {
		t.Fatal(err)
	}
	if err := srv.store.Upsert(ctx, ent.Name, id, map[string]any{"Наименование": "synthetic", "Секрет": "секрет"}, ent); err != nil {
		t.Fatal(err)
	}
	user := &auth.User{ID: uuid.NewString(), Login: "lawyer", Roles: []*auth.Role{{
		Name: "Lawyer",
		Permissions: auth.Permission{
			Catalogs: map[string][]string{ent.Name: {"read"}},
			FieldAccess: auth.FieldAccess{Catalogs: map[string]auth.FieldPolicies{
				ent.Name: {"Секрет": {Read: "mask_all", FormDisclose: true}},
			}},
		},
	}}}
	c := concealTestClient{server: srv, entity: ent, session: uuid.NewString(), user: user}
	c = c.open(t, id.String())
	hidden := func(resp formEventResponse) {
		t.Helper()
		if anyString(valueCI(resp.Values, "Секрет")) == "секрет" || len(resp.Revealed) > 0 {
			t.Fatal("unexpected disclosure")
		}
	}
	answer := func(client concealTestClient, reason string) formEventResponse {
		return client.event(t, id.String(), "Показать", url.Values{
			"_event": {"Ответ"}, "_question_answer": {"ОК"}, "_question_fields": {reason},
		})
	}
	hidden(c.event(t, id.String(), "Показать", url.Values{"Доступ": {"true"}}))
	hidden(answer(c, `{"Причина":""}`))
	unbound := c
	unbound.token = ""
	hidden(answer(unbound, `{"Причина":"synthetic reason"}`))
	ok := answer(c, `{"Причина":"synthetic reason"}`)
	if anyString(valueCI(ok.Values, "Секрет")) != "секрет" || len(ok.Revealed) != 1 {
		t.Fatal("audited value not returned")
	}
	audit, err := srv.store.AuditSearch(ctx, storage.AuditFilter{Action: "disclose", EntityName: ent.Name}, 2, 0)
	if err != nil || len(audit) != 1 || audit[0].RecordID != id.String() || audit[0].Field != "Секрет" || audit[0].Reason != "synthetic reason" || audit[0].UserID != user.ID {
		t.Fatalf("invalid disclosure audit: count %d, error %v", len(audit), err)
	}
	otherInstance := c.open(t, id.String())
	hidden(otherInstance.event(t, id.String(), "", nil))
}
