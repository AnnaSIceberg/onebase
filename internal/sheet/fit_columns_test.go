package sheet

import (
	"bytes"
	"strings"
	"testing"

	"github.com/xuri/excelize/v2"
)

func TestFittedReportColumnsAndGridAcrossExports(t *testing.T) {
	doc := NewDocument()
	doc.HideGrid = true
	doc.SetCell(0, 0, "Кем сформирован: "+strings.Repeat("ДлинныйФильтр", 30))
	doc.SetCell(1, 0, "Отчёт по диагностике")
	doc.GetCell(0, 0).Border = "none"
	doc.GetCell(1, 0).Border = "none"
	doc.Merge(2, 1, 2, 4)
	for i, value := range []string{"Дата закрытия", "03.09.2026", "04.09.2026", "05.09.2026"} {
		doc.SetCell(2, i, value)
		cell := doc.GetCell(2, i)
		cell.Bold, cell.Border = true, "thin"
	}
	doc.SetCell(3, 1, "99,99 %")
	doc.SetCell(3, 2, "100,00 %")
	doc.SetCell(3, 3, "-")
	doc.FitTableColumns(3, 2, 4)
	withoutPreamble := doc.Snapshot()
	delete(withoutPreamble.Cells, CellKey{Row: 0, Col: 0})
	delete(withoutPreamble.Cells, CellKey{Row: 1, Col: 0})
	withoutPreamble.FitTableColumns(3, 2, 4)
	for column := 1; column <= 4; column++ {
		if got, want := doc.ColumnWidth(column), withoutPreamble.ColumnWidth(column); got != want {
			t.Fatalf("column %d classifies preamble as data: got %g, want %g", column, got, want)
		}
	}
	for column := 3; column <= 4; column++ {
		if doc.ColumnWidth(column) != doc.ColumnWidth(2) {
			t.Fatalf("period column %d differs in width", column)
		}
	}
	if legacy := doc.HTML(HTMLOptions{}); strings.Contains(legacy, "td,th{border:none}") {
		t.Fatal("the legacy appearance must keep its default empty-cell borders")
	}
	view := doc.HTML(HTMLOptions{CellTextFlow: true})
	for _, fragment := range []string{"td,th{border:none}", "<colgroup data-sheet-columns>", ">Дата закрытия</td>", "data-sheet-flow-end=\"3\""} {
		if !strings.Contains(view, fragment) {
			t.Errorf("HTML missing %q", fragment)
		}
	}
	pdf, err := doc.PDF(PDFOptions{})
	if err != nil || !bytes.HasPrefix(pdf, []byte("%PDF")) {
		t.Fatalf("invalid PDF: %v", err)
	}
	book, err := doc.XLSX()
	if err != nil {
		t.Fatal(err)
	}
	file, err := excelize.OpenReader(bytes.NewReader(book))
	if err != nil {
		t.Fatal(err)
	}
	t.Cleanup(func() { _ = file.Close() })
	name := file.GetSheetName(0)
	grid, err := file.GetSheetView(name, 0)
	if err != nil || grid.ShowGridLines == nil || *grid.ShowGridLines {
		t.Fatalf("unexpected XLSX grid: %+v (%v)", grid, err)
	}
	for cell, wantBorder := range map[string]bool{"A1": false, "B1": false, "A3": true, "D3": true} {
		id, err := file.GetCellStyle(name, cell)
		if err != nil {
			t.Fatal(err)
		}
		style, err := file.GetStyle(id)
		if err != nil || (len(style.Border) > 0) != wantBorder {
			t.Fatalf("%s border: %+v %v, want %v", cell, style, err, wantBorder)
		}
	}
	width, err := file.GetColWidth(name, "B")
	if err != nil {
		t.Fatal(err)
	}
	for _, column := range []string{"C", "D"} {
		got, err := file.GetColWidth(name, column)
		if err != nil || got != width {
			t.Fatalf("XLSX %s width: %g (%v), want %g", column, got, err, width)
		}
	}
}
