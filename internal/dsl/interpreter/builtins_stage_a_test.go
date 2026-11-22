package interpreter

import (
	"testing"
	"time"
)

// callB вызывает builtin по имени и возвращает результат (для тестов Этапа A).
func callB(t *testing.T, name string, args ...any) any {
	t.Helper()
	fn, ok := builtins[name]
	if !ok {
		t.Fatalf("builtin %q не зарегистрирован", name)
	}
	v, err := fn(args, "", 0)
	if err != nil {
		t.Fatalf("%s вернул ошибку: %v", name, err)
	}
	return v
}

func TestTrimLeftRight(t *testing.T) {
	if got := callB(t, "сокрл", "  abc  "); got != "abc  " {
		t.Errorf("СокрЛ: got %q", got)
	}
	if got := callB(t, "сокрп", "  abc  "); got != "  abc" {
		t.Errorf("СокрП: got %q", got)
	}
}

func TestStrOccurrences(t *testing.T) {
	if got := callB(t, "стрчисловхождений", "a,b,c,d", ","); got != float64(3) {
		t.Errorf("СтрЧислоВхождений: got %v", got)
	}
	if got := callB(t, "стрчисловхождений", "abc", ""); got != float64(0) {
		t.Errorf("СтрЧислоВхождений с пустой подстрокой: got %v", got)
	}
}

func TestStrLineFns(t *testing.T) {
	s := "первая\nвторая\r\nтретья"
	if got := callB(t, "стрчислострок", s); got != float64(3) {
		t.Errorf("СтрЧислоСтрок: got %v", got)
	}
	if got := callB(t, "стрчислострок", ""); got != float64(1) {
		t.Errorf("СтрЧислоСтрок(\"\") должно быть 1: got %v", got)
	}
	if got := callB(t, "стрполучитьстроку", s, float64(2)); got != "вторая" {
		t.Errorf("СтрПолучитьСтроку(2): got %q", got)
	}
	if got := callB(t, "стрполучитьстроку", s, float64(9)); got != "" {
		t.Errorf("СтрПолучитьСтроку вне диапазона: got %q", got)
	}
}

func TestCharCode(t *testing.T) {
	if got := callB(t, "кодсимвола", "ABC", float64(1)); got != float64('A') {
		t.Errorf("КодСимвола(ABC,1): got %v", got)
	}
	// по умолчанию позиция 1
	if got := callB(t, "кодсимвола", "Z"); got != float64('Z') {
		t.Errorf("КодСимвола(Z): got %v", got)
	}
	if got := callB(t, "кодсимвола", "", float64(1)); got != float64(0) {
		t.Errorf("КодСимвола пустой строки: got %v", got)
	}
}

func TestStrCompare(t *testing.T) {
	if got := callB(t, "стрсравнить", "abc", "ABC"); got != float64(0) {
		t.Errorf("СтрСравнить без учёта регистра: got %v", got)
	}
	if got := callB(t, "стрсравнить", "a", "b"); got != float64(-1) {
		t.Errorf("СтрСравнить a<b: got %v", got)
	}
}

func TestIsBlankString(t *testing.T) {
	if got := callB(t, "пустаястрока", "   "); got != true {
		t.Errorf("ПустаяСтрока пробелов: got %v", got)
	}
	if got := callB(t, "пустаястрока", " x "); got != false {
		t.Errorf("ПустаяСтрока непустой: got %v", got)
	}
}

func TestTitleCase(t *testing.T) {
	if got := callB(t, "трег", "иван иВАНОВ"); got != "Иван Иванов" {
		t.Errorf("ТРег: got %q", got)
	}
}

func TestNStr(t *testing.T) {
	src := "ru = 'Привет'; en = 'Hello'"
	if got := callB(t, "нстр", src, "en"); got != "Hello" {
		t.Errorf("НСтр(en): got %q", got)
	}
	if got := callB(t, "нстр", src); got != "Привет" {
		t.Errorf("НСтр по умолчанию ru: got %q", got)
	}
	// неизвестный язык → первый сегмент
	if got := callB(t, "нстр", src, "de"); got != "Привет" {
		t.Errorf("НСтр(de)→первый: got %q", got)
	}
}

func TestQuarterAndOrdinalDates(t *testing.T) {
	d := time.Date(2026, 5, 15, 12, 30, 45, 0, time.Local)
	bq := callB(t, "началоквартала", d).(time.Time)
	if bq.Month() != time.April || bq.Day() != 1 || bq.Hour() != 0 {
		t.Errorf("НачалоКвартала: got %v", bq)
	}
	eq := callB(t, "конецквартала", d).(time.Time)
	if eq.Month() != time.June || eq.Day() != 30 || eq.Hour() != 23 || eq.Minute() != 59 {
		t.Errorf("КонецКвартала: got %v", eq)
	}
	if got := callB(t, "деньгода", time.Date(2026, 1, 10, 0, 0, 0, 0, time.Local)); got != float64(10) {
		t.Errorf("ДеньГода: got %v", got)
	}
}

func TestHourMinuteBounds(t *testing.T) {
	d := time.Date(2026, 5, 15, 12, 30, 45, 0, time.Local)
	bh := callB(t, "началочаса", d).(time.Time)
	if bh.Hour() != 12 || bh.Minute() != 0 || bh.Second() != 0 {
		t.Errorf("НачалоЧаса: got %v", bh)
	}
	eh := callB(t, "конецчаса", d).(time.Time)
	if eh.Minute() != 59 || eh.Second() != 59 {
		t.Errorf("КонецЧаса: got %v", eh)
	}
	em := callB(t, "конецминуты", d).(time.Time)
	if em.Second() != 59 {
		t.Errorf("КонецМинуты: got %v", em)
	}
}

func TestMathFns(t *testing.T) {
	if got := callB(t, "pow", float64(2), float64(10)); got != float64(1024) {
		t.Errorf("Pow(2,10): got %v", got)
	}
	if got := callB(t, "sqrt", float64(144)); got != float64(12) {
		t.Errorf("Sqrt(144): got %v", got)
	}
}

func TestBase64RoundTrip(t *testing.T) {
	enc := callB(t, "base64строка", "Привет").(string)
	dec := callB(t, "base64значение", enc).(string)
	if dec != "Привет" {
		t.Errorf("Base64 round-trip: got %q (enc=%q)", dec, enc)
	}
	if got := callB(t, "base64значение", "не-base64!!!"); got != "" {
		t.Errorf("Base64Значение на мусоре → \"\": got %q", got)
	}
}

func TestMD5BytesPreservesBinaryData(t *testing.T) {
	data := callB(t, "base64значение", "/9j/AIlQTkc=")
	for _, name := range []string{"md5данных", "md5bytes"} {
		if got := callB(t, name, data); got != "232b78c460093b91b664293edc4f281e" {
			t.Errorf("%s(binary) = %q", name, got)
		}
	}
}
