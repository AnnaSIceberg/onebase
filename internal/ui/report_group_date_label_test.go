package ui

import (
	"strings"
	"testing"
	"time"

	"github.com/ivantit66/onebase/internal/dsl/interpreter"
	"github.com/ivantit66/onebase/internal/metadata"
	"github.com/ivantit66/onebase/internal/query"
	"github.com/ivantit66/onebase/internal/report"
	"github.com/ivantit66/onebase/internal/report/compose"
)

// Группа-дата (НачалоДня/НачалоЧаса в А_ВторойКонтур) подписывалась сырым ISO:
// «2026-09-01 12:00:00». Подпись — как у даты детальной записи; путь группы
// (data-group) по-прежнему строится по сырому ключу.
func TestReportGroupDateLabel(t *testing.T) {
	moment := time.Date(2026, 9, 2, 13, 0, 0, 0, time.Local)
	rows := []compose.Row{
		{"Час": "2026-09-01 12:00:00", "День": "2026-09-01", "Вид": "Обычный", "Задач": "1"},
		{"Час": moment, "День": "2026-09-01", "Вид": "Обычный", "Задач": "1"},
	}
	spec := report.Composition{
		Groupings: []string{"День", "Час", "Вид"},
		Measures:  []report.Measure{{Field: "Задач", Agg: "sum"}},
		Totals:    report.Totals{Grand: true},
	}
	res, err := compose.Compose(rows, spec, newInterpEvaluator(interpreter.New()))
	if err != nil {
		t.Fatal(err)
	}
	out := string(renderComposedTable(res, &spec))
	for _, want := range []string{"> 01.09.2026</td>", "> 01.09.2026 12:00:00</td>", "> 02.09.2026 13:00:00</td>", "Обычный", `data-group="/2026-09-01/2026-09-01 12:00:00"`} {
		if !strings.Contains(out, want) {
			t.Fatalf("HTML без %q:\n%s", want, out)
		}
	}
	if strings.Contains(out, "> 2026-09-01") || strings.Contains(out, `aria-label="2026-`) {
		t.Fatalf("сырая дата в подписи группы:\n%s", out)
	}
	_, sheet := composedRows(res, &spec)
	var labels []string
	for _, r := range sheet {
		labels = append(labels, strings.TrimSpace(r[0].(string)))
	}
	joined := strings.Join(labels, "|")
	for _, want := range []string{"01.09.2026", "01.09.2026 12:00:00", "02.09.2026 13:00:00"} {
		if !strings.Contains(joined, want) {
			t.Fatalf("Excel без %q: %v", want, labels)
		}
	}

	cross := report.Composition{Groupings: []string{"Час"}, Columns: []string{"Вид"}, Measures: spec.Measures}
	cr, _ := compose.ComposeCross(rows, cross, nil)
	if out := string(renderCrossTable(cr, &cross)); !strings.Contains(out, "01.09.2026 12:00:00") {
		t.Fatalf("кросс-таблица: сырая дата в подписи:\n%s", out)
	}
	_, crossSheet := crossSheetRows(cr, &cross)
	if got := strings.TrimSpace(crossSheet[0][0].(string)); got != "01.09.2026 12:00:00" && got != "02.09.2026 13:00:00" {
		t.Fatalf("кросс-Excel: подпись %q", got)
	}
}

// Группа по функции периода (НачалоЧаса/НачалоДня в запросе) — в 1С значение
// типа дата-время: полночь подписывается «01.09.2026 0:00:00» и не сливается с
// группой-днём; группа по обычному полю-дате — как прежде.
func TestReportGroupPeriodCallLabel(t *testing.T) {
	doc := &metadata.Entity{Name: "ПериодПроба", Kind: metadata.KindDocument, Fields: []metadata.Field{{Name: "Дата", Type: metadata.FieldTypeDate}}}
	compiled, err := query.Compile(`ВЫБРАТЬ НачалоДня(З.Дата) КАК ДатаДня, НачалоЧаса(З.Дата) КАК Час, З.Дата КАК Дата, 1 КАК Задач
		ИЗ Документ.ПериодПроба КАК З`, query.CompileOpts{Entities: []*metadata.Entity{doc}})
	if err != nil {
		t.Fatal(err)
	}
	s, _ := newSubmitTestServer(t, []*metadata.Entity{doc})
	spec := report.Composition{Groupings: []string{"ДатаДня", "Час", "Дата"}, Measures: []report.Measure{{Field: "Задач", Agg: "sum"}}}
	presentation := s.compositionPresentation(&report.Report{}, &spec, &compiled, "ru")
	rows := []compose.Row{{"ДатаДня": "2026-09-01", "Час": "2026-09-01 00:00:00", "Дата": "2026-09-01 00:00:00", "Задач": "1"}}
	res, err := compose.Compose(rows, spec, newInterpEvaluator(interpreter.New()))
	if err != nil {
		t.Fatal(err)
	}
	out := string(renderComposedTablePresented(res, &spec, presentation))
	if strings.Count(out, "> 01.09.2026 0:00:00</td>") != 2 || !strings.Contains(out, "> 01.09.2026</td>") {
		t.Fatalf("подписи групп-периодов:\n%s", out)
	}
	if got := presentation.groupLabel("Час", "2026-09-01 09:05:00"); got != "01.09.2026 9:05:00" {
		t.Fatalf("час без ведущего нуля, как в 1С: %q", got)
	}
}
