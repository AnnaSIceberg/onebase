package configcheck

import (
	"fmt"
	"strings"

	"github.com/ivantit66/onebase/internal/metadata"
	"github.com/ivantit66/onebase/internal/project"
)

// CheckFormKPIChoice rejects this opt-in outside an internal KPI processor.
func CheckFormKPIChoice(proj *project.Project) []Issue {
	if proj == nil {
		return nil
	}
	entities := map[string]*metadata.Entity{}
	for _, e := range proj.Entities {
		if e != nil {
			entities[strings.ToLower(e.Name)] = e
		}
	}
	var issues []Issue
	check := func(owner *metadata.Entity, forms []*metadata.FormModule, permitted bool, file func(*metadata.FormModule) string) {
		for _, form := range forms {
			if form == nil {
				continue
			}
			form.Walk(func(el *metadata.FormElement) bool {
				if el == nil || el.KPIChoice == nil {
					return true
				}
				var err error
				if !permitted {
					err = fmt.Errorf("kpi_choice requires an internal kind:kpi processor")
				} else {
					target, _ := formChoiceRefSource(owner, form, el.DataPath, entities)
					err = metadata.ValidateKPIChoice(owner, form, el, target)
				}
				if err != nil {
					issues = append(issues, Issue{File: file(form), Object: owner.Name, Kind: "Управляемая форма", Code: "form.kpi-choice", Message: err.Error()})
				}
				return true
			})
		}
	}
	for _, e := range proj.Entities {
		if e != nil {
			check(e, e.Forms, false, func(f *metadata.FormModule) string { return formFileLabel(e, f) })
		}
	}
	for _, p := range proj.Processors {
		if p != nil {
			check(p.VirtualEntity(), p.Forms, p.Kind == "kpi" && !p.External, func(f *metadata.FormModule) string { return procFormFileLabel(p.Name, f) })
		}
	}
	return issues
}
