package query_test

import (
	"context"
	"strconv"
	"testing"
	"time"

	"github.com/ivantit66/onebase/internal/access"
	"github.com/ivantit66/onebase/internal/dbtest"
	"github.com/ivantit66/onebase/internal/metadata"
	"github.com/ivantit66/onebase/internal/query"
	"github.com/ivantit66/onebase/internal/storage"
)

func lineageInfoRegister() *metadata.InfoRegister {
	return &metadata.InfoRegister{Name: "LineageSlice", Periodic: true, Dimensions: []metadata.Field{{Name: "Phone", Type: "string", PII: true}}, Resources: []metadata.Field{{Name: "Value", Type: "number"}, {Name: "Owner", Type: "string"}}}
}

func TestReportLineageInfoSliceHiddenControls(t *testing.T) {
	ir := lineageInfoRegister()
	for _, method := range []string{"СрезПоследних", "LastSlice", "СрезПервых", "FirstSlice"} {
		t.Run(method, func(t *testing.T) {
			opts := query.CompileOpts{InfoRegs: []*metadata.InfoRegister{ir}, Params: map[string]any{"End": time.Date(2026, 1, 2, 12, 0, 0, 0, time.UTC)}}
			res, proof, err := query.CompileReportLineage("SELECT SUM(S.Value) AS Total FROM InformationRegister.LineageSlice."+method+"(&End,) AS S", opts)
			if err != nil {
				t.Fatal(err)
			}
			if !proof.Matches(res, opts) {
				t.Fatal("slice receipt mismatch")
			}
			partition, order, bound := false, false, false
			for _, use := range proof.Uses() {
				switch use.Clause {
				case "slice-partition":
					if len(use.Expression.Origins) != 1 || use.Expression.Origins[0].Field != "Phone" || use.Expression.Origins[0].Source.Kind != "inforeg" || use.Expression.Origins[0].Source.Name != ir.Name {
						t.Fatal("hidden partition origin lost")
					}
					partition = true
				case "slice-order-desc", "slice-order-asc":
					order = len(use.Expression.Origins) == 1 && use.Expression.Origins[0].Field == "period"
				case "slice-bound":
					bound = len(use.Expression.Args) == 2 && use.Expression.Args[1].Parameter == "End"
				}
			}
			if !partition || !order || !bound {
				t.Fatal("slice membership dependencies incomplete")
			}
			plan := access.ReportOutputQueryMaskPlan(nil, res, func(kind, name string) *metadata.Entity {
				if kind == "inforeg" && name == ir.Name {
					return storage.InfoRegisterPredicateEntity(ir)
				}
				return nil
			})
			if plan.Denied == "" {
				t.Error("mandatory output gate ignored hidden protected VT partition")
			}
		})
	}
}

func TestReportLineageInfoSliceRLSBeforeLatest(t *testing.T) {
	dbtest.ForEachDialect(t, func(t *testing.T, db *storage.DB) {
		ctx := context.Background()
		ir := lineageInfoRegister()
		if err := db.MigrateInfoRegisters(ctx, []*metadata.InfoRegister{ir}); err != nil {
			t.Fatal(err)
		}
		for _, item := range []struct {
			phone, owner string
			day          int
			value        int
		}{{"synthetic-A", "reader", 1, 10}, {"synthetic-A", "other", 2, 999}, {"synthetic-A", "reader", 3, 20}, {"synthetic-B", "reader", 1, 5}} {
			moment := time.Date(2026, 1, item.day, 0, 0, 0, 0, time.UTC)
			if err := db.InfoRegSet(ctx, ir, map[string]any{"Phone": item.phone}, map[string]any{"Owner": item.owner, "Value": item.value}, &moment); err != nil {
				t.Fatal(err)
			}
		}
		opts := query.CompileOpts{InfoRegs: []*metadata.InfoRegister{ir}, Dialect: db.Dialect(), Params: map[string]any{"End": time.Date(2026, 1, 2, 12, 0, 0, 0, time.UTC)}, RowFilters: map[query.SourceRef]*storage.Predicate{{Kind: "inforeg", Name: ir.Name}: {Field: "Owner", Op: "eq", Value: "reader"}}}
		res, _, err := query.CompileReportLineage("SELECT SUM(S.Value) AS Total FROM InformationRegister.LineageSlice.LastSlice(&End,) AS S", opts)
		if err != nil {
			t.Fatal(err)
		}
		rows, _, err := query.Run(ctx, db, &res)
		if err != nil {
			t.Fatal(err)
		}
		if len(rows) != 1 || rows[0]["total"] != int64(15) {
			t.Fatalf("excluded newer row influenced slice: %v", rows)
		}
	})
}

func TestReportLineageInfoSliceUnknownStops(t *testing.T) {
	for name, source := range map[string]string{"unknown_method": "InformationRegister.LineageSlice.Unknown() AS S", "nonempty_filter": "InformationRegister.LineageSlice.LastSlice(,Phone=&Key) AS S", "expression_boundary": "InformationRegister.LineageSlice.LastSlice(&End+1,) AS S", "extra_argument": "InformationRegister.LineageSlice.LastSlice(&End,,) AS S", "missing_alias": "InformationRegister.LineageSlice.LastSlice()", "accumulation": "AccumulationRegister.LineageSlice.Balances() AS S", "nonperiodic": "InformationRegister.LineageSlice.LastSlice() AS S", "empty_dimension": "InformationRegister.LineageSlice.LastSlice() AS S", "empty_boundary": "InformationRegister.LineageSlice.LastSlice(&,) AS S", "dimension_budget": "InformationRegister.LineageSlice.LastSlice() AS S", "no_dimensions": "InformationRegister.LineageSlice.LastSlice() AS S", "ambiguous": "InformationRegister.LineageSlice.LastSlice() AS S"} {
		t.Run(name, func(t *testing.T) {
			ir := lineageInfoRegister()
			opts := query.CompileOpts{InfoRegs: []*metadata.InfoRegister{ir}}
			if name == "nonperiodic" {
				ir.Periodic = false
			}
			if name == "no_dimensions" {
				ir.Dimensions = nil
			}
			if name == "empty_dimension" {
				ir.Dimensions = []metadata.Field{{Name: "", Type: "string"}}
			}
			if name == "dimension_budget" {
				ir.Dimensions = make([]metadata.Field, 65)
				for i := range ir.Dimensions {
					ir.Dimensions[i] = metadata.Field{Name: "D" + strconv.Itoa(i), Type: "string"}
				}
			}
			if name == "ambiguous" {
				opts.InfoRegs = append(opts.InfoRegs, ir)
			}
			res, p, err := query.CompileReportLineage("SELECT COUNT(*) AS N FROM "+source, opts)
			if err == nil || p != nil || res.SQL != "" {
				t.Fatal("unproved slice returned partial receipt")
			}
		})
	}
}
