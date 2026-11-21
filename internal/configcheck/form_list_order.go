package configcheck

import (
	"fmt"
	"slices"
	"strings"

	"github.com/ivantit66/onebase/internal/metadata"
	"github.com/ivantit66/onebase/internal/project"
)

// CheckFormListOrderBy verifies physical fields used as default sort keys.
func CheckFormListOrderBy(proj *project.Project) []Issue {
	var issues []Issue
	for _, entity := range proj.Entities {
		for _, form := range entity.Forms {
			if form == nil {
				continue
			}
			for _, spec := range form.ListOrderBy {
				name, _ := metadata.SplitOrderSpec(spec)
				known := slices.ContainsFunc(entity.Fields, func(field metadata.Field) bool {
					return strings.EqualFold(field.Name, name)
				})
				if strings.EqualFold(form.Kind, "list") && known {
					continue
				}
				issues = append(issues, Issue{
					File: formFileLabel(entity, form), Object: entity.Name, Kind: "Управляемая форма",
					Code:         "form.list-order-by",
					Message:      fmt.Sprintf("form.order_by: %q не является реквизитом формы списка %q", name, entity.Name),
					SuggestedFix: "укажите реквизиты сущности в шапке формы kind: list",
				})
			}
		}
	}
	return issues
}
