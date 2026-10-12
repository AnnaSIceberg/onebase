package api

import (
	"net/http"
	"net/http/httptest"
	"testing"
	"time"

	"github.com/google/uuid"

	"github.com/ivantit66/onebase/internal/metadata"
	"github.com/ivantit66/onebase/internal/runtime"
	"github.com/ivantit66/onebase/internal/storage"
)

func TestAPI_DeletePostingDocumentMissing(t *testing.T) {
	for _, version := range []string{"v1", "v2"} {
		t.Run(version, func(t *testing.T) {
			for _, scenario := range []string{"repeat", "unknown", "orphan_movements"} {
				t.Run(scenario, func(t *testing.T) {
					doc := &metadata.Entity{
						Name: "УстановкаЦен", Kind: metadata.KindDocument, Posting: true,
						Fields: []metadata.Field{{Name: "Номер", Type: metadata.FieldTypeString}},
					}
					ir := &metadata.InfoRegister{
						Name: "Цены", Periodic: true,
						Dimensions: []metadata.Field{{Name: "Товар", Type: metadata.FieldTypeString}},
						Resources:  []metadata.Field{{Name: "Цена", Type: metadata.FieldTypeNumber}},
					}
					h, ctx := newAPITestHandler(t, []*metadata.Entity{doc}, nil)
					if err := h.store.MigrateInfoRegisters(ctx, []*metadata.InfoRegister{ir}); err != nil {
						t.Fatal(err)
					}
					h.reg.Load(runtime.LoadOptions{Entities: []*metadata.Entity{doc}, InfoRegs: []*metadata.InfoRegister{ir}})
					id := uuid.New()
					requests := 1
					if scenario == "repeat" {
						if err := h.store.Upsert(ctx, doc.Name, id, map[string]any{"Номер": "1"}, doc); err != nil {
							t.Fatal(err)
						}
						if err := h.store.SetPosted(ctx, doc.Name, id, true); err != nil {
							t.Fatal(err)
						}
						requests = 2
					}
					if scenario != "unknown" {
						period := time.Date(2026, 10, 12, 12, 0, 0, 0, time.UTC)
						if err := h.store.WriteInfoMovements(ctx, ir.Name, doc.Name, id, []map[string]any{{
							"Товар": "Молоток", "Цена": float64(100),
						}}, ir, &period); err != nil {
							t.Fatal(err)
						}
					}

					endpoint := h.deleteObject(metadata.KindDocument)
					path := "/documents/" + doc.Name + "/" + id.String()
					params := map[string]string{"entity": doc.Name, "id": id.String()}
					if version == "v2" {
						endpoint = h.deleteObjectV2(metadata.KindDocument)
						path = "/api/v2/document/" + doc.Name + "/" + id.String()
						params = map[string]string{"name": doc.Name, "id": id.String()}
					}
					for attempt := 1; attempt <= requests; attempt++ {
						w := httptest.NewRecorder()
						endpoint.ServeHTTP(w, reqWithEntity(http.MethodDelete, path, nil, params, nil))
						if w.Code != http.StatusNoContent {
							t.Fatalf("DELETE #%d: expected 204, got %d: %s", attempt, w.Code, w.Body.String())
						}
					}
					if _, err := h.store.GetByID(ctx, doc.Name, id, doc); !storage.IsNotFound(err) {
						t.Fatalf("document should be absent: %v", err)
					}
					rows, err := h.store.InfoRegList(ctx, ir, storage.RegFilter{})
					if err != nil {
						t.Fatal(err)
					}
					if len(rows) != 0 {
						t.Fatalf("DELETE left %d orphan movements: %v", len(rows), rows)
					}
				})
			}
		})
	}
}
