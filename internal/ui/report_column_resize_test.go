package ui

import (
	"net/http"
	"net/http/httptest"
	"strings"
	"testing"

	"github.com/google/uuid"
	"github.com/ivantit66/onebase/internal/metadata"
	reportpkg "github.com/ivantit66/onebase/internal/report"
	"github.com/ivantit66/onebase/internal/runtime"
)

func TestReportResultColumnResizeHTML(t *testing.T) {
	entity := &metadata.Entity{
		Name: "ResizeRows", Kind: metadata.KindCatalog,
		Fields: []metadata.Field{
			{Name: "Bucket", Type: "string"},
			{Name: "PeriodName", Type: "string"},
			{Name: "Amount", Type: "number"},
		},
	}
	s, ctx := newSubmitTestServer(t, []*metadata.Entity{entity})
	t.Cleanup(s.Close)
	if err := s.store.EnsureServiceSchema(ctx); err != nil {
		t.Fatal(err)
	}
	values := map[string]any{"Bucket": "Group A", "PeriodName": "January", "Amount": 75}
	if err := s.store.Upsert(ctx, entity.Name, uuid.New(), values, entity); err != nil {
		t.Fatal(err)
	}

	group := reportpkg.Composition{
		Groupings: []string{"Bucket"},
		Measures:  []reportpkg.Measure{{Field: "Amount", Agg: "sum"}},
	}
	cross := group
	cross.Columns = []string{"PeriodName"}
	reports := []*reportpkg.Report{
		{Name: "ResizeFlat", Title: "Flat"},
		{Name: "ResizeGroup", Title: "Grouped", Composition: &group},
		{Name: "ResizeCross", Title: "Cross", Composition: &cross},
	}
	for _, rep := range reports {
		rep.Query = "ВЫБРАТЬ Bucket, PeriodName, Amount ИЗ Справочник.ResizeRows"
	}
	s.reg.Load(runtime.LoadOptions{Entities: []*metadata.Entity{entity}, Reports: reports})

	for _, rep := range reports {
		t.Run(rep.Name, func(t *testing.T) {
			w := httptest.NewRecorder()
			r := reqWithChi(http.MethodGet, "/ui/report/"+rep.Name+"?__run=1", nil, map[string]string{"name": rep.Name})
			s.reportForm(w, r)
			body := w.Body.String()
			for _, want := range []string{"Group A", "data-ob-report-columns", "setPointerCapture"} {
				if w.Code != http.StatusOK || !strings.Contains(body, want) {
					t.Fatalf("report result missing %q (HTTP %d)", want, w.Code)
				}
			}
		})
	}
}
