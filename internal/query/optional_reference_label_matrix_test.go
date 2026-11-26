package query_test

import (
	"testing"

	"github.com/google/uuid"
	"github.com/stretchr/testify/require"

	"github.com/ivantit66/onebase/internal/dbtest"
	"github.com/ivantit66/onebase/internal/metadata"
	"github.com/ivantit66/onebase/internal/query"
	"github.com/ivantit66/onebase/internal/storage"
)

func TestOptionalReferenceLabelMatrix(t *testing.T) {
	dbtest.ForEachDialect(t, func(t *testing.T, db *storage.DB) {
		ctx := t.Context()
		catalog := &metadata.Entity{Name: "Причины", Kind: metadata.KindCatalog,
			Fields: []metadata.Field{{Name: "Наименование", Type: metadata.FieldTypeString}}}
		history := &metadata.InfoRegister{Name: "История",
			Dimensions: []metadata.Field{{Name: "Событие", Type: metadata.FieldTypeString}},
			Resources:  []metadata.Field{{Name: "Причина", Type: "reference:Причины", RefEntity: "Причины"}}}
		entities := []*metadata.Entity{catalog}
		registers := []*metadata.InfoRegister{history}
		require.NoError(t, db.Migrate(ctx, entities))
		require.NoError(t, db.MigrateInfoRegisters(ctx, registers))
		why := uuid.New()
		require.NoError(t, db.Upsert(ctx, catalog.Name, why, map[string]any{"Наименование": "В работе"}, catalog))
		require.NoError(t, db.InfoRegSet(ctx, history, map[string]any{"Событие": "Получено"}, map[string]any{"Причина": why.String()}, nil))

		const call = "ПредставлениеПриДоступе(Запись.Причина)"
		const from = " ИЗ РегистрСведений.История КАК Запись"
		const source = "ВЫБРАТЬ Запись.Событие, " + call + " КАК Причина" + from
		ref := query.SourceRef{Kind: "catalog", Name: catalog.Name}
		opts := query.CompileOpts{Entities: entities, InfoRegs: registers, Dialect: db.Dialect()}
		for _, allowed := range []bool{true, false} {
			opts.CanReadReferenceLabel = func(name string) bool {
				require.Equal(t, catalog.Name, name)
				return allowed
			}
			compiled, err := query.Compile(source, opts)
			require.NoError(t, err)
			rows, columns, err := query.Run(ctx, db, &compiled)
			require.NoError(t, err, compiled.SQL)
			require.Equal(t, []string{"событие", "причина"}, columns)
			require.Len(t, rows, 1)
			require.Equal(t, "Получено", rows[0]["событие"])
			if allowed {
				require.Contains(t, compiled.Sources, ref)
				require.Equal(t, "В работе", rows[0]["причина"])
			} else {
				require.NotContains(t, compiled.Sources, ref)
				require.Nil(t, rows[0]["причина"])
			}
		}

		compiled, err := query.Compile(source+" ЛЕВОЕ СОЕДИНЕНИЕ Справочник.Причины КАК П ПО П.Ссылка = Запись.Причина", opts)
		require.NoError(t, err)
		require.Contains(t, compiled.Sources, ref, "explicit reads still require catalog access")

		for _, invalid := range []string{
			"ВЫБРАТЬ " + call + from,
			"ВЫБРАТЬ Запись.Событие" + from + " ГДЕ " + call + " = \"В работе\"",
			"ВЫБРАТЬ ЕСТЬNULL(" + call + ", \"\") КАК Причина" + from,
			"ВЫБРАТЬ ПредставлениеПриДоступе(Запись.Событие) КАК Причина" + from,
			"ВЫБРАТЬ Р.Причина ИЗ (" + source + ") КАК Р",
			"ВЫБРАТЬ " + call + " КАК Причина ИЗ (ВЫБРАТЬ * ИЗ РегистрСведений.История) КАК Запись",
		} {
			_, err := query.Compile(invalid, opts)
			require.Error(t, err, invalid)
		}
	})
}
