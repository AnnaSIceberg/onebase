package ui

import (
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
