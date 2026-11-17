package resultfilter_test

import (
	"testing"

	"github.com/ivantit66/onebase/internal/metadata"
	"github.com/ivantit66/onebase/internal/query"
	reportpkg "github.com/ivantit66/onebase/internal/report"
	"github.com/ivantit66/onebase/internal/report/resultfilter"
)

func TestReportResultFilterRebindUsesCompilerSource(t *testing.T) {
	entity := &metadata.Entity{Name: "Base", Kind: metadata.KindCatalog, Fields: []metadata.Field{{Name: "Amount", Type: "number"}}}
	other := &metadata.Entity{Name: "Other", Kind: metadata.KindCatalog, Fields: []metadata.Field{{Name: "Amount", Type: "number"}}}
	lookup := func(_, name string) *metadata.Entity {
		for _, meta := range []*metadata.Entity{entity, other} {
			if meta.Name == name {
				return meta
			}
		}
		return nil
	}
	allowed := func(query.SourceRef, *metadata.Entity, string, bool) bool { return true }
	compile := func(text string) []resultfilter.Field {
		t.Helper()
		res, err := query.Compile(text, query.CompileOpts{Entities: []*metadata.Entity{entity, other}})
		if err != nil {
			t.Fatal(err)
		}
		return resultfilter.Schema(res, lookup, allowed)
	}
	base := compile("ВЫБРАТЬ Amount КАК visible ИЗ Справочник.Base")
	filters := []reportpkg.Filter{{Field: "visible", Op: "ge", Value: "2.5"}}
	for _, tc := range []struct{ name, text, field string }{
		{"renamed-same-source", "ВЫБРАТЬ Amount КАК other_alias ИЗ Справочник.Base", "other_alias"},
		{"same-alias-wrong-source", "ВЫБРАТЬ Amount КАК visible ИЗ Справочник.Other", ""},
		{"derived-alias", "ВЫБРАТЬ Amount+1 КАК visible ИЗ Справочник.Base", ""},
		{"ambiguous-source", "ВЫБРАТЬ A.Amount КАК visible ИЗ Справочник.Base КАК A ВНУТРЕННЕЕ СОЕДИНЕНИЕ Справочник.Base КАК B ПО A.Amount = B.Amount", ""},
	} {
		t.Run(tc.name, func(t *testing.T) {
			got, err := resultfilter.Rebind(base, compile(tc.text), filters)
			if tc.field == "" {
				if err == nil {
					t.Fatal("unproven alias/source accepted")
				}
				return
			}
			if err != nil || len(got) != 1 || got[0].Field != tc.field || got[0].Op != "ge" || got[0].Value != "2.5" {
				t.Fatalf("rebind: %+v %v", got, err)
			}
			if filters[0].Field != "visible" {
				t.Fatal("host filters mutated")
			}
		})
	}
}
