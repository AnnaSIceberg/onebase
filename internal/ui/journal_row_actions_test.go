package ui

import (
	"net/http"
	"net/http/httptest"
	"os/exec"
	"strings"
	"testing"
	"time"

	"github.com/go-chi/chi/v5"
	"github.com/google/uuid"
	"github.com/ivantit66/onebase/internal/auth"
	"github.com/ivantit66/onebase/internal/metadata"
	"golang.org/x/net/html"
)

func TestJournalRowActionsRespectScopedCompactControlsHTTP(t *testing.T) {
	for _, tc := range []struct {
		name                   string
		roles                  []string
		enhanced, hide, hidden bool
	}{
		{"otk", []string{"ОТК"}, true, true, true},
		{"flag-off", []string{"ОТК"}, true, false, false},
		{"enhancements-off", []string{"ОТК"}, false, true, false},
		{"operator-kc", []string{"ОператорКЦ"}, true, true, false},
		{"operator-messenger", []string{"ОператорМессенджера"}, true, true, false},
		{"mixed-operator", []string{"ОТК", "ОператорКЦ"}, true, true, false},
	} {
		t.Run(tc.name, func(t *testing.T) {
			doc := &metadata.Entity{Name: "FineProbe", Kind: metadata.KindDocument, Fields: []metadata.Field{{Name: "Дата", Type: metadata.FieldTypeDate}, {Name: "Номер", Type: metadata.FieldTypeString}}}
			journal := &metadata.Journal{Name: "FineJournal", Documents: []string{doc.Name}, Columns: []metadata.JournalColumn{{Field: "Дата"}, {Field: "Номер"}}}
			s, ctx := newSubmitTestServer(t, []*metadata.Entity{doc})
			t.Cleanup(s.Close)
			s.reg.LoadJournals([]*metadata.Journal{journal})
			s.cfg.UIEnhancements, s.cfg.HideListRowActions = tc.enhanced, tc.hide
			s.cfg.UIRoles = []string{"ОТК"}
			s.cfg.UIExceptRoles = []string{"ОператорКЦ", "ОператорМессенджера"}
			id := uuid.New()
			if err := s.store.Upsert(ctx, doc.Name, id, map[string]any{"Дата": time.Date(2026, 11, 17, 9, 0, 0, 0, time.UTC), "Номер": "FINE-001"}, doc); err != nil {
				t.Fatal(err)
			}
			actor := &auth.User{ID: uuid.NewString(), Login: tc.name}
			for _, name := range tc.roles {
				actor.Roles = append(actor.Roles, &auth.Role{Name: name, Permissions: auth.Permission{Documents: map[string][]string{doc.Name: {"read"}}}})
			}
			router := chi.NewRouter()
			s.Mount(router)
			r := httptest.NewRequest(http.MethodGet, "/ui/journal/"+journal.Name, nil)
			r = r.WithContext(auth.ContextWithUser(r.Context(), actor))
			response := httptest.NewRecorder()
			router.ServeHTTP(response, r)
			if response.Code != http.StatusOK {
				t.Fatalf("journal HTTP %d: %s", response.Code, response.Body.String())
			}
			page, err := html.Parse(strings.NewReader(response.Body.String()))
			if err != nil {
				t.Fatal(err)
			}
			row := semanticFind(page, "data-ob-list-row", "")
			if row == nil || semanticAttr(row, "data-open-url") != "/ui/document/fineprobe/"+id.String() || semanticAttr(row, "tabindex") != "-1" {
				t.Fatal("journal lost native row navigation")
			}
			table := row.Parent.Parent
			var headers, cells, buttons int
			var walk func(*html.Node)
			walk = func(node *html.Node) {
				if node.Type == html.ElementNode && node.Data == "th" {
					headers++
				}
				for child := node.FirstChild; child != nil; child = child.NextSibling {
					walk(child)
				}
			}
			walk(table)
			for cell := row.FirstChild; cell != nil; cell = cell.NextSibling {
				if cell.Type != html.ElementNode || cell.Data != "td" {
					continue
				}
				cells++
				for child := cell.FirstChild; child != nil; child = child.NextSibling {
					if child.Type == html.ElementNode && child.Data == "a" {
						buttons++
					}
				}
			}
			want := 4
			if tc.hidden {
				want = 3
			}
			if cells != want || headers != want || (buttons == 0) != tc.hidden || !strings.Contains(response.Body.String(), "FINE-001") {
				t.Fatalf("hidden=%t: headers=%d cells=%d buttons=%d", tc.hidden, headers, cells, buttons)
			}
			if s.cfg.HideListRowActions != tc.hide {
				t.Fatal("request mutated shared presentation config")
			}
		})
	}
}

func TestJournalRowsKeepNativeOpeningInNode(t *testing.T) {
	node, err := exec.LookPath("node")
	if err != nil {
		t.Fatal("node is required for this affected navigation test")
	}
	cmd := exec.Command(node, "--test", "static/journal_row_open_behavior_test.js") //nolint:gosec // Fixed local test, same Node used by UI tests.
	hideAudioTestProcess(cmd)
	if output, err := cmd.CombinedOutput(); err != nil {
		t.Fatalf("journal row opening: %v\n%s", err, output)
	}
}
