// Package format1c — представление значений по строке формата 1С (ДФ, ДЛФ),
// общее для Формат() модулей и ячеек табличного документа: правила не должны
// расходиться между модулем, печатной формой и выгрузкой.
package format1c

import (
	"strconv"
	"strings"
	"time"
)

// Param extracts a parameter value from a format string like "ЧДЦ=2; ЧРГ=' '"
func Param(fmtStr, key string) string {
	// Ключ («дф=», «чдц=») ищем без учёта регистра, а значение отдаём из
	// ИСХОДНОЙ строки: шаблон даты регистрозависим. Ищем по рунам, потому что
	// Unicode-приведение регистра может менять число UTF-8-байтов (например,
	// «İ»), и тогда байтовый индекс lower-case копии неприменим к оригиналу.
	formatRunes := []rune(fmtStr)
	keyRunes := []rune(key)
	idx := -1
	for i := 0; i+len(keyRunes) <= len(formatRunes); i++ {
		if strings.EqualFold(string(formatRunes[i:i+len(keyRunes)]), key) {
			idx = i
			break
		}
	}
	if idx < 0 {
		return ""
	}
	rest := string(formatRunes[idx+len(keyRunes):])
	// Skip optional quote
	if len(rest) > 0 && rest[0] == '\'' {
		rest = rest[1:]
		end := strings.Index(rest, "'")
		if end >= 0 {
			return rest[:end]
		}
	}
	// Read until ; or end
	end := strings.Index(rest, ";")
	if end >= 0 {
		return rest[:end]
	}
	return rest
}

// Pattern converts a 1C-style date pattern to Go format and formats.
//
// formatDate переводит шаблон 1С в раскладку Go. Токены РЕГИСТРОЗАВИСИМЫ, как в
// 1С: «MM» — месяц, «mm» — минуты, «HH» — часы 24-часовые, «hh» — 12-часовые.
// Русские написания («дд.ММ.гггг ЧЧ:мм:сс») понимаются наравне с латинскими.
//
// РАНЬШЕ ВРЕМЕНИ НЕ БЫЛО ВОВСЕ: строка формата приводилась к нижнему регистру
// целиком, и в замене участвовали только yyyy/yy/mm/dd. Привычное
// «ДФ=dd.MM.yyyy HH:mm» печатало «09.09.2026 hh:09» — «hh» уходило в вывод
// литералом, а на месте минут оказывался МЕСЯЦ. Ошибки при этом не было, поэтому
// такая метка времени спокойно доезжала до пользователя.
//
// СОВМЕСТИМОСТЬ. До этой правки строчное «mm» означало МЕСЯЦ (регистр не
// различался), и конфигурации с «ДФ=dd.mm.yyyy» существуют. Прочитать его как
// минуты значило бы молча заменить месяц минутами — ровно тот дефект, который
// здесь и чинится. Поэтому строчное «mm»/«мм» читается минутами, когда шаблон
// говорит временем: ЕСТЬ часы, ЛИБО есть секунды и при этом нет ни года, ни
// дня — «мм:сс» это минуты:секунды, а «месяц:секунды» не осмысленный шаблон
// (#1541). БЕЗ ЧАСОВ шаблон с годом или днём («дд.мм.гггг») печатает месяц.
// При наличии часов строчное «mm»/«мм» всегда означает минуты, даже рядом
// с годом или днём. Для месяца нужно явное «MM»/«ММ»: «дд.ММ.гггг ЧЧ:мм:сс».
func Pattern(t time.Time, pattern string) string {
	// Текст в двойных кавычках — литерал, как в 1С: ДФ='d MMMM yyyy "года"'.
	// Он выводится как есть и не участвует ни в замене токенов, ни в выборе
	// падежа, ни в разборе шаблона Go (каждая часть форматируется отдельно).
	parts := strings.Split(pattern, `"`)
	var tokens strings.Builder
	for i := 0; i < len(parts); i += 2 {
		tokens.WriteString(parts[i])
		tokens.WriteByte(' ')
	}
	template := tokens.String()
	// Замены не перекрываются и не перечитывают уже подставленное, а порядок
	// аргументов задаёт приоритет: «yyyy» пробуется раньше «yy», иначе год
	// превратился бы в «0606».
	minute := goLayoutMonth
	if hasHourToken(template) || isMinutesSecondsPattern(template) {
		minute = goLayoutMinute
	}
	// «ММММ»/«МММ» — название месяца, как в 1С: ДФ='ММММ гг' → «Сентябрь 26»,
	// а с днём — родительный падеж: ДФ='д ММММ гггг' → «1 сентября 2026».
	// Раньше «ММММ» печаталось двумя номерами — «0909». Название вставляется
	// после форматирования через метку, чтобы токены шаблона («д» в
	// «декабря») его не задели. Одиночное «д»/«d» — день без ведущего нуля.
	month := int(t.Month()) - 1
	full := monthNamesRU[month]
	if strings.ContainsAny(template, "dд") {
		full = monthGenitiveRU[month]
	}
	fullMark, shortMark, hourMark := string(rune(1)), string(rune(2)), string(rune(3))
	layout := strings.NewReplacer(
		"MMMM", fullMark, "ММММ", fullMark,
		"MMM", shortMark, "МММ", shortMark,
		"yyyy", goLayoutYear4, "гггг", goLayoutYear4,
		"yy", goLayoutYear2, "гг", goLayoutYear2,
		"MM", goLayoutMonth, "ММ", goLayoutMonth,
		"dd", goLayoutDay, "дд", goLayoutDay,
		"d", goLayoutDayShort, "д", goLayoutDayShort,
		"HH", goLayoutHour24, "ЧЧ", goLayoutHour24,
		"hh", goLayoutHour12, "чч", goLayoutHour12,
		// Одиночные «H»/«Ч» — час (24) без ведущего нуля, «h»/«ч» — 12-часовой.
		"H", hourMark, "Ч", hourMark,
		"h", goLayoutHour12Short, "ч", goLayoutHour12Short,
		"ss", goLayoutSecond, "сс", goLayoutSecond,
		"mm", minute, "мм", minute,
	)
	names := strings.NewReplacer(fullMark, full, shortMark, monthShortRU[month], hourMark, strconv.Itoa(t.Hour()))
	var out strings.Builder
	for i, part := range parts {
		if i%2 == 1 {
			out.WriteString(part)
		} else if part != "" {
			out.WriteString(names.Replace(t.Format(layout.Replace(part))))
		}
	}
	return out.String()
}

var monthNamesRU = [12]string{"Январь", "Февраль", "Март", "Апрель", "Май", "Июнь", "Июль", "Август", "Сентябрь", "Октябрь", "Ноябрь", "Декабрь"}
var monthGenitiveRU = [12]string{"января", "февраля", "марта", "апреля", "мая", "июня", "июля", "августа", "сентября", "октября", "ноября", "декабря"}
var monthShortRU = [12]string{"янв", "фев", "мар", "апр", "май", "июн", "июл", "авг", "сен", "окт", "ноя", "дек"}

// Токены эталонного времени Go (Mon Jan 2 15:04:05 MST 2006) — именами, чтобы
// таблица замен читалась как «год → год», а не как набор чисел.
const (
	goLayoutYear4       = "2006"
	goLayoutYear2       = "06"
	goLayoutMonth       = "01"
	goLayoutDay         = "02"
	goLayoutDayShort    = "2"
	goLayoutHour24      = "15"
	goLayoutHour12      = "03"
	goLayoutHour12Short = "3"
	goLayoutMinute      = "04"
	goLayoutSecond      = "05"
)

// hasHourToken — есть ли в шаблоне часы. От этого зависит прочтение строчного
// «mm»: минуты или (по совместимости) месяц.
func hasHourToken(pattern string) bool {
	for _, tok := range []string{"H", "h", "Ч", "ч"} {
		if strings.Contains(pattern, tok) {
			return true
		}
	}
	return false
}

// isMinutesSecondsPattern — шаблон вида «мм:сс»: есть секунды и нет ни года,
// ни дня. «Месяц:секунды» не осмысленный шаблон, поэтому в этом контексте
// строчное «mm»/«мм» читается минутами (#1541). Год или день возвращают
// старое прочтение-совместимость только без часов: hasHourToken в formatDate
// имеет приоритет. «дд.мм.гггг» без часов печатает месяц, как и раньше.
func isMinutesSecondsPattern(pattern string) bool {
	if !strings.Contains(pattern, "ss") && !strings.Contains(pattern, "сс") {
		return false
	}
	if strings.Contains(pattern, "yy") || strings.Contains(pattern, "гг") {
		return false
	}
	return !strings.Contains(pattern, "dd") && !strings.Contains(pattern, "дд")
}

// DateTime — представление даты-времени по умолчанию, как Строка(Дата) в 1С:
// «18.11.2026 3:07:09» (час без ведущего нуля), пустая дата — «01.01.0001 0:00:00».
func DateTime(t time.Time) string {
	return t.Format("02.01.2006") + " " + strconv.Itoa(t.Hour()) + t.Format(":04:05")
}

// Date форматирует дату по строке формата 1С: ДЛФ (локальный формат) или ДФ
// (шаблон). Второе значение false — в строке нет ни того, ни другого.
// ДЛФ, как в 1С: D/Д — «18.11.2026», T/В — «3:07:09», DT/ДВ — «18.11.2026
// 3:07:09», DD/ДД — «18 ноября 2026 г.», DDT/ДДВ — «18 ноября 2026 г. 3:07:09».
func Date(t time.Time, format string) (string, bool) {
	if lf := strings.ToUpper(strings.TrimSpace(firstParam(format, "длф=", "dlf="))); lf != "" {
		long := Pattern(t, "d MMMM yyyy") + " г."
		clock := strconv.Itoa(t.Hour()) + t.Format(":04:05")
		switch lf {
		case "D", "Д":
			return t.Format("02.01.2006"), true
		case "T", "В":
			return clock, true
		case "DT", "ДВ":
			return DateTime(t), true
		case "DD", "ДД":
			return long, true
		case "DDT", "ДДВ":
			return long + " " + clock, true
		}
	}
	if pattern := firstParam(format, "дф=", "df="); pattern != "" {
		return Pattern(t, pattern), true
	}
	return "", false
}

func firstParam(format string, keys ...string) string {
	for _, key := range keys {
		if v := Param(format, key); v != "" {
			return v
		}
	}
	return ""
}
