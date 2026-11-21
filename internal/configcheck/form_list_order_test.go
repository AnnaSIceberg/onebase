package configcheck

import (
	"strings"
	"testing"

	"github.com/ivantit66/onebase/internal/metadata"
	"github.com/ivantit66/onebase/internal/project"
)

func TestCheckFormListOrderBy(t *testing.T) {
	form := &metadata.FormModule{Name: "Список", Kind: "list", ListOrderBy: []string{"ранг desc", "Наименование"}}
	proj := &project.Project{Entities: []*metadata.Entity{{
		Name: "Мастера", Fields: []metadata.Field{{Name: "Ранг"}, {Name: "Наименование"}},
		Forms: []*metadata.FormModule{form},
	}}}
	if issues := CheckFormListOrderBy(proj); len(issues) != 0 {
		t.Fatalf("valid order: %v", issues)
	}

	form.ListOrderBy = append(form.ListOrderBy, "НетПоля")
	if issues := CheckFormListOrderBy(proj); len(issues) != 1 ||
		issues[0].Code != "form.list-order-by" || !strings.Contains(issues[0].Message, "НетПоля") {
		t.Fatalf("unknown field: %v", issues)
	}
}
