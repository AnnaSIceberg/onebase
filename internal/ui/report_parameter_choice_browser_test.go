package ui

import (
	"context"
	"encoding/json"
	"fmt"
	"net/http"
	"net/http/httptest"
	"testing"
	"time"

	"github.com/go-chi/chi/v5"
	"github.com/google/uuid"
	"github.com/ivantit66/onebase/internal/auth"
	"github.com/ivantit66/onebase/internal/dsl/interpreter"
	"github.com/ivantit66/onebase/internal/metadata"
	reportpkg "github.com/ivantit66/onebase/internal/report"
	"github.com/ivantit66/onebase/internal/runtime"
)

// Use the mounted report and shipped assets: both the short list and the full
// picker must follow the current branch, not the list rendered on initial GET.
func TestReportParameterChoiceBrowserBranchRefresh(t *testing.T) {
	browserPath := managedLayoutTestBrowser(t)
	branches := &metadata.Entity{Name: "BrowserBranches", Kind: metadata.KindCatalog, Fields: []metadata.Field{{Name: "Name", Type: metadata.FieldTypeString}}}
	directions := &metadata.Entity{Name: "BrowserDirections", Kind: metadata.KindCatalog, OrderBy: []string{"Priority", "Name"}, Fields: []metadata.Field{
		{Name: "Name", Type: metadata.FieldTypeString}, {Name: "Priority", Type: metadata.FieldTypeNumber},
		{Name: "Branch", Type: "reference:BrowserBranches", RefEntity: branches.Name},
	}}
	rep := &reportpkg.Report{Name: "BrowserChoice", Query: "ВЫБРАТЬ Name ИЗ Справочник.BrowserDirections", Params: []reportpkg.Param{
		{Name: "Branch", Type: "reference:BrowserBranches"},
		{Name: "Direction", Type: "reference:BrowserDirections", ChoiceFilter: []reportpkg.ParamChoiceFilter{{Field: "Branch", Param: "Branch"}}, ChoiceLabelSuffix: "Branch"},
	}}
	fixture := newReportExportTestServer(t, rep)
	entities := []*metadata.Entity{branches, directions}
	if err := fixture.store.Migrate(t.Context(), entities); err != nil {
		t.Fatal(err)
	}
	b1, b2, d1, d2, d3 := uuid.New(), uuid.New(), uuid.New(), uuid.New(), uuid.New()
	for id, name := range map[uuid.UUID]string{b1: "Branch A", b2: "Branch B"} {
		if err := fixture.store.Upsert(t.Context(), branches.Name, id, map[string]any{"Name": name}, branches); err != nil {
			t.Fatal(err)
		}
	}
	for _, row := range []struct {
		id, branch uuid.UUID
		name       string
		priority   int
	}{{d1, b1, "Z first", 1}, {d2, b1, "A second", 2}, {d3, b2, "Other", 3}} {
		if err := fixture.store.Upsert(t.Context(), directions.Name, row.id, map[string]any{"Name": row.name, "Branch": row.branch, "Priority": row.priority}, directions); err != nil {
			t.Fatal(err)
		}
	}
	fixture.reg.Load(runtime.LoadOptions{Entities: entities, Reports: []*reportpkg.Report{rep}})
	s := New(fixture.reg, fixture.store, interpreter.New(), nil, Config{}, nil)
	t.Cleanup(s.Close)
	router := chi.NewRouter()
	router.Use(func(next http.Handler) http.Handler {
		return http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
			next.ServeHTTP(w, r.WithContext(auth.ContextWithOpenAccess(r.Context())))
		})
	})
	s.MountStatic(router)
	s.Mount(router)
	server := httptest.NewServer(router)
	t.Cleanup(func() { s.CloseEventStreams(); server.Close() })
	b := newMobileLayoutBrowser(t, browserPath)
	ctx, cancel := context.WithTimeout(t.Context(), 60*time.Second)
	defer cancel()
	var page struct {
		ID string `json:"targetId"`
	}
	if err := b.call(ctx, "", "Target.createTarget", map[string]any{"url": server.URL + "/ui/report/BrowserChoice?lang=ru"}, &page); err != nil {
		t.Fatal(err)
	}
	var session struct {
		ID string `json:"sessionId"`
	}
	if err := b.call(ctx, "", "Target.attachToTarget", map[string]any{"targetId": page.ID, "flatten": true}, &session); err != nil {
		t.Fatal(err)
	}
	eval := func(expression string) string {
		t.Helper()
		var out struct {
			Result struct {
				Value json.RawMessage `json:"value"`
			} `json:"result"`
			Exception json.RawMessage `json:"exceptionDetails"`
		}
		if err := b.call(ctx, session.ID, "Runtime.evaluate", map[string]any{"expression": expression, "returnByValue": true, "awaitPromise": true}, &out); err != nil {
			t.Fatal(err)
		}
		if len(out.Exception) != 0 {
			t.Fatalf("browser script failed: %s", out.Exception)
		}
		return string(out.Result.Value)
	}
	wait := func(expression string) {
		t.Helper()
		tick := time.NewTicker(20 * time.Millisecond)
		defer tick.Stop()
		for eval("Boolean("+expression+")") != "true" {
			select {
			case <-ctx.Done():
				t.Fatalf("browser condition timed out: %s", expression)
			case <-tick.C:
			}
		}
	}
	assert := func(expression string) {
		t.Helper()
		if got := eval("Boolean(" + expression + ")"); got != "true" {
			t.Fatalf("browser assertion: %s = %s", expression, got)
		}
	}
	wait(`document.querySelector('#rp-Direction') && window.obRefreshChoiceFilters`)
	eval(`window.branch=document.querySelector('#rp-Branch');window.direction=document.querySelector('#rp-Direction');window.allLabel=direction.options[0].textContent`)
	assert(`branch.value==='' && direction.value==='' && direction.options.length===4 && direction.options[1].textContent==='Z first (Branch A)' && direction.options[2].textContent==='A second (Branch A)'`)
	changeBranch := func(id string) {
		t.Helper()
		eval(fmt.Sprintf(`(async()=>{branch.value=%q;branch.dispatchEvent(new Event('change',{bubbles:true}));await window.obRefreshChoiceFilters();return true})()`, id))
		assert(`!direction.hasAttribute('data-ob-choice-error') && direction.options[0].textContent===allLabel`)
	}
	changeBranch(b1.String())
	assert(`direction.options.length===3 && direction.options[1].textContent==='Z first (Branch A)' && direction.options[2].textContent==='A second (Branch A)'`)
	eval(fmt.Sprintf(`direction.value=%q;direction.dispatchEvent(new Event('change',{bubbles:true}))`, d2.String()))
	changeBranch(b2.String())
	assert(`direction.value==='' && direction.options.length===2 && direction.options[1].textContent==='Other (Branch B)'`)
	// The actual picker button fetches the same scoped list and keeps its label
	// when the user selects a row back into the report parameter.
	eval(`document.querySelector('[data-ob-ref-picker="rp-Direction"]').click()`)
	wait(`document.querySelectorAll('#_rp-list [data-id]').length===1`)
	assert(`document.querySelector('#_rp-list [data-id]').getAttribute('data-label')==='Other (Branch B)'`)
	eval(`document.querySelector('#_rp-list [data-id]').click()`)
	assert(fmt.Sprintf(`direction.value===%q && direction.selectedOptions[0].textContent==='Other (Branch B)'`, d3.String()))
	changeBranch("")
	assert(fmt.Sprintf(`direction.value===%q && direction.options.length===4 && direction.options[0].textContent===allLabel`, d3.String()))
	// Clearing both parameters means all; no branch predicate is retained.
	eval(`direction.value='';direction.dispatchEvent(new Event('change',{bubbles:true}))`)
	assert(`branch.value==='' && direction.value==='' && direction.options[0].textContent===allLabel`)
}
