package ui

import (
	"strings"

	"github.com/ivantit66/onebase/internal/auth"
	"github.com/ivantit66/onebase/internal/metadata"
)

type infoRegListColumn struct {
	Field    metadata.Field
	Resource bool
}

func infoRegListColumns(ir *metadata.InfoRegister, user *auth.User, periodTitle string) []infoRegListColumn {
	fields := infoRegisterDetailFields(ir, periodTitle)
	resources := make(map[string]bool, len(ir.Resources))
	for _, field := range ir.Resources {
		resources[field.Name] = true
	}
	ordinary := make([]infoRegListColumn, 0, len(fields))
	for _, field := range fields {
		ordinary = append(ordinary, infoRegListColumn{Field: field, Resource: resources[field.Name]})
	}
	layout := ir.List
	if layout == nil || !auth.RoleScopeMatches(user, nil, layout.ExceptRoles, false) {
		return ordinary
	}
	byName := make(map[string]infoRegListColumn, len(ordinary))
	for _, column := range ordinary {
		byName[strings.ToLower(column.Field.Name)] = column
	}
	visible := make([]infoRegListColumn, 0, len(layout.Columns))
	for _, spec := range layout.Columns {
		if spec.AdminOnly && (user == nil || !user.IsAdmin) {
			continue
		}
		if !auth.RoleScopeMatches(user, spec.Roles, nil, false) {
			continue
		}
		column, ok := byName[strings.ToLower(spec.Field)]
		if !ok {
			continue
		}
		if spec.Label != "" {
			column.Field.Title = spec.Label
			column.Field.Titles = nil
		}
		visible = append(visible, column)
	}
	return visible
}
