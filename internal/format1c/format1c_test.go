package format1c

import (
	"testing"
	"time"
)

// Заказ kc2 (финотчёт как в 1С): ДЛФ, час без ведущего нуля, Строка(Дата).
func TestDateLocalFormatsLike1C(t *testing.T) {
	moment := time.Date(2026, 11, 18, 3, 7, 9, 0, time.UTC)
	for format, want := range map[string]string{
		"ДЛФ=D":                   "18.11.2026",
		"ДЛФ=Д":                   "18.11.2026",
		"ДЛФ=DT":                  "18.11.2026 3:07:09",
		"ДЛФ=ДВ":                  "18.11.2026 3:07:09",
		"ДЛФ=T":                   "3:07:09",
		"ДЛФ=В":                   "3:07:09",
		"ДЛФ=DD":                  "18 ноября 2026 г.",
		"ДЛФ=DDT":                 "18 ноября 2026 г. 3:07:09",
		"DLF=D":                   "18.11.2026",
		"ДФ='dd.MM.yyyy H:mm:ss'": "18.11.2026 3:07:09",
		"ДФ='Ч:мм'":               "3:07",
		"ДФ='HH:mm'":              "03:07",
		"ДФ='h:mm'":               "3:07",
	} {
		got, ok := Date(moment, format)
		if !ok || got != want {
			t.Fatalf("%s: %q %v (want %q)", format, got, ok, want)
		}
	}
	if _, ok := Date(moment, "ЧДЦ=2"); ok {
		t.Fatal("числовой формат принят за дату")
	}
	if got := DateTime(moment); got != "18.11.2026 3:07:09" {
		t.Fatal(got)
	}
	if got := DateTime(time.Date(2026, 11, 18, 13, 0, 0, 0, time.UTC)); got != "18.11.2026 13:00:00" {
		t.Fatal(got)
	}
}
