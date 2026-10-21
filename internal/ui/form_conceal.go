package ui

import (
	"net/http"
	"strconv"
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
func (s *Server) concealEventValues(form *metadata.FormModule, entity *metadata.Entity, values map[string]any) []string {
	fields := concealedFormFields(form)
	if len(fields) == 0 {
		return nil
	}
	var revealed []string
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
			revealed = append(revealed, name)
			continue
		}
		for k := range values {
			if strings.EqualFold(k, name) {
				values[k] = ""
			}
		}
	}
	return revealed
}

// unrevealedConcealedFields — скрытые поля, присланное значение которых
// принимать нельзя: reveal_when ложно по присланным реквизитам формы (или его
// нет). Присланное значение такого поля — подделка или остаток старой разметки;
// база в обоих случаях не меняется.
func (s *Server) unrevealedConcealedFields(r *http.Request, entity *metadata.Entity, form *metadata.FormModule) map[string]bool {
	fields := concealedFormFields(form)
	if len(fields) == 0 || r == nil {
		return nil
	}
	header := map[string]any{}
	for k, v := range r.PostForm {
		if len(v) > 0 {
			header[k] = v[0]
		}
	}
	// Реквизиты формы приходят строками: булево и число приводим по типу, иначе
	// «false» читалось бы непустой строкой, то есть истиной.
	for _, attr := range form.Attributes {
		if attr == nil {
			continue
		}
		raw, ok := header[attr.Name].(string)
		if !ok {
			continue
		}
		switch strings.ToLower(attr.TypeRef) {
		case "bool", "boolean":
			header[attr.Name] = raw == "true"
		case "number":
			if n, err := strconv.ParseFloat(raw, 64); err == nil {
				header[attr.Name] = n
			}
		}
	}
	row := compose.Row(managedFormHeaderValues(entity, header))
	out := map[string]bool{}
	for name, expr := range fields {
		reveal := false
		if expr != "" && s.interp != nil {
			ok, err := newInterpEvaluator(s.interp).EvalBool(expr, row)
			reveal = err == nil && ok
		}
		if !reveal {
			out[name] = true
		}
	}
	return out
}
