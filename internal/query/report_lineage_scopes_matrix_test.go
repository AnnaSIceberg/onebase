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

func TestReportLineageDerivedAggregateMatrix(t *testing.T) {
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
		inner := `SELECT A.Phone AS K,A.Owner AS Owner,COUNT(*) AS C FROM Document.LineageRequests AS A GROUP BY A.Phone,A.Owner`
		for _, text := range []string{
			`SELECT T.Owner AS Owner,SUM(T.C) AS Total FROM (` + inner + `) AS T GROUP BY T.Owner`,
			`SELECT U.Total AS Total,U.Owner AS Owner FROM (SELECT T.Owner AS Owner,SUM(T.C) AS Total FROM (` + inner + `) AS T GROUP BY T.Owner) AS U`,
		} {
			res, p, err := query.CompileReportLineage(text, opts)
			if err != nil {
				t.Fatal(err)
			}
			rows, _, err := query.Run(ctx, db, &res)
			if err != nil {
				t.Fatal(err)
			}
			if len(rows) != 1 || rows[0]["owner"] != "reader" || fmt.Sprint(rows[0]["total"]) != "3" {
				t.Fatalf("RLS/count grain changed: %v", rows)
			}
			if len(rows[0]) != 2 {
				t.Fatal("internal grouping key escaped SQL")
			}
			seenPhone, innerProjection := false, false
			for _, use := range p.Uses() {
				if use.Scope > 0 && use.Clause == "inner-projection" {
					innerProjection = true
				}
				walkReportExpression(use.Expression, func(e query.ReportExpression) {
					for _, origin := range e.Origins {
						if origin.Field == "Phone" && origin.Source.Name == "LineageRequests" {
							seenPhone = true
						}
					}
				})
			}
			if !seenPhone || !innerProjection {
				t.Fatal("inner key/group/control provenance lost")
			}
			if !p.Matches(res, opts) {
				t.Fatal("fresh derived receipt rejected")
			}
			plan := access.ReportOutputQueryMaskPlan(nil, res, func(kind, name string) *metadata.Entity {
				for _, entity := range entities {
					if kind == string(entity.Kind) && name == entity.Name {
						return entity
					}
				}
				return nil
			})
			if plan.Denied == "" {
				t.Fatal("derived trace granted protected aggregation")
			}
		}
	})
}
func TestReportLineageDerivedOccurrenceAndGroupShadow(t *testing.T) {
	opts := query.CompileOpts{Entities: reportLineageEntities()}
	text := `SELECT A.Number AS N,T.C AS C FROM Document.LineageRequests AS A INNER JOIN (SELECT B.Phone AS K,COUNT(*) AS C FROM Document.LineageRequests AS B GROUP BY B.Phone) AS T ON A.Phone=T.K`
	_, p, err := query.CompileReportLineage(text, opts)
	if err != nil {
		t.Fatal(err)
	}
	occurrences := map[int]bool{}
	for _, use := range p.Uses() {
		if use.Scope == 0 && use.Clause == "join" {
			walkReportExpression(use.Expression, func(e query.ReportExpression) {
				for _, origin := range e.Origins {
					if origin.Field == "Phone" {
						occurrences[origin.Occurrence] = true
					}
				}
			})
		}
	}
	if !occurrences[0] || !occurrences[1] || len(occurrences) != 2 {
		t.Fatal("outer/derived physical sources merged")
	}
	_, p, err = query.CompileReportLineage(`SELECT "PUBLIC" AS Phone,COUNT(*) AS N FROM (SELECT A.Phone AS Phone FROM Document.LineageRequests AS A) AS T GROUP BY Phone`, opts)
	if err != nil {
		t.Fatal(err)
	}
	seen := false
	for _, use := range p.Uses() {
		if use.Scope == 0 && use.Clause == "group" {
			walkReportExpression(use.Expression, func(e query.ReportExpression) {
				for _, origin := range e.Origins {
					seen = seen || origin.Field == "Phone"
				}
			})
		}
	}
	if !seen {
		t.Fatal("GROUP alias hid a derived protected field")
	}
}
func TestReportLineageDerivedUnknownAndExpansionBudget(t *testing.T) {
	opts := query.CompileOpts{Entities: reportLineageEntities()}
	for _, text := range []string{
		`SELECT T.N AS N FROM (SELECT A.Number AS N,A.Phone AS N FROM Document.LineageRequests AS A) AS T`,
		`SELECT T.K AS K FROM (SELECT Magical(A.Phone) AS K FROM Document.LineageRequests AS A) AS T`,
		`SELECT T.ID AS ID FROM (SELECT A.Ссылка AS ID FROM Document.LineageRequests AS A) AS T`,
		`SELECT K AS K FROM (SELECT A.Number AS K FROM Document.LineageRequests AS A) AS T INNER JOIN (SELECT B.Phone AS K FROM Document.LineageRequests AS B) AS U ON 1=1`,
	} {
		res, p, err := query.CompileReportLineage(text, opts)
		if err == nil || p != nil || res.SQL != "" {
			t.Fatal("unknown/ambiguous derived lineage accepted")
		}
	}
	text := `SELECT 1 AS X FROM Document.LineageRequests AS A`
	for i := 0; i < 24; i++ {
		text = `SELECT T.X+T.X AS X FROM (` + text + `) AS T`
	}
	if res, p, err := query.CompileReportLineage(text, opts); err == nil || p != nil || res.SQL != "" {
		t.Fatal("exponential detached inspection was not bounded")
	} else if !strings.Contains(err.Error(), "budget") {
		t.Fatalf("wrong stop: %v", err)
	}
}
