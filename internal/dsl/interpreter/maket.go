package interpreter

import (
	"strings"

	"github.com/ivantit66/onebase/internal/printform"
)

// Макет wraps a LayoutTemplate as a DSL-accessible object.
// DSL code uses it: Макет.Область("Заголовок") → returns SpreadsheetDocumentArea.
type Макет struct {
	template *printform.LayoutTemplate
}

// NewMaket creates a Макет DSL object from a layout template.
func NewMaket(lt *printform.LayoutTemplate) *Макет {
	return &Макет{template: lt}
}

// InjectMaket adds the «Макет» DSL variable to vars when a layout template is
// present. With a nil layout it is a no-op (the variable is not added, so DSL
// without a макет behaves exactly as before). Used by all processor run paths
// (UI, CLI procrun, scheduler) to expose src/<имя>.proc.layout.yaml as Макет.
func InjectMaket(vars map[string]any, lt *printform.LayoutTemplate) {
	if vars == nil || lt == nil {
		return
	}
	vars["Макет"] = NewMaket(lt)
}

// Get allows property access: Макет.Заголовок → same as Макет.Область("Заголовок").
func (m *Макет) Get(field string) any {
	return areaValue(m.getArea(field), field)
}

// Set is not supported on макет.
func (m *Макет) Set(field string, v any) {}

// CallMethod handles method calls on the макет.
func (m *Макет) CallMethod(name string, args []any) any {
	switch strings.ToLower(name) {
	case "область", "area", "получитьобласть", "getarea":
		if len(args) > 0 {
			return areaValue(m.getArea(strArg(args, 0)), strArg(args, 0))
		}
	case "имя", "name":
		return m.template.Name
	}
	return nil
}

// getArea returns a FRESH SpreadsheetDocumentArea for the named layout area.
// Each call creates a new area with its own cell storage, so the same area
// definition can be used multiple times with different parameter values (e.g., in loops).
//
// Ядро материализации (LayoutArea → sheet.Area со статическими текстами и
// именами параметров) вынесено в printform.BuildAreaCells — общее с
// декларативным движком BuildSheet (план 64, этап 3). DSL-путь не интерполирует
// {{...}} в text-ячейках: значения подставляются через Область.Параметры.X.
func (m *Макет) getArea(name string) *SpreadsheetDocumentArea {
	if m.template == nil {
		return nil
	}
	areaDef := m.template.Area(name) // case-insensitive (см. LayoutTemplate.Area)
	if areaDef == nil {
		// «Строки|Колонки» — пересечение строковой и колоночной областей, как
		// в 1С; ширины — по исходным колонкам макета. Только если области с
		// таким буквальным именем нет: макеты с областями «Шапка|Секция_1»
		// (разрезанными вручную) продолжают работать как раньше.
		if strings.Contains(name, "|") {
			if area, widths, ok := m.template.IntersectionArea(name); ok {
				return &SpreadsheetDocumentArea{Area: area, colWidths: widths}
			}
		}
		return nil
	}
	return &SpreadsheetDocumentArea{
		Area:      printform.BuildAreaCells(areaDef),
		colWidths: m.template.ColumnWidthsPx(),
	}
}

// areaValue — область как значение DSL. Ненайденная область — исключение
// модуля, как в 1С («Область не найдена»), его ловит Попытка…Исключение.
// Раньше возвращался типизированный nil-указатель: модуль видел его как
// значение, и обращение Область.Параметры… роняло сервер паникой nil pointer.
func areaValue(a *SpreadsheetDocumentArea, name string) any {
	if a == nil {
		RaiseUserError("Область макета не найдена: " + name)
	}
	return a
}
