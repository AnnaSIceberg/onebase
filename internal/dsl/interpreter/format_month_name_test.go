package interpreter

import (
	"testing"
	"time"
)

// ДФ='ММММ гг' (Формат поля СКД) печатал «0909 26» вместо «Сентябрь 26».
func TestFormatMonthName(t *testing.T) {
	moment := time.Date(2026, 9, 1, 0, 0, 0, 0, time.Local)
	for format, want := range map[string]string{
		"ДФ='ММММ гг'":    "Сентябрь 26",
		"ДФ='MMMM yyyy'":  "Сентябрь 2026",
		"ДФ='МММ гггг'":   "сен 2026",
		"ДФ='дд.ММ.гггг'": "01.09.2026",
		"ДФ='ММММ'":       "Сентябрь",
	} {
		got, err := FormatValue(moment, format)
		if err != nil || got != want {
			t.Fatalf("%s: %q, %v (want %q)", format, got, err, want)
		}
	}
	if got, _ := FormatValue("2026-05-01", "ДФ='ММММ гг'"); got != "Май 26" {
		t.Fatalf("строка-дата SQLite: %q", got)
	}
}

// С днём в шаблоне месяц — в родительном падеже, как в 1С.
func TestFormatMonthNameGenitiveWithDay(t *testing.T) {
	moment := time.Date(2026, 9, 1, 0, 0, 0, 0, time.Local)
	for format, want := range map[string]string{
		"ДФ='д ММММ гггг'":  "1 сентября 2026",
		"ДФ='dd MMMM yyyy'": "01 сентября 2026",
		"ДФ='ММММ гггг'":    "Сентябрь 2026",
	} {
		if got, err := FormatValue(moment, format); err != nil || got != want {
			t.Fatalf("%s: %q, %v (want %q)", format, got, err, want)
		}
	}
}

// Литерал в двойных кавычках не трогается: ни токены, ни падеж.
func TestFormatDateQuotedLiteral(t *testing.T) {
	moment := time.Date(2026, 9, 1, 0, 0, 0, 0, time.Local)
	for format, want := range map[string]string{
		`ДФ='d MMMM yyyy "года"'`: "1 сентября 2026 года",
		`ДФ='MMMM "дня" yyyy'`:    "Сентябрь дня 2026",
		`ДФ='"Jan 1 2006" yyyy'`:  "Jan 1 2006 2026",
	} {
		if got, err := FormatValue(moment, format); err != nil || got != want {
			t.Fatalf("%s: %q, %v (want %q)", format, got, err, want)
		}
	}
}
