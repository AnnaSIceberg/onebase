package report

import (
	"os"
	"strings"

	"gopkg.in/yaml.v3"
)

type Param struct {
	Name    string            `yaml:"name"`
	Type    string            `yaml:"type"`    // string, date, datetime, number, bool, select, reference:Entity
	Label   string            `yaml:"label"`   // display label; falls back to Name
	Labels  map[string]string `yaml:"labels"`  // per-language labels (lang code → translation)
	Options []string          `yaml:"options"` // for type: select
	// ChoiceShowDeleted — для reference-параметра: показывать ли в выборе
	// помеченные на удаление (`choice_show_deleted`). nil — показывать с
	// отметкой: параметр отчёта ищет по истории, а не вводит данные.
	ChoiceShowDeleted *bool `yaml:"choice_show_deleted"`
	// ChoiceFilter связывает ссылочный выбор с другими параметрами отчёта.
	// Пустой ведущий параметр не ограничивает выбор (поиск по всем филиалам).
	ChoiceFilter []ParamChoiceFilter `yaml:"choice_filter,omitempty"`
	// ChoiceLabelSuffix — ссылочный реквизит, чья подпись добавляется в скобках.
	ChoiceLabelSuffix string `yaml:"choice_label_suffix,omitempty"`
	// Default — значение параметра, когда пользователь его не задал. Понимает ту
	// же грамматику подстановок, что параметры виджетов и регламентных заданий
	// ({{today}}, {{now|-7d}} и т.п.). Раньше линтер ключ `default` принимал, а
	// модель его не знала: отчёт с необязательной датой молча приходил пустым,
	// потому что «Срок < NULL» не выбирает ничего.
	//
	// Результат подстановки форматируется ПО ТИПУ параметра: `datetime` получает
	// момент со временем суток (2006-01-02T15:04:05), `date` и все прочие типы —
	// дату. Поэтому `{{now | minus_hours:6}}` у `datetime` отличается от
	// `{{now}}`, а у `date` усечение до даты — правило типа, а не потеря (#1204).
	Default string `yaml:"default"`
	// PeriodEnd — только для `date`: значение берётся концом дня (23:59:59),
	// как КонецПериода(…, День) в 1С. Нужен параметру «По»: документы и движения
	// хранят время суток, и «Дата <= &По» с полночью теряло всё, что было в этот
	// день после 00:00.
	PeriodEnd bool `yaml:"period_end"`
	// VisibleWhen controls presentation only. Hidden values remain independent
	// and are still submitted/validated; the server query chooses its own bounds.
	VisibleWhen *ParamVisibility `yaml:"visible_when,omitempty"`
	// Context is a trusted server binding, not an editable/default form value.
	Context *ParamContext `yaml:"context,omitempty"`
	// Enhanced changes the input control while keeping the parameter value and type.
	Enhanced ParamEnhancements `yaml:"enhanced,omitempty"`
}

type ParamEnhancements struct {
	DateOnly bool `yaml:"date_only,omitempty"`
}

type ParamChoiceFilter struct {
	Field string `yaml:"field"`
	Param string `yaml:"param"`
}

type Report struct {
	FormActions *FormActions      `yaml:"form_actions,omitempty"`
	Name        string            `yaml:"name"`
	Title       string            `yaml:"title"`
	Titles      map[string]string `yaml:"titles"`
	Params      []Param           `yaml:"params"`
	Query       string            `yaml:"query"`
	ChartProc   string            `yaml:"chart_proc"`
	ResultProc  string            `yaml:"result_proc,omitempty"`
	Composition *Composition      `yaml:"composition"`          // nil = плоская таблица (старое поведение)
	Variants    []ReportVariant   `yaml:"variants"`             // именованные компоновки и источники данных
	Drilldowns  []Drilldown       `yaml:"drilldowns,omitempty"` // trusted root metadata, shared by variants

	// ComparePhoneField opts a project report's result procedure into request-local comparison.
	ComparePhoneField string `yaml:"compare_phone_field,omitempty"`
	// CompareRequestPhones enables the protected predicate in the native request-match report.
	CompareRequestPhones bool `yaml:"compare_request_phones,omitempty"`

	// OutputFormat — предпочтительный формат выгрузки отчёта (issue #218): пусто|html
	// (интерактивный просмотр + кнопки Excel/PDF), pdf, excel или совместимый
	// legacy-синоним excel_html. Реальная выгрузка идёт через /ui/report/<name>/pdf
	// и /excel независимо от этого поля; формат лишь подсказывает приоритет в UI.
	OutputFormat string `yaml:"output_format,omitempty"`

	// External помечает отчёт из внешнего контура (таблица _ext_reports),
	// а не из конфигурации проекта. Заполняется программно при загрузке;
	// в YAML не сериализуется.
	External bool `yaml:"-"`
}

// AllowsRequestPhoneJoin binds the comparison to the opted-in project report.
// The external flag is set by the loader and cannot be overridden in YAML.
func (r *Report) AllowsRequestPhoneJoin() bool {
	return r.CompareRequestPhones && !r.External && r.ResultProc == "" && strings.EqualFold(r.Name, "А_ЗаявкиПоНомеруТелефона")
}

// FormActions configures the action row below the report parameters.
type FormActions struct {
	PrimaryBackground string `yaml:"primary_background,omitempty"`
	Cancel            bool   `yaml:"cancel,omitempty"`
}

// ReportVariant — именованный вариант компоновки и данных отчёта. Пустой Query
// наследует основной запрос; Filters задают постоянный отбор до лимита строк.
// Пустой выбор соответствует основному запросу и блоку composition.
type ReportVariant struct {
	Name        string          `yaml:"name"`
	Composition *Composition    `yaml:"composition"`
	Query       string          `yaml:"query,omitempty"`
	Filters     []VariantFilter `yaml:"filters,omitempty"`
}

// Composition описывает настройки компоновки данных отчёта.
type Composition struct {
	Groupings    []string   `yaml:"groupings"`
	Columns      []string   `yaml:"columns"` // непусто = режим кросс-таблицы: измерения в колонки
	Measures     []Measure  `yaml:"measures"`
	Totals       Totals     `yaml:"totals"`
	Detail       bool       `yaml:"detail"`
	Sort         []SortKey  `yaml:"sort"`
	Conditional  []CondRule `yaml:"conditional"`
	Appearance   Appearance `yaml:"appearance"` // общее оформление вывода (линии, зебра)
	Chart        *ChartSpec `yaml:"chart"`
	DetailLink   string     `yaml:"detail_link"`   // поле строки с UUID регистратора/ссылки
	DetailEntity string     `yaml:"detail_entity"` // имя сущности для перехода по ссылке
	// DetailFields — поля детальной записи (аналог «Выбранных полей» 1С):
	// колонки, которые в строке детализации показывают значения записи
	// (номер, наименование, даты), а в строках групп и итогов пусты.
	DetailFields []DetailField `yaml:"detail_fields,omitempty"`
	// Formats — формат подписи значения группировки или колонки кросс-таблицы,
	// строкой Формат() 1С (как «Формат» поля схемы СКД): {Месяц: "ДФ='ММММ гг'"}
	// → «Сентябрь 26». Меняет только подпись: группировка и сортировка — по
	// значению.
	Formats map[string]string `yaml:"formats,omitempty"`
	// ColumnGroupTotals — в кросс-таблице с несколькими группировками колонок
	// после подколонок каждого значения верхней группировки идёт её итог
	// (1С: РасположениеГруппировки = Конец).
	ColumnGroupTotals bool `yaml:"column_group_totals,omitempty"`
	// RowTotals — общий итог строки справа в кросс-таблице; nil = выводится,
	// false = нет (1С: ГоризонтальноеРасположениеОбщихИтогов = Нет).
	RowTotals *bool `yaml:"row_totals,omitempty"`
}

// ShowRowTotals — выводить ли общий итог строки справа в кросс-таблице.
func (c *Composition) ShowRowTotals() bool { return c == nil || c.RowTotals == nil || *c.RowTotals }

// DetailField — колонка детальной записи отчёта: поле строки запроса и
// заголовок (пустой = имя поля).
type DetailField struct {
	Field string `yaml:"field"`
	Title string `yaml:"title,omitempty"`
}

// Measure описывает измеримый показатель (поле + агрегат) в компоновке.
type Measure struct {
	Field  string `yaml:"field"`
	Agg    string `yaml:"agg"` // sum|count|count_distinct|avg|min|max ("" = sum)
	Title  string `yaml:"title"`
	Align  string `yaml:"align"`  // left|right|center ("" = right)
	Format string `yaml:"format"` // "" | "#,##0.00" | "#,##0" | "0.0%" и т.п.
	Expr   string `yaml:"expr"`   // непустое = вычисляемый показатель (DSL-выражение по другим полям)
	// Scope limits resource values to named row groupings, $grand and/or $detail.
	// Empty means every level, preserving existing reports.
	Scope        []string             `yaml:"scope,omitempty" json:",omitempty"`
	Calculations []MeasureCalculation `yaml:"calculations,omitempty" json:",omitempty"`
}

// Totals управляет выводом итогов в отчёте.
type Totals struct {
	Grand     bool `yaml:"grand"`
	Subtotals bool `yaml:"subtotals"`
}

// SortKey задаёт поле и направление сортировки.
type SortKey struct {
	Field string `yaml:"field"`
	Dir   string `yaml:"dir"` // asc|desc
}

// CondRule описывает правило условного оформления строки или ячейки.
type CondRule struct {
	When  string    `yaml:"when"`
	Field string    `yaml:"field"` // "" = вся строка
	Style CellStyle `yaml:"style"`
}

// CellStyle определяет стиль ячейки для условного оформления.
type CellStyle struct {
	Color      string `yaml:"color"`
	Background string `yaml:"background"`
	Bold       bool   `yaml:"bold"`
	Italic     bool   `yaml:"italic"`
}

// Appearance — общее оформление вывода компоновки (план 71): свойства всей
// таблицы, не зависящие от условий (в отличие от Conditional). Нулевое значение
// = исторический вид (горизонтальные линии, без зебры) — существующие отчёты не
// меняются. Lines: "" или "horizontal" — только нижние границы (как было);
// "vertical" — только вертикальные; "both" — полная сетка; "none" — без линий.
//
// CollapseTo — начальное состояние групп при открытии отчёта (issue #575):
// уровни 0..N развёрнуты, глубже — свёрнуто. 0 — видны только группы верхнего
// уровня со своими итогами. nil (ключа нет) — всё развёрнуто, как было. Поле
// указательное именно поэтому: `collapse_to: 0` — осмысленное значение, и от
// «не задано» его надо отличать.
type Appearance struct {
	Lines      string `yaml:"lines" json:"lines,omitempty"`
	Zebra      bool   `yaml:"zebra" json:"zebra,omitempty"`
	CollapseTo *int   `yaml:"collapse_to,omitempty" json:"collapse_to,omitempty"`
}

// ChartSpec задаёт параметры диаграммы, встроенной в отчёт.
type ChartSpec struct {
	Type     string   `yaml:"type"` // bar|line|pie
	Category string   `yaml:"category"`
	Series   []string `yaml:"series"`
}

// DisplayLabel возвращает подпись параметра с учётом языка.
func (p *Param) DisplayLabel(lang string) string {
	if lang != "" {
		if v, ok := p.Labels[lang]; ok && v != "" {
			return v
		}
	}
	if p.Label != "" {
		return p.Label
	}
	return p.Name
}

// DisplayName возвращает заголовок отчёта с учётом языка.
func (r *Report) DisplayName(lang string) string {
	if lang != "" {
		if v, ok := r.Titles[lang]; ok && v != "" {
			return v
		}
	}
	if r.Title != "" {
		return r.Title
	}
	return r.Name
}

// ActiveComposition возвращает компоновку выбранного варианта по его имени.
// Пустое имя или несовпадение → основной composition (вариант по умолчанию).
func (r *Report) ActiveComposition(name string) *Composition {
	if name != "" {
		for i := range r.Variants {
			if r.Variants[i].Name == name {
				return r.Variants[i].Composition
			}
		}
	}
	return r.Composition
}

func LoadFile(path string) (*Report, error) {
	data, err := os.ReadFile(path)
	if err != nil {
		return nil, err
	}
	return ParseBytes(data)
}

// ParseBytes разбирает YAML отчёта из памяти (без файла). Нужна для внешних
// отчётов, хранящихся в БД (см. internal/extform).
func ParseBytes(data []byte) (*Report, error) {
	var r Report
	if err := yaml.Unmarshal(data, &r); err != nil {
		return nil, err
	}
	return &r, nil
}
