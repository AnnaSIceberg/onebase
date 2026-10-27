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

// formHintFromValues — «реквизит → текст» для всех hint_from формы по
// значениям после обработчика события. Пустое значение тоже отдаётся: клиент
// по нему возвращает статический hint.
func formHintFromValues(form *metadata.FormModule, entity *metadata.Entity, values map[string]any) map[string]string {
	if form == nil {
		return nil
	}
	var attrs []string
	walkFormHintFrom(form.Elements, &attrs)
	if len(attrs) == 0 {
		return nil
	}
	header := managedFormHeaderValues(entity, values)
	out := make(map[string]string, len(attrs))
	for _, attr := range attrs {
		v, ok := values[attr]
		if !ok {
			v, ok = lookupFold(header, attr)
		}
		if !ok || v == nil {
			out[attr] = ""
			continue
		}
		out[attr] = strings.TrimSpace(fmt.Sprint(v))
	}
	return out
}

func walkFormHintFrom(elements []*metadata.FormElement, out *[]string) {
	for _, el := range elements {
		if el == nil {
			continue
		}
		if attr := hintFromAttr(el); attr != "" {
			*out = append(*out, attr)
		}
		walkFormHintFrom(el.Children, out)
	}
}

func lookupFold(m map[string]any, key string) (any, bool) {
	if v, ok := m[key]; ok {
		return v, true
	}
	for k, v := range m {
		if strings.EqualFold(k, key) {
			return v, true
		}
	}
	return nil, false
}
