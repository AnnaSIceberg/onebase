package compose_test

import (
	"strings"
	"testing"

	"github.com/ivantit66/onebase/internal/report"
	"github.com/ivantit66/onebase/internal/report/compose"
	"github.com/ivantit66/onebase/internal/report/expreval"
	"github.com/shopspring/decimal"
)

type availabilityEvaluator struct {
	inner compose.Evaluator
	calls map[string]int
}

func (e *availabilityEvaluator) EvalBool(expr string, row compose.Row) (bool, error) {
	return e.inner.EvalBool(expr, row)
}

func (e *availabilityEvaluator) EvalNum(expr string, row compose.Row) (decimal.Decimal, bool, error) {
	e.calls[expr]++
	return e.inner.EvalNum(expr, row)
}

func TestScopeUnavailableTransitiveAndOrdinaryNil(t *testing.T) {
	for _, tc := range []struct {
		name     string
		value    any
		excluded bool
	}{
		{name: "ordinary nil", value: nil},
		{name: "ordinary zero", value: 0},
		{name: "scope unavailable", value: 7, excluded: true},
	} {
		for _, reverse := range []bool{false, true} {
			t.Run(tc.name+map[bool]string{false: "/ordered", true: "/reordered"}[reverse], func(t *testing.T) {
				n := report.Measure{Field: "N", Agg: "min"}
				if tc.excluded {
					n.Scope = []string{report.MeasureScopeGrand}
				}
				measures := []report.Measure{n,
					{Field: "X", Expr: "0", Calculations: []report.MeasureCalculation{{Level: "G", Expr: "N * 2"}}},
					{Field: "Y", Expr: "1", Calculations: []report.MeasureCalculation{{Level: "G", Expr: "X + 1"}}},
					{Field: "Z", Expr: "2", Calculations: []report.MeasureCalculation{{Level: "G", Expr: "Y + 1"}}},
				}
				if reverse {
					measures = []report.Measure{measures[3], measures[2], measures[1], measures[0]}
				}
				ev := &availabilityEvaluator{inner: expreval.New(nil, expreval.DefaultProfile()), calls: map[string]int{}}
				result, err := compose.Compose([]compose.Row{{"G": "a", "N": tc.value}}, report.Composition{Groupings: []string{"G"}, Measures: measures}, ev)
				if err != nil || len(result.Warnings) != 0 {
					t.Fatalf("compose: %v %+v", err, result)
				}
				got := result.Groups[0].Subtotals
				if tc.excluded {
					for _, field := range []string{"N", "X", "Y", "Z"} {
						if got[field] != nil {
							t.Fatalf("unavailable %s became %v", field, got[field])
						}
					}
				} else {
					number(t, got["X"], "0")
					number(t, got["Y"], "1")
					number(t, got["Z"], "2")
				}
				for _, expr := range []string{"N * 2", "X + 1", "Y + 1"} {
					want := 1
					if tc.excluded {
						want = 0
					}
					if ev.calls[expr] != want {
						t.Fatalf("EvalNum(%q) calls=%d want=%d", expr, ev.calls[expr], want)
					}
				}
			})
		}
	}
}

func TestScopeUnavailableDoesNotPoisonActiveDuplicateBinding(t *testing.T) {
	for _, tc := range []struct {
		name string
		m    report.Measure
		v    any
		want string
	}{
		{name: "aggregate", m: report.Measure{Field: "N", Agg: "min"}, v: 7, want: "14"},
		{name: "available nil", m: report.Measure{Field: "N", Agg: "min"}, want: "0"},
		{name: "expression", m: report.Measure{Field: "N", Expr: "7"}, want: "14"},
	} {
		for _, excludedFirst := range []bool{false, true} {
			t.Run(tc.name+map[bool]string{false: "/excluded-last", true: "/excluded-first"}[excludedFirst], func(t *testing.T) {
				excluded := report.Measure{Field: "N", Agg: "sum", Scope: []string{report.MeasureScopeGrand}}
				measures := []report.Measure{tc.m, excluded}
				if excludedFirst {
					measures = []report.Measure{excluded, tc.m}
				}
				measures = append(measures, report.Measure{Field: "X", Expr: "N * 2"})
				result, err := compose.Compose([]compose.Row{{"G": "a", "N": tc.v}}, report.Composition{Groupings: []string{"G"}, Measures: measures}, expreval.New(nil, expreval.DefaultProfile()))
				if err != nil || len(result.Warnings) != 0 {
					t.Fatalf("compose: %v %+v", err, result)
				}
				number(t, result.Groups[0].Subtotals["X"], tc.want)
			})
		}
	}
}

func TestScopeUnavailableSelfReadUsesActualPreviousWriter(t *testing.T) {
	for _, scoped := range []bool{false, true} {
		previous := report.Measure{Field: "N", Agg: "min"}
		value := any(nil)
		if scoped {
			previous.Scope = []string{report.MeasureScopeGrand}
			value = 7
		}
		spec := report.Composition{Groupings: []string{"G"}, Measures: []report.Measure{previous, {Field: "N", Expr: "N + 1"}, {Field: "X", Expr: "N * 2"}}}
		ev := &availabilityEvaluator{inner: expreval.New(nil, expreval.DefaultProfile()), calls: map[string]int{}}
		result, err := compose.Compose([]compose.Row{{"G": "a", "N": value}}, spec, ev)
		if err != nil || len(result.Warnings) != 0 {
			t.Fatalf("self binding: %v %+v", err, result)
		}
		if scoped {
			number(t, result.Grand["N"], "8")
			number(t, result.Grand["X"], "16")
			if result.Groups[0].Subtotals["N"] != nil || result.Groups[0].Subtotals["X"] != nil || ev.calls["N + 1"] != 1 || ev.calls["N * 2"] != 1 {
				t.Fatal("pending self slot replaced an actual excluded writer")
			}
		} else {
			number(t, result.Groups[0].Subtotals["N"], "1")
			number(t, result.Groups[0].Subtotals["X"], "2")
			if ev.calls["N + 1"] != 2 || ev.calls["N * 2"] != 2 {
				t.Fatal("ordinary available nil self-read changed")
			}
		}
	}
}

func TestScopeUnavailableMissingP6PropagatesAndZeroRemains(t *testing.T) {
	spec := report.Composition{Groupings: []string{"G"}, Columns: []string{"Year"}, Measures: []report.Measure{
		{Field: "Amount", Agg: "sum"}, {Field: "X", Expr: "Amount - Previous_Amount"}, {Field: "Y", Expr: "X + 1"},
	}}
	result, err := compose.ComposeCross([]compose.Row{{"G": "a", "Year": 1, "Amount": 0}, {"G": "a", "Year": 2, "Amount": 0}}, spec, expreval.New(nil, expreval.DefaultProfile()))
	if err != nil || len(result.Warnings) != 0 {
		t.Fatalf("cross: %v %+v", err, result)
	}
	for _, col := range result.Cols {
		if col.Measure != "X" && col.Measure != "Y" {
			continue
		}
		got := result.Rows[0].Cells[col.Key()]
		if col.Path[0] == "1" {
			if got != nil {
				t.Fatalf("missing P6 %s became %v", col.Measure, got)
			}
		} else if col.Measure == "X" {
			number(t, got, "0")
		} else {
			number(t, got, "1")
		}
	}
}

func TestScopeUnavailableCycleClosureKeepsDiagnostics(t *testing.T) {
	spec := report.Composition{Groupings: []string{"G"}, Measures: []report.Measure{
		{Field: "N", Agg: "sum", Scope: []string{report.MeasureScopeGrand}},
		{Field: "X", Expr: "Y + N"}, {Field: "Y", Expr: "X + 1"},
	}}
	ev := &availabilityEvaluator{inner: expreval.New(nil, expreval.DefaultProfile()), calls: map[string]int{}}
	result, err := compose.Compose([]compose.Row{{"G": "a", "N": 7}}, spec, ev)
	if err != nil || result.Groups[0].Subtotals["X"] != nil || result.Groups[0].Subtotals["Y"] != nil {
		t.Fatalf("cycle availability: %v %+v", err, result)
	}
	if ev.calls["Y + N"] != 1 || ev.calls["X + 1"] != 1 {
		t.Fatal("scope-blocked cycle called evaluator")
	}
	control := spec
	control.Measures = append([]report.Measure(nil), spec.Measures...)
	control.Measures[0].Scope = nil
	baseline, baselineErr := compose.Compose([]compose.Row{{"G": "a", "N": 7}}, control, expreval.New(nil, expreval.DefaultProfile()))
	if baselineErr != nil || strings.Join(result.Warnings, "\n") != strings.Join(baseline.Warnings, "\n") || !strings.Contains(strings.Join(result.Warnings, "\n"), "циклическая зависимость") {
		t.Fatalf("genuine active cycle diagnostic changed: %+v", result.Warnings)
	}
}

func TestScopeUnavailableCasefoldPrefixedNameAndAvailableNil(t *testing.T) {
	for _, scoped := range []bool{false, true} {
		m := report.Measure{Field: "Grand_Amount", Agg: "min"}
		if scoped {
			m.Scope = []string{report.MeasureScopeGrand}
		}
		spec := report.Composition{Groupings: []string{"G"}, Measures: []report.Measure{{Field: "Amount", Agg: "sum"}, m, {Field: "X", Expr: "gRaNd_AmOuNt * 2"}}}
		result, err := compose.Compose([]compose.Row{{"G": "a", "Amount": 100, "Grand_Amount": nil}}, spec, expreval.New(nil, expreval.DefaultProfile()))
		if err != nil || len(result.Warnings) != 0 {
			t.Fatalf("prefixed binding: %v %+v", err, result)
		}
		number(t, result.Grand["X"], "0")
		if scoped {
			if result.Groups[0].Subtotals["X"] != nil {
				t.Fatal("casefold scope-excluded declared name became a synthetic P6 reference")
			}
		} else {
			number(t, result.Groups[0].Subtotals["X"], "0")
		}
	}
}

func TestScopeUnavailableIgnoresFunctionAndStringNames(t *testing.T) {
	spec := report.Composition{Groupings: []string{"G"}, Measures: []report.Measure{
		{Field: "Round", Agg: "sum", Scope: []string{report.MeasureScopeGrand}},
		{Field: "Marker", Agg: "sum", Scope: []string{report.MeasureScopeGrand}},
		{Field: "X", Expr: "Round(1.25, 0)"}, {Field: "Y", Expr: `StrLen("Marker")`},
	}}
	result, err := compose.Compose([]compose.Row{{"G": "a"}}, spec, expreval.New(nil, expreval.DefaultProfile()))
	if err != nil || len(result.Warnings) != 0 {
		t.Fatalf("non-binding identifiers: %v %+v", err, result)
	}
	number(t, result.Groups[0].Subtotals["X"], "1")
	number(t, result.Groups[0].Subtotals["Y"], "6")
}
