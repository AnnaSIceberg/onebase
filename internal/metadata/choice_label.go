package metadata

import (
	"fmt"
	"strings"
)

type ChoiceLabel struct {
	SuffixField string   `yaml:"suffix_field"`
	ExceptRoles []string `yaml:"except_roles,omitempty"`
}

func validateChoiceLabel(e *Entity) error {
	if e.ChoiceLabel == nil {
		return nil
	}
	if e.Kind != KindCatalog {
		return fmt.Errorf("entity %s: choice_label is available for catalogs only", e.Name)
	}
	suffix := strings.TrimSpace(e.ChoiceLabel.SuffixField)
	for _, field := range e.Fields {
		if strings.EqualFold(field.Name, suffix) && field.RefEntity != "" && !IsSystemRefTarget(field.RefEntity) {
			return nil
		}
	}
	return fmt.Errorf("entity %s: choice_label.suffix_field must name a reference to a configuration entity", e.Name)
}
