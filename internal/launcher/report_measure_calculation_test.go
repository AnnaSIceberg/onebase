package launcher

import (
	"html"
	"net/url"
	"reflect"
	"strings"
	"testing"

	"github.com/ivantit66/onebase/internal/report"
	"gopkg.in/yaml.v3"
)

func TestReportMeasureCalculationEditorRoundTrip(t *testing.T) {
	rules := []report.MeasureCalculation{
		{Level: "D", Agg: "avg", GroupBy: "B", InnerAgg: "avg"},
		{Level: report.MeasureScopeGrand, Agg: "count_distinct"},
	}
	comp := &report.Composition{Groupings: []string{"D", "B"}, Measures: []report.Measure{{Field: "X", Agg: "sum", Calculations: rules, NullWhenEmpty: true}}}
	page := renderConfiguratorReport(t, comp)
	tag := `name="comp.measure.0.calculations" value="`
	at := strings.Index(page, tag)
	if at < 0 {
		t.Fatal("calculation metadata not rendered")
	}
	value := strings.SplitN(page[at+len(tag):], `"`, 2)[0]
	if !strings.Contains(page, `name="comp.measure.0.null_when_empty" value="true"`) {
		t.Fatal("empty sum metadata not rendered")
	}
	form := url.Values{"comp.present": {"1"}, "comp.grouping.0": {"D"}, "comp.grouping.1": {"B"}, "comp.measure.0.field": {"X"}, "comp.measure.0.agg": {"sum"}, "comp.measure.0.calculations": {html.UnescapeString(value)}, "comp.measure.0.null_when_empty": {"true"}}
	raw, err := applyReportComposition([]byte("name: Levels\nquery: SELECT 1\n"), form)
	if err != nil {
		t.Fatal(err)
	}
	var restored report.Report
	if err := yaml.Unmarshal(raw, &restored); err != nil {
		t.Fatal(err)
	}
	if !restored.Composition.Measures[0].NullWhenEmpty {
		t.Fatal("empty sum metadata lost on save")
	}
	if !reflect.DeepEqual(restored.Composition.Measures[0].Calculations, rules) {
		t.Fatalf("calculations lost: %+v", restored.Composition)
	}
	for _, invalid := range []string{`[{"Level":"D","Unknown":true}]`, `[`, strings.Repeat(" ", 65537) + `[]`, `[] []`} {
		form.Set("comp.measure.0.calculations", invalid)
		if _, err := applyReportComposition(raw, form); err == nil {
			t.Fatal("malformed calculation silently saved")
		}
	}
	// User-controlled expression text in trusted YAML stays inside the input.
	rules[0].Expr = `"/><script>unsafe()</script>`
	rules[0].GroupBy = ""
	rules[0].InnerAgg = ""
	page = renderConfiguratorReport(t, comp)
	if strings.Contains(page, "<script>unsafe()") {
		t.Fatal("hidden JSON attribute injection")
	}
}
