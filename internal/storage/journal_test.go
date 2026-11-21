package storage

import (
	"context"
	"path/filepath"
	"testing"

	"github.com/google/uuid"
	"github.com/ivantit66/onebase/internal/metadata"
)

// явный Map имеет приоритет над exact match и fallback.
func TestColExprForDoc_MapWins(t *testing.T) {
	entity := &metadata.Entity{
		Name: "ПоступлениеТоваров",
		Fields: []metadata.Field{
			{Name: "Контрагент", Type: metadata.FieldTypeString},
			{Name: "Поставщик", Type: metadata.FieldTypeString},
		},
	}
	jcol := metadata.JournalColumn{
		Field: "Контрагент",
		Map: map[string]string{
			"ПоступлениеТоваров": "Поставщик",
		},
		Fallback: []string{"Поставщик", "Покупатель"},
	}
	expr, f := colExprForDoc(jcol, entity)
	// Map → Поставщик
	if expr != "поставщик" {
		t.Errorf("Map должен выиграть: ожидался «поставщик», получили %q", expr)
	}
	if f == nil || f.Name != "Поставщик" {
		t.Errorf("вернулось не то поле: %v", f)
	}
}

// Без Map — exact match.
func TestColExprForDoc_ExactMatch(t *testing.T) {
	entity := &metadata.Entity{
		Name: "ПоступлениеТоваров",
		Fields: []metadata.Field{
			{Name: "Контрагент", Type: metadata.FieldTypeString},
		},
	}
	jcol := metadata.JournalColumn{Field: "Контрагент"}
	expr, f := colExprForDoc(jcol, entity)
	if expr != "контрагент" {
		t.Errorf("exact match не сработал: %q", expr)
	}
	if f.Name != "Контрагент" {
		t.Errorf("не то поле: %s", f.Name)
	}
}

// Без Map и без exact — fallback.
func TestColExprForDoc_FallbackUsed(t *testing.T) {
	entity := &metadata.Entity{
		Name: "ПоступлениеТоваров",
		Fields: []metadata.Field{
			{Name: "Поставщик", Type: metadata.FieldTypeString},
		},
	}
	jcol := metadata.JournalColumn{
		Field:    "Контрагент",
		Fallback: []string{"Поставщик", "Покупатель"},
	}
	expr, _ := colExprForDoc(jcol, entity)
	if expr != "поставщик" {
		t.Errorf("fallback не сработал: %q", expr)
	}
}

// Map указывает на несуществующее поле — NULL, не молчаливый fallback.
func TestColExprForDoc_MapMissingFieldGivesNull(t *testing.T) {
	entity := &metadata.Entity{
		Name: "ПоступлениеТоваров",
		Fields: []metadata.Field{
			{Name: "Поставщик", Type: metadata.FieldTypeString},
		},
	}
	jcol := metadata.JournalColumn{
		Field: "Контрагент",
		Map: map[string]string{
			"ПоступлениеТоваров": "НетТакого",
		},
		Fallback: []string{"Поставщик"}, // не должен сработать — Map важнее
	}
	expr, _ := colExprForDoc(jcol, entity)
	if expr != "NULL" {
		t.Errorf("неверная колонка в Map → должна быть NULL, получили %q", expr)
	}
}

// Несколько fallback — COALESCE.
func TestColExprForDoc_MultiFallbackCoalesce(t *testing.T) {
	entity := &metadata.Entity{
		Name: "СоставнойДокумент",
		Fields: []metadata.Field{
			{Name: "Поставщик", Type: metadata.FieldTypeString},
			{Name: "Покупатель", Type: metadata.FieldTypeString},
		},
	}
	jcol := metadata.JournalColumn{
		Field:    "Контрагент",
		Fallback: []string{"Поставщик", "Покупатель"},
	}
	expr, _ := colExprForDoc(jcol, entity)
	if expr != "COALESCE(поставщик, покупатель)" {
		t.Errorf("ожидался COALESCE, получили %q", expr)
	}
}

func TestJournalQueryRowFiltersApplyInsideDocumentUnion(t *testing.T) {
	ctx := context.Background()
	db, err := ConnectSQLite(ctx, filepath.Join(t.TempDir(), "journal-rls.db"))
	if err != nil {
		t.Fatalf("ConnectSQLite: %v", err)
	}
	defer db.Close()

	doc := &metadata.Entity{
		Name: "Заказ",
		Kind: metadata.KindDocument,
		Fields: []metadata.Field{
			{Name: "Owner", Type: metadata.FieldTypeString},
			{Name: "Номер", Type: metadata.FieldTypeString},
		},
	}
	if err := db.Migrate(ctx, []*metadata.Entity{doc}); err != nil {
		t.Fatalf("Migrate: %v", err)
	}
	if err := db.Upsert(ctx, doc.Name, uuid.New(), map[string]any{"Owner": "u", "Номер": "1"}, doc); err != nil {
		t.Fatalf("upsert allowed: %v", err)
	}
	if err := db.Upsert(ctx, doc.Name, uuid.New(), map[string]any{"Owner": "other", "Номер": "2"}, doc); err != nil {
		t.Fatalf("upsert hidden: %v", err)
	}

	j := &metadata.Journal{
		Name:      "Заказы",
		Documents: []string{doc.Name},
		Columns:   []metadata.JournalColumn{{Field: "Owner"}, {Field: "Номер"}},
	}
	rows, total, _, err := db.JournalQuery(ctx, j, map[string]*metadata.Entity{doc.Name: doc}, ListParams{
		JournalRowFilters: map[string]*Predicate{
			doc.Name: {Field: "Owner", Op: "eq", Value: "u"},
		},
	}, 20, 0)
	if err != nil {
		t.Fatalf("JournalQuery: %v", err)
	}
	if total != 1 || len(rows) != 1 {
		t.Fatalf("total=%d rows=%#v, want one allowed row", total, rows)
	}
	if rows[0]["Owner"] != "u" {
		t.Fatalf("row filter leaked hidden row: %#v", rows[0])
	}
}

func TestJournalReferenceProjectionKeepsPageOrder(t *testing.T) {
	ctx := t.Context()
	db, err := ConnectSQLite(ctx, filepath.Join(t.TempDir(), "journal-order.db"))
	if err != nil {
		t.Fatal(err)
	}
	t.Cleanup(func() { db.Close() })

	master := &metadata.Entity{Name: "Master", Kind: metadata.KindCatalog}
	doc := &metadata.Entity{Name: "JournalOrderDoc", Kind: metadata.KindDocument, Fields: []metadata.Field{
		{Name: "Дата", Type: metadata.FieldTypeDate},
		{Name: "Мастер", Type: "reference:Master", RefEntity: master.Name},
	}}
	if err := db.Migrate(ctx, []*metadata.Entity{master, doc}); err != nil {
		t.Fatal(err)
	}
	older, newer := uuid.New(), uuid.New()
	for _, row := range []struct {
		docID uuid.UUID
		refID uuid.UUID
		date  string
	}{
		{older, uuid.MustParse("00000000-0000-0000-0000-000000000002"), "2026-09-01T00:00:00Z"},
		{newer, uuid.MustParse("00000000-0000-0000-0000-000000000001"), "2026-09-02T00:00:00Z"},
	} {
		if err := db.Upsert(ctx, master.Name, row.refID, nil, master); err != nil {
			t.Fatal(err)
		}
		if err := db.Upsert(ctx, doc.Name, row.docID, map[string]any{"Дата": row.date, "Мастер": row.refID.String()}, doc); err != nil {
			t.Fatal(err)
		}
	}
	docs := map[string]*metadata.Entity{doc.Name: doc}
	base := metadata.Journal{Name: "Orders", Documents: []string{doc.Name}, Columns: []metadata.JournalColumn{{Field: "Дата"}}}
	projected := base
	projected.Columns = append([]metadata.JournalColumn{{Field: "ДатаФилиала", ReferencePath: "Мастер.Филиал"}}, base.Columns...)
	for _, j := range []*metadata.Journal{&base, &projected} {
		for offset, want := range []uuid.UUID{older, newer} {
			rows, total, _, err := db.JournalQuery(ctx, j, docs, ListParams{ChronologicalDocuments: true}, 1, offset)
			if err != nil {
				t.Fatal(err)
			}
			if total != 2 || len(rows) != 1 || rows[0]["id"] != want.String() {
				t.Fatalf("offset %d: total=%d rows=%#v, want %s", offset, total, rows, want)
			}
		}
	}
}
