package ui

import (
	"context"
	"net/http"
	"net/http/httptest"
	"net/url"
	"os"
	"path/filepath"
	"strings"
	"testing"

	"github.com/ivantit66/onebase/internal/dbtest"
	"github.com/ivantit66/onebase/internal/metadata"
	"github.com/ivantit66/onebase/internal/storage"
	"golang.org/x/net/html"
	"gopkg.in/yaml.v3"
)

// Follow the script URL emitted by the real page, through MountStatic+Mount.
// A successful template render alone does not prove the controller is served.
func TestRecursiveNavigationHTTPAsset(t *testing.T) {
	f := newRecursiveHTTPFixture(t)
	page := f.request(t, http.MethodGet, "/ui/?subsystem=School", "alice", nil)
	if page.Code != http.StatusOK {
		t.Fatalf("page: %d", page.Code)
	}
	doc, err := html.Parse(strings.NewReader(page.Body.String()))
	if err != nil {
		t.Fatal(err)
	}
	script := semanticFind(doc, "src", "/static/navigation-tree.js")
	if script == nil || script.Data != "script" {
		t.Fatal("recursive controller missing from page")
	}
	path := semanticAttr(script, "src")
	asset := f.request(t, http.MethodGet, path, "", nil)
	if asset.Code != http.StatusOK || asset.Body.String() != string(navigationTreeJS) || !strings.Contains(asset.Header().Get("Content-Type"), "application/javascript") {
		t.Fatalf("script response: %d %s", asset.Code, asset.Header().Get("Content-Type"))
	}
	etag := asset.Header().Get("ETag")
	if etag == "" || asset.Header().Get("Cache-Control") != "no-cache" {
		t.Fatal("controller must revalidate after engine updates")
	}
	req := httptest.NewRequest(http.MethodGet, path, nil)
	req.Header.Set("If-None-Match", etag)
	rec := httptest.NewRecorder()
	f.http.ServeHTTP(rec, req)
	if rec.Code != http.StatusNotModified || rec.Body.Len() != 0 {
		t.Fatalf("revalidation: %d", rec.Code)
	}
}

func recursiveProjectDir(t *testing.T) string {
	t.Helper()
	dir := t.TempDir()
	if err := os.CopyFS(dir, os.DirFS("testdata/navigation-school")); err != nil {
		t.Fatal(err)
	}
	path := filepath.Join(dir, "subsystems", personalSchool+".yaml")
	raw, err := os.ReadFile(path)
	if err != nil {
		t.Fatal(err)
	}
	var sub metadata.Subsystem
	if err := yaml.Unmarshal(raw, &sub); err != nil {
		t.Fatal(err)
	}
	menu := strings.NewReplacer("catalog:Classes", "catalog:Классы", "catalog:Private", "catalog:ClosedSentinel", "document:Orders", "document:ПриказОНачалеУчебногоГода", "processor:ClassJournal", "processor:КлассныйЖурнал").Replace(recursiveSchoolYAML)
	sub.Menu = new(metadata.Menu)
	if err := yaml.Unmarshal([]byte(menu), sub.Menu); err != nil {
		t.Fatal(err)
	}
	raw, err = yaml.Marshal(sub)
	if err != nil {
		t.Fatal(err)
	}
	if err := os.WriteFile(path, raw, 0600); err != nil {
		t.Fatal(err)
	}
	return dir
}

func saveRecursivePersonalFolder(t *testing.T, f navigationHTTPFixture, login string) {
	t.Helper()
	state := f.personalEditor(t, login, personalSchool)
	state.Desired.Sections[0].Nodes[1].Nodes[0].Title = login + " nested folder"
	state.Renamed = []string{"cfg:held"}
	personalSave(t, f, login, state)
}

func assertRecursivePersistedHTTP(t *testing.T, f navigationHTTPFixture) {
	t.Helper()
	for _, login := range []string{"alice", "bob"} {
		page := f.request(t, http.MethodGet, "/ui/?subsystem="+url.QueryEscape(personalSchool), login, nil)
		if page.Code != http.StatusOK {
			t.Fatalf("page: %d", page.Code)
		}
		doc, err := html.Parse(strings.NewReader(page.Body.String()))
		if err != nil {
			t.Fatal(err)
		}
		nav := semanticFind(doc, "id", "ob-nav")
		if nav == nil {
			t.Fatal("sidebar missing")
		}
		held := semanticFind(nav, "data-nav-id", "cfg:held")
		if held == nil || !strings.Contains(semanticHeading(held), login+" nested folder") || semanticFind(held, "data-nav-id", "cfg:full") == nil {
			t.Fatal("nested personal folder/descendant lost", login)
		}
		for _, id := range []string{"cfg:empty", "cfg:forbidden", "cfg:chief", "cfg:chief-command"} {
			if semanticFind(nav, "data-nav-id", id) != nil {
				t.Fatal("restricted descendant exposed", login, id)
			}
		}
	}
	if rec := f.request(t, http.MethodGet, "/ui/catalog/ClosedSentinel?subsystem="+url.QueryEscape(personalSchool), "alice", nil); rec.Code != http.StatusForbidden {
		t.Fatalf("direct ACL: %d", rec.Code)
	}
}

func TestRecursiveNavigationHTTPPersistenceMatrix(t *testing.T) {
	dir := recursiveProjectDir(t)
	dbtest.ForEachDialect(t, func(t *testing.T, db *storage.DB) {
		f := schoolNavigationFixture(t, db, dir, true)
		for _, login := range []string{"alice", "bob"} {
			saveRecursivePersonalFolder(t, f, login)
		}
		assertRecursivePersistedHTTP(t, f)
		// Recreate the host and load configuration/settings from storage again.
		f.server.Close()
		assertRecursivePersistedHTTP(t, schoolNavigationFixture(t, db, dir, false))
	})
}

func TestRecursiveNavigationHTTPDatabaseRestart(t *testing.T) {
	dir := recursiveProjectDir(t)
	path := filepath.Join(t.TempDir(), "recursive-restart.sqlite")
	db, err := storage.ConnectSQLite(context.Background(), path)
	if err != nil {
		t.Fatal(err)
	}
	t.Cleanup(db.Close)
	f := schoolNavigationFixture(t, db, dir, true)
	for _, login := range []string{"alice", "bob"} {
		saveRecursivePersonalFolder(t, f, login)
	}
	f.server.Close()
	db.Close()
	reopened, err := storage.ConnectSQLite(context.Background(), path)
	if err != nil {
		t.Fatal(err)
	}
	t.Cleanup(reopened.Close)
	assertRecursivePersistedHTTP(t, schoolNavigationFixture(t, reopened, dir, false))
}
