package sheet

import (
	"bytes"
	"testing"
	"time"
	_ "time/tzdata"

	"github.com/xuri/excelize/v2"
)

// Дата в ячейке макета с форматом ДЛФ=D: момент из SQLite приходит в UTC, и
// 10.09 00:00 МСК выводилось 09.09. Формат применяется в местном времени.
func TestCellDateFormatUsesLocalTime(t *testing.T) {
	loc, err := time.LoadLocation("Europe/Moscow")
	if err != nil {
		t.Fatal(err)
	}
	previous := time.Local
	time.Local = loc
	t.Cleanup(func() { time.Local = previous })
	moment := time.Date(2026, 9, 9, 21, 0, 0, 0, time.UTC)
	for format, want := range map[string]string{"ДЛФ=D": "10.09.2026", "ДЛФ=DT": "10.09.2026 0:00:00", "ДФ=dd.MM.yyyy": "10.09.2026"} {
		c := &Cell{Text: "raw", Value: moment, Format: format}
		if got := c.DisplayText(); got != want {
			t.Fatalf("%s: %q (want %q)", format, got, want)
		}
	}
	if got := (&Cell{Text: "raw", Value: moment}).DisplayText(); got != "raw" {
		t.Fatalf("без формата текст ячейки не меняется: %q", got)
	}
}

// Формат ячейки 1С совпадает с Формат() модуля (kc2, финотчёт): без ЧДЦ —
// знаки значения, ноль без ЧН — пусто.
func TestCellNumberFormatLikeFormat(t *testing.T) {
	for _, tc := range []struct {
		value  any
		format string
		want   string
	}{
		{1, "ЧН=0", "1"},
		{0, "ЧН=0", "0"},
		{0, "ЧДЦ=2", ""},
		{33.333, "ЧДЦ=2", "33,33"},
		{1250.5, "ЧГ=3", "1 250,5"},
		{0, "ЧН='-'", "-"},
	} {
		c := &Cell{Text: "raw", Value: tc.value, Format: tc.format}
		if got := c.DisplayText(); got != tc.want {
			t.Fatalf("%v %q: %q (want %q)", tc.value, tc.format, got, tc.want)
		}
	}
}

// XLSX: дата с ДЛФ=D — местный день и маска Excel, а не «ДЛФ=D» и UTC.
func TestXLSXDateLocalFormatMask(t *testing.T) {
	loc, err := time.LoadLocation("Europe/Moscow")
	if err != nil {
		t.Fatal(err)
	}
	previous := time.Local
	time.Local = loc
	t.Cleanup(func() { time.Local = previous })
	d := NewDocument()
	c := d.GetOrCreateCell(0, 0)
	c.Text, c.Value, c.Format = "x", time.Date(2026, 9, 9, 21, 0, 0, 0, time.UTC), "ДЛФ=D"
	data, err := d.XLSX()
	if err != nil {
		t.Fatal(err)
	}
	f, err := excelize.OpenReader(bytes.NewReader(data))
	if err != nil {
		t.Fatal(err)
	}
	sh := f.GetSheetName(0)
	if raw, _ := f.GetCellValue(sh, "A1", excelize.Options{RawCellValue: true}); raw != "46275" {
		t.Fatalf("дата 10.09.2026 как серийный номер Excel: %q", raw)
	}
	id, _ := f.GetCellStyle(sh, "A1")
	st, err := f.GetStyle(id)
	if err != nil || st.CustomNumFmt == nil || *st.CustomNumFmt != "dd.mm.yyyy" {
		t.Fatalf("маска Excel: %+v %v", st, err)
	}
}
