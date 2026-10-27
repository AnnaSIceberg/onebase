package ui

import (
	"fmt"
	"html"
	"html/template"
	"strings"

	"github.com/ivantit66/onebase/internal/metadata"
)

// hintFromAttr — имя реквизита формы из hint_from («Форма.» допускается).
func hintFromAttr(el *metadata.FormElement) string {
	if el == nil {
		return ""
	}
	return strings.TrimPrefix(strings.TrimSpace(el.HintFrom), "Форма.")
}

// elHintText — подсказка элемента: значение реквизита hint_from, а если оно
// пусто (или ключа нет) — статический hint; hotKey дописывается в скобках.
func elHintText(values map[string]any, el *metadata.FormElement, hotKey string) string {
	text := ""
	if attr := hintFromAttr(el); attr != "" && values != nil {
		if v, ok := values[attr]; ok && v != nil {
			text = strings.TrimSpace(fmt.Sprint(v))
		}
	}
	if text == "" && el != nil {
		text = el.Hint
	}
	switch {
	case hotKey == "":
		return text
	case text == "":
		return hotKey
	default:
		return text + " (" + hotKey + ")"
	}
}

// elHintAttrs — атрибуты подсказки элемента. Элемент с hint_from несёт
// data-ob-hint-from (реквизит), data-ob-hint (статический hint) и
// data-ob-hint-key (горячая клавиша): клиент обновляет title после события
// формы, ничего не пересылая отдельно — значения реквизитов уже в ответе.
func elHintAttrs(ctx any, el *metadata.FormElement, hotKey string) template.HTMLAttr {
	if el == nil {
		return ""
	}
	m, _ := ctx.(map[string]any)
	values, _ := m["Values"].(map[string]any)
	if values == nil {
		if sv, ok := m["Values"].(map[string]string); ok {
			values = make(map[string]any, len(sv))
			for k, v := range sv {
				values[k] = v
			}
		}
	}
	var b strings.Builder
	if text := elHintText(values, el, hotKey); text != "" {
		b.WriteString(` title="` + html.EscapeString(text) + `"`)
	}
	if attr := hintFromAttr(el); attr != "" {
		b.WriteString(` data-ob-hint-from="` + html.EscapeString(attr) + `"`)
		if el.Hint != "" {
			b.WriteString(` data-ob-hint="` + html.EscapeString(el.Hint) + `"`)
		}
		if hotKey != "" {
			b.WriteString(` data-ob-hint-key="` + html.EscapeString(hotKey) + `"`)
		}
	}
	return template.HTMLAttr(b.String()) //nolint:gosec // G203: значения экранированы html.EscapeString
}
