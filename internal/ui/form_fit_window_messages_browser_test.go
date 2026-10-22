package ui

import (
	"context"
	"encoding/json"
	"net/http"
	"net/http/httptest"
	"os"
	"path/filepath"
	"strings"
	"sync/atomic"
	"testing"
	"time"

	"github.com/go-chi/chi/v5"

	"github.com/ivantit66/onebase/internal/auth"
	"github.com/ivantit66/onebase/internal/dsl/interpreter"
	"github.com/ivantit66/onebase/internal/dsl/loader"
	"github.com/ivantit66/onebase/internal/metadata"
	orun "github.com/ivantit66/onebase/internal/runtime"
	"github.com/ivantit66/onebase/internal/storage"
)

// The message bar lives outside main. Showing it changes main's available
// height without resizing the window or any of main's observed child blocks.
// Exercise the rendered card, production scripts and message reload/clear UI.
func TestFitWindowMessagePanel_Browser(t *testing.T) {
	browser := newMobileLayoutBrowser(t, managedLayoutTestBrowser(t))
	for _, kind := range []string{"СтраницыФормы", "ГруппаФормы"} {
		t.Run(kind, func(t *testing.T) {
			db, err := storage.ConnectSQLite(t.Context(), filepath.Join(t.TempDir(), "fit.db"))
			if err != nil {
				t.Fatal(err)
			}
			t.Cleanup(func() { db.Close() })
			src := "schema: onebase.form/v1\nform:\n  name: ФормаДокумента\n  kind: object\n  entity: Заявка\nelements:\n" +
				"  - kind: " + kind + "\n    name: Область\n    fit_window: true\n    children:\n"
			if kind == "СтраницыФормы" {
				src += "      - kind: Страница\n        name: Заказчик\n        children:\n" +
					"          - kind: ПолеВвода\n            name: ПолеКлиент\n            data_path: Объект.Клиент\n"
			} else {
				src += "      - kind: ПолеВвода\n        name: ПолеКлиент\n        data_path: Объект.Клиент\n"
			}
			path := filepath.Join(t.TempDir(), "ФормаДокумента.form.yaml")
			if err := os.WriteFile(path, []byte(src), 0o600); err != nil {
				t.Fatal(err)
			}
			fm, err := loader.NewManagedFormLoader().LoadFormFile(path, "Заявка")
			if err != nil {
				t.Fatal(err)
			}
			ent := &metadata.Entity{Name: "Заявка", Kind: metadata.KindDocument, Posting: true,
				Fields: []metadata.Field{{Name: "Клиент", Type: metadata.FieldTypeString}}, Forms: []*metadata.FormModule{fm}}
			if err := db.Migrate(t.Context(), []*metadata.Entity{ent}); err != nil {
				t.Fatal(err)
			}
			reg := orun.NewRegistry()
			reg.Load(orun.LoadOptions{Entities: []*metadata.Entity{ent}})
			srv := New(reg, db, interpreter.New(), auth.NewRepo(db), Config{}, nil)
			t.Cleanup(srv.BeginShutdown)
			router := chi.NewRouter()
			srv.MountStatic(router)
			srv.Mount(router)
			rec := httptest.NewRecorder()
			router.ServeHTTP(rec, httptest.NewRequest(http.MethodGet, "/ui/document/заявка/new", nil))
			if rec.Code != http.StatusOK {
				t.Fatalf("form: %d", rec.Code)
			}
			page := rec.Body.String()
			end := strings.LastIndex(page, "</body>")
			if end < 0 {
				t.Fatal("form has no body")
			}
			page = page[:end] + fitWindowMessagesProbe + page[end:]
			var messages atomic.Bool
			server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
				switch {
				case r.URL.Path == "/":
					w.Header().Set("Content-Type", "text/html; charset=utf-8")
					_, _ = w.Write([]byte(page))
				case r.URL.Path == "/test/show-message":
					messages.Store(true)
				case r.URL.Path == "/ui/messages/clear":
					messages.Store(false)
				case r.URL.Path == "/ui/messages":
					w.Header().Set("Content-Type", "application/json")
					if messages.Load() {
						_, _ = w.Write([]byte(`{"messages":[{"time":"2026-10-22T12:00:00Z","text":"Сообщение"}]}`))
					} else {
						_, _ = w.Write([]byte(`{"messages":[]}`))
					}
				case strings.HasPrefix(r.URL.Path, "/static/"), strings.HasPrefix(r.URL.Path, "/vendor/"):
					router.ServeHTTP(w, r)
				default:
					http.NotFound(w, r)
				}
			}))
			defer server.Close()
			ctx, cancel := context.WithTimeout(t.Context(), 30*time.Second)
			defer cancel()
			data, err := browser.measure(ctx, server.URL)
			if err != nil {
				t.Fatalf("browser: %v\n%s", err, browser.diagnostics())
			}
			var got struct {
				Phases []struct {
					Name                   string
					Scroll, Client, Height int
				}
				Resizes int
			}
			if err := json.Unmarshal(data, &got); err != nil {
				t.Fatal(err)
			}
			if len(got.Phases) != 3 {
				t.Fatalf("missing phases: %s", data)
			}
			for _, phase := range got.Phases {
				if phase.Scroll > phase.Client+1 || phase.Height <= 120 {
					t.Errorf("%s: fit_window did not fill available height without scrolling: %+v", phase.Name, phase)
				}
			}
			before, shown, cleared := got.Phases[0], got.Phases[1], got.Phases[2]
			if before.Client-shown.Client != 32 || before.Height-shown.Height != 32 {
				t.Errorf("showing messages must give up 32px: %s", data)
			}
			if cleared.Client != before.Client || cleared.Height != before.Height {
				t.Errorf("clearing messages must restore available height: %s", data)
			}
			if got.Resizes != 0 {
				t.Errorf("unexpected window resize: %s", data)
			}
			t.Logf("message panel geometry: %s", data)
		})
	}
}

// Use the existing CDP helper's measurement channel; do not call layout
// functions from the probe, since that would conceal a missing observer.
const fitWindowMessagesProbe = `<script>
window.addEventListener('load', async function () {
  var sleep = ms => new Promise(resolve => setTimeout(resolve, ms));
  var waitFor = async predicate => {
    var deadline = Date.now() + 5000;
    while (!predicate()) {
      if (Date.now() > deadline) throw new Error('message panel did not change visibility');
      await sleep(20);
    }
    await sleep(250); // let ResizeObserver and its deferred refit settle
  };
  var publish = (id, text) => {
    var pre = document.createElement('pre'); pre.id = id; pre.textContent = text;
    document.body.appendChild(pre);
  };
  try {
    await document.fonts.ready;
    await waitFor(() => document.getElementById('ob-msg-bar') && window.obReloadMessages);
    var resizes = 0;
    window.addEventListener('resize', () => resizes++);
    var phases = [];
    var measure = name => {
      var main = document.querySelector('main');
      var fitted = document.querySelector('.managed-tab-content') || document.querySelector('[data-ob-fit-group="1"]');
      phases.push({name:name, scroll:main.scrollHeight, client:main.clientHeight, height:fitted.offsetHeight});
    };
    measure('hidden');
    await fetch('/test/show-message', {method:'POST'});
    window.obReloadMessages();
    await waitFor(() => !document.getElementById('ob-msg-bar').classList.contains('hidden'));
    measure('shown');
    document.getElementById('ob-msg-clear').click();
    await waitFor(() => document.getElementById('ob-msg-bar').classList.contains('hidden'));
    measure('cleared');
    publish('list-mobile-measure', JSON.stringify({phases:phases, resizes:resizes}));
  } catch (error) { publish('list-mobile-error', String(error)); }
});
</script>`
