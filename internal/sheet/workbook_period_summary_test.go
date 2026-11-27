package sheet

import (
	"bytes"
	"testing"

	"github.com/shopspring/decimal"
	"github.com/xuri/excelize/v2"
)

func TestReplacePeriodSummaryKeepsClosedColumnsAndUpdatesYear(t *testing.T) {
	for _, existing := range []bool{true, false} {
		book := excelize.NewFile()
		name := book.GetSheetName(0)
		rows := [][]any{
			{"", "", "дек.25", "Итого 2025", "окт26", "Итого 2026"},
			{},
			{"Заявки", "с", 77, 124, 20, 20},
			{"Звонки", "", 55, 394, 40, 40},
			{"Конверсия", "", 122, 8, 50, 50},
			{"Архивная метрика", "код", 706, 706, 123, 123},
			{"Нулевая строка", "", 2, 2, 1, 1},
		}
		for i, row := range rows {
			if existing {
				old := []any{"нояб.26", nil, 21, 19, 110, 111, 3}[i]
				if len(row) > 0 {
					result := append([]any(nil), row[:5]...)
					result = append(result, old, row[5])
					row = result
				}
			}
			ref, _ := excelize.CoordinatesToCellName(1, i+1)
			if err := book.SetSheetRow(name, ref, &row); err != nil {
				t.Fatal(err)
			}
		}
		style, err := book.NewStyle(&excelize.Style{Font: &excelize.Font{Bold: true}})
		if err != nil {
			t.Fatal(err)
		}
		if err := book.SetCellStyle(name, "C3", "D3", style); err != nil {
			t.Fatal(err)
		}
		if _, err := book.NewSheet("Ранее"); err != nil {
			t.Fatal(err)
		}
		if err := book.SetCellInt("Ранее", "A1", 701); err != nil {
			t.Fatal(err)
		}
		raw, err := book.WriteToBuffer()
		if err != nil {
			t.Fatal(err)
		}
		if err := book.Close(); err != nil {
			t.Fatal(err)
		}

		source := NewDocument()
		put := func(row, col int, value any) {
			cell := source.GetOrCreateCell(row-1, col-1)
			if number, ok := value.(int); ok {
				cell.Value = decimal.NewFromInt(int64(number))
				cell.Text = cell.Value.(decimal.Decimal).String()
			} else {
				cell.Text, _ = value.(string)
			}
		}
		put(1, 3, "нояб.26")
		put(3, 1, "Звонки")
		put(3, 3, 1200)
		put(4, 1, "Заявки")
		put(4, 2, "с")
		put(4, 3, 60)
		put(5, 1, "Конверсия")
		put(5, 3, 5)
		put(6, 1, "Нулевая строка")

		snapshot := NewWorkbook()
		err = snapshot.AddPeriodSummary(name, source, PeriodSummary{
			Period: "нояб.26", Total: "Итого 2026", Previous: []string{"янв.26", "окт.26"},
			Rows: []PeriodSummaryRow{{Row: 3}, {Row: 4}, {Row: 5, Numerator: 4, Denominator: 3}, {Row: 6}},
		})
		if err != nil {
			t.Fatal(err)
		}
		withPatch, err := snapshot.ReplaceXLSX(bytes.NewReader(raw.Bytes()))
		if err != nil {
			t.Fatal(err)
		}
		for i := 0; i < 2; i++ {
			file, err := excelize.OpenReader(bytes.NewReader(withPatch))
			if err != nil {
				t.Fatal(err)
			}
			for _, test := range []struct{ sheet, ref, want string }{
				{name, "C3", "77"}, {name, "D3", "124"}, {name, "E3", "20"}, {name, "F3", "60"}, {name, "G3", "80"},
				{name, "F4", "1200"}, {name, "G4", "1240"},
				{name, "F5", "5"}, {name, "G5", "6.5"},
				{name, "C6", "706"}, {name, "E6", "123"}, {name, "G6", "123"},
				{name, "F7", ""}, {name, "G7", "1"}, {"Ранее", "A1", "701"},
			} {
				got, err := file.GetCellValue(test.sheet, test.ref, excelize.Options{RawCellValue: true})
				if err != nil || got != test.want {
					t.Errorf("existing=%t pass=%d %s!%s = %q, %v, want %q", existing, i, test.sheet, test.ref, got, err, test.want)
				}
			}
			got, err := file.GetCellStyle(name, "C3")
			if err != nil || got != style {
				t.Errorf("previous formatting changed: %d, %v", got, err)
			}
			if err := file.Close(); err != nil {
				t.Fatal(err)
			}
			if i == 0 {
				withPatch, err = snapshot.ReplaceXLSX(bytes.NewReader(withPatch))
				if err != nil {
					t.Fatal(err)
				}
			}
		}
	}
}

func TestReplacePeriodSummaryKeepsNewSiteInItsSection(t *testing.T) {
	book := excelize.NewFile()
	defer book.Close()
	name := book.GetSheetName(0)
	old := [][]any{
		{"", "", "окт.26", "Итого 2026"}, {},
		{"Детализация"}, {"old.test", "Получено задач", 3}, {},
		{"SEO", "Получено задач"}, {"shared.test", "Получено задач", 2}, {},
		{"Direct", "Получено задач"}, {"direct.test", "Получено задач", 4},
	}
	for i := range old {
		ref, _ := excelize.CoordinatesToCellName(1, i+1)
		if err := book.SetSheetRow(name, ref, &old[i]); err != nil {
			t.Fatal(err)
		}
	}
	raw, err := book.WriteToBuffer()
	if err != nil {
		t.Fatal(err)
	}
	source := NewDocument()
	source.GetOrCreateCell(0, 2).Text = "нояб.26"
	newRows := []struct {
		row   int
		label string
		value int64
	}{
		{4, "old.test", 0}, {5, "shared.test", 1},
		{7, "SEO", 0}, {8, "shared.test", 5},
		{10, "Direct", 0}, {11, "direct.test", 0}, {12, "shared.test", 7},
	}
	source.GetOrCreateCell(2, 0).Text = "Детализация"
	var rules []PeriodSummaryRow
	for _, row := range newRows {
		source.GetOrCreateCell(row.row-1, 0).Text = row.label
		source.GetOrCreateCell(row.row-1, 1).Text = "Получено задач"
		if row.value != 0 {
			n := decimal.NewFromInt(row.value)
			cell := source.GetOrCreateCell(row.row-1, 2)
			cell.Text, cell.Value = n.String(), n
		}
		rules = append(rules, PeriodSummaryRow{Row: row.row})
	}
	patch := NewWorkbook()
	if err := patch.AddPeriodSummary(name, source, PeriodSummary{
		Period: "нояб.26", Total: "Итого 2026", Previous: []string{"окт.26"},
		Rows: rules, GroupMarker: "Получено задач",
	}); err != nil {
		t.Fatal(err)
	}
	data := raw.Bytes()
	for pass := 1; pass <= 2; pass++ {
		data, err = patch.ReplaceXLSX(bytes.NewReader(data))
		if err != nil {
			t.Fatal(err)
		}
		merged, err := excelize.OpenReader(bytes.NewReader(data))
		if err != nil {
			t.Fatal(err)
		}
		rows, err := readSummaryCells(merged, name)
		merged.Close()
		if err != nil {
			t.Fatal(err)
		}
		expected := map[string][3]string{
			"\x00shared.test":       {"", "1", "1"},
			"seo\x00shared.test":    {"2", "5", "7"},
			"direct\x00shared.test": {"", "7", "7"},
		}
		for i, key := range summaryKeysForRows(rows, "Получено задач") {
			if key.Label != "shared.test" {
				continue
			}
			got := [3]string{summaryText(rows[i], 3), summaryText(rows[i], 4), summaryText(rows[i], 5)}
			want, ok := expected[key.Group]
			if !ok || got != want {
				t.Fatalf("pass %d: section %q values %v, want %v", pass, key.Group, got, want)
			}
			delete(expected, key.Group)
		}
		if len(expected) != 0 {
			t.Fatalf("pass %d: missing site blocks %v", pass, expected)
		}
	}
}
