package query_test

import (
	"context"
	"testing"

	"github.com/google/uuid"
	"github.com/ivantit66/onebase/internal/access"
	"github.com/ivantit66/onebase/internal/dbtest"
	"github.com/ivantit66/onebase/internal/metadata"
	"github.com/ivantit66/onebase/internal/query"
	"github.com/ivantit66/onebase/internal/storage"
)

func TestReportLineageReferencePredicateMatching(t *testing.T) {
	dbtest.ForEachDialect(t, func(t *testing.T, db *storage.DB) {
		ctx := context.Background()
		entities := reportLineageEntities()
		entities[0].Fields[3].PII = true // UUID must retain this protected root.
		if err := db.Migrate(ctx, entities); err != nil {
			t.Fatal(err)
		}
		branch := uuid.New()
		if err := db.Upsert(ctx, entities[1].Name, branch, map[string]any{"Title": "Safe branch"}, entities[1]); err != nil {
			t.Fatal(err)
		}
		for _, row := range []struct{ number, owner string }{{"R1", "reader"}, {"R2", "reader"}, {"R3", "other"}} {
			if err := db.Upsert(ctx, entities[0].Name, uuid.New(), map[string]any{"Number": row.number, "Owner": row.owner, "Phone": "synthetic-match-key", "Branch": branch.String()}, entities[0]); err != nil {
				t.Fatal(err)
			}
		}
		opts := query.CompileOpts{Entities: entities, Dialect: db.Dialect(), Params: map[string]any{"Branch": branch.String()}, RowFilters: map[query.SourceRef]*storage.Predicate{{Kind: "document", Name: "LineageRequests"}: {Field: "Owner", Op: "eq", Value: "reader"}}}
		text := `SELECT A.Number AS N,COUNT(*) AS C FROM Document.LineageRequests AS A
		 INNER JOIN Document.LineageRequests AS B ON A.Ссылка<>B.Ссылка AND A.Phone=B.Phone AND A.Branch=B.Branch
		 LEFT JOIN Catalog.LineageBranches AS T ON T.Ссылка=A.Branch
		 WHERE (A.Branch=&Branch AND A.Branch IS NOT NULL) AND T.Ссылка IS NOT NULL
		 GROUP BY A.Number ORDER BY N`
		res, proof, err := query.CompileReportLineage(text, opts)
		if err != nil {
			t.Fatal(err)
		}
		if !proof.Matches(res, opts) {
			t.Fatal("fresh reference predicate receipt mismatch")
		}
		seen := map[int]bool{}
		for _, use := range proof.Uses() {
			walkReportExpression(use.Expression, func(e query.ReportExpression) {
				for _, origin := range e.Origins {
					if origin.Field == "Branch" {
						if origin.Source.Kind != "document" || origin.Source.Name != "LineageRequests" || origin.Type != "reference:LineageBranches" || len(origin.Path) != 1 || origin.Path[0] != "Branch" {
							t.Fatal("FK owner/type/root lost")
						}
						seen[origin.Occurrence] = true
					}
					if origin.Field == "Title" {
						t.Fatal("predicate ID became a label")
					}
				}
			})
		}
		if !seen[0] || !seen[1] {
			t.Fatal("self JOIN FK occurrences collapsed")
		}
		rows, _, err := query.Run(ctx, db, &res)
		if err != nil {
			t.Fatal(err)
		}
		if len(rows) != 2 {
			t.Fatalf("RLS matching rows: %d", len(rows))
		}
		for _, row := range rows {
			if row["c"] != int64(1) {
				t.Fatal("excluded JOIN side changed count")
			}
		}
		plan := access.ReportOutputQueryMaskPlan(nil, res, func(kind, name string) *metadata.Entity {
			for _, e := range entities {
				if string(e.Kind) == kind && e.Name == name {
					return e
				}
			}
			return nil
		})
		if plan.Denied == "" {
			t.Fatal("reference lineage authorized protected use")
		}
	})
}

func TestReportLineageReferencePredicateSections(t *testing.T) {
	opts := query.CompileOpts{Entities: reportLineageEntities(), Params: map[string]any{"Branch": uuid.New().String()}}
	for name, text := range map[string]string{
		"where":       "SELECT A.Number AS N FROM Document.LineageRequests AS A WHERE A.Branch=&Branch",
		"nested_null": "SELECT A.Number AS N FROM Document.LineageRequests AS A WHERE NOT(A.Branch IS NULL)",
		"having":      "SELECT COUNT(*) AS N FROM Document.LineageRequests AS A GROUP BY A.Number HAVING COUNT(A.Branch)>0",
	} {
		t.Run(name, func(t *testing.T) {
			if _, p, err := query.CompileReportLineage(text, opts); err != nil || p == nil {
				t.Fatalf("predicate ID STOP: %v", err)
			}
		})
	}
	for name, text := range map[string]string{
		"select":              "SELECT A.Branch AS N FROM Document.LineageRequests AS A",
		"select_expression":   "SELECT COUNT(A.Branch) AS N FROM Document.LineageRequests AS A",
		"select_case":         "SELECT CASE WHEN A.Branch IS NULL THEN 0 ELSE 1 END AS N FROM Document.LineageRequests AS A",
		"group":               "SELECT COUNT(*) AS N FROM Document.LineageRequests AS A GROUP BY A.Branch",
		"order":               "SELECT A.Number AS N FROM Document.LineageRequests AS A ORDER BY A.Branch",
		"derived_passthrough": "SELECT COUNT(*) AS N FROM (SELECT A.Branch AS K FROM Document.LineageRequests AS A) AS T WHERE T.K=&Branch",
	} {
		t.Run(name, func(t *testing.T) {
			res, p, err := query.CompileReportLineage(text, opts)
			if err == nil || p != nil || res.SQL != "" {
				t.Fatal("presentation returned partial reference receipt")
			}
		})
	}
}
