package ui

import (
	"context"
	"regexp"
	"testing"

	"github.com/google/uuid"

	"github.com/ivantit66/onebase/internal/metadata"
)

// hint_from через публичный GET формы: подсказка кнопки и надписи берётся из
// значения реквизита, пустое значение — статический hint; разметка несёт
// data-ob-hint-from, по которому клиент обновляет подсказку после события.
// Заодно — у кнопки теперь есть и статический hint (раньше title был только
// у горячей клавиши, и hint кнопки не показывался вовсе).
func TestManagedForm_HintFromAttribute(t *testing.T) {
	srv, ent := setupManagedEventsServer(t, ``, nil, []*metadata.FormElement{
		{Kind: metadata.FormElementButton, Name: "КнПримечание", Hint: "Примечание направления", HintFrom: "Наименование"},
		{Kind: metadata.FormElementLabel, Name: "НдПримечание", HintFrom: "Наименование"},
		{Kind: metadata.FormElementButton, Name: "КнПросто", Hint: "Статическая подсказка"},
	})
	ctx := context.Background()
	render := func(name string) string {
		t.Helper()
		id := uuid.New()
		if err := srv.store.Upsert(ctx, ent.Name, id, map[string]any{"Наименование": name}, ent); err != nil {
			t.Fatal(err)
		}
		return renderObjectFormGET(t, srv, ent, id)
	}
	titleOf := func(body, el string) string {
		t.Helper()
		m := regexp.MustCompile(`data-ob-el="` + el + `"[^>]*?title="([^"]*)"|title="([^"]*)"[^>]*?data-ob-el="` + el + `"`).FindStringSubmatch(body)
		if m == nil {
			return ""
		}
		return m[1] + m[2]
	}
	body := render("Только после 18:00")
	if got := titleOf(body, "КнПримечание"); got != "Только после 18:00" {
		t.Errorf("кнопка: title %q, ждали значение реквизита", got)
	}
	if got := titleOf(body, "НдПримечание"); got != "Только после 18:00" {
		t.Errorf("надпись: title %q, ждали значение реквизита", got)
	}
	if got := titleOf(body, "КнПросто"); got != "Статическая подсказка" {
		t.Errorf("статический hint кнопки не показан: %q", got)
	}
	if !regexp.MustCompile(`data-ob-el="КнПримечание"[^>]*data-ob-hint-from="Наименование"`).MatchString(body) {
		t.Error("нет data-ob-hint-from — клиент не обновит подсказку после события")
	}
	if got := titleOf(render(""), "КнПримечание"); got != "Примечание направления" {
		t.Errorf("пустой реквизит: title %q, ждали статический hint", got)
	}
}
