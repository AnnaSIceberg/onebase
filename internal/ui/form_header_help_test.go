package ui

import (
	"fmt"
	"net/http"
	"net/http/httptest"
	"os"
	"path/filepath"
	"strings"
	"testing"

	"github.com/go-chi/chi/v5"
	"github.com/ivantit66/onebase/internal/dsl/interpreter"
	"github.com/ivantit66/onebase/internal/metadata"
	"github.com/ivantit66/onebase/internal/userhelp"
)

// Exercise the public form routes with a real help binding. Popup forms keep
// help in their title and only need a separate command row for header buttons.
func TestManagedHeaderContextHelpPublicRoutes(t *testing.T) {
	for _, popup := range []bool{false, true} {
		for _, saved := range []bool{false, true} {
			// The edit endpoint renders a regular form even with _popup=1.
			if popup && saved {
				continue
			}
			for _, help := range []bool{false, true} {
				for _, header := range []bool{false, true} {
					t.Run(fmt.Sprintf("popup=%t/saved=%t/help=%t/header=%t", popup, saved, help, header), func(t *testing.T) {
						button := &metadata.FormElement{Kind: metadata.FormElementButton, Name: "HeaderAction", Placement: metadata.FormPlacementHeader}
						ent := layoutTestEntity(button)
						if !header {
							ent.Forms[0].Elements = nil
						}
						fixture, _ := newSubmitTestServer(t, []*metadata.Entity{ent})
						srv := New(fixture.reg, fixture.store, interpreter.New(), nil, Config{}, nil)
						defer srv.BeginShutdown()
						if help {
							dir := t.TempDir()
							if err := userhelp.Scaffold(dir); err != nil {
								t.Fatal(err)
							}
							index := fmt.Sprintf("title: Help\nchapters:\n  - id: form\n    title: Form\n    file: introduction.md\n    objects: [catalog/%s]\n", ent.Name)
							if err := os.WriteFile(filepath.Join(dir, "docs/help/index.yaml"), []byte(index), 0o644); err != nil {
								t.Fatal(err)
							}
							srv.reg.LoadHelp(userhelp.Load(dir))
						}
						path := "/ui/catalog/" + ent.Name + "/new"
						if saved {
							path = "/ui/catalog/" + ent.Name + "/" + insertContragent(t, srv, ent, "Example").String()
						}
						if popup {
							path += "?_popup=1"
						}
						router := chi.NewRouter()
						srv.Mount(router)
						rec := httptest.NewRecorder()
						router.ServeHTTP(rec, httptest.NewRequest(http.MethodGet, path, nil))
						if rec.Code != http.StatusOK {
							t.Fatalf("GET %s: %d %s", path, rec.Code, rec.Body.String())
						}
						body := rec.Body.String()
						wantHelp := 0
						if help {
							wantHelp = 1
						}
						if got := strings.Count(body, "data-ob-context-help"); got != wantHelp {
							t.Errorf("context help count=%d, want %d", got, wantHelp)
						}
						if help && !strings.Contains(body, `href="/ui/help/chapter/form"`) {
							t.Error("missing bound help link")
						}
						wantButton := 0
						if header {
							wantButton = 1
						}
						if got := strings.Count(body, `data-ob-el="HeaderAction"`); got != wantButton {
							t.Errorf("header button count=%d, want %d", got, wantButton)
						}
						if got := strings.Contains(body, `class="ob-popup-commands"`); got != (popup && header) {
							t.Errorf("popup command row present=%t, want %t", got, popup && header)
						}
					})
				}
			}
		}
	}
}
