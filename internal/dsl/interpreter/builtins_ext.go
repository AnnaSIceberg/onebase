package interpreter

import (
	"fmt"
	"github.com/ivantit66/onebase/internal/format1c"
	"strconv"
	"strings"
	"time"
)

// isBlankVal checks if a value is considered empty (nil, "", 0, false, empty collection).
func isBlankVal(v any) bool {
	if v == nil {
		return true
	}
	// Числовой ноль пуст в любом Go-типе (см. truthy): булево поле из запроса на
	// SQLite приходит как int64, и ЗначениеЗаполнено(Ложь) отвечало «истина».
	if zero, ok := numericZero(v); ok {
		return zero
	}
	switch t := v.(type) {
	case string:
		return t == ""
	case bool:
		return !t
	case []any:
		return len(t) == 0
	case *Array:
		return len(t.items) == 0
	case *Map:
		return t.count() == 0
	case *Ref:
		return isEmptyRefUUID(t.UUID)
	case time.Time:
		// Пустая дата — нулевая, как в 1С: ЗначениеЗаполнено('00010101') = Ложь.
		// Раньше её считали заполненной, и это молча выключало проверки: типовой
		// «БизнесДата = Об.Дата; Если НЕ ЗначениеЗаполнено(БизнесДата) Тогда
		// БизнесДата = ТекущаяДата()» переставал подставлять текущую дату, срез
		// версионного регистра на 01.01.0001 возвращал пусто — и правило, которое
		// должно было заблокировать запись, просто не находилось. Отказ был бы
		// заметен сразу; тишина обнаруживается через месяц по кривым данным.
		// Сравниваем локальные компоненты, а не через IsZero: тот верен только для
		// UTC, а «пустая» дата приезжает и в другой зоне (разбор строки, драйвер
		// БД) — там IsZero уже ложь, и проверка снова молчала бы. Время проверяем
		// полностью: 01.01.0001 с ненулевым временем — уже заполненное значение.
		y, mo, d := t.Date()
		h, min, s := t.Clock()
		return y == 1 && mo == time.January && d == 1 &&
			h == 0 && min == 0 && s == 0 && t.Nanosecond() == 0
	}
	return false
}

// isEmptyRefVal — узкое определение «пустой ссылки», в отличие от isBlankVal,
// которая считает пустым и 0, и false. Используется в ПустаяСсылка(x).
func isEmptyRefVal(v any) bool {
	if v == nil {
		return true
	}
	switch t := v.(type) {
	case string:
		return isEmptyRefUUID(t)
	case *Ref:
		return isEmptyRefUUID(t.UUID)
	}
	return false
}

// isEmptyRefUUID — пустой UUID = "" или нули.
func isEmptyRefUUID(s string) bool {
	return s == "" || s == "00000000-0000-0000-0000-000000000000"
}

// formatValue implements Формат(value, formatString) with minimal format support.
func fmtBuiltin(args []any) (string, error) {
	return fmtBuiltinBounded(args, 0)
}

// FormatValue — Формат(значение, строкаФормата) для Go-кода вне модулей:
// подписи группировок и колонок компоновки отчёта (composition.formats).
func FormatValue(v any, format string) (string, error) {
	return fmtBuiltin([]any{v, format})
}

func fmtBuiltinBounded(args []any, maxDecimalPlaces int32) (string, error) {
	if len(args) == 0 {
		return "", nil
	}
	if len(args) < 2 {
		return fmt.Sprintf("%v", args[0]), nil
	}
	val := args[0]
	if maxDecimalPlaces > 0 {
		// Numeric strings reach Decimal.String/StringFixed too. Bound their
		// lexical size before parsing, then the coefficient and exponent before
		// either formatting path can expand them (including float underflow).
		if text, ok := val.(string); ok && len(text) > maxSandboxDecimalTextBytes {
			return "", fmt.Errorf("формат: numeric text exceeds the sandbox decimal limit")
		}
		if d, ok := toDecimal(val); ok && !decimalWithinExpansionBounds(d, maxDecimalPlaces) {
			return "", fmt.Errorf("формат: число вне безопасного диапазона")
		}
	}
	// Строку формата НЕ приводим к нижнему регистру: шаблон даты
	// регистрозависим (MM — месяц, mm — минуты, HH — часы), а ключи «ДФ=»/
	// «ЧДЦ=» ищет сам extractFormatParam, без учёта регистра.
	fmtStr := strArg(args, 1)
	fmtKeys := strings.ToLower(fmtStr)

	// Date formatting: ДЛФ (D, DT, T, DD, DDT) и ДФ — общие с ячейками
	// табличного документа (format1c).
	if strings.Contains(fmtKeys, "дф=") || strings.Contains(fmtKeys, "df=") || strings.Contains(fmtKeys, "длф=") || strings.Contains(fmtKeys, "dlf=") {
		if t, ok := toTime(args, 0); ok {
			if text, ok := format1c.Date(t, fmtStr); ok {
				return text, nil
			}
		}
	}

	// Number formatting
	if f, ok := toFloat(val); ok {
		// Ноль, как в 1С, — пустая строка, если не задано ЧН («ЧН=0» → «0»):
		// Формат(0, "") = «», Формат(0, "ЧДЦ=2") = «».
		if f == 0 {
			if _, isBool := val.(bool); !isBool {
				return extractFormatParam(fmtStr, "чн="), nil
			}
		}
		// Без ЧДЦ — столько знаков, сколько есть в значении, как в 1С:
		// Формат(21, "ЧГ=0") = «21». Раньше подставлялось два знака, и номер
		// дня или года печатался «21.00» / «2 026.00».
		decimals := -1
		if d := extractFormatParam(fmtStr, "чдц="); d != "" {
			if n, err := strconv.Atoi(d); err == nil {
				if maxDecimalPlaces > 0 && (n < -int(maxDecimalPlaces) || n > int(maxDecimalPlaces)) {
					return "", fmt.Errorf("формат: точность вне безопасного диапазона")
				}
				decimals = n
			}
		}
		sep := " "
		if s := extractFormatParam(fmtStr, "чрг="); s != "" {
			if maxDecimalPlaces > 0 && len(s) > 64 {
				return "", fmt.Errorf("формат: разделитель разрядов слишком длинный")
			}
			sep = s
		}
		// ЧГ=0 — без группировки разрядов, как в 1С. Раньше ключ не читался
		// вовсе, и год печатался «2 026».
		if strings.TrimSpace(extractFormatParam(fmtStr, "чг=")) == "0" {
			sep = ""
		}
		if decimals < 0 {
			return formatNumberAsIs(val, f, sep), nil
		}
		// Округление — по правилам 1С, половина от нуля: Формат(1234.5,
		// "ЧДЦ=0") = «1235». Через float половина уходила к чётному («1234»).
		if d, ok := toDecimal(val); ok && decimals <= 64 {
			return groupNumberDigits(d.StringFixed(int32(decimals)), sep), nil //nolint:gosec // G115: 0 ≤ decimals ≤ 64 проверено выше
		}
		return formatNumber(f, decimals, sep), nil
	}

	return fmt.Sprintf("%v", val), nil
}

// extractFormatParam — параметр строки формата 1С (format1c.Param).
func extractFormatParam(fmtStr, key string) string { return format1c.Param(fmtStr, key) }

// formatDate — шаблон даты 1С (format1c.Pattern).
func formatDate(t time.Time, pattern string) string { return format1c.Pattern(t, pattern) }

// formatNumberAsIs — число без заданной точности: десятичная запись значения
// без незначащих нулей (21 → «21», 21.50 → «21.5»). Десятичное значение берётся
// как есть, а не через float: 0.1 + 0.2 не превращается в «0.30000000000000004».
func formatNumberAsIs(val any, f float64, sep string) string {
	s := strconv.FormatFloat(f, 'f', -1, 64)
	if d, ok := toDecimal(val); ok {
		s = d.String()
	}
	return groupNumberDigits(s, sep)
}

// formatNumber formats a float with given decimal places and thousands separator.
func formatNumber(f float64, decimals int, sep string) string {
	return groupNumberDigits(strconv.FormatFloat(f, 'f', decimals, 64), sep)
}

// groupNumberDigits расставляет разделитель разрядов в целой части записи числа.
func groupNumberDigits(s, sep string) string {
	parts := strings.Split(s, ".")
	intPart := parts[0]
	if sep != "" && len(intPart) > 3 {
		sign := ""
		if intPart[0] == '-' {
			sign = "-"
			intPart = intPart[1:]
		}
		var buf []byte
		for i, c := range intPart {
			if i > 0 && (len(intPart)-i)%3 == 0 {
				buf = append(buf, sep...)
			}
			buf = append(buf, byte(c)) //nolint:gosec // G115: значение приходит из проверенной модели и заведомо укладывается в целевой тип
		}
		intPart = sign + string(buf)
	}
	if len(parts) > 1 {
		return intPart + "." + parts[1]
	}
	return intPart
}
