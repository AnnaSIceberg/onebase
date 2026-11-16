package rowsource

import (
	"github.com/shopspring/decimal"
	"testing"
)

func TestPlatform3AdditionalQueryInputContract(t *testing.T) {
	t.Run("join_number_matches_initial_number", func(t *testing.T) {
		ctx, db, reg, _, src, _ := fixture(t)
		src.Code = `Функция Build() Экспорт
   Для Каждого Р Из Результат Цикл
    З=Новый Запрос;
    З.Текст="ВЫБРАТЬ Р.Amount КАК A ИЗ Документ.RowRequest КАК Р ВНУТРЕННЕЕ СОЕДИНЕНИЕ Документ.RowRequest.Works КАК Т ПО Т.Ссылка=Р.Ссылка";
    Для Каждого Д Из З.Выполнить() Цикл
     Если ТипЗнч(Д.A) = ТипЗнч(Р.Amount) Тогда Р.Вставить("Works","same"); Иначе Р.Вставить("Works","wrong_type"); КонецЕсли;
    КонецЦикла;
   КонецЦикла;
   Возврат Результат;
  КонецФункции`
		result, err := Run(ctx, db, reg, Query{Text: initialQuery}, src)
		if err != nil {
			t.Fatal(err)
		}
		if len(result.Rows) != 1 || result.Rows[0]["Works"] != "same" {
			t.Fatalf("additional JOIN Query lost proven number type: %v", result.Rows)
		}
	})
	t.Run("additional_date_null_stays_null", func(t *testing.T) {
		ctx, db, reg, ent, src, id := fixture(t)
		if err := db.Upsert(ctx, ent.Name, id, map[string]any{"Label": "safe", "Note": "", "Amount": decimal.NewFromInt(1), "Count": decimal.NewFromInt(1), "Moment": nil, "Owner": "reader"}, ent); err != nil {
			t.Fatal(err)
		}
		src.Code = `Функция Build() Экспорт
   Для Каждого Р Из Результат Цикл
    З=Новый Запрос;
    З.Текст="ВЫБРАТЬ Р.Moment КАК M ИЗ Документ.RowRequest КАК Р";
    Для Каждого Д Из З.Выполнить() Цикл
     Если Д.M = Неопределено Тогда Р.Вставить("Works","null"); Иначе Р.Вставить("Works","not_null"); КонецЕсли;
    КонецЦикла;
   КонецЦикла;
   Возврат Результат;
  КонецФункции`
		result, err := Run(ctx, db, reg, Query{Text: initialQuery}, src)
		if err != nil {
			t.Fatal(err)
		}
		if len(result.Rows) != 1 || result.Rows[0]["Works"] != "null" {
			t.Fatalf("additional Query changed SQL NULL before DSL: %v", result.Rows)
		}
	})
}
