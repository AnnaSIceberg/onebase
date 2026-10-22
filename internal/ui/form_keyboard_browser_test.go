package ui

import (
	"bytes"
	"context"
	"encoding/json"
	"html"
	"net/http"
	"net/http/httptest"
	"net/url"
	"os/exec"
	"runtime"
	"strings"
	"testing"
	"time"

	"github.com/go-chi/chi/v5"
	"github.com/ivantit66/onebase/internal/metadata"
)

// Проверяем production-форму и настоящие ui.js/managed.js/SlickGrid в браузере:
// readonly-таблица не должна оставлять в Tab ни focus sinks, ни текущую tr.
func TestFormKeyboard_ReadonlyTablesBrowser(t *testing.T) {
	browser := managedLayoutTestBrowser(t)
	for _, tc := range []struct {
		name                   string
		noGrid, readOnly, skip bool
	}{
		{"slick readonly skip", false, true, true},
		{"DOM readonly skip", true, true, true},
		{"slick readonly default", false, true, false},
		{"DOM readonly default", true, true, false},
		{"slick editable skip", false, false, true},
		{"DOM editable skip", true, false, true},
	} {
		t.Run(tc.name, func(t *testing.T) {
			table := &metadata.FormElement{Kind: metadata.FormElementTablePart, Name: "History", DataPath: "Object.Lines", NoGrid: tc.noGrid}
			form := managedObjectForm(fieldEl("FirstField", "Object.First"), table, fieldEl("LastField", "Object.Last"))
			form.TabSkipReadonly = tc.skip
			form.Autofocus = "first_empty"
			ent := &metadata.Entity{
				Name: "KeyboardRecord", Kind: metadata.KindCatalog,
				Fields:     []metadata.Field{{Name: "First", Type: metadata.FieldTypeString}, {Name: "Last", Type: metadata.FieldTypeString}},
				TableParts: []metadata.TablePart{{Name: "Lines", Fields: []metadata.Field{{Name: "Text", Type: metadata.FieldTypeString}}}},
				Forms:      []*metadata.FormModule{form},
			}
			srv, _ := newSubmitTestServer(t, []*metadata.Entity{ent})
			written := httptest.NewRecorder()
			srv.submit(written, reqWithChi(http.MethodPost, "/ui/catalog/KeyboardRecord/new", url.Values{
				"tp_present.Lines": {"1"}, "tp.Lines.0.Text": {"history"},
			}, map[string]string{"entity": ent.Name}))
			if written.Code != http.StatusSeeOther {
				t.Fatalf("create: %d %s", written.Code, written.Body.String())
			}
			table.ReadOnly = tc.readOnly
			id := onlyStoredID(t, srv, ent).String()
			opened := httptest.NewRecorder()
			srv.formEdit(opened, reqWithChi(http.MethodGet, "/ui/catalog/KeyboardRecord/"+id, nil, map[string]string{"entity": ent.Name, "id": id}))
			if opened.Code != http.StatusOK {
				t.Fatalf("open: %d %s", opened.Code, opened.Body.String())
			}

			const probe = `<script>
window.addEventListener('load', async function () {
  const result = {};
  const settle = () => new Promise(resolve => setTimeout(resolve, 20));
  const table = document.querySelector('.ob-grid, .tp-table');
  const stops = () => Array.from(table.querySelectorAll('[tabindex]')).filter(el => el.tabIndex >= 0).length;
  try {
    await settle();
    result.initial = stops();
    result.autofocus = document.activeElement.name;
    document.querySelector('[name="Last"]').focus();
    // Настоящий obFire и применение ответа: транспорт возвращает новые строки.
    await window.obFire('History', 'Refresh');
    await settle();
    result.afterEvent = stops();
    result.manualFocus = document.activeElement.name;
    if (table.matches('.tp-table')) {
      result.updated = table.tBodies[0].rows.length === 2;
      window.obDOMSetCurrentRow(table, table.tBodies[0].rows[1], false);
    } else {
      const grid = window._obGrids.Lines;
      result.updated = grid.dataView.getLength() === 2;
      // Новый focus sink, как при повторном создании внутренних узлов грида.
      const sink = table.querySelector('[tabindex]').cloneNode(false);
      sink.removeAttribute('data-ob-tab-orig');
      sink.setAttribute('tabindex', '0');
      table.appendChild(sink);
    }
    await settle();
    result.afterRecreate = stops();
  } catch (e) { result.error = String(e.stack || e); }
  const out = document.createElement('pre');
  out.id = 'keyboard-result';
  out.textContent = JSON.stringify(result);
  document.body.appendChild(out);
});
</script>`
			page := strings.Replace(opened.Body.String(), "</body>", probe+"</body>", 1)
			router := chi.NewRouter()
			mountStatic(router)
			router.Post("/ui/catalog/KeyboardRecord/form-event", func(w http.ResponseWriter, r *http.Request) {
				w.Header().Set("Content-Type", "application/json")
				_, _ = w.Write([]byte(`{"ok":true,"tableparts":{"Lines":[{"Text":"new history"},{"Text":"second row"}]}}`))
			})
			router.Get("/", func(w http.ResponseWriter, r *http.Request) {
				w.Header().Set("Content-Type", "text/html; charset=utf-8")
				_, _ = w.Write([]byte(page))
			})
			server := httptest.NewServer(router)
			t.Cleanup(server.Close)
			args := []string{"--headless=new", "--disable-gpu", "--disable-background-networking", "--disable-component-update", "--disable-extensions", "--disable-sync", "--no-first-run", "--no-default-browser-check", "--user-data-dir=" + t.TempDir(), "--timeout=10000", "--virtual-time-budget=1500", "--dump-dom", server.URL}
			if runtime.GOOS == "linux" {
				args = append([]string{"--no-sandbox"}, args...)
			}
			ctx, cancel := context.WithTimeout(t.Context(), 30*time.Second)
			defer cancel()
			cmd := exec.CommandContext(ctx, browser, args...) //nolint:gosec // closed browser allow-list in managedLayoutTestBrowser
			var stderr bytes.Buffer
			cmd.Stderr = &stderr
			dumped, err := cmd.Output()
			if err != nil {
				t.Fatalf("browser: %v\n%s", err, stderr.String())
			}
			const marker = `<pre id="keyboard-result">`
			_, data, found := strings.Cut(string(dumped), marker)
			if !found {
				t.Fatalf("browser probe missing: %s", stderr.String())
			}
			data, _, _ = strings.Cut(data, "</pre>")
			var got struct {
				Initial, AfterEvent, AfterRecreate int
				Autofocus, ManualFocus, Error      string
				Updated                            bool
			}
			if err := json.Unmarshal([]byte(html.UnescapeString(data)), &got); err != nil {
				t.Fatal(err)
			}
			if got.Error != "" || !got.Updated || got.Autofocus != "First" || got.ManualFocus != "Last" {
				t.Fatalf("form did not complete browser scenario: %s", data)
			}
			for phase, count := range map[string]int{"initial": got.Initial, "event": got.AfterEvent, "recreate": got.AfterRecreate} {
				if tc.skip && tc.readOnly {
					if count != 0 {
						t.Errorf("%s: readonly table has %d Tab stops", phase, count)
					}
				} else if count == 0 {
					t.Errorf("%s: table lost its Tab stops", phase)
				}
			}
		})
	}
}
