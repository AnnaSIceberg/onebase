package interpreter_test

// Сортировка по ссылке в сгруппированном запросе (матрица SQLite/PostgreSQL).
//
// «СГРУППИРОВАТЬ ПО Р.Номенклатура УПОРЯДОЧИТЬ ПО Номенклатура»: группировка
// идёт по колонке ссылки, а сортировка — по её представлению из авто-JOIN
// (ref_номенклатура.наименование). Этой колонки в GROUP BY нет, и PostgreSQL
// отвергал запрос: column must appear in the GROUP BY clause (42803). SQLite
// такой запрос молча принимает — отсюда матрица. Найдено на отчёте «По штрафам»
// конфигурации колл-центра при пробе выпуска на PostgreSQL.

import (
	"context"
	"testing"
	"time"

	"github.com/google/uuid"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"

	"github.com/ivantit66/onebase/internal/dbtest"
	"github.com/ivantit66/onebase/internal/metadata"
	"github.com/ivantit66/onebase/internal/storage"
)

func TestQueryGroupedOrderByRefPresentation(t *testing.T) {
	dbtest.ForEachDialect(t, func(t *testing.T, db *storage.DB) {
		ctx := context.Background()
		ents := refQueryEntities()
		ents[1].Fields = append(ents[1].Fields, metadata.Field{ID: "d_date", Name: "Дата", Type: metadata.FieldTypeDate})
		require.NoError(t, db.Migrate(ctx, ents))

		bolt, anchor := uuid.New(), uuid.New()
		require.NoError(t, db.Upsert(ctx, "Номенклатура", bolt, map[string]any{"Наименование": "Болт"}, ents[0]))
		require.NoError(t, db.Upsert(ctx, "Номенклатура", anchor, map[string]any{"Наименование": "Анкер"}, ents[0]))
		when := time.Date(2026, 10, 1, 10, 0, 0, 0, time.Local)
		for _, nom := range []uuid.UUID{bolt, bolt, anchor} {
			require.NoError(t, db.Upsert(ctx, "РасходТовара", uuid.New(),
				map[string]any{"Номер": "1", "Номенклатура": nom.String(), "Дата": when}, ents[1]))
		}

		result := runOnDBEntities(t, db, ents, `
Функция Тест()
	З = Новый Запрос;
	З.Текст = "ВЫБРАТЬ ГОД(Р.Дата) КАК Год, Р.Номенклатура КАК Номенклатура, КОЛИЧЕСТВО(*) КАК Ч
	|ИЗ Документ.РасходТовара КАК Р
	|СГРУППИРОВАТЬ ПО ГОД(Р.Дата), Р.Номенклатура
	|УПОРЯДОЧИТЬ ПО Год, Номенклатура";
	Итог = "";
	Для Каждого С Из З.Выполнить() Цикл
		Итог = Итог + Строка(С.Номенклатура) + "=" + Строка(С.Ч) + ";";
	КонецЦикла;
	Возврат Итог;
КонецФункции
`)
		// «Анкер» раньше «Болта» — порядок по представлению, а не по UUID.
		assert.Equal(t, anchor.String()+"=1;"+bolt.String()+"=2;", result, "сортировка по представлению ссылки в сгруппированной выборке")
	})
}
