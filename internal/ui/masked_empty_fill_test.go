package ui

import (
	"encoding/json"
	"net/http"
	"net/http/httptest"
	"net/url"
	"strings"
	"testing"

	"github.com/go-chi/chi/v5"
	"github.com/google/uuid"

	"github.com/ivantit66/onebase/internal/auth"
	"github.com/ivantit66/onebase/internal/metadata"
)

// Документ, созданный кодом и открытый для оформления, уже записан: при его
// сохранении защита «нельзя изменить то, что не видно» восстанавливала поля
// под маской из базы. Пустой телефон роль видит пустым — скрывать там нечего,
// — но и введённый ею номер молча отбрасывался. Пустое под маской заполнить
// можно; заполненное по-прежнему не меняется; hide не заполняется никогда.
func TestUI_SaveCard_MaskedEmptyFieldCanBeFilled(t *testing.T) {
	cases := []struct {
		name   string
		meta   func() *metadata.Entity
		policy auth.FieldPolicies
		stored string
		sent   string
		want   string
	}{
		{name: "mask_tail, пусто — заполняется", meta: uiClientEntity,
			policy: auth.FieldPolicies{"Телефон": {Read: "mask_tail", Keep: 4}}, stored: "", sent: "(903)222-33-44", want: "(903)222-33-44"},
		{name: "pii без политики (mask_all), пусто — заполняется", meta: piiClientEntity,
			stored: "", sent: "(903)222-33-44", want: "(903)222-33-44"},
		{name: "mask_all, заполнено — не меняется", meta: piiClientEntity,
			stored: "(916)111-22-33", sent: "(903)222-33-44", want: "(916)111-22-33"},
		{name: "mask_all, маска в ответ — не меняется", meta: piiClientEntity,
			stored: "(916)111-22-33", sent: "••••••", want: "(916)111-22-33"},
		{name: "mask_all, пусто, в ответ маска — звёздочки не пишутся", meta: piiClientEntity,
			stored: "", sent: "••••••", want: ""},
		{name: "hide, пусто — не заполняется", meta: uiClientEntity,
			policy: auth.FieldPolicies{"Телефон": {Read: "hide"}}, stored: "", sent: "(903)222-33-44", want: ""},
	}
	for _, tc := range cases {
		t.Run(tc.name, func(t *testing.T) {
			cat := tc.meta()
			s, ctx := newSubmitTestServer(t, []*metadata.Entity{cat})
			id := uuid.New()
			initial := map[string]any{"Наименование": "Иванов"}
			if tc.stored != "" {
				initial["Телефон"] = tc.stored
			}
			if err := s.store.Upsert(ctx, cat.Name, id, initial, cat); err != nil {
				t.Fatal(err)
			}
			router := chi.NewRouter()
			s.Mount(router)
			user := uiMaskUser([]string{"read", "write"}, tc.policy)
			form := url.Values{"Наименование": {"Петров"}, "Телефон": {tc.sent}}
			r := httptest.NewRequest(http.MethodPost, "/ui/catalog/"+url.PathEscape(cat.Name)+"/"+id.String(), strings.NewReader(form.Encode()))
			r.Header.Set("Content-Type", "application/x-www-form-urlencoded")
			r = r.WithContext(auth.ContextWithUser(r.Context(), user))
			w := httptest.NewRecorder()
			router.ServeHTTP(w, r)
			if w.Code != http.StatusSeeOther {
				t.Fatalf("сохранение карточки: ожидался 303, получено %d: %s", w.Code, w.Body.String())
			}
			row, err := s.store.GetByID(ctx, cat.Name, id, cat)
			if err != nil {
				t.Fatal(err)
			}
			if got := strings.TrimSpace(toStringOrEmpty(row["Телефон"])); got != tc.want {
				t.Fatalf("Телефон = %q, ожидалось %q", got, tc.want)
			}
			if row["Наименование"] != "Петров" {
				t.Fatalf("видимое поле должно обновиться, получено %v", row["Наименование"])
			}
		})
	}
}

func piiClientEntity() *metadata.Entity {
	cat := uiClientEntity()
	for i := range cat.Fields {
		if cat.Fields[i].Name == "Телефон" {
			cat.Fields[i].PII = true
		}
	}
	return cat
}

func toStringOrEmpty(v any) string {
	if v == nil {
		return ""
	}
	if s, ok := v.(string); ok {
		return s
	}
	return ""
}

// Номер, который пользователь только что набрал в поле под маской, ответ
// события формы возвращает как есть: маска на обратном пути подменяла его
// звёздочками, и следующее событие или запись присылали на сервер их — номер
// терялся, а с заполнением пустого поля в базу легли бы сами звёздочки.
// Значение из базы (клиент прислал маску) и поставленное обработчиком
// по-прежнему уходят маской.
func TestUI_FormEvent_EchoesTypedMaskedValue(t *testing.T) {
	cat := piiClientEntity()
	form := managedObjectForm(fieldEl("ПолеНаименование", "Объект.Наименование"), fieldEl("ПолеТелефон", "Объект.Телефон"),
		maskEventButton("КнПроверить", "КнПроверитьНажатие"), maskEventButton("КнПодменить", "КнПодменитьНажатие"))
	form.EntityName = cat.Name
	form.ProgramAST = mustParse(t, `
Процедура КнПроверитьНажатие()
КонецПроцедуры

Процедура КнПодменитьНажатие()
	Объект.Телефон = "(111)000-00-00";
КонецПроцедуры
`)
	cat.Forms = []*metadata.FormModule{form}
	s, ctx := newSubmitTestServer(t, []*metadata.Entity{cat})
	router := chi.NewRouter()
	s.Mount(router)
	user := uiMaskUser([]string{"read", "write"}, nil)

	event := func(id uuid.UUID, element, phone string) map[string]any {
		t.Helper()
		body := url.Values{"_id": {id.String()}, "_element": {element}, "_event": {"Нажатие"},
			"Наименование": {"Иванов"}, "Телефон": {phone}}
		r := httptest.NewRequest(http.MethodPost, "/ui/catalog/"+url.PathEscape(cat.Name)+"/form-event", strings.NewReader(body.Encode()))
		r.Header.Set("Content-Type", "application/x-www-form-urlencoded; charset=utf-8")
		r = r.WithContext(auth.ContextWithUser(r.Context(), user))
		w := httptest.NewRecorder()
		router.ServeHTTP(w, r)
		var resp struct {
			Values map[string]any `json:"values"`
			Error  string         `json:"error"`
		}
		if err := json.Unmarshal(w.Body.Bytes(), &resp); err != nil || resp.Error != "" {
			t.Fatalf("form-event: %d %s (%v)", w.Code, w.Body.String(), err)
		}
		return resp.Values
	}
	seed := func(phone string) uuid.UUID {
		t.Helper()
		id := uuid.New()
		fields := map[string]any{"Наименование": "Иванов"}
		if phone != "" {
			fields["Телефон"] = phone
		}
		if err := s.store.Upsert(ctx, cat.Name, id, fields, cat); err != nil {
			t.Fatal(err)
		}
		return id
	}

	empty := seed("")
	if got := event(empty, "КнПроверить", "(903)222-33-44")["Телефон"]; got != "(903)222-33-44" {
		t.Fatalf("набранный номер должен вернуться как есть, получено %v", got)
	}
	if got := event(empty, "КнПодменить", "(903)222-33-44")["Телефон"]; got == "(111)000-00-00" {
		t.Fatalf("значение, поставленное обработчиком, должно уйти маской, получено %v", got)
	}
	filled := seed("(916)111-22-33")
	if got := event(filled, "КнПроверить", "••••••")["Телефон"]; got == "(916)111-22-33" {
		t.Fatalf("значение из базы должно уйти маской, получено %v", got)
	}
}

// Поле под маской в управляемой форме помечено data-ob-protected и показывается
// точками, как пароль; у пользователя без маски пометки нет.
func TestUI_ManagedForm_ProtectedInputRenderedAsPassword(t *testing.T) {
	cat := piiClientEntity()
	form := managedObjectForm(fieldEl("ПолеНаименование", "Объект.Наименование"), fieldEl("ПолеТелефон", "Объект.Телефон"))
	form.EntityName = cat.Name
	cat.Forms = []*metadata.FormModule{form}
	s, ctx := newSubmitTestServer(t, []*metadata.Entity{cat})
	id := uuid.New()
	if err := s.store.Upsert(ctx, cat.Name, id, map[string]any{"Наименование": "Иванов", "Телефон": "(916)111-22-33"}, cat); err != nil {
		t.Fatal(err)
	}
	router := chi.NewRouter()
	s.Mount(router)
	page := func(user *auth.User) string {
		t.Helper()
		r := httptest.NewRequest(http.MethodGet, "/ui/catalog/"+url.PathEscape(cat.Name)+"/"+id.String(), nil)
		r = r.WithContext(auth.ContextWithUser(r.Context(), user))
		w := httptest.NewRecorder()
		router.ServeHTTP(w, r)
		if w.Code != http.StatusOK {
			t.Fatalf("GET формы: %d %s", w.Code, w.Body.String())
		}
		return w.Body.String()
	}
	masked := page(uiMaskUser([]string{"read", "write"}, nil))
	if !strings.Contains(masked, `name="Телефон" value="••••••" data-ob-protected`) {
		t.Fatalf("поле телефона под маской должно быть помечено data-ob-protected")
	}
	if strings.Contains(masked, "(916)111-22-33") {
		t.Fatal("настоящего номера не должно быть в разметке")
	}
	if strings.Contains(masked, `name="Наименование" value="Иванов" data-ob-protected`) {
		t.Fatal("обычное поле не должно помечаться")
	}
	full := page(uiMaskUser([]string{"read", "write"}, auth.FieldPolicies{"Телефон": {Read: "full"}}))
	if !strings.Contains(full, `name="Телефон" value="(916)111-22-33"`) || strings.Contains(full, `value="(916)111-22-33" data-ob-protected`) {
		t.Fatal("у пользователя без маски значение видно и поле не помечается")
	}
}

func maskEventButton(name, handler string) *metadata.FormElement {
	return &metadata.FormElement{
		Kind: metadata.FormElementButton, Name: name,
		Handlers: map[metadata.FormEventType]string{metadata.FormEventOnClick: handler},
	}
}
