package query_test

import (
	"context"
	"fmt"
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

func TestQueryStartOfHourMatrix(t *testing.T) {
	for _, zone := range []string{"Europe/Moscow", "Asia/Kolkata", "America/New_York"} {
		t.Run(zone, func(t *testing.T) {
			loc, err := time.LoadLocation(zone)
			require.NoError(t, err)
			previous := time.Local
			time.Local = loc
			t.Cleanup(func() { time.Local = previous })
			dbtest.ForEachDialect(t, func(t *testing.T, db *storage.DB) {
				ctx := context.Background()
				doc := &metadata.Entity{Name: "HourProbe", Kind: metadata.KindDocument, Fields: []metadata.Field{
					{Name: "Номер", Type: metadata.FieldTypeString}, {Name: "Дата", Type: metadata.FieldTypeDate},
				}}
				require.NoError(t, db.Migrate(ctx, []*metadata.Entity{doc}))
				moments := []time.Time{
					time.Date(2026, 11, 17, 0, 45, 37, 123000000, loc),
					time.Date(2026, 11, 17, 0, 59, 59, 999000000, loc),
					time.Date(2026, 11, 18, 0, 1, 2, 0, loc),
					time.Date(2026, 11, 17, 1, 2, 3, 0, loc),
				}
				for i := 0; i <= len(moments); i++ {
					var value any
					if i < len(moments) {
						value = moments[i]
					}
					require.NoError(t, db.Upsert(ctx, doc.Name, uuid.New(), map[string]any{"Номер": fmt.Sprint(i), "Дата": value}, doc))
				}
				run := func(text string, params map[string]any) (query.Result, []map[string]any) {
					t.Helper()
					compiled, err := query.Compile(text, query.CompileOpts{Entities: []*metadata.Entity{doc}, Dialect: db.Dialect(), Params: params})
					require.NoError(t, err, text)
					rows, _, err := query.Run(ctx, db, &compiled)
					require.NoError(t, err, compiled.SQL)
					return compiled, rows
				}
				compiled, rows := run("ВЫБРАТЬ НачалоЧаса(З.Дата) КАК Час ИЗ Документ.HourProbe КАК З УПОРЯДОЧИТЬ ПО З.Номер", nil)
				require.Contains(t, compiled.DateColumns, "час")
				require.Len(t, rows, 5)
				for i, moment := range moments {
					got, ok := rows[i]["час"].(time.Time)
					require.True(t, ok, "datetime result: %T %v", rows[i]["час"], rows[i]["час"])
					want := time.Date(moment.Year(), moment.Month(), moment.Day(), moment.Hour(), 0, 0, 0, loc)
					require.True(t, got.Equal(want), "%s != %s", got, want)
					require.Zero(t, got.Nanosecond())
				}
				require.Nil(t, rows[4]["час"])

				_, groups := run("ВЫБРАТЬ НачалоЧаса(З.Дата) КАК Час, КОЛИЧЕСТВО(*) КАК Число ИЗ Документ.HourProbe КАК З СГРУППИРОВАТЬ ПО НачалоЧаса(З.Дата)", nil)
				counts := map[string]int{}
				for _, group := range groups {
					key := "NULL"
					if group["час"] != nil {
						got, ok := group["час"].(time.Time)
						require.True(t, ok, "group key must be datetime, not an hour number")
						key = got.In(loc).Format("2006-01-02 15:04:05")
					}
					counts[key] = intValue(t, group["число"])
				}
				require.Equal(t, map[string]int{"2026-11-17 00:00:00": 2, "2026-11-18 00:00:00": 1, "2026-11-17 01:00:00": 1, "NULL": 1}, counts)

				for _, value := range []any{moments[0], nil} {
					_, parameter := run("ВЫБРАТЬ НачалоЧаса(&Момент) КАК Час ИЗ Документ.HourProbe КАК З ГДЕ З.Номер = &Номер", map[string]any{"Момент": value, "Номер": "0"})
					require.Len(t, parameter, 1)
					if value == nil {
						require.Nil(t, parameter[0]["час"])
					} else {
						assertLocalClock(t, parameter[0]["час"], 2026, 11, 17, 0, 0, 0)
					}
				}
				_, nested := run("ВЫБРАТЬ НачалоЧаса(НачалоЧаса(З.Дата)) КАК Час, День(НачалоЧаса(З.Дата)) КАК День ИЗ Документ.HourProbe КАК З ГДЕ З.Номер = &Номер", map[string]any{"Номер": "0"})
				require.Len(t, nested, 1)
				assertLocalClock(t, nested[0]["час"], 2026, 11, 17, 0, 0, 0)
				require.Equal(t, 17, intValue(t, nested[0]["день"]))

				_, extremes := run("ВЫБРАТЬ НачалоЧаса(МИНИМУМ(З.Дата)) КАК Первый, НачалоЧаса(MAX(З.Дата)) КАК Последний ИЗ Документ.HourProbe КАК З", nil)
				require.Len(t, extremes, 1)
				require.IsType(t, time.Time{}, extremes[0]["первый"])
				require.IsType(t, time.Time{}, extremes[0]["последний"])
				assertLocalClock(t, extremes[0]["первый"], 2026, 11, 17, 0, 0, 0)
				assertLocalClock(t, extremes[0]["последний"], 2026, 11, 18, 0, 0, 0)
				_, empty := run("ВЫБРАТЬ НачалоЧаса(MIN(З.Дата)) КАК Час ИЗ Документ.HourProbe КАК З ГДЕ З.Номер = &Номер", map[string]any{"Номер": "missing"})
				require.Len(t, empty, 1)
				require.Nil(t, empty[0]["час"])
			})
		})
	}
}

func TestQueryStartOfHourRejectsWrongArity(t *testing.T) {
	for _, dialect := range []storage.Dialect{storage.SQLiteDialect{}, storage.PgDialect{}} {
		for _, expression := range []string{"НачалоЧаса()", "НачалоЧаса(&Дата, &Дата)", "НачалоЧаса(НачалоЧаса())"} {
			_, err := query.Compile("ВЫБРАТЬ "+expression+" КАК Час", query.CompileOpts{Dialect: dialect})
			require.ErrorContains(t, err, "один аргумент даты")
		}
	}
}
