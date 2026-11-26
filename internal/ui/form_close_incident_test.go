package ui

import (
	"net/http"
	"net/http/httptest"
	"strings"
	"testing"

	"github.com/ivantit66/onebase/internal/incident"
)

func TestManagedFormCloseIntentRecordsEarlyPanic(t *testing.T) {
	s := &Server{incidents: incident.NewStore(8)}
	request := httptest.NewRequest(http.MethodPost, "/ui/document/А_Задача/form-close-intent", nil)
	response := httptest.NewRecorder()
	s.captureFormCloseResponse(response, request, &formCloseInvocation{}, func(http.ResponseWriter) {
		panic("synthetic failure before close envelope")
	})
	if response.Code != http.StatusInternalServerError ||
		!strings.Contains(response.Body.String(), "код инцидента") ||
		strings.Contains(response.Body.String(), "synthetic failure") {
		t.Fatalf("unexpected response: %d %s", response.Code, response.Body.String())
	}
	records := s.incidents.Recent("", 8)
	if len(records) != 1 || records[0].Kind != incident.KindPanic ||
		records[0].Text != "synthetic failure before close envelope" || records[0].Stack == "" {
		t.Fatalf("diagnostic missing: %+v", records)
	}
}
