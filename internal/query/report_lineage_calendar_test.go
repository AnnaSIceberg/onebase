package query_test

import (
	"context"
	"testing"
	"time"

	"github.com/google/uuid"
	"github.com/ivantit66/onebase/internal/access"
	"github.com/ivantit66/onebase/internal/dbtest"
	"github.com/ivantit66/onebase/internal/metadata"
	"github.com/ivantit66/onebase/internal/query"
	"github.com/ivantit66/onebase/internal/storage"
)

func calendarLineageEntities() []*metadata.Entity {
	entities := reportLineageEntities()
	entities[0].Fields = append(entities[0].Fields, metadata.Field{Name: "Moment", Type: metadata.FieldTypeDate, PII: true})
	return entities
}

func TestReportLineageCalendarOrigins(t *testing.T) {
	opts := query.CompileOpts{Entities: calendarLineageEntities()}
	for _, name := range []string{"Год", "YEAR", "Месяц", "MONTH", "День", "DAY"} {
		t.Run(name, func(t *testing.T) {
			_, proof, err := query.CompileReportLineage("SELECT "+name+"(A.Moment) AS N FROM Document.LineageRequests AS A GROUP BY "+name+"(A.Moment)", opts)
			if err != nil {
				t.Fatal(err)
			}
			out := proof.Outputs()[0].Expression
			if out.Type != metadata.FieldTypeNumber || len(out.Args) != 1 || len(out.Args[0].Origins) != 1 {
				t.Fatal("calendar input lineage missing")
			}
			origin := out.Args[0].Origins[0]
			if origin.Source.Kind != "document" || origin.Source.Name != "LineageRequests" || origin.Field != "Moment" || origin.Type != metadata.FieldTypeDate {
				t.Fatal("calendar input was relabelled")
			}
			uses := proof.Uses()
			if len(uses) != 1 || uses[0].Clause != "group" || uses[0].Expression.Op != out.Op || len(uses[0].Expression.Args[0].Origins) != 1 {
				t.Fatal("calendar grouping dependency missing")
			}
		})
	}
}

func TestReportLineageCalendarMatchingMatrix(t *testing.T) {
	dbtest.ForEachDialect(t, func(t *testing.T, db *storage.DB) {
		ctx := context.Background()
		entities := calendarLineageEntities()
		if err := db.Migrate(ctx, entities); err != nil {
			t.Fatal(err)
		}
		for _, item := range []struct {
			month, day int
			owner      string
		}{{1, 31, "reader"}, {2, 1, "reader"}, {2, 1, "other"}} {
			if err := db.Upsert(ctx, entities[0].Name, uuid.New(), map[string]any{"Moment": time.Date(2026, time.Month(item.month), item.day, 12, 0, 0, 0, time.UTC), "Owner": item.owner}, entities[0]); err != nil {
				t.Fatal(err)
			}
		}
		opts := query.CompileOpts{Entities: entities, Dialect: db.Dialect(), RowFilters: map[query.SourceRef]*storage.Predicate{{Kind: "document", Name: "LineageRequests"}: {Field: "Owner", Op: "eq", Value: "reader"}}}
		res, proof, err := query.CompileReportLineage(`SELECT Год(A.Moment)*100+Месяц(A.Moment) AS Bucket,DAY(A.Moment) AS D,COUNT(*) AS N
		 FROM Document.LineageRequests AS A GROUP BY YEAR(A.Moment)*100+MONTH(A.Moment),День(A.Moment) ORDER BY Bucket,D`, opts)
		if err != nil {
			t.Fatal(err)
		}
		if !proof.Matches(res, opts) {
			t.Fatal("calendar receipt mismatch")
		}
		rows, _, err := query.Run(ctx, db, &res)
		if err != nil {
			t.Fatal(err)
		}
		if len(rows) != 2 || rows[0]["bucket"] != int64(202601) || rows[0]["d"] != int64(31) || rows[1]["bucket"] != int64(202602) || rows[1]["d"] != int64(1) {
			t.Fatalf("calendar buckets changed: %v", rows)
		}
		for _, row := range rows {
			if row["n"] != int64(1) {
				t.Fatal("RLS excluded row changed calendar count")
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
			t.Fatal("calendar operation erased protected input policy")
		}
	})
}

func TestReportLineageCalendarUnknownStops(t *testing.T) {
	opts := query.CompileOpts{Entities: calendarLineageEntities()}
	for name, expr := range map[string]string{"string": "YEAR(A.Phone)", "number": "MONTH(1)", "literal_date": "YEAR('2026-01-01')", "unknown_param": "DAY(&Moment)", "arity": "YEAR(A.Moment,A.Moment)", "distinct": "YEAR(DISTINCT A.Moment)", "unknown_function": "Quarter(A.Moment)", "unsupported_datediff": "РАЗНОСТЬДАТ(A.Moment,A.Moment,ДЕНЬ)"} {
		t.Run(name, func(t *testing.T) {
			res, p, err := query.CompileReportLineage("SELECT "+expr+" AS N FROM Document.LineageRequests AS A", opts)
			if err == nil || p != nil || res.SQL != "" {
				t.Fatal("unproved calendar input returned a receipt")
			}
		})
	}
}
