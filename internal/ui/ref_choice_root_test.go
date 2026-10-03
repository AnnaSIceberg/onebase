package ui

import (
	"net/http"
	"net/http/httptest"
	"net/url"
	"strings"
	"testing"

	"github.com/go-chi/chi/v5"
	"github.com/google/uuid"
	"github.com/ivantit66/onebase/internal/auth"
	"github.com/ivantit66/onebase/internal/metadata"
)

// The public picker, not an internal predicate helper, must use is_root for
// items, total and selected_allowed under the operator's row access policy.
func TestRefOptionsIsRoot(t *testing.T) {
	f := newParentChoiceFixture(t)
	element := f.owner.Forms[0].Elements[1]
	fetch := func(selected uuid.UUID) choiceHTTPResponse {
		t.Helper()
		query := url.Values{
			"form_entity": {f.owner.Name}, "form": {"ФормаОбъекта"},
			"element": {element.ID}, "sources": {"{}"}, "limit": {"100"},
		}
		if selected != uuid.Nil {
			query.Set("selected_id", selected.String())
		}
		router := chi.NewRouter()
		f.server.Mount(router)
		request := httptest.NewRequest(http.MethodGet, "/ui/_ref-options/"+url.PathEscape(f.target.Name)+"?"+query.Encode(), nil)
		request = request.WithContext(auth.ContextWithUser(request.Context(), f.user))
		recorder := httptest.NewRecorder()
		router.ServeHTTP(recorder, request)
		if recorder.Code != http.StatusOK {
			t.Fatalf("status=%d body=%s", recorder.Code, recorder.Body.String())
		}
		return decodeChoiceHTTP(t, recorder)
	}

	element.ChoiceFilter = []metadata.FormChoiceCondition{
		{Field: "is_root", Op: metadata.FormChoiceOpEqual, Value: boolPtr(true)},
		{Field: "is_folder", Op: metadata.FormChoiceOpEqual, Value: boolPtr(true)},
	}
	root := fetch(f.tech)
	if labels := strings.Join(parentChoiceLabels(root), ","); labels != "Прочее,Техника" || root.Total != 2 {
		t.Fatalf("root folders: total=%d items=%s", root.Total, labels)
	}
	if root.SelectedAllowed == nil || !*root.SelectedAllowed {
		t.Fatalf("root selection rejected: %+v", root.SelectedAllowed)
	}
	if nested := fetch(f.kitchen); nested.SelectedAllowed == nil || *nested.SelectedAllowed {
		t.Fatalf("nested folder allowed as root: %+v", nested.SelectedAllowed)
	}

	element.ChoiceFilter = []metadata.FormChoiceCondition{
		{Field: "is_root", Op: metadata.FormChoiceOpEqual, Value: boolPtr(false)},
		{Field: "is_folder", Op: metadata.FormChoiceOpEqual, Value: boolPtr(false)},
	}
	nonRoot := fetch(f.kettle)
	if labels := strings.Join(parentChoiceLabels(nonRoot), ","); labels != "лампа,утюг,чайник" || nonRoot.Total != 3 {
		t.Fatalf("non-root items/RLS: total=%d items=%s", nonRoot.Total, labels)
	}
	if nonRoot.SelectedAllowed == nil || !*nonRoot.SelectedAllowed {
		t.Fatalf("nested selection rejected: %+v", nonRoot.SelectedAllowed)
	}
	if closed := fetch(f.dryer); closed.SelectedAllowed == nil || *closed.SelectedAllowed {
		t.Fatalf("RLS-protected record allowed: %+v", closed.SelectedAllowed)
	}
}
