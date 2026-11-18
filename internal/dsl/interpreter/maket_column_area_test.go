package interpreter

import (
	"strings"
	"testing"

	"github.com/ivantit66/onebase/internal/dsl/lexer"
	"github.com/ivantit66/onebase/internal/dsl/parser"
	"github.com/ivantit66/onebase/internal/metadata"
	"github.com/ivantit66/onebase/internal/printform"
	"github.com/ivantit66/onebase/internal/runtime"
)

// Заказ kc2 (финотчёт как в 1С): ПолучитьОбласть("Строки|Колонки") —
// пересечение строковой и колоночной областей макета; Вывести/Присоединить
// переносят ширины ИСХОДНЫХ колонок макета, а не с первой.
func TestMaketRowColumnIntersection(t *testing.T) {
	lt, err := printform.ParseLayoutBytes([]byte(`name: Фин
columns:
  - width: 60px
  - width: 70px
  - width: 80px
  - width: 90px
areas:
  - name: Шапка
    rows:
      - cells:
          - text: A
          - text: B
          - text: C
            colspan: 2
column_areas:
  - name: Месяц
    from: 3
    to: 3
  - name: Итог
    from: 4
    to: 4
  - name: Показатели
    from: 1
    to: 2
`))
	if err != nil {
		t.Fatal(err)
	}
	src := `Процедура Main()
		Д = Новый ТабличныйДокумент;
		Д.Вывести(Макет.ПолучитьОбласть("Шапка|Показатели"));
		Д.Присоединить(Макет.ПолучитьОбласть("Шапка|Месяц"));
		Д.Присоединить(Макет.ПолучитьОбласть("Шапка|Месяц"));
		Д.Присоединить(Макет.ПолучитьОбласть("Шапка|Итог"));
		this.Result = Д;
	КонецПроцедуры`
	p, e := parser.New(lexer.New(src, "fin.os")).ParseProgram()
	if e != nil {
		t.Fatal(e)
	}
	o := runtime.NewObject("Own", metadata.KindDocument)
	i := New()
	vars := NewSpreadsheetFunctions()
	vars["Макет"] = NewMaket(lt)
	if e = i.Run(p.Procedures[0], o, vars); e != nil {
		t.Fatal(e)
	}
	d, ok := o.Get("Result").(*SpreadsheetDocument)
	if !ok {
		t.Fatal("no document")
	}
	var got []string
	for c := 0; c < 5; c++ {
		text := ""
		if cell := d.Doc.GetCell(0, c); cell != nil {
			text = cell.Text
		}
		got = append(got, text)
	}
	// Колонка 4 («Итог») в макете накрыта объединением C (colspan 2) — пусто.
	if got[0] != "A" || got[1] != "B" || got[2] != "C" || got[3] != "C" || got[4] != "" {
		t.Fatalf("ячейки: %q", got)
	}
	if cell := d.Doc.GetCell(0, 2); cell == nil || cell.ColSpan > 1 {
		t.Fatal("объединение на границе колоночной области не обрезано")
	}
	for col, want := range map[int]float64{1: 60, 2: 70, 3: 80, 4: 80, 5: 90} {
		if w := d.Doc.ColumnWidth(col); w != want {
			t.Fatalf("ширина колонки %d = %v, want %v (по исходной колонке макета)", col, w, want)
		}
	}
	if NewMaket(lt).getArea("Шапка|Нет") != nil || NewMaket(lt).getArea("Нет|Месяц") != nil {
		t.Fatal("несуществующая область пересечения")
	}
	if _, err := printform.ParseLayoutBytes([]byte("column_areas:\n  - name: X\n    from: 3\n    to: 2\n")); err == nil {
		t.Fatal("from > to принят")
	}
}

// Регрессия 96a202410 (kc2): макеты с областями, буквально названными «X|Y»
// (ОтчетKPI «Шапка|Секция_1», финотчёт «ШапкаТаблицы|Раздел»), получали
// Неопределено и падали nil pointer. Точное имя области — первым; пересечение —
// только если такой области нет. Ненайденная область — исключение модуля.
func TestMaketLiteralPipeAreaAndMissingAreaError(t *testing.T) {
	lt, err := printform.ParseLayoutBytes([]byte(`name: KPI
areas:
  - name: Шапка|Секция_1
    rows:
      - cells:
          - text: LITERAL
  - name: Шапка
    rows:
      - cells:
          - text: A
          - text: B
column_areas:
  - name: Секция_1
    from: 2
    to: 2
`))
	if err != nil {
		t.Fatal(err)
	}
	run := func(src string) any {
		t.Helper()
		p, e := parser.New(lexer.New(src, "kpi.os")).ParseProgram()
		if e != nil {
			t.Fatal(e)
		}
		o := runtime.NewObject("Own", metadata.KindDocument)
		vars := NewSpreadsheetFunctions()
		vars["Макет"] = NewMaket(lt)
		if e = New().Run(p.Procedures[0], o, vars); e != nil {
			t.Fatal(e)
		}
		return o.Get("Result")
	}
	d, ok := run(`Процедура Main() Д = Новый ТабличныйДокумент; Д.Вывести(Макет.ПолучитьОбласть("Шапка|Секция_1")); this.Result = Д; КонецПроцедуры`).(*SpreadsheetDocument)
	if !ok {
		t.Fatal("no document")
	}
	if cell := d.Doc.GetCell(0, 0); cell == nil || cell.Text != "LITERAL" {
		t.Fatal("буквальная область «Шапка|Секция_1» должна побеждать пересечение")
	}
	msg := run(`Процедура Main()
		Попытка
			О = Макет.ПолучитьОбласть("Нет|Такой");
			this.Result = "без ошибки";
		Исключение
			this.Result = ОписаниеОшибки();
		КонецПопытки;
	КонецПроцедуры`)
	if s, _ := msg.(string); !strings.Contains(s, "Область макета не найдена: Нет|Такой") {
		t.Fatalf("ненайденная область — исключение модуля: %v", msg)
	}
}
