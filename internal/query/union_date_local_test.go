package query_test

import (
	"context"
	"testing"
	"time"
	_ "time/tzdata"

	"github.com/google/uuid"
	"github.com/ivantit66/onebase/internal/dbtest"
	"github.com/ivantit66/onebase/internal/metadata"
	"github.com/ivantit66/onebase/internal/query"
	"github.com/ivantit66/onebase/internal/storage"
	"github.com/stretchr/testify/require"
)

// А_ЗвонкиПоНаправлениям «по месяцам (с архивом)»: ДЕНЬ/МЕСЯЦ/ГОД/НачалоМесяца
// от даты подзапроса-ОБЪЕДИНИТЬ считались на SQLite по UTC — 05.08.2026 00:00 МСК
// (в базе 2026-08-04 21:00:00+00:00) давал день 4. Дата во всех ветвях
// доказывает колонку-дату, и функция берёт местное время.
func TestQueryCalendarOfUnionDerivedDate(t *testing.T) {
	loc, err := time.LoadLocation("Europe/Moscow")
	require.NoError(t, err)
	previous := time.Local
	time.Local = loc
	t.Cleanup(func() { time.Local = previous })
	dbtest.ForEachDialect(t, func(t *testing.T, db *storage.DB) {
		ctx := context.Background()
		calls := &metadata.Entity{Name: "UnionCallProbe", Kind: metadata.KindDocument, Fields: []metadata.Field{{Name: "Дата", Type: metadata.FieldTypeDate}}}
		orders := &metadata.Entity{Name: "UnionOrderProbe", Kind: metadata.KindDocument, Fields: []metadata.Field{{Name: "Дата", Type: metadata.FieldTypeDate}}}
		require.NoError(t, db.Migrate(ctx, []*metadata.Entity{calls, orders}))
		moment := time.Date(2026, 8, 5, 0, 0, 0, 0, loc)
		require.NoError(t, db.Upsert(ctx, calls.Name, uuid.New(), map[string]any{"Дата": moment}, calls))
		require.NoError(t, db.Upsert(ctx, orders.Name, uuid.New(), map[string]any{"Дата": moment}, orders))
		compiled, err := query.Compile(`ВЫБРАТЬ ДЕНЬ(Д.Дата) КАК День, МЕСЯЦ(Д.Дата) КАК Месяц, ГОД(Д.Дата) КАК Год, НАЧАЛОМЕСЯЦА(Д.Дата) КАК НачалоМ, Д.Звонки КАК Звонки
			ИЗ (ВЫБРАТЬ Зв.Дата КАК Дата, 1 КАК Звонки ИЗ Документ.UnionCallProbe КАК Зв
				ОБЪЕДИНИТЬ ВСЕ
				ВЫБРАТЬ З.Дата, 0 ИЗ Документ.UnionOrderProbe КАК З) КАК Д`,
			query.CompileOpts{Entities: []*metadata.Entity{calls, orders}, Dialect: db.Dialect()})
		require.NoError(t, err)
		rows, _, err := query.Run(ctx, db, &compiled)
		require.NoError(t, err, compiled.SQL)
		require.Len(t, rows, 2)
		for _, row := range rows {
			require.Equal(t, 5, intValue(t, row["день"]), compiled.SQL)
			require.Equal(t, 8, intValue(t, row["месяц"]))
			require.Equal(t, 2026, intValue(t, row["год"]))
			switch v := row["началом"].(type) {
			case time.Time:
				local := v.In(loc)
				require.Equal(t, [3]int{2026, 8, 1}, [3]int{local.Year(), int(local.Month()), local.Day()})
			case string:
				require.Equal(t, "2026-08-01", v[:10])
			default:
				t.Fatalf("НачалоМесяца: %T %v", v, v)
			}
		}
	})
}
