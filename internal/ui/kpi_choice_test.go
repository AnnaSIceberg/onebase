package ui

import (
	"encoding/json"
	"net/http"
	"net/http/httptest"
	"net/url"
	"strings"
	"testing"

	"github.com/go-chi/chi/v5"
	"github.com/google/uuid"
	"github.com/ivantit66/onebase/internal/auth"
	"github.com/ivantit66/onebase/internal/dbtest"
	"github.com/ivantit66/onebase/internal/dsl/interpreter"
	"github.com/ivantit66/onebase/internal/metadata"
	"github.com/ivantit66/onebase/internal/processor"
	"github.com/ivantit66/onebase/internal/runtime"
	"github.com/ivantit66/onebase/internal/storage"
	"golang.org/x/net/html"
	"gopkg.in/yaml.v3"
)

type kpiChoiceFixture struct {
	s                                *Server
	proc                             *processor.Processor
	branches, directions             *metadata.Entity
	field                            *metadata.FormElement
	user                             *auth.User
	b1, b2, hiddenBranch, d1, d2, d3 uuid.UUID
}

func newKPIChoiceFixture(t *testing.T, db *storage.DB) kpiChoiceFixture {
	t.Helper()
	f := kpiChoiceFixture{b1: uuid.New(), b2: uuid.New(), hiddenBranch: uuid.New(), d1: uuid.New(), d2: uuid.New(), d3: uuid.New()}
	f.branches = &metadata.Entity{Name: "KPIBranches", Kind: metadata.KindCatalog, Fields: []metadata.Field{
		{Name: "Name", Type: metadata.FieldTypeString}, {Name: "Audience", Type: metadata.FieldTypeString},
		{Name: "Phone", Type: metadata.FieldTypeString, PII: true},
	}}
	f.directions = &metadata.Entity{Name: "KPIDirections", Kind: metadata.KindCatalog, OrderBy: []string{"Priority", "Name"}, Fields: []metadata.Field{
		{Name: "Name", Type: metadata.FieldTypeString}, {Name: "Branch", Type: "reference:KPIBranches", RefEntity: f.branches.Name},
		{Name: "Priority", Type: metadata.FieldTypeNumber}, {Name: "Enabled", Type: metadata.FieldTypeBool}, {Name: "Audience", Type: metadata.FieldTypeString},
	}}
	entities := []*metadata.Entity{f.branches, f.directions}
	if err := db.Migrate(t.Context(), entities); err != nil {
		t.Fatal(err)
	}
	if err := db.EnsureSettingsSchema(t.Context()); err != nil {
		t.Fatal(err)
	}
	for _, row := range []struct {
		id             uuid.UUID
		name, audience string
	}{{f.b1, "Branch A", "anna"}, {f.b2, "Branch B", "anna"}, {f.hiddenBranch, "Hidden branch", "bob"}} {
		if err := db.Upsert(t.Context(), f.branches.Name, row.id, map[string]any{"Name": row.name, "Audience": row.audience, "Phone": "PRIVATE_PHONE"}, f.branches); err != nil {
			t.Fatal(err)
		}
	}
	for i, row := range []struct {
		id, branch     uuid.UUID
		name, audience string
		enabled        bool
	}{
		{f.d1, f.b1, "Z first", "anna", true}, {f.d2, f.b1, "A second", "anna", true}, {f.d3, f.b2, "Other", "anna", true},
		{uuid.New(), f.b1, "Disabled", "anna", false}, {uuid.New(), f.b1, "Hidden direction", "bob", true},
	} {
		if err := db.Upsert(t.Context(), f.directions.Name, row.id, map[string]any{"Name": row.name, "Branch": row.branch, "Priority": i + 1, "Enabled": row.enabled, "Audience": row.audience}, f.directions); err != nil {
			t.Fatal(err)
		}
	}
	var form metadata.FormModule
	if err := yaml.Unmarshal([]byte(`name: Form
kind: object
layout_kind: managed
elements:
  - {id: branch, name: BranchField, kind: ПолеВвода, data_path: Объект.Branch}
  - id: direction
    name: DirectionField
    kind: ПолеВвода
    data_path: Объект.Direction
    choice_filter:
      - {field: Branch, op: eq, from: Объект.Branch}
      - {field: Enabled, op: eq, value: true}
    kpi_choice: {all_when_empty: Объект.Branch, label_suffix: Branch}
`), &form); err != nil {
		t.Fatal(err)
	}
	f.proc = &processor.Processor{Name: "KPIChoice", Kind: "kpi", Params: []processor.Param{{Name: "Branch", Type: "reference:KPIBranches"}, {Name: "Direction", Type: "reference:KPIDirections"}}, Forms: []*metadata.FormModule{&form}}
	f.field = form.Elements[1]
	reg := runtime.NewRegistry()
	reg.Load(runtime.LoadOptions{Entities: entities})
	reg.LoadProcessors([]*processor.Processor{f.proc})
	f.s = New(reg, db, interpreter.New(), nil, Config{}, nil)
	t.Cleanup(f.s.Close)
	f.user = &auth.User{Login: "anna", Roles: []*auth.Role{{Permissions: auth.Permission{
		Processors: map[string][]string{f.proc.Name: {"run"}},
		Catalogs:   map[string][]string{f.branches.Name: {"read"}, f.directions.Name: {"read"}},
		RowAccess: auth.RowAccess{Catalogs: map[string]auth.RowPolicies{
			f.branches.Name:   {"read": {Field: "Audience", Op: "eq", Value: auth.RowValue{User: "login"}}},
			f.directions.Name: {"read": {Field: "Audience", Op: "eq", Value: auth.RowValue{User: "login"}}},
		}},
	}}}}
	return f
}

func (f kpiChoiceFixture) query(source string) url.Values {
	sources, _ := json.Marshal(map[string]string{"Объект.Branch": source})
	return url.Values{"form_kind": {"processor"}, "form_entity": {f.proc.Name}, "form": {"Form"}, "element": {"direction"}, "sources": {string(sources)}}
}

func (f kpiChoiceFixture) serve(t *testing.T, user *auth.User, path string) *httptest.ResponseRecorder {
	t.Helper()
	router := chi.NewRouter()
	f.s.Mount(router)
	r := httptest.NewRequest(http.MethodGet, path, nil)
	if user != nil {
		r = r.WithContext(auth.ContextWithUser(r.Context(), user))
	}
	w := httptest.NewRecorder()
	router.ServeHTTP(w, r)
	return w
}

func (f kpiChoiceFixture) picker(t *testing.T, user *auth.User, q url.Values) *httptest.ResponseRecorder {
	return f.serve(t, user, "/ui/_ref-options/"+f.directions.Name+"?"+q.Encode())
}

func TestKPIManagedChoiceHTTPMatrix(t *testing.T) {
	dbtest.ForEachDialect(t, func(t *testing.T, db *storage.DB) {
		f := newKPIChoiceFixture(t, db)
		get := func(q url.Values) choiceHTTPResponse {
			t.Helper()
			w := f.picker(t, f.user, q)
			if w.Code != 200 {
				t.Fatalf("%d %s", w.Code, w.Body.String())
			}
			return decodeChoiceHTTP(t, w)
		}
		all := get(f.query(""))
		if all.Total != 3 || len(all.Items) != 3 || all.Items[0]["_label"] != "Z first (Branch A)" || all.Items[1]["_label"] != "A second (Branch A)" {
			t.Fatalf("ALL lost RLS/static filter/order/suffix: %+v", all)
		}
		if strings.Contains(mustJSON(t, all), "PRIVATE_PHONE") {
			t.Fatal("PII in suffix response")
		}
		q := f.query(f.b1.String())
		q.Set("limit", "1")
		q.Set("selected_id", f.d2.String())
		page := get(q)
		if page.Total != 2 || len(page.Items) != 1 || page.SelectedAllowed == nil || !*page.SelectedAllowed {
			t.Fatalf("selected outside first page: %+v", page)
		}
		q.Set("selected_id", f.d3.String())
		if got := get(q); got.SelectedAllowed == nil || *got.SelectedAllowed {
			t.Fatalf("incompatible selected accepted: %+v", got)
		}
		q.Set("q", "second")
		q.Set("offset", "0")
		if got := get(q); got.Total != 1 || got.Items[0]["_label"] != "A second (Branch A)" {
			t.Fatalf("search lost suffix/scope: %+v", got)
		}
		q = f.query("")
		q.Set("sources", "{}")
		if got := get(q); got.Total != 0 {
			t.Fatalf("missing source opened ALL: %+v", got)
		}
		if got := get(f.query(f.hiddenBranch.String())); got.Total != 0 {
			t.Fatalf("unreadable source opened choices: %+v", got)
		}
		for _, raw := range []string{"bad", " ", "\t", uuid.Nil.String(), uuid.NewString()} {
			w := f.picker(t, f.user, f.query(raw))
			if raw == "bad" || strings.TrimSpace(raw) == "" || raw == uuid.Nil.String() {
				if w.Code != 400 {
					t.Fatalf("invalid UUID accepted: %d", w.Code)
				}
			} else if got := get(f.query(raw)); got.Total != 0 {
				t.Fatal("absent branch broadened choice")
			}
		}
		// Initial and selected options use the exact same FK suffix.
		f.proc.Params[0].Default = f.b1.String()
		f.proc.Params[1].Default = f.d2.String()
		w := f.serve(t, f.user, "/ui/processor/"+f.proc.Name)
		if w.Code != 200 || !strings.Contains(w.Body.String(), "A second (Branch A)") || strings.Contains(w.Body.String(), "PRIVATE_PHONE") {
			t.Fatalf("initial form: %d %s", w.Code, w.Body.String())
		}
		doc, err := html.Parse(strings.NewReader(w.Body.String()))
		if err != nil {
			t.Fatal(err)
		}
		sel := findSelectByName(doc, "Direction")
		if sel == nil {
			t.Fatal("direction control removed")
		}
		if ctx, ok := htmlAttribute(sel, "data-ref-choice-context"); !ok || !strings.Contains(ctx, `"form_kind":"processor"`) {
			t.Fatal("no server-owned processor context")
		}
	})
}

func TestKPIManagedChoiceGuards(t *testing.T) {
	dbtest.ForEachDialect(t, func(t *testing.T, db *storage.DB) {
		f := newKPIChoiceFixture(t, db)
		if w := f.picker(t, nil, f.query("")); w.Code != 400 {
			t.Fatalf("missing actor accepted: %d", w.Code)
		}
		for _, kind := range []string{"", "test"} {
			f.proc.Kind = kind
			if w := f.picker(t, f.user, f.query("")); w.Code != 400 {
				t.Fatalf("non-KPI processor accepted: %d", w.Code)
			}
		}
		f.proc.Kind = "kpi"
		f.proc.External = true
		if w := f.picker(t, f.user, f.query("")); w.Code != 400 {
			t.Fatalf("external processor accepted: %d", w.Code)
		}
		f.proc.External = false
		f.user.Roles[0].Permissions.Processors[f.proc.Name] = nil
		if w := f.picker(t, f.user, f.query("")); w.Code != 400 {
			t.Fatalf("processor run denial bypassed: %d", w.Code)
		}
		f.user.Roles[0].Permissions.Processors[f.proc.Name] = []string{"run"}
		for _, target := range []string{f.branches.Name, f.directions.Name} {
			permissions := &f.user.Roles[0].Permissions
			saved := permissions.Catalogs[target]
			permissions.Catalogs[target] = nil
			w := f.picker(t, f.user, f.query(""))
			if target == f.directions.Name {
				if w.Code != 403 {
					t.Fatalf("target read bypass: %d", w.Code)
				}
			} else if got := decodeChoiceHTTP(t, w); got.Total != 0 {
				t.Fatal("source read denial opened ALL")
			}
			permissions.Catalogs[target] = saved
		}
		f.user.Roles[0].Permissions.FieldAccess = auth.FieldAccess{Catalogs: map[string]auth.FieldPolicies{f.branches.Name: {"Name": {Read: "hide"}}}}
		if got := decodeChoiceHTTP(t, f.picker(t, f.user, f.query(""))); got.Total != 0 {
			t.Fatal("hidden suffix label used")
		}
		f.user.Roles[0].Permissions.FieldAccess = auth.FieldAccess{}
		f.user.Roles[0].Permissions.FieldAccess = auth.FieldAccess{Catalogs: map[string]auth.FieldPolicies{f.directions.Name: {"Branch": {Read: "hide"}}}}
		if got := decodeChoiceHTTP(t, f.picker(t, f.user, f.query(""))); got.Total != 0 {
			t.Fatal("hidden suffix FK opened ALL")
		}
		f.user.Roles[0].Permissions.FieldAccess = auth.FieldAccess{}
		f.branches.Fields[0].PII = true
		admin := &auth.User{Login: "admin", IsAdmin: true}
		if got := decodeChoiceHTTP(t, f.picker(t, admin, f.query(""))); got.Total != 0 {
			t.Fatal("admin disclosed PII suffix")
		}
		f.branches.Fields[0].PII = false
		// Ordinary forms retain their existing empty-source semantics.
		f.field.KPIChoice = nil
		f.proc.Kind = ""
		if got := decodeChoiceHTTP(t, f.picker(t, f.user, f.query(""))); got.Total != 0 {
			t.Fatal("ordinary eq changed")
		}
	})
}
