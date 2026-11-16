package configcheck

import (
	"testing"

	"github.com/ivantit66/onebase/internal/metadata"
	"github.com/ivantit66/onebase/internal/project"
	"github.com/ivantit66/onebase/internal/report"
)

func TestDetailSortNativeVariantActualProjection(t *testing.T) {
	entity := &metadata.Entity{Name: "NativeDetailLine", Kind: metadata.KindCatalog, Fields: []metadata.Field{{Name: "Label", Type: "string"}, {Name: "Amount", Type: "number"}}}
	first := "ВЫБРАТЬ Label КАК first ИЗ Справочник.NativeDetailLine"
	second := "ВЫБРАТЬ Amount КАК second ИЗ Справочник.NativeDetailLine"
	for _, tc := range []struct {
		name, main, variant, sort string
		wantIssue                 bool
	}{
		{name: "variant owns projection", main: first, variant: second, sort: "second"},
		{name: "main alias cannot leak", main: first, variant: second, sort: "first", wantIssue: true},
		{name: "queryless main", variant: second, sort: "second"},
		{name: "queryless main unknown", variant: second, sort: "first", wantIssue: true},
		{name: "inherits main", main: first, sort: "first"},
		{name: "inherits main rejects foreign", main: first, sort: "second", wantIssue: true},
		{name: "whitespace inherits main", main: first, variant: " \n\t ", sort: "first"},
		{name: "whitespace inheritance rejects foreign", main: first, variant: " \n\t ", sort: "second", wantIssue: true},
	} {
		t.Run(tc.name, func(t *testing.T) {
			rep := &report.Report{Name: "NativeVariantDetail", Query: tc.main, Variants: []report.ReportVariant{{Name: "variant", Query: tc.variant, Composition: &report.Composition{Detail: true, Sort: []report.SortKey{{Field: tc.sort, Dir: "asc"}}}}}}
			issues := CheckReportComposition(&project.Project{Entities: []*metadata.Entity{entity}, Reports: []*report.Report{rep}})
			if got := issuesContain(issues, "поле сортировки"); got != tc.wantIssue {
				t.Fatalf("actual variant projection issue=%v want=%v: %+v", got, tc.wantIssue, issues)
			}
		})
	}
}
