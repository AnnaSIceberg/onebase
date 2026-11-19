package sheet

import (
	"strings"
	"testing"
)

func TestHTMLTextFlowStopsAtOccupiedAndMergedCells(t *testing.T) {
	d := NewDocument()
	d.SetCell(0, 0, "Отчёт с длинной шапкой")
	d.SetCell(0, 2, "Вторая непустая ячейка")
	d.SetCell(0, 3, "Граница")
	d.SetCell(1, 0, "Переносить слова")
	d.GetCell(1, 0).WrapText = true
	d.SetCell(1, 2, "Объединённая область")
	d.Merge(2, 3, 3, 4)
	d.SetCell(2, 0, "Оборвать у объединения")
	d.SetCell(2, 4, "После области")
	d.ColWidths[1] = 55

	view := d.HTML(HTMLOptions{Title: "Отчёт & KPI", CellTextFlow: true})
	for _, want := range []string{
		`<meta name="ob-tab-title" content="Отчёт &amp; KPI">`,
		`data-sheet-flow-end="1"><span>Отчёт с длинной шапкой</span>`,
		`data-sheet-flow-end="1"><span>Оборвать у объединения</span>`,
		`class="sheet-wrap-text"`,
		`colspan="2" rowspan="2"`,
	} {
		if !strings.Contains(view, want) {
			t.Errorf("нет %q", want)
		}
	}
	if strings.Contains(view, `data-sheet-flow-end="3"><span>Вторая непустая ячейка`) {
		t.Error("текст вышел за непустую соседнюю ячейку")
	}
	if strings.Contains(d.HTML(HTMLOptions{}), "data-sheet-flow-end") {
		t.Error("прежний режим просмотра изменился")
	}
}
