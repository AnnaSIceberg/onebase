package query_test

import (
	"testing"
	"time"

	"github.com/google/uuid"
	"github.com/stretchr/testify/require"

	"github.com/ivantit66/onebase/internal/dbtest"
	"github.com/ivantit66/onebase/internal/metadata"
	"github.com/ivantit66/onebase/internal/query"
	"github.com/ivantit66/onebase/internal/storage"
)

func TestISODateTimeCallDuration(t *testing.T) {
	const fallback = 71
	cases := map[string]struct {
		start, end any
		seconds    any
		duration   int
	}{
		"normal":    {"2026-09-01T09:00:00Z", "2026-09-01T09:00:33Z", 33, 33},
		"offsets":   {"2026-09-01T12:00:00+03:00", "2026-09-01T05:00:33-04:00", 33, 33},
		"fraction":  {"2026-09-01T09:00:00.999999Z", "2026-09-01T09:00:01.000001Z", 1, 1},
		"empty":     {"", "2026-09-01T09:00:33Z", nil, fallback},
		"missing":   {"2026-09-01T09:00:00Z", nil, nil, fallback},
		"malformed": {"2026-09-01T09:00:00Z", "bad", nil, fallback},
		"calendar":  {"2026-02-30T09:00:00Z", "2026-09-01T09:00:33Z", nil, fallback},
		"no zone":   {"2026-09-01T09:00:00", "2026-09-01T09:00:33Z", nil, fallback},
		"year zero": {"0000-09-01T09:00:00Z", "2026-09-01T09:00:33Z", nil, fallback},
		"reverse":   {"2026-09-01T12:00:33+03:00", "2026-09-01T09:00:00Z", -33, fallback},
	}
	dbtest.ForEachDialect(t, func(t *testing.T, db *storage.DB) {
		ctx := t.Context()
		ent := &metadata.Entity{Name: "Разговоры", Kind: metadata.KindCatalog, Fields: []metadata.Field{
			{Name: "Наименование", Type: metadata.FieldTypeString},
			{Name: "Начало", Type: metadata.FieldTypeString},
			{Name: "Окончание", Type: metadata.FieldTypeString},
			{Name: "CDR", Type: metadata.FieldTypeNumber},
		}}
		require.NoError(t, db.Migrate(ctx, []*metadata.Entity{ent}))
		for name, tc := range cases {
			require.NoError(t, db.Upsert(ctx, ent.Name, uuid.New(), map[string]any{
				"Наименование": name, "Начало": tc.start, "Окончание": tc.end, "CDR": fallback,
			}, ent))
		}
		opts := query.CompileOpts{Dialect: db.Dialect(), Entities: []*metadata.Entity{ent}}
		compiled, err := query.Compile(`
			ВЫБРАТЬ
				З.Наименование,
				ДатаИзISO(З.Начало) КАК Момент,
				РАЗНОСТЬДАТ(ДатаИзISO(З.Начало), ISODateTime(З.Окончание), СЕКУНДА) КАК Секунд,
				ВЫБОР КОГДА РАЗНОСТЬДАТ(ДатаИзISO(З.Начало), ДатаИзISO(З.Окончание), СЕКУНДА) >= 0
					ТОГДА РАЗНОСТЬДАТ(ДатаИзISO(З.Начало), ДатаИзISO(З.Окончание), СЕКУНДА)
					ИНАЧЕ З.CDR + 0 КОНЕЦ КАК Длительность
			ИЗ Справочник.Разговоры КАК З`, opts)
		require.NoError(t, err)
		rows, _, err := query.Run(ctx, db, &compiled)
		require.NoError(t, err, compiled.SQL)
		require.Len(t, rows, len(cases))
		for _, row := range rows {
			name, ok := row["наименование"].(string)
			require.True(t, ok)
			tc, known := cases[name]
			require.True(t, known, name)
			require.Equal(t, tc.duration, intValue(t, row["длительность"]), name)
			if tc.seconds == nil {
				require.Nil(t, row["секунд"], name)
			} else {
				require.Equal(t, tc.seconds, intValue(t, row["секунд"]), name)
			}
			if name == "offsets" {
				got, ok := row["момент"].(time.Time)
				require.True(t, ok, "expected time.Time, got %T", row["момент"])
				require.True(t, got.Equal(time.Date(2026, 9, 1, 9, 0, 0, 0, time.UTC)))
			}
		}
	})
}

func TestISODateTimeRequiresOneArgument(t *testing.T) {
	for _, src := range []string{"ВЫБРАТЬ ДатаИзISO()", "ВЫБРАТЬ ISODateTime(1, 2)"} {
		_, err := query.Compile(src, query.CompileOpts{})
		require.ErrorContains(t, err, "ДатаИзISO")
	}
}
