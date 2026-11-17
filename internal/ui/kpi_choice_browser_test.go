package ui

import (
	"context"
	"encoding/json"
	"fmt"
	"net/http"
	"net/http/httptest"
	"path/filepath"
	"testing"
	"time"

	"github.com/go-chi/chi/v5"
	"github.com/ivantit66/onebase/internal/auth"
	"github.com/ivantit66/onebase/internal/storage"
)

func TestKPIManagedChoiceBrowser(t *testing.T) {
	browserPath := managedLayoutTestBrowser(t)
	db, err := storage.ConnectSQLite(t.Context(), filepath.Join(t.TempDir(), "kpi.db"))
	if err != nil {
		t.Fatal(err)
	}
	t.Cleanup(func() { _ = db.Close() })
	f := newKPIChoiceFixture(t, db)
	router := chi.NewRouter()
	router.Use(func(next http.Handler) http.Handler {
		return http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
			next.ServeHTTP(w, r.WithContext(auth.ContextWithUser(r.Context(), f.user)))
		})
	})
	f.s.MountStatic(router)
	f.s.Mount(router)
	server := httptest.NewServer(router)
	t.Cleanup(func() { f.s.CloseEventStreams(); server.Close() })
	b := newMobileLayoutBrowser(t, browserPath)
	ctx, cancel := context.WithTimeout(t.Context(), 60*time.Second)
	defer cancel()
	var page struct {
		ID string `json:"targetId"`
	}
	if err := b.call(ctx, "", "Target.createTarget", map[string]any{"url": server.URL + "/ui/processor/" + f.proc.Name}, &page); err != nil {
		t.Fatal(err)
	}
	var session struct {
		ID string `json:"sessionId"`
	}
	if err := b.call(ctx, "", "Target.attachToTarget", map[string]any{"targetId": page.ID, "flatten": true}, &session); err != nil {
		t.Fatal(err)
	}
	eval := func(expr string) string {
		t.Helper()
		var out struct {
			Result struct {
				Value json.RawMessage `json:"value"`
			} `json:"result"`
			Exception json.RawMessage `json:"exceptionDetails"`
		}
		if err := b.call(ctx, session.ID, "Runtime.evaluate", map[string]any{"expression": expr, "returnByValue": true, "awaitPromise": true}, &out); err != nil {
			t.Fatal(err)
		}
		if len(out.Exception) != 0 {
			t.Fatalf("browser script: %s", out.Exception)
		}
		return string(out.Result.Value)
	}
	wait := func(expr string) {
		t.Helper()
		tick := time.NewTicker(20 * time.Millisecond)
		defer tick.Stop()
		for eval("Boolean("+expr+")") != "true" {
			select {
			case <-ctx.Done():
				t.Fatalf("browser timed out: %s", expr)
			case <-tick.C:
			}
		}
	}
	assert := func(expr string) {
		t.Helper()
		if eval("Boolean("+expr+")") != "true" {
			t.Fatalf("browser assertion: %s", expr)
		}
	}
	wait(`document.querySelector('select[name="Direction"]') && window.obRefreshChoiceFilters`)
	eval(`window.branch=document.querySelector('select[name="Branch"]');window.direction=document.querySelector('select[name="Direction"]')`)
	assert(`branch && branch.value==='' && direction.options.length===4 && direction.options[1].textContent==='Z first (Branch A)' && direction.options[2].textContent==='A second (Branch A)'`)
	change := func(id string) {
		t.Helper()
		eval(fmt.Sprintf(`(async()=>{branch.value=%q;branch.dispatchEvent(new Event('change',{bubbles:true}));await window.obRefreshChoiceFilters();return true})()`, id))
		assert(`!direction.hasAttribute('data-ob-choice-error')`)
	}
	change(f.b1.String())
	assert(`direction.options.length===3 && direction.options[1].textContent==='Z first (Branch A)'`)
	eval(fmt.Sprintf(`direction.value=%q;direction.dispatchEvent(new Event('change',{bubbles:true}))`, f.d2.String()))
	change(f.b2.String())
	assert(`direction.value==='' && direction.options.length===2 && direction.options[1].textContent==='Other (Branch B)'`)
	eval(`direction.parentElement.querySelector('[data-ob-ref-picker]').click()`)
	wait(`document.querySelectorAll('#_rp-list [data-id]').length===1`)
	assert(`document.querySelector('#_rp-list [data-id]').getAttribute('data-label')==='Other (Branch B)'`)
	eval(`document.querySelector('#_rp-list [data-id]').click()`)
	assert(fmt.Sprintf(`direction.value===%q && direction.selectedOptions[0].textContent==='Other (Branch B)'`, f.d3.String()))
	change("")
	assert(fmt.Sprintf(`direction.options.length===4 && direction.value===%q && document.querySelector('select[name="Branch"]')===branch`, f.d3.String()))
}
