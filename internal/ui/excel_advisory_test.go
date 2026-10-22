package ui

import (
	"net/http"
	"net/http/httptest"
	"testing"

	"github.com/go-chi/chi/v5"
	"github.com/google/uuid"
	"github.com/ivantit66/onebase/internal/metadata"
	"github.com/ivantit66/onebase/internal/printform"
	"github.com/ivantit66/onebase/internal/runtime"
	"github.com/ivantit66/onebase/internal/testutil/xlsxadvisory"
	"strings"
)

func TestPrintDocumentXLSX_PublicAdvisories(t *testing.T) {
	for _, scenario := range xlsxadvisory.Cases {
		t.Run(scenario, func(t *testing.T) {
			xlsxadvisory.Run(t, func() {
				ent := &metadata.Entity{Name: "Карточка", Kind: metadata.KindCatalog, Fields: []metadata.Field{{Name: "Наименование", Type: metadata.FieldTypeString}}}
				s, ctx := newSubmitTestServer(t, []*metadata.Entity{ent})
				s.reg.Load(runtime.LoadOptions{Entities: []*metadata.Entity{ent}})
				id := uuid.New()
				if err := s.store.Upsert(ctx, ent.Name, id, map[string]any{"Наименование": "007"}, ent); err != nil {
					t.Fatal(err)
				}
				s.reg.LoadLayoutForms([]*printform.LayoutForm{{Name: "Бланк", Document: ent.Name, Layout: &printform.LayoutTemplate{}, XLSXTemplate: xlsxadvisory.Book(t, scenario)}})
				router := chi.NewRouter()
				s.Mount(router)
				rec := httptest.NewRecorder()
				router.ServeHTTP(rec, httptest.NewRequest(http.MethodGet, "/ui/catalog/Карточка/"+id.String()+"/print/Бланк/xlsx", nil))
				if scenario == "cyclic-array" {
					if rec.Code != 200 || !strings.HasPrefix(rec.Body.String(), "PK") {
						t.Fatalf("cached workbook render: %d %s", rec.Code, rec.Body.String())
					}
				} else {
					if rec.Code != http.StatusInternalServerError || strings.HasPrefix(rec.Body.String(), "PK") || rec.Header().Get("Content-Disposition") != "" {
						t.Fatalf("bad template produced download: %d %v", rec.Code, rec.Header())
					}
				}
			})
		})
	}
}
