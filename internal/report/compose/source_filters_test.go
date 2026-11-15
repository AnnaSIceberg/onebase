package compose

import (
	"github.com/google/uuid"
	"github.com/ivantit66/onebase/internal/report"
	"github.com/shopspring/decimal"
	"reflect"
	"strings"
	"testing"
)

func TestApplySourceFilters(t *testing.T) {
	id := uuid.New()
	other := uuid.New()
	rows := []Row{{"Label": "ALPHA", "Amount": decimal.NewFromInt(1), "Ref": id}, {"Label": "BETA", "Amount": decimal.NewFromInt(2), "Ref": other}, {"Label": "EMPTY", "Amount": nil, "Ref": nil}}
	before := make([]Row, len(rows))
	for i, r := range rows {
		before[i] = make(Row)
		for k, v := range r {
			before[i][k] = v
		}
	}
	for _, tc := range []struct {
		name  string
		f     report.Filter
		count int
		deny  bool
	}{
		{"alpha", report.Filter{Field: "label", Op: "eq", Value: "ALPHA"}, 1, false},
		{"numeric", report.Filter{Field: "Amount", Op: "ge", Value: "2"}, 1, false},
		{"reference", report.Filter{Field: "Ref", Op: "eq", Value: strings.ToUpper(id.String())}, 1, false},
		{"reference-ne", report.Filter{Field: "Ref", Op: "ne", Value: id.String()}, 2, false},
		{"empty-reference", report.Filter{Field: "Ref", Op: "eq", Value: ""}, 1, false},
		{"contains", report.Filter{Field: "Label", Op: "contains", Value: "alp"}, 1, false},
		{"empty-result", report.Filter{Field: "Label", Op: "eq", Value: "NONE"}, 0, false},
		{"unknown-field", report.Filter{Field: "Unknown", Op: "eq", Value: "A"}, 0, true},
		{"unknown-op", report.Filter{Field: "Label", Op: "bogus", Value: "A"}, 0, true},
		{"ref-display-label", report.Filter{Field: "Ref", Op: "eq", Value: "Display A"}, 0, true},
		{"ref-contains", report.Filter{Field: "Ref", Op: "contains", Value: "A"}, 0, true},
	} {
		t.Run(tc.name, func(t *testing.T) {
			got, err := ApplySourceFilters(rows, []report.Filter{tc.f}, []string{"Label", "Amount", "Ref"})
			if (err != nil) != tc.deny {
				t.Fatalf("deny=%v err=%v", tc.deny, err)
			}
			if !tc.deny && len(got) != tc.count {
				t.Fatalf("rows%d want%d", len(got), tc.count)
			}
			if !reflect.DeepEqual(rows, before) {
				t.Fatal("source rows or reference types mutated")
			}
		})
	}
}
