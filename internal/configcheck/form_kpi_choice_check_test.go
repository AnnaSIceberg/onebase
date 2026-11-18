package configcheck

import (
	"path/filepath"
	"testing"

	"github.com/ivantit66/onebase/internal/project"
)

func TestRunFullKPIChoiceOptIn(t *testing.T) {
	for _, tc := range []struct {
		name, kind, config string
		bad                bool
	}{
		{"valid", "kpi", `{all_when_empty: Объект.Branch, label_suffix: Branch}`, false},
		{"ordinary processor", "", `{all_when_empty: Объект.Branch, label_suffix: Branch}`, true},
		{"form source", "kpi", `{all_when_empty: Форма.Branch, label_suffix: Branch}`, true},
		{"missing source", "kpi", `{label_suffix: Branch}`, true},
		{"foreign suffix", "kpi", `{all_when_empty: Объект.Branch, label_suffix: Missing}`, true},
		{"unknown key", "kpi", `{all_when_empty: Объект.Branch, label_suffix: Branch, fallback: all}`, true},
	} {
		t.Run(tc.name, func(t *testing.T) {
			dir := t.TempDir()
			mkFile(t, filepath.Join(dir, "catalogs", "Branches.yaml"), "name: Branches\nfields:\n  - {name: Name, type: string}\n")
			mkFile(t, filepath.Join(dir, "catalogs", "Directions.yaml"), "name: Directions\nfields:\n  - {name: Name, type: string}\n  - {name: Branch, type: 'reference:Branches'}\n")
			mkFile(t, filepath.Join(dir, "processors", "KPI.yaml"), "name: KPI\nkind: "+tc.kind+"\nparams:\n  - {name: Branch, type: 'reference:Branches'}\n  - {name: Direction, type: 'reference:Directions'}\n")
			mkFile(t, filepath.Join(dir, "forms", "kpi", "Form.form.yaml"), `schema: onebase.form/v1
form:
  name: Form
  kind: object
  entity: KPI
elements:
  - id: direction
    name: DirectionField
    kind: ПолеВвода
    data_path: Объект.Direction
    choice_filter:
      - {field: Branch, op: eq, from: Объект.Branch}
    kpi_choice: `+tc.config+"\n")
			if !tc.bad {
				proj, err := project.Load(dir)
				if err != nil || len(proj.Processors) != 1 || len(proj.Processors[0].Forms) != 1 || proj.Processors[0].Forms[0].Elements[0].KPIChoice == nil {
					t.Fatalf("public project loader lost KPI metadata: %v", err)
				}
			}
			result := RunFull(dir)
			if tc.bad {
				if result.OK {
					t.Fatalf("invalid KPI metadata accepted: %+v", result)
				}
			} else {
				for _, issue := range result.Issues {
					if issue.Code == "form.kpi-choice" || issue.Code == "form.choice-filter" {
						t.Fatalf("valid KPI contract rejected: %+v", issue)
					}
				}
			}
		})
	}
}
