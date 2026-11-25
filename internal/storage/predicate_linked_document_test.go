package storage_test

import (
	"context"
	"testing"

	"github.com/google/uuid"
	"github.com/ivantit66/onebase/internal/dbtest"
	"github.com/ivantit66/onebase/internal/metadata"
	"github.com/ivantit66/onebase/internal/storage"
)

func TestLinkedDocumentReadPredicate(t *testing.T) {
	dbtest.ForEachDialect(t, func(t *testing.T, db *storage.DB) {
		ctx := context.Background()
		call := &metadata.Entity{Name: "CallLink", Kind: metadata.KindDocument}
		field := metadata.Field{Name: "Call", Type: "reference:CallLink", RefEntity: call.Name}
		tp := metadata.TablePart{Name: "Links", Fields: []metadata.Field{field}}
		doc := &metadata.Entity{
			Name: "CaseLink", Kind: metadata.KindDocument,
			Fields: []metadata.Field{field, {Name: "Scope", Type: metadata.FieldTypeString}}, TableParts: []metadata.TablePart{tp},
		}
		recording := &metadata.Entity{Name: "RecordingLink", Kind: metadata.KindCatalog, Fields: []metadata.Field{field}}
		if err := db.Migrate(ctx, []*metadata.Entity{call, doc, recording}); err != nil {
			t.Fatal(err)
		}
		cases := []struct {
			link, scope string
			visible     bool
		}{
			{"header", "allowed", true}, {"part", "allowed", true}, {"header", "denied", false}, {"none", "allowed", false},
		}
		linked := &storage.LinkedDocument{
			Document: doc, Links: []storage.LinkedDocumentPath{{Field: field}, {Field: field, TablePart: &tp}},
			Scope: &storage.Predicate{Field: "Scope", Op: "eq", Value: "allowed"},
		}
		linked.Match = func(id uuid.UUID) (bool, error) { return db.HasLinkedDocument(ctx, linked, id) }
		pred := &storage.Predicate{Field: "Call", LinkedDocument: linked}
		for _, tc := range cases {
			callID, docID, recID := uuid.New(), uuid.New(), uuid.New()
			if err := db.Upsert(ctx, call.Name, callID, nil, call); err != nil {
				t.Fatal(err)
			}
			fields := map[string]any{"Scope": tc.scope}
			if tc.link == "header" {
				fields["Call"] = callID
			}
			if err := db.Upsert(ctx, doc.Name, docID, fields, doc); err != nil {
				t.Fatal(err)
			}
			if tc.link == "part" {
				if err := db.UpsertTablePartRows(ctx, doc.Name, tp.Name, docID, []map[string]any{{"Call": callID}}, tp); err != nil {
					t.Fatal(err)
				}
			}
			if err := db.Upsert(ctx, recording.Name, recID, map[string]any{"Call": callID}, recording); err != nil {
				t.Fatal(err)
			}
			if got := storage.MatchPredicate(map[string]any{"Call": callID}, pred); got != tc.visible {
				t.Errorf("direct %s/%s: %v", tc.link, tc.scope, got)
			}
		}
		rows, err := db.List(ctx, recording.Name, recording, storage.ListParams{RowFilter: pred, RowFilterEvaluated: true})
		if err != nil || len(rows) != 2 {
			t.Fatalf("list allows two visible links: got %d, %v", len(rows), err)
		}
	})
}
