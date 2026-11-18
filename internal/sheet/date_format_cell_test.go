package sheet

import (
	"testing"
	"time"
	_ "time/tzdata"
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
