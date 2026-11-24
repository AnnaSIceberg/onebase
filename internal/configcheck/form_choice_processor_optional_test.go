package configcheck

import (
	"path/filepath"
	"testing"
)

func TestRunFullProcessorOptionalChoice(t *testing.T) {
	for _, tc := range []struct {
		name, processor, source, field string
		valid                          bool
	}{
		{"processor header", "", "Объект.Branch", "Branch", true},
		{"internal KPI", "kind: kpi", "Объект.Branch", "Branch", false},
		{"deep source", "", "Объект.Branch.Branch", "Branch", false},
		{"wrong target field", "", "Объект.Branch", "Name", false},
	} {
		t.Run(tc.name, func(t *testing.T) {
			dir := t.TempDir()
			mkFile(t, filepath.Join(dir, "catalogs", "Branches.yaml"), "name: Branches\nfields:\n  - {name: Name, type: string}\n")
			mkFile(t, filepath.Join(dir, "catalogs", "Directions.yaml"), "name: Directions\nfields:\n  - {name: Name, type: string}\n  - {name: Branch, type: 'reference:Branches'}\n")
			mkFile(t, filepath.Join(dir, "processors", "Filtered.yaml"), "name: Filtered\n"+tc.processor+"\nparams:\n  - {name: Branch, type: 'reference:Branches'}\n  - {name: Direction, type: 'reference:Directions'}\n")
			const form = `schema: onebase.form/v1
form:
  name: Form
  kind: processor
  entity: Filtered
elements:
  - id: direction
    name: DirectionField
    kind: ПолеВвода
    data_path: Объект.Direction
`
			mkFile(t, filepath.Join(dir, "forms", "filtered", "Form.form.yaml"), form+"    choice_filter: [{field: "+tc.field+", op: eq, from: "+tc.source+", skip_when_empty: true}]\n")
			found := false
			for _, issue := range RunFull(dir).Issues {
				found = found || issue.Code == "form.choice-filter"
			}
			if found == tc.valid {
				t.Fatalf("choice-filter validation error = %t, want %t", found, !tc.valid)
			}
		})
	}
}
