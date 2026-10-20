package ui

import (
	"net/http"
	"net/http/httptest"
	"os"
	"path/filepath"
	"strings"
	"testing"

	"github.com/go-chi/chi/v5"
	"github.com/ivantit66/onebase/internal/dsl/interpreter"
	"github.com/ivantit66/onebase/internal/metadata"
	"github.com/ivantit66/onebase/internal/runtime"
	"github.com/ivantit66/onebase/internal/userhelp"
)

func TestHelpPublicRoutesAndContext(t *testing.T) {
	s := newServerForFormMode(t)
	s = New(s.reg, s.store, interpreter.New(), nil, Config{}, nil)
	defer s.BeginShutdown()
	dir := t.TempDir()
	if err := userhelp.Scaffold(dir); err != nil {
		t.Fatal(err)
	}
	writeHelp := func(name, data string) {
		t.Helper()
		root, err := os.OpenRoot(filepath.Join(dir, "docs/help"))
		if err != nil {
			t.Fatal(err)
		}
		defer func() { _ = root.Close() }()
		if err := root.WriteFile(name, []byte(data), 0o644); err != nil {
			t.Fatal(err)
		}
	}
	writeHelp("index.yaml", `title: Руководство
chapters:
  - id: calls
    title: Звонки
    file: introduction.md
    objects: [document/Call]
`)
	writeHelp("introduction.md", "# Звонки\n\nСоздайте заявку.\n\n![Пример](images/call.png)\n")
	writeHelp("images/call.png", "image-bytes")
	b := userhelp.Load(dir)
	s.reg.LoadHelp(b)
	r := chi.NewRouter()
	s.Mount(r)
	for _, tc := range []struct {
		path, want string
		status     int
	}{
		{"/ui/help", "Звонки", 200},
		{"/ui/help?q=СОЗДАЙТЕ", "Звонки", 200},
		{"/ui/help?q=absent", "Ничего не найдено", 200},
		{"/ui/help/chapter/calls", `src="/ui/help/assets/images/call.png"`, 200},
		{"/ui/help/assets/images/call.png", "image-bytes", 200},
		{"/ui/help/assets/index.yaml", "", 404},
		{"/ui/help/chapter/missing", "", 404},
	} {
		rec := httptest.NewRecorder()
		r.ServeHTTP(rec, httptest.NewRequest(http.MethodGet, tc.path, nil))
		if rec.Code != tc.status || (tc.want != "" && !strings.Contains(rec.Body.String(), tc.want)) {
			t.Fatalf("GET %s: %d %s", tc.path, rec.Code, rec.Body.String())
		}
	}
	// Exercise the actual entity form endpoint, not only context helpers.
	ent := &metadata.Entity{Name: "Call", Kind: metadata.KindDocument, Fields: []metadata.Field{{Name: "Номер", Type: metadata.FieldTypeString}}}
	s.reg.Load(runtime.LoadOptions{Entities: []*metadata.Entity{ent}})
	if err := s.store.Migrate(t.Context(), []*metadata.Entity{ent}); err != nil {
		t.Fatal(err)
	}
	rec := httptest.NewRecorder()
	r.ServeHTTP(rec, httptest.NewRequest("GET", "/ui/document/Call/new", nil))
	if rec.Code != 200 || !strings.Contains(rec.Body.String(), `href="/ui/help/chapter/calls"`) {
		t.Fatalf("form missing context help: %d %s", rec.Code, rec.Body.String())
	}
	// Publish a new registry generation, as --watch does. Removed help must vanish.
	next := runtime.NewRegistry()
	s.reg.ReplaceProjectFrom(next)
	rec = httptest.NewRecorder()
	r.ServeHTTP(rec, httptest.NewRequest("GET", "/ui/help", nil))
	if rec.Code != 404 {
		t.Fatalf("stale documentation survived reload: %d", rec.Code)
	}
}
