package ui

import (
	"context"
	"fmt"
	"net/http"
	"net/http/httptest"
	"net/url"
	"strings"
	"testing"

	"github.com/go-chi/chi/v5"

	"github.com/ivantit66/onebase/internal/metadata"
)

// conceal: true — значение поля объекта не уходит в браузер, пока форма его не
// раскроет (reveal_when по реквизитам формы), как ПоказатьЛичныеДанные в 1С.
// Обработчики видят настоящее значение, запись формы его не затирает. Всё —
// через публичные обработчики: GET карточки, form-event, POST записи.
func TestManagedFormConcealedField(t *testing.T) {
	const secret = "ул. Тайная, д. 7, тел. +7 900 000-00-00"
	srv, ent := setupManagedEventsServer(t, `
Процедура ПриОткрытииФормы()
	Объект.Маской = Истина;
	Объект.Длина = СтрДлина(Объект.Наименование);
КонецПроцедуры

Процедура ПоказатьНажатие()
	Объект.Маской = Ложь;
КонецПроцедуры
`, map[metadata.FormEventType]string{metadata.FormEventOnOpen: "ПриОткрытииФормы"}, []*metadata.FormElement{
		{Kind: metadata.FormElementField, Name: "ПолеНаим", DataPath: "Объект.Наименование", Conceal: true, RevealWhen: "НЕ Маской"},
		{Kind: metadata.FormElementButton, Name: "Показать", Handlers: map[metadata.FormEventType]string{metadata.FormEventOnClick: "ПоказатьНажатие"}},
	})
	ent.Forms[0].Attributes = []*metadata.FormAttribute{{Name: "Маской", TypeRef: "bool"}, {Name: "Длина", TypeRef: "number"}}
	id := insertContragent(t, srv, ent, secret)

	// 1. Открытие: значения нет в разметке, поле отключено (не отправляется).
	rec := executeFormEditGET(t, srv, ent, id)
	if rec.Code != http.StatusOK {
		t.Fatalf("formEdit: %d", rec.Code)
	}
	// Наименование здесь ещё и представление записи (заголовок вкладки) — его
	// conceal не прячет; проверяем сам контрол поля.
	if strings.Contains(rec.Body.String(), `value="`+secret) {
		t.Fatal("скрытое значение попало в поле формы")
	}
	if !strings.Contains(rec.Body.String(), "data-ob-concealed") {
		t.Fatal("скрытое поле не помечено и не отключено")
	}

	// 2. ПриОткрытии: обработчик видит настоящее значение, ответ его не несёт.
	open := url.Values{"_id": {id.String()}, "_element": {""}, "_event": {string(metadata.FormEventOnOpen)}}
	resp := decodeFormEventResponse(t, executeFormEvent(t, srv, ent, open).Body.Bytes())
	if !resp.OK {
		t.Fatalf("ПриОткрытии: %q", resp.Error)
	}
	if v := strings.TrimSpace(anyString(valueCI(resp.Values, "Наименование"))); v != "" {
		t.Fatalf("ответ ПриОткрытии раскрыл значение: %q (values=%v)", v, resp.Values)
	}
	if l := anyString(valueCI(resp.Values, "Длина")); l == "" || l == "0" {
		t.Fatalf("обработчик не видел настоящего значения: Длина=%q", l)
	}

	// 3. «Показать» снимает флаг — ответ несёт настоящее значение.
	show := url.Values{"_id": {id.String()}, "_element": {"Показать"}, "_event": {string(metadata.FormEventOnClick)}, "Маской": {"true"}}
	resp = decodeFormEventResponse(t, executeFormEvent(t, srv, ent, show).Body.Bytes())
	if !resp.OK {
		t.Fatalf("Показать: %q", resp.Error)
	}
	if v := anyString(valueCI(resp.Values, "Наименование")); v != secret {
		t.Fatalf("после раскрытия значение не пришло: %q", v)
	}

	// 4. Запись формы без скрытого поля (оно отключено) значение не затирает.
	req := httptest.NewRequest("POST", "/ui/catalog/"+ent.Name+"/"+id.String(), strings.NewReader(url.Values{"_action": {""}}.Encode()))
	req.Header.Set("Content-Type", "application/x-www-form-urlencoded")
	rctx := chi.NewRouteContext()
	rctx.URLParams.Add("entity", ent.Name)
	rctx.URLParams.Add("id", id.String())
	req = req.WithContext(context.WithValue(req.Context(), chi.RouteCtxKey, rctx))
	srv.submitEdit(httptest.NewRecorder(), req)
	row, err := srv.store.GetByID(context.Background(), ent.Name, id, ent)
	if err != nil {
		t.Fatal(err)
	}
	if row["Наименование"] != secret {
		t.Fatalf("запись формы затёрла скрытое значение: %v", row["Наименование"])
	}
}

func valueCI(values map[string]any, name string) any {
	for k, v := range values {
		if strings.EqualFold(k, name) {
			return v
		}
	}
	return nil
}

func anyString(v any) string {
	if v == nil {
		return ""
	}
	if s, ok := v.(string); ok {
		return s
	}
	return strings.TrimSpace(fmt.Sprint(v))
}
