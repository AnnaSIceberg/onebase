package metadata

import "github.com/ivantit66/onebase/internal/csssafe"

// FormElementBackgroundCSS возвращает CSS-объявление фона элемента формы
// или пустую строку. Фон читают ГруппаФормы (#1547) и Кнопка — кнопки разных
// действий различают цветом, как в формах 1С. Значение обязано пройти
// csssafe.Color (hex, rgb, именованный цвет), иначе объявление не выдаётся —
// рантайм не применяет произвольные строки, а недопустимое значение называет
// check (form.background-color).
func FormElementBackgroundCSS(el *FormElement) string {
	if el == nil || !FormElementSupportsBackground(el.Kind) || el.Background == "" {
		return ""
	}
	if c := csssafe.Color(el.Background); c != "" {
		return "background:" + c + ";"
	}
	return ""
}

// FormElementSupportsBackground — виды элементов, для которых ключ background
// применяется: контейнеры (группа; набор страниц — весь набор с полосой закладок), кнопка и поле ввода (у поля — фон самого ввода,
// FormElementFieldColors).
func FormElementSupportsBackground(kind FormElementType) bool {
	return kind == FormElementGroupBox || kind == FormElementPages || kind == FormElementButton || kind == FormElementField
}

// FormElementSupportsBorderColor — ключ border_color читает только ПолеВвода.
func FormElementSupportsBorderColor(kind FormElementType) bool {
	return kind == FormElementField
}

// FormElementFieldColors — классы и CSS-переменные обёртки ПолеВвода для фона
// и цвета рамки самого ввода (background / border_color). Поле окрашено
// всегда, в том числе нередактируемое: так в ТЗ выделяют поле, на которое
// оператор обязан посмотреть («Приём заявок до»). Каждый цвет проходит
// csssafe.Color; нераспознанный не выдаётся, его называет check.
func FormElementFieldColors(el *FormElement) (classes, vars string) {
	if el == nil || el.Kind != FormElementField {
		return "", ""
	}
	if c := csssafe.Color(el.Background); el.Background != "" && c != "" {
		classes += " ob-field-bg"
		vars += "--ob-field-bg:" + c + ";"
	}
	if c := csssafe.Color(el.BorderColor); el.BorderColor != "" && c != "" {
		classes += " ob-field-border"
		vars += "--ob-field-border:" + c + ";"
	}
	return classes, vars
}

// FormTableHeaderColors — классы и CSS-переменные контейнера таблицы
// ТабличнаяЧасть для фона и границы шапки (header_background /
// header_border_color): SlickGrid, простая таблица и таблица значений.
// Каждый цвет проходит csssafe.Color; нераспознанный не выдаётся, его
// называет check.
func FormTableHeaderColors(el *FormElement) (classes, vars string) {
	if el == nil || el.Kind != FormElementTablePart {
		return "", ""
	}
	if c := csssafe.Color(el.HeaderBackground); el.HeaderBackground != "" && c != "" {
		classes += " ob-th-bg"
		vars += "--ob-th-bg:" + c + ";"
	}
	if c := csssafe.Color(el.HeaderBorderColor); el.HeaderBorderColor != "" && c != "" {
		classes += " ob-th-border"
		vars += "--ob-th-border:" + c + ";"
	}
	return classes, vars
}
