package ui

import (
	"fmt"
	"strings"

	"github.com/ivantit66/onebase/internal/dsl/interpreter"
	"github.com/ivantit66/onebase/internal/metadata"
)

// addPageChangeContext кладёт в обработчик события ПриСменеСтраницы имя
// открытой страницы — переменную ТекущаяСтраница (параметр ТекущаяСтраница
// ПриСменеСтраницы панели 1С). Имя приходит из браузера, поэтому принимается
// только страница ТОГО ЖЕ набора СтраницыФормы: чужое или выдуманное имя —
// отказ, а не произвольная строка в обработчике.
func addPageChangeContext(eventName, page string, target browserFormEventTarget, vars map[string]any) error {
	if eventName != string(metadata.FormEventOnPageChange) {
		return nil
	}
	if target.element == nil || target.element.Kind != metadata.FormElementPages {
		return fmt.Errorf("событие %q отправляет только набор страниц", eventName)
	}
	page = strings.TrimSpace(page)
	if page != "" {
		for _, child := range target.element.Children {
			if child != nil && child.Kind == metadata.FormElementPage && child.Name == page {
				vars["ТекущаяСтраница"] = page
				vars["CurrentPage"] = page
				return nil
			}
		}
	}
	return fmt.Errorf("страница %q не входит в набор страниц %q", page, target.element.Name)
}

// installFormPageCommand limits navigation to a unique page of the open form.
func installFormPageCommand(form *metadata.FormModule, vars map[string]any, page *string) {
	activate := interpreter.BuiltinFunc(func(args []any, _ string, _ int) (any, error) {
		if len(args) != 1 {
			return nil, fmt.Errorf("ОткрытьСтраницуФормы: укажите имя страницы")
		}
		name, ok := args[0].(string)
		if !ok || strings.TrimSpace(name) == "" {
			return nil, fmt.Errorf("ОткрытьСтраницуФормы: укажите имя страницы")
		}
		matches := 0
		form.Walk(func(el *metadata.FormElement) bool {
			if el.Kind == metadata.FormElementPage && el.Name == name {
				matches++
			}
			return true
		})
		if matches != 1 {
			return nil, fmt.Errorf("ОткрытьСтраницуФормы: страница %q отсутствует или её имя неоднозначно", name)
		}
		*page = name
		return nil, nil
	})
	vars["ОткрытьСтраницуФормы"], vars["ActivateFormPage"] = activate, activate
}
