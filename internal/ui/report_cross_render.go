package ui

// report_cross_render.go — HTML-рендер кросс-таблицы (pivot). Строковые измерения
// уходят влево, значения измерений-колонок — в шапку, на пересечении выводятся
// агрегаты показателей. Выравнивание, формат и условное оформление — общие с
// обычным режимом (measureAlign/fmtMeasure/cssOf), чтобы вид был единым.

import (
	"fmt"
	"html"
	"html/template"
	"strings"

	"github.com/ivantit66/onebase/internal/report"
	"github.com/ivantit66/onebase/internal/report/compose"
)

// measureByIdx возвращает функцию выбора показателя колонки по CrossCol.MeasureIdx.
// Два показателя с одинаковым Field — разные колонки (issue #17): сопоставление по
// индексу не даёт им затирать подпись/формат/выравнивание друг друга. Общая для
// HTML- (renderCrossTable) и Excel-рендера (crossSheetRows), чтобы логика не
// разъезжалась — именно расхождение этих двух путей и было причиной #17.
func measureByIdx(spec *report.Composition) func(compose.CrossCol) report.Measure {
	byIdx := spec.Measures
	return func(c compose.CrossCol) report.Measure {
		if c.MeasureIdx >= 0 && c.MeasureIdx < len(byIdx) {
			return byIdx[c.MeasureIdx]
		}
		return report.Measure{Field: c.Measure}
	}
}

func renderCrossTable(cr *compose.CrossResult, spec *report.Composition) template.HTML {
	return renderCrossTablePresented(cr, spec, reportCompositionPresentation{})
}

func renderCrossTablePresented(cr *compose.CrossResult, spec *report.Composition, presentation reportCompositionPresentation) template.HTML {
	measureAt := measureByIdx(spec)
	multiMeasure := len(spec.Measures) > 1

	var b strings.Builder
	b.WriteString(`<table class="` + appearanceClass(spec) + ` report-cross">`)

	// Шапка: первый столбец — строковые измерения; далее по колонке на CrossCol.
	b.WriteString(`<thead><tr><th>` + html.EscapeString(presentation.groupHeader(spec)) + `</th>`)
	for _, c := range cr.Cols {
		m := measureAt(c)
		b.WriteString(`<th class="num" style="` + html.EscapeString(measureAlign(m)) + `">` +
			html.EscapeString(crossColTitle(c, m, multiMeasure, spec, presentation)) + `</th>`)
	}
	// Итог строки по всем колонкам — справа, как в СКД 1С (row_totals: false
	// его убирает).
	for _, m := range crossTotalMeasures(spec) {
		b.WriteString(`<th class="num rc-row-total" style="` + html.EscapeString(measureAlign(m)) + `">` +
			html.EscapeString(crossTotalTitle(presentation.grandTitle(), m, multiMeasure)) + `</th>`)
	}
	b.WriteString(`</tr></thead><tbody>`)

	// Тело: дерево строк.
	for _, row := range cr.Rows {
		writeCrossRow(&b, row, cr.Cols, measureAt, 0, "", spec, presentation)
	}

	// Общий итог по колонкам.
	b.WriteString(`<tr class="grand"><td>` + html.EscapeString(presentation.grandTitle()) + `</td>`)
	for _, c := range cr.Cols {
		m := measureAt(c)
		b.WriteString(`<td class="num" style="` + html.EscapeString(measureAlign(m)) + `">` +
			html.EscapeString(fmtMeasure(cr.RowTotal[c.Key()], m)) + `</td>`)
	}
	writeCrossTotals(&b, cr.GrandTotal, spec)
	b.WriteString(`</tr></tbody></table>`)
	return template.HTML(b.String()) //nolint:gosec // G203: HTML собран с html.EscapeString на каждом значении
}

// writeCrossRow рисует строку дерева и рекурсивно её детей с нарастающим отступом.
// Узлы с детьми (промежуточные группы) несут подытоги по колонкам.
func writeCrossRow(b *strings.Builder, row *compose.CrossRow, cols []compose.CrossCol, measureAt func(compose.CrossCol) report.Measure, level int, parentPath string, spec *report.Composition, presentation reportCompositionPresentation) {
	pad := fmt.Sprintf("padding-left:%dpx", 8+level*18)
	path := parentPath + "/" + pathSeg(fmtVal(row.Key))
	rowStyle := "--rc-group-fill:" + composedGroupFill(level, len(spec.Groupings)) + ";--rc-group-dark:" + composedGroupDarkFill(level, len(spec.Groupings))
	depth, collapsed := collapseLevel(spec)
	if collapsed && level > depth {
		rowStyle += ";display:none"
	}
	marker := ""
	if len(row.Children) > 0 {
		open, text := !collapsed || level < depth, "+"
		if open {
			text = "−"
		}
		marker = fmt.Sprintf(`<button type="button" class="rc-fold" aria-expanded="%t" aria-label="%s">%s</button> `, open, html.EscapeString(presentation.groupLabel(row.Field, row.Key)), text)
	}
	fmt.Fprintf(b, `<tr class="grp" data-group="%s" data-level="%d" style="%s"><td style="%s">%s%s</td>`, html.EscapeString(path), level, html.EscapeString(rowStyle), pad, marker, html.EscapeString(presentation.groupLabel(row.Field, row.Key)))
	for _, c := range cols {
		m := measureAt(c)
		ck := c.Key()
		cell := joinStyles(measureAlign(m), cssOf(row.Styles[ck]))
		fmt.Fprintf(b, `<td class="num" style="%s">%s</td>`, html.EscapeString(cell), html.EscapeString(fmtMeasure(row.Cells[ck], m)))
	}
	writeCrossTotals(b, row.Total, spec)
	b.WriteString(`</tr>`)
	for _, ch := range row.Children {
		writeCrossRow(b, ch, cols, measureAt, level+1, path, spec, presentation)
	}
}

// crossColTitle — подпись колонки: значение(я) пути через « / »; при нескольких
// показателях добавляется название показателя, чтобы колонки различались.
func crossColTitle(c compose.CrossCol, m report.Measure, multiMeasure bool, spec *report.Composition, presentation reportCompositionPresentation) string {
	parts := make([]string, len(c.Path))
	for i, p := range c.Path {
		field := ""
		if i < len(spec.Columns) {
			field = spec.Columns[i]
		}
		// Колонка-дата подписывается так же, как группа: «01.09.2026», а для
		// функции периода — «01.09.2026 0:00:00».
		parts[i] = presentation.groupLabel(field, p)
	}
	if c.Subtotal {
		parts = append(parts, presentation.grandTitle())
	}
	title := strings.Join(parts, " / ")
	if multiMeasure {
		title += " · " + measureTitle(m)
	}
	return title
}

// writeCrossTotals — ячейки итога строки по всем колонкам (CrossRow.Total).
func writeCrossTotals(b *strings.Builder, totals map[string]any, spec *report.Composition) {
	for mi, m := range crossTotalMeasures(spec) {
		fmt.Fprintf(b, `<td class="num rc-row-total" style="%s">%s</td>`, html.EscapeString(measureAlign(m)), html.EscapeString(fmtMeasure(totals[compose.CrossTotalKey(mi)], m)))
	}
}

// crossTotalTitle — подпись колонки итога строки; при нескольких показателях
// добавляется название показателя, как у обычных колонок.
func crossTotalTitle(title string, m report.Measure, multiMeasure bool) string {
	if multiMeasure {
		return title + " · " + measureTitle(m)
	}
	return title
}

// crossTotalMeasures — показатели колонки «Итого» справа; пусто, если
// composition.row_totals: false.
func crossTotalMeasures(spec *report.Composition) []report.Measure {
	if !spec.ShowRowTotals() {
		return nil
	}
	return spec.Measures
}
