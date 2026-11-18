package query_test

import (
	"context"
	"path/filepath"
	"testing"
	"time"
	_ "time/tzdata"

	"github.com/google/uuid"
	"github.com/ivantit66/onebase/internal/metadata"
	"github.com/ivantit66/onebase/internal/query"
	"github.com/ivantit66/onebase/internal/storage"
	"github.com/stretchr/testify/require"
)

// Дата старого формата SQLite без смещения («2026-09-01 12:05:00» — местные
// стенные часы) давала NULL в НачалоДня/НачалоЧаса: группировка отчёта по дню
// и часу (А_ВторойКонтур) выводилась с пустой подписью.
func TestQueryStartOfPeriodLegacySQLiteDate(t *testing.T) {
	loc, err := time.LoadLocation("Europe/Moscow")
	require.NoError(t, err)
	previous := time.Local
	time.Local = loc
	t.Cleanup(func() { time.Local = previous })
	ctx := context.Background()
	db, err := storage.ConnectSQLite(ctx, filepath.Join(t.TempDir(), "legacy.db"))
	require.NoError(t, err)
	t.Cleanup(db.Close)
	doc := &metadata.Entity{Name: "LegacyDateProbe", Kind: metadata.KindDocument, Fields: []metadata.Field{
		{Name: "Номер", Type: metadata.FieldTypeString}, {Name: "Дата", Type: metadata.FieldTypeDate},
	}}
	require.NoError(t, db.Migrate(ctx, []*metadata.Entity{doc}))
	require.NoError(t, db.Upsert(ctx, doc.Name, uuid.New(), map[string]any{"Номер": "1", "Дата": time.Date(2026, 9, 1, 12, 5, 0, 0, loc)}, doc))
	var date metadata.Field
	for _, f := range doc.Fields {
		if f.Name == "Дата" {
			date = f
		}
	}
	_, err = db.Exec(ctx, `UPDATE "`+metadata.TableName(doc.Name)+`" SET "`+metadata.ColumnName(date)+`" = '2026-09-01 12:05:00'`)
	require.NoError(t, err)
	compiled, err := query.Compile("ВЫБРАТЬ НачалоДня(З.Дата) КАК День, НачалоЧаса(З.Дата) КАК Час ИЗ Документ.LegacyDateProbe КАК З", query.CompileOpts{Entities: []*metadata.Entity{doc}, Dialect: db.Dialect()})
	require.NoError(t, err)
	rows, _, err := query.Run(ctx, db, &compiled)
	require.NoError(t, err, compiled.SQL)
	require.Len(t, rows, 1)
	require.NotNil(t, rows[0]["день"], "НачалоДня of a legacy SQLite date")
	require.NotNil(t, rows[0]["час"], "НачалоЧаса of a legacy SQLite date")
	assertLocalClock(t, rows[0]["час"], 2026, 9, 1, 12, 0, 0)
}
