package metadata

import (
	"fmt"
	"strings"

	"gopkg.in/yaml.v3"
)

// FormKPIChoice belongs to one field of a managed, internal kind:kpi processor.
// AllWhenEmpty names the one direct source whose explicit empty value removes
// its eq predicate. LabelSuffix names that predicate's target reference field.
type FormKPIChoice struct {
	AllWhenEmpty string `yaml:"all_when_empty"`
	LabelSuffix  string `yaml:"label_suffix"`
}

func (c *FormKPIChoice) UnmarshalYAML(node *yaml.Node) error {
	if node.Kind != yaml.MappingNode {
		return fmt.Errorf("kpi_choice must be a mapping")
	}
	seen := map[string]bool{}
	for i := 0; i+1 < len(node.Content); i += 2 {
		key, value := node.Content[i].Value, node.Content[i+1]
		if seen[key] || (key != "all_when_empty" && key != "label_suffix") || value.Kind != yaml.ScalarNode || value.Tag != "!!str" {
			return fmt.Errorf("invalid kpi_choice key or value: %s", key)
		}
		seen[key] = true
		if key == "all_when_empty" {
			c.AllWhenEmpty = value.Value
		} else {
			c.LabelSuffix = value.Value
		}
	}
	return nil
}

// ValidateKPIChoice checks the narrow metadata shape. The caller additionally
// proves processor kind, registered managed form, actor, run/read and row scope.
func ValidateKPIChoice(owner *Entity, form *FormModule, el *FormElement, target *Entity) error {
	if el == nil || el.KPIChoice == nil {
		return nil
	}
	c := el.KPIChoice
	source, ok := ParseFormChoiceSource(c.AllWhenEmpty)
	if !ok || source.Root != "Объект" || source.Deep() || strings.TrimSpace(c.AllWhenEmpty) != c.AllWhenEmpty || c.LabelSuffix == "" || strings.TrimSpace(c.LabelSuffix) != c.LabelSuffix {
		return fmt.Errorf("kpi_choice requires direct Объект source and label_suffix")
	}
	if owner == nil || form == nil || !form.IsManaged() || el.Kind != FormElementField || el.ID == "" || target == nil || target.Kind != KindCatalog {
		return fmt.Errorf("kpi_choice requires a managed reference field with id")
	}
	if _, tp := FormChoiceTablePart(form, el); tp != "" || !strings.HasPrefix(el.DataPath, "Объект.") {
		return fmt.Errorf("kpi_choice supports only processor header fields")
	}
	var from, suffix, selected *Field
	for i := range owner.Fields {
		f := &owner.Fields[i]
		if f.Name == source.Field {
			from = f
		}
		if el.DataPath == "Объект."+f.Name {
			selected = f
		}
	}
	for i := range target.Fields {
		if target.Fields[i].Name == c.LabelSuffix {
			suffix = &target.Fields[i]
		}
	}
	if from == nil || from.RefEntity == "" || from.PII || suffix == nil || suffix.PII || suffix.RefEntity != from.RefEntity || selected == nil || selected.RefEntity != target.Name {
		return fmt.Errorf("kpi_choice source, target and suffix must be compatible public references")
	}
	matches := 0
	for _, condition := range el.ChoiceFilter {
		if condition.From != c.AllWhenEmpty {
			continue
		}
		if condition.Op != FormChoiceOpEqual || condition.Field != c.LabelSuffix || condition.Value != nil || condition.Ref != "" || condition.IsQuery() || len(condition.Params) != 0 {
			return fmt.Errorf("kpi_choice all_when_empty requires one direct eq to label_suffix")
		}
		matches++
	}
	if matches != 1 {
		return fmt.Errorf("kpi_choice all_when_empty must match exactly one eq")
	}
	return nil
}
