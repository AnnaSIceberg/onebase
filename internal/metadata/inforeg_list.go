package metadata

import (
	"fmt"
	"strings"
)

// InfoRegisterList configures the presentation of the register list only.
type InfoRegisterList struct {
	ExceptRoles []string                 `yaml:"except_roles,omitempty"`
	Columns     []InfoRegisterListColumn `yaml:"columns"`
}

type InfoRegisterListColumn struct {
	Field     string   `yaml:"field"`
	Label     string   `yaml:"label,omitempty"`
	Roles     []string `yaml:"roles,omitempty"`
	AdminOnly bool     `yaml:"admin_only,omitempty"`
}

func validateInfoRegisterList(ir *InfoRegister) error {
	if ir.List == nil {
		return nil
	}
	if len(ir.List.Columns) == 0 {
		return fmt.Errorf("list.columns: укажите хотя бы одну колонку")
	}
	known := make(map[string]string, len(ir.Dimensions)+len(ir.Resources)+1)
	if ir.Periodic {
		known["period"] = "period"
	}
	for _, fields := range [][]Field{ir.Dimensions, ir.Resources} {
		for _, field := range fields {
			known[strings.ToLower(field.Name)] = field.Name
		}
	}
	seen := make(map[string]bool, len(ir.List.Columns))
	for i := range ir.List.Columns {
		column := &ir.List.Columns[i]
		key := strings.ToLower(strings.TrimSpace(column.Field))
		name, ok := known[key]
		if !ok {
			return fmt.Errorf("list.columns: поле %q не найдено в регистре", column.Field)
		}
		if seen[key] {
			return fmt.Errorf("list.columns: поле %q указано дважды", column.Field)
		}
		seen[key] = true
		column.Field = name
		column.Roles = trimStringList(column.Roles)
	}
	ir.List.ExceptRoles = trimStringList(ir.List.ExceptRoles)
	return nil
}
