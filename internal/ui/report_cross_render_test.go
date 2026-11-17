package ui

import (
	"strings"
	"testing"

	"github.com/ivantit66/onebase/internal/report"
	"github.com/ivantit66/onebase/internal/report/compose"
)

func TestRenderCrossTable(t *testing.T) {
	rows := []compose.Row{
		{"Товар": "А", "Месяц": "Янв", "Сумма": "100"},
		{"Товар": "А", "Месяц": "Фев", "Сумма": "200"},
		{"Товар": "Б", "Месяц": "Янв", "Сумма": "30"},
	}
	spec := &report.Composition{
		Groupings: []string{"Товар"},
		Columns:   []string{"Месяц"},
		Measures:  []report.Measure{{Field: "Сумма", Agg: "sum"}},
	}
	cr, _ := compose.ComposeCross(rows, *spec, nil)
	out := string(renderCrossTable(cr, spec))
	for _, want := range []string{"<table", "Товар", "Янв", "Фев", "100", "200", "Итого", "130"} {
		if !strings.Contains(out, want) {
			t.Fatalf("HTML кросс-таблицы не содержит %q:\n%s", want, out)
		}
	}
}

func TestRenderCrossFormatAlign(t *testing.T) {
	// Формат и выравнивание показателя (этап A) применяются и к ячейкам кросс-таблицы.
	rows := []compose.Row{
		{"Товар": "А", "Месяц": "Янв", "Сумма": "12333.32"},
	}
	spec := &report.Composition{
		Groupings: []string{"Товар"},
		Columns:   []string{"Месяц"},
		Measures:  []report.Measure{{Field: "Сумма", Agg: "sum", Format: "#,##0.00", Align: "center"}},
	}
	cr, _ := compose.ComposeCross(rows, *spec, nil)
	out := string(renderCrossTable(cr, spec))
	// Разделитель разрядов — неразрывный пробел (U+00A0), как в FormatNumber (этап A4).
	if !strings.Contains(out, "12 333,32") {
		t.Fatalf("ожидали форматированное число 12 333,32:\n%s", out)
	}
	if !strings.Contains(out, "text-align:center") {
		t.Fatalf("ожидали выравнивание text-align:center:\n%s", out)
	}
}

func TestRenderCrossMultiMeasureTitle(t *testing.T) {
	// При нескольких показателях подпись колонки включает название показателя.
	rows := []compose.Row{
		{"Товар": "А", "Месяц": "Янв", "Сумма": "100", "Кол": "2"},
	}
	spec := &report.Composition{
		Groupings: []string{"Товар"},
		Columns:   []string{"Месяц"},
		Measures: []report.Measure{
			{Field: "Сумма", Agg: "sum", Title: "Сумма"},
			{Field: "Кол", Agg: "sum", Title: "Кол-во"},
		},
	}
	cr, _ := compose.ComposeCross(rows, *spec, nil)
	out := string(renderCrossTable(cr, spec))
	if !strings.Contains(out, "Сумма") || !strings.Contains(out, "Кол-во") {
		t.Fatalf("ожидали названия показателей в шапке:\n%s", out)
	}
}

func TestRenderCrossAppearance(t *testing.T) {
	// Оформление компоновки даёт класс таблицы и в кросс-режиме; report-cross
	// сохраняется. Дефолт (нулевой Appearance) — прежний класс без rep-*.
	rows := []compose.Row{{"Товар": "А", "Месяц": "Янв", "Сумма": "100"}}
	spec := &report.Composition{
		Groupings:  []string{"Товар"},
		Columns:    []string{"Месяц"},
		Measures:   []report.Measure{{Field: "Сумма", Agg: "sum"}},
		Appearance: report.Appearance{Lines: "vertical"},
	}
	cr, _ := compose.ComposeCross(rows, *spec, nil)
	out := string(renderCrossTable(cr, spec))
	if !strings.Contains(out, `class="report-composed rep-lines-v report-cross"`) {
		t.Fatalf("ожидали класс оформления на кросс-таблице:\n%s", out)
	}

	specDef := *spec
	specDef.Appearance = report.Appearance{}
	cr2, _ := compose.ComposeCross(rows, specDef, nil)
	out2 := string(renderCrossTable(cr2, &specDef))
	if !strings.Contains(out2, `class="report-composed report-cross"`) {
		t.Fatalf("дефолт кросс-таблицы должен сохранять прежний класс:\n%s", out2)
	}
}

func TestCrossSheetRowsDuplicateField(t *testing.T) {
	// Регрессия #17 (Excel): два показателя на одном Field не должны затирать
	// друг друга. У каждой колонки своя подпись (Title) и свой формат — как уже
	// сделано в HTML-рендере (renderCrossTable берёт показатель по MeasureIdx).
	rows := []compose.Row{
		{"Товар": "А", "Месяц": "01", "Сумма": "100"},
	}
	spec := &report.Composition{
		Groupings: []string{"Товар"},
		Columns:   []string{"Месяц"},
		Measures: []report.Measure{
			{Field: "Сумма", Agg: "sum", Title: "Сумма"},                           // без формата → float64
			{Field: "Сумма", Agg: "sum", Title: "Сумма с НДС", Format: "#,##0.00"}, // с форматом → строка
		},
	}
	cr, _ := compose.ComposeCross(rows, *spec, nil)
	headers, sheet := crossSheetRows(cr, spec)
	// Измерение + 2 показателя + по колонке «Итого» на каждый показатель.
	if len(headers) != 5 {
		t.Fatalf("ожидали 5 заголовков (измерение + 2 показателя + 2 итога строки), получили %v", headers)
	}
	if headers[3] == headers[4] || !strings.Contains(headers[4], "НДС") {
		t.Fatalf("итоги строки по показателям с одним Field затёрты: %v", headers)
	}
	// Подписи колонок различаются — каждая несёт свой Title показателя.
	if headers[1] == headers[2] {
		t.Fatalf("подписи колонок совпали (показатель затёрт по Field): %q == %q", headers[1], headers[2])
	}
	if !strings.Contains(headers[2], "НДС") {
		t.Fatalf("во второй колонке ожидали Title «Сумма с НДС», получили %q", headers[2])
	}
	// Формат берётся из своего показателя: 1-я без формата → float64, 2-я с форматом → строка.
	last := sheet[len(sheet)-1] // строка ВСЕГО
	if _, ok := last[1].(float64); !ok {
		t.Fatalf("колонка без формата должна быть числом (float64), получили %v (%T)", last[1], last[1])
	}
	if _, ok := last[2].(string); !ok {
		t.Fatalf("колонка с форматом должна быть отформатированной строкой, получили %v (%T)", last[2], last[2])
	}
}

func TestCrossSheetRows(t *testing.T) {
	// Excel-выгрузка кросс-таблицы: шапка = строковые измерения + колонки + Итого;
	// строки данных + нижняя строка ВСЕГО; значения — float64.
	rows := []compose.Row{
		{"Товар": "А", "Месяц": "01", "Сумма": "100"},
		{"Товар": "А", "Месяц": "02", "Сумма": "200"},
		{"Товар": "Б", "Месяц": "01", "Сумма": "30"},
	}
	spec := &report.Composition{
		Groupings: []string{"Товар"},
		Columns:   []string{"Месяц"},
		Measures:  []report.Measure{{Field: "Сумма", Agg: "sum"}},
	}
	cr, _ := compose.ComposeCross(rows, *spec, nil)
	headers, sheet := crossSheetRows(cr, spec)
	if len(headers) != 4 || headers[0] != "Товар" || headers[3] != "Итого" {
		t.Fatalf("headers: %v", headers)
	}
	// Строки: А, Б, ВСЕГО.
	if len(sheet) != 3 {
		t.Fatalf("ожидали 3 строки (А, Б, ВСЕГО), получили %d: %v", len(sheet), sheet)
	}
	last := sheet[len(sheet)-1]
	if last[0] != "Итого" {
		t.Fatalf("последняя строка — Итого, получили %v", last[0])
	}
	// Колонки отсортированы: 01, 02. Итог 01 = 100+30 = 130, 02 = 200.
	if last[1] != float64(130) {
		t.Fatalf("итог колонки 01 ожидали 130.0, получили %v (%T)", last[1], last[1])
	}
	if last[2] != float64(200) {
		t.Fatalf("итог колонки 02 ожидали 200.0, получили %v (%T)", last[2], last[2])
	}
}

// Колонка «Итого» по строке справа (как в СКД 1С) — в HTML и в Excel.
func TestCrossRowTotalColumnHTMLAndExcel(t *testing.T) {
	rows := []compose.Row{
		{"Товар": "А", "Месяц": "Янв", "Сумма": "100"},
		{"Товар": "А", "Месяц": "Фев", "Сумма": "200"},
		{"Товар": "Б", "Месяц": "Янв", "Сумма": "30"},
	}
	spec := &report.Composition{
		Groupings: []string{"Товар"},
		Columns:   []string{"Месяц"},
		Measures:  []report.Measure{{Field: "Сумма", Agg: "sum"}},
	}
	cr, _ := compose.ComposeCross(rows, *spec, nil)
	out := string(renderCrossTable(cr, spec))
	if strings.Count(out, `rc-row-total`) != 1+2+1 { // шапка + две строки + угловой итог
		t.Fatalf("колонка итога строки:\n%s", out)
	}
	for _, want := range []string{">300<", ">330<"} {
		if !strings.Contains(out, want) {
			t.Fatalf("HTML без итога строки %q:\n%s", want, out)
		}
	}
	headers, sheet := crossSheetRows(cr, spec)
	if len(headers) != 4 || headers[3] != "Итого" {
		t.Fatalf("Excel: шапка %v", headers)
	}
	last := len(headers) - 1
	if got := fmtVal(sheet[0][last]); got != "300" {
		t.Fatalf("Excel: итог строки А = %v, строки %v", got, sheet)
	}
	if got := fmtVal(sheet[len(sheet)-1][last]); got != "330" {
		t.Fatalf("Excel: угловой итог = %v", got)
	}
}

// row_totals: false убирает колонку «Итого» справа (1С
// ГоризонтальноеРасположениеОбщихИтогов = Нет); колонка-итог года подписана.
func TestCrossRowTotalsOffAndGroupTotalTitle(t *testing.T) {
	rows := []compose.Row{
		{"Товар": "А", "Год": "2026", "Месяц": "01", "Сумма": "100"},
		{"Товар": "А", "Год": "2026", "Месяц": "02", "Сумма": "200"},
	}
	off := false
	spec := &report.Composition{
		Groupings: []string{"Товар"}, Columns: []string{"Год", "Месяц"},
		Measures:          []report.Measure{{Field: "Сумма", Agg: "sum"}},
		ColumnGroupTotals: true, RowTotals: &off,
	}
	cr, _ := compose.ComposeCross(rows, *spec, nil)
	out := string(renderCrossTable(cr, spec))
	if strings.Contains(out, "rc-row-total") || !strings.Contains(out, ">2026 / Итого</th>") {
		t.Fatalf("HTML:\n%s", out)
	}
	headers, sheet := crossSheetRows(cr, spec)
	if len(headers) != 4 || headers[3] != "2026 / Итого" || len(sheet[0]) != 4 {
		t.Fatalf("Excel: %v %v", headers, sheet)
	}
}
