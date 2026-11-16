package query_test

import (
	"context"
	"fmt"
	"strings"
	"testing"

	"github.com/google/uuid"
	"github.com/ivantit66/onebase/internal/access"
	"github.com/ivantit66/onebase/internal/dbtest"
	"github.com/ivantit66/onebase/internal/metadata"
	"github.com/ivantit66/onebase/internal/query"
	"github.com/ivantit66/onebase/internal/storage"
)

func TestReportLineageUnionAggregateMatrix(t *testing.T) {
	dbtest.ForEachDialect(t, func(t *testing.T, db *storage.DB) {
		ctx := context.Background()
		entities := reportLineageEntities()
		if err := db.Migrate(ctx, entities); err != nil {
			t.Fatal(err)
		}
		for i, key := range []string{"synthetic-A", "synthetic-A", "synthetic-B", "synthetic-A"} {
			owner := "reader"
			if i == 3 {
				owner = "other"
			}
			if err := db.Upsert(ctx, entities[0].Name, uuid.New(), map[string]any{"Number": fmt.Sprint(i), "Owner": owner, "Phone": key}, entities[0]); err != nil {
				t.Fatal(err)
			}
		}
		opts := query.CompileOpts{Entities: entities, Dialect: db.Dialect(), RowFilters: map[query.SourceRef]*storage.Predicate{{Kind: "document", Name: "LineageRequests"}: {Field: "Owner", Op: "eq", Value: "reader"}}}
		left := `SELECT A.Phone AS K,A.Owner AS O,COUNT(*) AS C FROM Document.LineageRequests AS A GROUP BY A.Phone,A.Owner`
		right := `SELECT B.Phone AS OtherK,B.Owner AS OtherO,COUNT(*) AS OtherC FROM Document.LineageRequests AS B GROUP BY B.Phone,B.Owner`
		for _, tc := range []struct{ op, want string }{{"UNION ALL", "6"}, {"UNION", "3"}} {
			text := `SELECT U.O AS Owner,SUM(U.C) AS Total FROM (` + left + ` ` + tc.op + ` ` + right + `) AS U GROUP BY U.O ORDER BY Owner`
			res, p, err := query.CompileReportLineage(text, opts)
			if err != nil {
				t.Fatal(err)
			}
			rows, _, err := query.Run(ctx, db, &res)
			if err != nil {
				t.Fatal(err)
			}
			if len(rows) != 1 || len(rows[0]) != 2 || rows[0]["owner"] != "reader" || fmt.Sprint(rows[0]["total"]) != tc.want {
				t.Fatalf("union grain/RLS changed: %s %v", tc.op, rows)
			}
			phoneIDs := map[int]bool{}
			distinct := false
			for _, use := range p.Uses() {
				distinct = distinct || use.Clause == "distinct"
				walkReportExpression(use.Expression, func(e query.ReportExpression) {
					for _, o := range e.Origins {
						if o.Field == "Phone" {
							phoneIDs[o.Occurrence] = true
						}
					}
				})
			}
			if len(phoneIDs) != 2 || distinct != (tc.op == "UNION") {
				t.Fatal("branch origins or duplicate elimination lost")
			}
			if !p.Matches(res, opts) {
				t.Fatal("fresh UNION receipt rejected")
			}
			plan := access.ReportOutputQueryMaskPlan(nil, res, func(kind, name string) *metadata.Entity {
				for _, e := range entities {
					if kind == string(e.Kind) && name == e.Name {
						return e
					}
				}
				return nil
			})
			if plan.Denied == "" {
				t.Fatal("UNION trace silently granted protected aggregation")
			}
		}
	})
}
func TestReportLineageUnionGlobalOrderUsesEveryBranch(t *testing.T) {
	opts := query.CompileOpts{Entities: reportLineageEntities()}
	_, p, err := query.CompileReportLineage(`SELECT A.Number AS K FROM Document.LineageRequests AS A UNION ALL SELECT B.Phone AS DifferentName FROM Document.LineageRequests AS B ORDER BY K`, opts)
	if err != nil {
		t.Fatal(err)
	}
	phone, number := false, false
	for _, use := range p.Uses() {
		if use.Scope == 0 && use.Clause == "order" {
			walkReportExpression(use.Expression, func(e query.ReportExpression) {
				for _, o := range e.Origins {
					phone = phone || o.Field == "Phone"
					number = number || o.Field == "Number"
				}
			})
		}
	}
	if !phone || !number {
		t.Fatal("global ORDER bound only one UNION branch")
	}
}
func TestReportLineageUnionUnknownStops(t *testing.T) {
	opts := query.CompileOpts{Entities: reportLineageEntities()}
	for _, text := range []string{
		`SELECT A.Number AS K FROM Document.LineageRequests AS A UNION ALL SELECT B.Phone AS K,B.Number AS N FROM Document.LineageRequests AS B`,
		`SELECT A.Number AS K FROM Document.LineageRequests AS A UNION ALL SELECT COUNT(*) AS N FROM Document.LineageRequests AS B`,
		`SELECT A.Number AS K FROM Document.LineageRequests AS A UNION ALL SELECT B.Phone AS K FROM Document.LineageRequests AS B ORDER BY B.Phone`,
		`SELECT A.Number AS K FROM Document.LineageRequests AS A UNION ALL SELECT B.Phone AS DifferentName FROM Document.LineageRequests AS B ORDER BY DifferentName`,
		`SELECT A.Number AS K FROM Document.LineageRequests AS A UNION ALL SELECT B.Phone AS K FROM Document.LineageRequests AS B ORDER BY +1`,
		`SELECT A.Number AS K FROM Document.LineageRequests AS A UNION ALL SELECT B.Phone AS K FROM Document.LineageRequests AS B UNION ALL SELECT Magical(C.Phone) AS K FROM Document.LineageRequests AS C`,
	} {
		res, p, err := query.CompileReportLineage(text, opts)
		if err == nil || p != nil || res.SQL != "" {
			t.Fatalf("unsupported UNION returned partial receipt: %s", text)
		}
	}
	left := `SELECT A.Number AS K FROM Document.LineageRequests AS A`
	many := strings.Repeat(left+` UNION ALL `, 18) + left
	if _, p, err := query.CompileReportLineage(many, opts); err == nil || p != nil {
		t.Fatal("union budget not enforced")
	}
}

func TestReportLineageSelectDistinctIsMembershipDependency(t *testing.T) {
	opts := query.CompileOpts{Entities: reportLineageEntities()}
	_, p, err := query.CompileReportLineage(`SELECT DISTINCT A.Phone AS HiddenPhone FROM Document.LineageRequests AS A`, opts)
	if err != nil {
		t.Fatal(err)
	}
	seen := false
	for _, use := range p.Uses() {
		if use.Clause == "select-distinct" {
			walkReportExpression(use.Expression, func(e query.ReportExpression) {
				for _, origin := range e.Origins {
					seen = seen || origin.Field == "Phone"
				}
			})
		}
	}
	if !seen {
		t.Fatal("DISTINCT hidden output had no membership dependency")
	}
}
