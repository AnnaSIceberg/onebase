package interpreter

import (
	"testing"
	"time"

	"github.com/shopspring/decimal"
)

// Ноль в параметре ячейки макета без формата, как в 1С, пуст; дата — местный день.
func TestCellTextZeroAndLocalDate(t *testing.T) {
	for _, v := range []any{0, 0.0, int64(0), decimal.Zero} {
		if got := cellText(v); got != "" {
			t.Fatalf("cellText(%T 0) = %q", v, got)
		}
	}
	if got := cellText("0"); got != "0" {
		t.Fatalf("строка «0» — не число: %q", got)
	}
	if got := cellText(false); got == "" {
		t.Fatal("булево ЛОЖЬ не ноль")
	}
	if got := cellText(12.5); got != "12.5" {
		t.Fatal(got)
	}
	loc, err := time.LoadLocation("Europe/Moscow")
	if err != nil {
		t.Fatal(err)
	}
	previous := time.Local
	time.Local = loc
	t.Cleanup(func() { time.Local = previous })
	if got := cellText(time.Date(2026, 9, 9, 21, 0, 0, 0, time.UTC)); got != "10.09.2026" {
		t.Fatalf("дата в ячейке — местный день: %q", got)
	}
}
