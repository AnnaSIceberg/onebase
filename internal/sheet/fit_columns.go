package sheet

import (
	"math"
	"strings"

	"github.com/go-pdf/fpdf"
)

// FitTableColumns fits the table starting at the given 1-based row. Rows above
// it, and cell text spanning several columns, do not constrain a single column.
func (d *Document) FitTableColumns(firstRow, equalFrom, equalTo int) {
	if firstRow < 1 {
		return
	}
	widths := make(map[int]float64)
	last := 0
	measure := fpdf.New("P", "mm", "A4", "")
	registerFonts(measure)
	for key, cell := range d.Cells {
		if key.Row < firstRow-1 || cell == nil {
			continue
		}
		col := key.Col + 1
		last = max(last, col+max(1, cell.ColSpan)-1)
		if cell.ColSpan > 1 {
			continue
		}
		family, style := resolveFont(cell.FontFamily, cell.Bold, cell.Italic)
		measure.SetFont(family, style, fontSizeOr(cell))
		text := cell.DisplayText()
		var lineWidth, wordWidth float64
		for _, line := range strings.Split(text, "\n") {
			lineWidth = max(lineWidth, measure.GetStringWidth(line))
		}
		for _, word := range strings.Fields(text) {
			wordWidth = max(wordWidth, measure.GetStringWidth(word))
		}
		// Leave space for borders, padding and browser font differences.
		preferred := min(260, lineWidth*pxPerInch/mmPerInch*1.08+18)
		minimum := wordWidth*pxPerInch/mmPerInch*1.08 + 18
		widths[col] = max(widths[col], math.Ceil(max(preferred, minimum)))
	}
	if last == 0 {
		return
	}
	if equalFrom > 0 && equalFrom <= equalTo && equalTo <= last {
		uniform := 72.0
		for col := equalFrom; col <= equalTo; col++ {
			uniform = max(uniform, widths[col])
		}
		for col := equalFrom; col <= equalTo; col++ {
			widths[col] = uniform
		}
	}
	for col := 1; col <= last; col++ {
		d.SetColumnWidth(col, max(72, widths[col]))
	}
	d.FittedTableFirstRow = firstRow
}

func (d *Document) fittedHTMLColumnWidth(col int) float64 {
	if width := d.ColWidths[col+1]; width > 0 {
		return width
	}
	return 72
}
