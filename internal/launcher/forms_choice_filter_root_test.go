package launcher

import (
	"context"
	"encoding/json"
	"net/http"
	"net/http/httptest"
	"net/url"
	"path/filepath"
	"strings"
	"testing"

	"github.com/go-chi/chi/v5"
	"github.com/ivantit66/onebase/internal/metadata"
)

func TestConfiguratorChoiceFilterIsRootHTTP(t *testing.T) {
	s := &Store{path: filepath.Join(t.TempDir(), "ibases.yaml")}
	b := &Base{Path: t.TempDir(), ConfigSource: "file"}
	if err := s.Add(b); err != nil {
		t.Fatal(err)
	}
	h := &handler{store: s}
	const src = `schema: onebase.form/v1
form:
  name: ФормаОбъекта
  kind: object
  entity: Заявка
elements:
  - id: city-picker
    kind: ПолеВвода
    name: ПолеГород
    data_path: Объект.Город
`
	post := func(condition string) editOpResponse {
		t.Helper()
		form := url.Values{"op": {"setChoiceFilter"}, "node": {"elements.0"}, "choice_filter": {condition}, "yaml": {src}}
		request := httptest.NewRequest(http.MethodPost, "/bases/"+b.ID+"/configurator/forms/edit-op", strings.NewReader(form.Encode()))
		request.Header.Set("Content-Type", "application/x-www-form-urlencoded")
		rctx := chi.NewRouteContext()
		rctx.URLParams.Add("id", b.ID)
		request = request.WithContext(context.WithValue(request.Context(), chi.RouteCtxKey, rctx))
		recorder := httptest.NewRecorder()
		h.configuratorFormsEditOp(recorder, request)
		var response editOpResponse
		if err := json.Unmarshal(recorder.Body.Bytes(), &response); err != nil {
			t.Fatalf("response: %v; body=%s", err, recorder.Body.String())
		}
		return response
	}
	for _, value := range []string{"true", "false"} {
		response := post(`[{"field":"is_root","op":"eq","value":` + value + `}]`)
		if !response.OK {
			t.Fatalf("is_root=%s rejected: %v", value, response.Errors)
		}
		conditions := response.Model["elements.0"].ChoiceFilter
		if len(conditions) != 1 || conditions[0].Field != metadata.FormChoiceRootField || conditions[0].Value == nil || *conditions[0].Value != (value == "true") {
			t.Fatalf("is_root=%s lost on HTTP round trip: %+v", value, conditions)
		}
		if !strings.Contains(response.YAML, "value: "+value) {
			t.Fatalf("typed value missing from YAML: %s", response.YAML)
		}
	}
	for _, condition := range []string{
		`[{"field":"is_root","op":"eq","from":"Объект.Город"}]`,
		`[{"field":"is_root","op":"eq","ref":"d4ba641c-70ae-4632-bf99-035f4caaa0af"}]`,
		`[{"field":"is_root","op":"in_hierarchy","value":true}]`,
	} {
		if response := post(condition); response.OK {
			t.Fatalf("invalid is_root saved: %s", condition)
		}
	}
}
