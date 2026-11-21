package metadata

import (
	"fmt"
	"os"
	"path/filepath"
	"strings"

	"gopkg.in/yaml.v3"
)

type Journal struct {
	Name        string
	Title       string
	Titles      map[string]string
	Documents   []string
	Columns     []JournalColumn
	Filters     []JournalFilter
	Conditional []JournalCondRule
}

// DisplayName возвращает заголовок журнала документов с учётом языка.
func (j *Journal) DisplayName(lang string) string {
	if lang != "" {
		if v, ok := j.Titles[lang]; ok && v != "" {
			return v
		}
	}
	if j.Title != "" {
		return j.Title
	}
	return j.Name
}

// JournalDocumentKindField is the built-in source document type column.
const JournalDocumentKindField = "_doc_kind"

// JournalDeletionMarkField exposes the document's stored deletion flag.
const JournalDeletionMarkField = "deletion_mark"

type JournalColumn struct {
	Field    string
	Label    string
	Labels   map[string]string
	Fallback []string
	// Map — явное соответствие docTypeName → fieldName в этом документе.
	// лучшая альтернатива fallback, потому что:
	//   1. Explicit — видно для какого документа какое поле.
	//   2. Не зависит от порядка обхода fallback (raw COALESCE мог давать
	//      нежелательные совпадения если в документе два совпадающих поля).
	//   3. UI может показывать tooltip «в документе X это поле называется Y».
	// Если Map[doc] есть — используется он, fallback не применяется. Если
	// не задан — работает старый fallback-механизм (back compat).
	Map    map[string]string
	Format string // "date" | "number" | "" (auto)
	// ReferencePath is an opt-in display projection through one linked record.
	ReferencePath string
}

func (c JournalColumn) IsDocumentKind() bool {
	return strings.EqualFold(strings.TrimSpace(c.Field), JournalDocumentKindField)
}

func (c JournalColumn) IsDeletionMark() bool {
	return strings.EqualFold(strings.TrimSpace(c.Field), JournalDeletionMarkField)
}

func (c JournalColumn) referenceParts() (source, attribute string, ok bool) {
	source, attribute, ok = strings.Cut(c.ReferencePath, ".")
	return source, attribute, ok && ValidIdent(source) && ValidIdent(attribute)
}

// ReferenceSource returns the header field selected from the document.
func (c JournalColumn) ReferenceSource(doc *Entity) *Field {
	root, _, ok := c.referenceParts()
	if !ok || doc == nil {
		return nil
	}
	field := findEntityFieldFold(doc, root)
	if field == nil || field.RefEntity == "" {
		return nil
	}
	return field
}

type JournalReferenceProjection struct {
	Related *Entity
	Field   string
	Target  *Entity
}

func (c JournalColumn) ResolveReference(doc *Entity, lookup func(string) *Entity) (JournalReferenceProjection, bool) {
	source := c.ReferenceSource(doc)
	if source == nil {
		return JournalReferenceProjection{}, false
	}
	related := lookup(source.RefEntity)
	if related == nil {
		return JournalReferenceProjection{}, false
	}
	_, attribute, _ := c.referenceParts()
	field := findEntityFieldFold(related, attribute)
	if field == nil || field.RefEntity == "" {
		return JournalReferenceProjection{}, false
	}
	target := lookup(field.RefEntity)
	if target == nil {
		return JournalReferenceProjection{}, false
	}
	return JournalReferenceProjection{Related: related, Field: field.Name, Target: target}, true
}

func (j *Journal) HasDocumentKindColumn() bool {
	for _, column := range j.Columns {
		if column.IsDocumentKind() {
			return true
		}
	}
	return false
}

// DisplayLabel возвращает подпись колонки журнала с учётом языка.
func (c JournalColumn) DisplayLabel(lang string) string {
	if lang != "" {
		if v, ok := c.Labels[lang]; ok && v != "" {
			return v
		}
	}
	if c.Label != "" {
		return c.Label
	}
	return c.Field
}

type JournalFilter struct {
	Field  string
	Label  string
	Labels map[string]string
	Type   string // "date_range" | "reference:X" | "string"
}

// DisplayLabel возвращает подпись фильтра журнала с учётом языка.
func (f JournalFilter) DisplayLabel(lang string) string {
	if lang != "" {
		if v, ok := f.Labels[lang]; ok && v != "" {
			return v
		}
	}
	if f.Label != "" {
		return f.Label
	}
	return f.Field
}

type JournalCondRule struct {
	When  string
	Field string // "" = вся строка
	Style JournalCellStyle
}

type JournalCellStyle struct {
	Color      string
	Background string
	Bold       bool
	Italic     bool
}

type rawJournal struct {
	Name      string            `yaml:"name"`
	Title     string            `yaml:"title"`
	Titles    map[string]string `yaml:"titles"`
	Documents []string          `yaml:"documents"`
	Columns   []struct {
		Field         string            `yaml:"field"`
		Label         string            `yaml:"label"`
		Labels        map[string]string `yaml:"labels"`
		Fallback      []string          `yaml:"fallback"`
		Map           map[string]string `yaml:"map"`
		Format        string            `yaml:"format"`
		ReferencePath string            `yaml:"reference_path"`
	} `yaml:"columns"`
	Filters []struct {
		Field  string            `yaml:"field"`
		Label  string            `yaml:"label"`
		Labels map[string]string `yaml:"labels"`
		Type   string            `yaml:"type"`
	} `yaml:"filters"`
	Conditional           []rawJournalCondRule `yaml:"conditional"`
	ConditionalFormatting []rawJournalCondRule `yaml:"conditional_formatting"`
}

type rawJournalCondRule struct {
	When  string           `yaml:"when"`
	Field string           `yaml:"field"`
	Style JournalCellStyle `yaml:"style"`
	Then  JournalCellStyle `yaml:"then"`
}

func LoadJournalFile(path string) (*Journal, error) {
	data, err := os.ReadFile(path)
	if err != nil {
		return nil, err
	}
	var raw rawJournal
	if err := yaml.Unmarshal(data, &raw); err != nil {
		return nil, err
	}
	if raw.Name == "" {
		raw.Name = strings.TrimSuffix(filepath.Base(path), ".yaml")
	}
	if raw.Title == "" {
		raw.Title = raw.Name
	}
	j := &Journal{
		Name:        raw.Name,
		Title:       raw.Title,
		Titles:      raw.Titles,
		Documents:   raw.Documents,
		Conditional: rawJournalConditional(raw),
	}
	kindSeen, markSeen := false, false
	referenceColumns := make(map[string]bool)
	usedColumns := make(map[string]bool)
	for _, rc := range raw.Columns {
		if strings.EqualFold(strings.TrimSpace(rc.Field), JournalDocumentKindField) {
			if kindSeen || len(rc.Map) != 0 || len(rc.Fallback) != 0 || rc.Format != "" || rc.ReferencePath != "" {
				return nil, fmt.Errorf("журнал %s: %s допускается один раз без map, fallback и format", j.Name, JournalDocumentKindField)
			}
			kindSeen = true
			rc.Field = JournalDocumentKindField
		}
		if strings.EqualFold(strings.TrimSpace(rc.Field), JournalDeletionMarkField) {
			if markSeen || len(rc.Map) != 0 || len(rc.Fallback) != 0 || rc.Format != "" || rc.ReferencePath != "" {
				return nil, fmt.Errorf("журнал %s: %s допускается один раз без дополнительных настроек источника", j.Name, JournalDeletionMarkField)
			}
			markSeen = true
			rc.Field = JournalDeletionMarkField
		}
		rc.ReferencePath = strings.TrimSpace(rc.ReferencePath)
		alias := strings.ToLower(rc.Field)
		if usedColumns[alias] && (rc.ReferencePath != "" || referenceColumns[alias]) {
			return nil, fmt.Errorf("журнал %s: повторная колонка %q конфликтует с reference_path", j.Name, rc.Field)
		}
		usedColumns[alias] = true
		if rc.ReferencePath != "" {
			column := JournalColumn{ReferencePath: rc.ReferencePath}
			_, _, valid := column.referenceParts()
			if !valid || !ValidIdent(rc.Field) || systemColumns[strings.ToLower(rc.Field)] ||
				len(rc.Map) != 0 || len(rc.Fallback) != 0 || rc.Format != "" {
				return nil, fmt.Errorf("журнал %s: колонка %q: reference_path ожидает Ссылка.Реквизит без map, fallback и format", j.Name, rc.Field)
			}
			referenceColumns[alias] = true
		}
		label := rc.Label
		if label == "" {
			label = rc.Field
		}
		j.Columns = append(j.Columns, JournalColumn{
			Field:         rc.Field,
			Label:         label,
			Labels:        rc.Labels,
			Fallback:      rc.Fallback,
			Map:           rc.Map,
			Format:        rc.Format,
			ReferencePath: rc.ReferencePath,
		})
	}
	for _, rf := range raw.Filters {
		if referenceColumns[strings.ToLower(rf.Field)] {
			return nil, fmt.Errorf("журнал %s: колонка reference_path %q доступна только для отображения", j.Name, rf.Field)
		}
		ft := rf.Type
		if ft == "" {
			ft = "string"
		}
		j.Filters = append(j.Filters, JournalFilter{Field: rf.Field, Label: rf.Label, Labels: rf.Labels, Type: ft})
	}
	return j, nil
}

func rawJournalConditional(raw rawJournal) []JournalCondRule {
	rawRules := append([]rawJournalCondRule{}, raw.Conditional...)
	rawRules = append(rawRules, raw.ConditionalFormatting...)
	out := make([]JournalCondRule, 0, len(rawRules))
	for _, rr := range rawRules {
		style := rr.Style
		if journalStyleZero(style) {
			style = rr.Then
		}
		out = append(out, JournalCondRule{When: rr.When, Field: rr.Field, Style: style})
	}
	return out
}

func journalStyleZero(s JournalCellStyle) bool {
	return s.Color == "" && s.Background == "" && !s.Bold && !s.Italic
}

func LoadJournalDir(dir string) ([]*Journal, error) {
	items, err := os.ReadDir(dir)
	if os.IsNotExist(err) {
		return nil, nil
	}
	if err != nil {
		return nil, err
	}
	var journals []*Journal
	for _, item := range items {
		if item.IsDir() || !strings.HasSuffix(item.Name(), ".yaml") {
			continue
		}
		j, err := LoadJournalFile(filepath.Join(dir, item.Name()))
		if err != nil {
			return nil, err
		}
		journals = append(journals, j)
	}
	return journals, nil
}
