package storage_test

import (
	"context"
	"fmt"
	"reflect"
	"testing"

	"github.com/google/uuid"
	"github.com/ivantit66/onebase/internal/dbtest"
	"github.com/ivantit66/onebase/internal/metadata"
	"github.com/ivantit66/onebase/internal/storage"
)

func TestProjectedListKeepsFiltersAndMetadataOrder(t *testing.T) {
	dbtest.ForEachDialect(t, func(t *testing.T, db *storage.DB) {
		ctx := context.Background()
		ent := &metadata.Entity{
			Name: "WideListDocument", Kind: metadata.KindDocument, OrderBy: []string{"Number", "Label"},
			Fields: []metadata.Field{
				{Name: "Number", Type: metadata.FieldTypeString},
				{Name: "Label", Type: metadata.FieldTypeString},
				{Name: "Imported", Type: metadata.FieldTypeBool},
			},
		}
		if err := db.Migrate(ctx, []*metadata.Entity{ent}); err != nil {
			t.Fatal(err)
		}
		fixtures := []struct {
			number   any
			label    string
			imported any
		}{
			{"A", "b", nil}, {"A", "a", false}, {"A", "a", nil},
			{"B", "a", false}, {nil, "a", false}, {"", "a", nil},
			{"A", "a", true}, {"", "b", true},
		}
		for i, item := range fixtures {
			id := uuid.MustParse(fmt.Sprintf("%d0000000-0000-0000-0000-000000000000", i+1))
			if err := db.Upsert(ctx, ent.Name, id, map[string]any{
				"Number": item.number, "Label": item.label, "Imported": item.imported,
			}, ent); err != nil {
				t.Fatal(err)
			}
		}
		params := storage.ListParams{RowFilterEvaluated: true, RowFilter: &storage.Predicate{Any: []storage.Predicate{
			{Field: "Imported", Op: "empty"},
			{Field: "Imported", Op: "eq", Value: false},
		}}}
		unlimited, err := db.List(ctx, ent.Name, ent, params)
		if err != nil || len(unlimited) != 6 {
			t.Fatalf("filtered rows = %d, error = %v", len(unlimited), err)
		}
		var pages []map[string]any
		params.OutputFields = ent.Fields[:1]
		params.Limit = 2
		for params.Offset = 0; params.Offset <= len(unlimited); params.Offset += params.Limit {
			page, err := db.List(ctx, ent.Name, ent, params)
			if err != nil {
				t.Fatal(err)
			}
			pages = append(pages, page...)
		}
		for _, row := range unlimited {
			delete(row, "Label")
			delete(row, "Imported")
		}
		if !reflect.DeepEqual(pages, unlimited) {
			t.Fatal("projection changed visible rows, number, system fields, or metadata order")
		}
	})
}
