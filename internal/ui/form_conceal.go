package ui

import (
	"strings"

	"github.com/ivantit66/onebase/internal/metadata"
	"github.com/ivantit66/onebase/internal/report/compose"
)

// concealedFormFields — поля объекта, которые форма прячет от браузера
// (conceal: true у ПолеВвода), с условием раскрытия reveal_when.
func concealedFormFields(form *metadata.FormModule) map[string]string {
	if form == nil {
		return nil
	}
	out := map[string]string{}
	form.Walk(func(el *metadata.FormElement) bool {
		if el != nil && el.Conceal && el.DataPath != "" && strings.Count(el.DataPath, ".") <= 1 {
			name := dpFieldName(el.DataPath)
			if prev, ok := out[name]; ok && prev == "" {
				return true // уже есть размещение без условия раскрытия — строже
			}
			out[name] = strings.TrimSpace(el.RevealWhen)
		}
		return true
	})
	return out
}

// blankConcealedValues стирает значения скрытых полей перед отрисовкой формы:
// при открытии значение не раскрыто никогда — условие раскрытия вычисляет
// обработчик формы (обычно ПриОткрытии), и решает его ответ события.
func blankConcealedValues(form *metadata.FormModule, values any) {
	fields := concealedFormFields(form)
	if len(fields) == 0 {
		return
	}
	for name := range fields {
		switch m := values.(type) {
		case map[string]string:
			for k := range m {
				if strings.EqualFold(k, name) {
					m[k] = ""
				}
			}
		case map[string]any:
			for k := range m {
				if strings.EqualFold(k, name) {
					m[k] = ""
				}
			}
		}
	}
}

// concealEventValues прячет значения скрытых полей в ответе события формы,
// пока их reveal_when ложно. Ошибка в выражении — поле остаётся скрытым
// (fail closed): раскрыть данные из-за опечатки хуже, чем не показать их.
func (s *Server) concealEventValues(form *metadata.FormModule, entity *metadata.Entity, values map[string]any) {
	fields := concealedFormFields(form)
	if len(fields) == 0 {
		return
	}
	var row compose.Row
	ev := newInterpEvaluator(s.interp)
	if s.interp != nil {
		row = compose.Row(managedFormHeaderValues(entity, values))
	}
	for name, expr := range fields {
		reveal := false
		if expr != "" && s.interp != nil {
			ok, err := ev.EvalBool(expr, row)
			reveal = err == nil && ok
		}
		if reveal {
			continue
		}
		for k := range values {
			if strings.EqualFold(k, name) {
				values[k] = ""
			}
		}
	}
}
