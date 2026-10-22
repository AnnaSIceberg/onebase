package storage

import (
	"context"
	"path/filepath"
	"testing"

	"github.com/google/uuid"

	"github.com/ivantit66/onebase/internal/metadata"
)

// Счётчик записей сдвигают изменяющие операторы — и через DB, и через
// транзакцию; чтение и служебные команды транзакции его не трогают.
func TestWriteSeqCountsWritesNotReads(t *testing.T) {
	ctx := context.Background()
	db, err := ConnectSQLite(ctx, filepath.Join(t.TempDir(), "seq.db"))
	if err != nil {
		t.Fatal(err)
	}
	defer db.Close()
	ent := &metadata.Entity{Name: "Товары", Kind: metadata.KindCatalog, Fields: []metadata.Field{{Name: "Наименование", Type: metadata.FieldTypeString}}}
	if err := db.Migrate(ctx, []*metadata.Entity{ent}); err != nil {
		t.Fatal(err)
	}
	id := uuid.New()
	if err := db.Upsert(ctx, ent.Name, id, map[string]any{"Наименование": "до"}, ent); err != nil {
		t.Fatal(err)
	}

	before := WriteSeq()
	if _, err := db.GetByID(ctx, ent.Name, id, ent); err != nil {
		t.Fatal(err)
	}
	if _, err := db.QueryAll(ctx, `SELECT 1`); err != nil {
		t.Fatal(err)
	}
	if got := WriteSeq(); got != before {
		t.Fatalf("чтение сдвинуло счётчик записей: %d → %d", before, got)
	}

	if err := db.Upsert(ctx, ent.Name, id, map[string]any{"Наименование": "после"}, ent); err != nil {
		t.Fatal(err)
	}
	afterWrite := WriteSeq()
	if afterWrite == before {
		t.Fatal("запись не сдвинула счётчик")
	}

	tx, txCtx, err := db.BeginTx(ctx)
	if err != nil {
		t.Fatal(err)
	}
	if _, err := tx.Exec(txCtx, `UPDATE "товары" SET "наименование" = 'в транзакции'`); err != nil {
		t.Fatal(err)
	}
	if err := tx.Commit(txCtx); err != nil {
		t.Fatal(err)
	}
	if WriteSeq() == afterWrite {
		t.Fatal("запись в транзакции не сдвинула счётчик")
	}
}
