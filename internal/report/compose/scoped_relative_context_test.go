package compose_test

import (
	"testing"

	"github.com/ivantit66/onebase/internal/report"
	"github.com/ivantit66/onebase/internal/report/compose"
	"github.com/ivantit66/onebase/internal/report/expreval"
)

func TestScopedRelativeGrandAndChart(t *testing.T) {
	spec := report.Composition{Groupings: []string{"G"}, Measures: []report.Measure{
		{Field: "Amount", Agg: "sum"},
		{Field: "Share", Scope: []string{"G"}, Calculations: []report.MeasureCalculation{{Level: "G", Expr: "Amount / Grand_Amount * 100"}}},
	}}
	rows := []compose.Row{{"G": "a", "Amount": 10}, {"G": "b", "Amount": 30}}
	ev := expreval.New(nil, expreval.DefaultProfile())
	result, err := compose.Compose(rows, spec, ev)
	if err != nil || len(result.Warnings) != 0 {
		t.Fatalf("compose: %v, result=%+v", err, result)
	}
	if result.Grand["Share"] != nil {
		t.Fatal("named level calculation escaped to grand")
	}
	number(t, result.Groups[0].Subtotals["Share"], "25")
	number(t, result.Groups[1].Subtotals["Share"], "75")
	_, chart := compose.GroupByField(rows, spec, "G", ev)
	number(t, chart["a"]["Share"], "25")
	number(t, chart["b"]["Share"], "75")
}

func TestScopedRelativeCrossColumnsAndRowTotal(t *testing.T) {
	spec := report.Composition{Groupings: []string{"G"}, Columns: []string{"Year"}, Measures: []report.Measure{
		{Field: "Amount", Agg: "sum"},
		{Field: "Change", Scope: []string{"G"}, Calculations: []report.MeasureCalculation{{Level: "G", Expr: "Amount - Previous_Amount"}}},
		{Field: "FirstRatio", Expr: "Amount / First_Amount * 100", Scope: []string{"G"}},
		{Field: "Share", Expr: "Amount / Grand_Amount * 100", Scope: []string{"G"}},
		{Field: "AllChange", Expr: "Amount - Previous_Amount"},
		{Field: "Doc", Agg: "count_distinct", Scope: []string{report.MeasureScopeGrand}},
	}}
	rows := []compose.Row{
		{"G": "a", "Year": 3, "Amount": 30, "Doc": "one"}, {"G": "a", "Year": 1, "Amount": 10, "Doc": "one"},
		{"G": "a", "Year": 2, "Amount": 20, "Doc": "one"}, {"G": "b", "Year": 1, "Amount": 40, "Doc": "two"},
		{"G": "b", "Year": 3, "Amount": 0, "Doc": "two"}, {"G": "c", "Year": 2, "Amount": 0, "Doc": "two"},
	}
	result, err := compose.ComposeCross(rows, spec, expreval.New(nil, expreval.DefaultProfile()))
	if err != nil || len(result.Warnings) != 0 {
		t.Fatalf("cross: %v, result=%+v", err, result)
	}
	cell := func(group, year, field string) any {
		t.Helper()
		for _, row := range result.Rows {
			if row.Key == group {
				for _, col := range result.Cols {
					if col.Path[0] == year && col.Measure == field {
						return row.Cells[col.Key()]
					}
				}
			}
		}
		t.Fatalf("missing cell %s/%s/%s", group, year, field)
		return nil
	}
	number(t, cell("a", "2", "Change"), "10")
	number(t, cell("a", "3", "FirstRatio"), "300")
	number(t, cell("a", "3", "Share"), "30")
	if cell("a", "1", "Change") != nil || cell("b", "3", "Change") != nil || cell("c", "2", "FirstRatio") != nil {
		t.Fatal("missing previous/first period was replaced by a populated period")
	}
	if cell("a", "1", "Doc") != nil || result.Grand["Share"] != nil {
		t.Fatal("scope changed between ordinary grand and cross row levels")
	}
	number(t, result.Grand["Doc"], "2")
	for _, col := range result.Cols {
		if col.Measure == "Change" && result.RowTotal[col.Key()] != nil {
			t.Fatal("named-level expression escaped to period grand")
		}
		if col.Path[0] == "2" && col.Measure == "AllChange" {
			number(t, result.RowTotal[col.Key()], "-30")
		}
	}
}

func TestScopedRelativeDeclaredNameKeepsPriority(t *testing.T) {
	for _, prefix := range []string{"Grand_", "Previous_", "First_"} {
		t.Run(prefix, func(t *testing.T) {
			name := prefix + "Amount"
			spec := report.Composition{Groupings: []string{"G"}, Columns: []string{"Year"}, Measures: []report.Measure{
				{Field: "Amount", Agg: "sum"}, {Field: name, Agg: "sum", Scope: []string{report.MeasureScopeGrand}},
				{Field: "X", Expr: name + " * 2"},
			}}
			rows := []compose.Row{{"G": "a", "Year": 1, "Amount": 10, name: 3}, {"G": "a", "Year": 2, "Amount": 20, name: 4}}
			result, err := compose.ComposeCross(rows, spec, expreval.New(nil, expreval.DefaultProfile()))
			if err != nil || len(result.Warnings) != 0 {
				t.Fatalf("cross: %v, result=%+v", err, result)
			}
			number(t, result.Grand["X"], "14")
			for _, col := range result.Cols {
				if col.Measure == "X" && result.Rows[0].Cells[col.Key()] != nil {
					t.Fatal("excluded declared name became a P6 context reference")
				}
			}
		})
	}
}

func TestScopedRelativeDeclaredNameDoesNotInventCycle(t *testing.T) {
	spec := report.Composition{Groupings: []string{"G"}, Measures: []report.Measure{
		{Field: "Amount", Expr: "X + 1"},
		{Field: "Grand_Amount", Agg: "sum", Scope: []string{report.MeasureScopeGrand}},
		{Field: "X", Expr: "Grand_Amount * 2"},
	}}
	result, err := compose.Compose([]compose.Row{{"G": "a", "Grand_Amount": 7}}, spec, expreval.New(nil, expreval.DefaultProfile()))
	if err != nil || len(result.Warnings) != 0 {
		t.Fatalf("false dependency cycle: %v, result=%+v", err, result)
	}
	number(t, result.Grand["Amount"], "15")
	if result.Groups[0].Subtotals["X"] != nil || result.Groups[0].Subtotals["Amount"] != nil {
		t.Fatal("scope produced a value through an invented context dependency")
	}
}
