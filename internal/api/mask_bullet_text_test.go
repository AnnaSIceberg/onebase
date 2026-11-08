package api

import (
	"bytes"
	"encoding/json"
	"net/http"
	"net/http/httptest"
	"testing"

	"github.com/go-chi/chi/v5"
	"github.com/google/uuid"
	"github.com/ivantit66/onebase/internal/auth"
	"github.com/ivantit66/onebase/internal/metadata"
)

func TestRESTV2SavesOrdinaryBulletText(t *testing.T) {
	entity := &metadata.Entity{Name: "Note", Kind: metadata.KindCatalog,
		Fields:     []metadata.Field{{Name: "Comment", Type: metadata.FieldTypeString}},
		TableParts: []metadata.TablePart{{Name: "Lines", Fields: []metadata.Field{{Name: "Comment", Type: metadata.FieldTypeString}}}},
	}
	h, ctx := newAPITestHandler(t, []*metadata.Entity{entity}, nil)
	router := chi.NewRouter()
	h.mountV2(router)
	body := []byte(`{"Comment":"• Позвонить клиенту","__tableparts":{"Lines":[{"Comment":"• Отправить письмо"}]}}`)
	request := httptest.NewRequest(http.MethodPost, "/api/v2/catalog/Note", bytes.NewReader(body))
	request.Header.Set("Content-Type", "application/json")
	request = request.WithContext(auth.ContextWithOpenAccess(request.Context()))
	response := httptest.NewRecorder()
	router.ServeHTTP(response, request)
	if response.Code != http.StatusOK {
		t.Fatalf("POST: %d %s", response.Code, response.Body.String())
	}
	var result struct {
		Data struct {
			ID string `json:"id"`
		} `json:"data"`
	}
	if err := json.Unmarshal(response.Body.Bytes(), &result); err != nil {
		t.Fatal(err)
	}
	id, err := uuid.Parse(result.Data.ID)
	if err != nil {
		t.Fatalf("response: %s: %v", response.Body.String(), err)
	}
	row, err := h.store.GetByID(ctx, entity.Name, id, entity)
	if err != nil {
		t.Fatal(err)
	}
	if row["Comment"] != "• Позвонить клиенту" && row["comment"] != "• Позвонить клиенту" {
		t.Fatalf("header: %v", row)
	}
	rows, err := h.store.GetTablePartRows(ctx, entity.Name, "Lines", id, entity.TableParts[0])
	if err != nil {
		t.Fatal(err)
	}
	if len(rows) != 1 || (rows[0]["Comment"] != "• Отправить письмо" && rows[0]["comment"] != "• Отправить письмо") {
		t.Fatalf("tablepart: %v", rows)
	}
}
