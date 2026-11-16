package ui

import (
	"context"
	"encoding/json"
	"net/http/httptest"
	"net/url"
	"strings"
	"testing"

	"github.com/google/uuid"
	"github.com/ivantit66/onebase/internal/auth"
	"github.com/ivantit66/onebase/internal/dbtest"
	"github.com/ivantit66/onebase/internal/metadata"
	reportpkg "github.com/ivantit66/onebase/internal/report"
	"github.com/ivantit66/onebase/internal/runtime"
	"github.com/ivantit66/onebase/internal/storage"
)

func TestReportParameterChoiceNativeContext(t *testing.T) {
	dbtest.ForEachDialect(t, func(t *testing.T, db *storage.DB) {
		ctx := context.Background()
		repo := auth.NewRepo(db)
		if err := repo.EnsureSchema(ctx); err != nil {
			t.Fatal(err)
		}
		user, err := repo.Create(ctx, "choice-context", "test-choice-password", "Synthetic", true)
		if err != nil {
			t.Fatal(err)
		}
		branch := &metadata.Entity{Name: "ChoiceNativeBranch", Kind: metadata.KindCatalog, Fields: []metadata.Field{{Name: "Name", Type: metadata.FieldTypeString}}}
		card := &metadata.Entity{Name: "ChoiceNativeCard", Kind: metadata.KindCatalog, Fields: []metadata.Field{
			{Name: "Account", Type: "reference:_users", RefEntity: "_users"}, {Name: "Branch", Type: "reference:ChoiceNativeBranch", RefEntity: branch.Name},
		}}
		direction := &metadata.Entity{Name: "ChoiceNativeDirection", Kind: metadata.KindCatalog, Fields: []metadata.Field{
			{Name: "Name", Type: metadata.FieldTypeString}, {Name: "Branch", Type: "reference:ChoiceNativeBranch", RefEntity: branch.Name},
		}}
		entities := []*metadata.Entity{branch, card, direction}
		if err := db.Migrate(ctx, entities); err != nil {
			t.Fatal(err)
		}
		b1, b2, d1, d2 := uuid.New(), uuid.New(), uuid.New(), uuid.New()
		for _, id := range []uuid.UUID{b1, b2} {
			if err := db.Upsert(ctx, branch.Name, id, map[string]any{"Name": "Synthetic branch"}, branch); err != nil {
				t.Fatal(err)
			}
		}
		if err := db.Upsert(ctx, card.Name, uuid.New(), map[string]any{"Account": user.ID, "Branch": b1}, card); err != nil {
			t.Fatal(err)
		}
		for id, b := range map[uuid.UUID]uuid.UUID{d1: b1, d2: b2} {
			if err := db.Upsert(ctx, direction.Name, id, map[string]any{"Name": "Synthetic direction", "Branch": b}, direction); err != nil {
				t.Fatal(err)
			}
		}
		rep := &reportpkg.Report{Name: "NativeChoice", Params: []reportpkg.Param{
			{Name: "Branch", Type: "reference:ChoiceNativeBranch", Context: &reportpkg.ParamContext{UserCard: &auth.UserCardSource{Catalog: card.Name, AccountField: "Account", Field: "Branch"}}},
			{Name: "Direction", Type: "reference:ChoiceNativeDirection", ChoiceFilter: []reportpkg.ParamChoiceFilter{{Field: "Branch", Param: "Branch"}}},
		}}
		reg := runtime.NewRegistry()
		reg.Load(runtime.LoadOptions{Entities: entities, Reports: []*reportpkg.Report{rep}})
		s := &Server{store: db, reg: reg}
		boundCtx := auth.ContextWithUser(ctx, user)
		public := map[string]any{"Branch": b2.String()}
		params := s.buildReportParams(boundCtx, "ru", rep.Params, public, rep)
		if len(params) != 1 || len(params[0].Opts) != 1 || refValueString(params[0].Opts[0]["id"]) != d1.String() || public["Branch"] != b2.String() {
			t.Fatalf("initial options did not use detached trusted binding: %#v", params)
		}
		var choice managedChoiceContext
		if err := json.Unmarshal([]byte(params[0].ChoiceContext(rep.Name, rep.Params)), &choice); err != nil || len(choice.Sources) != 0 {
			t.Fatalf("hidden binding was sent as a browser source: %+v %v", choice, err)
		}
		request := func(sources string, actor *auth.User) (*resolvedChoiceRequest, error) {
			q := url.Values{"form_entity": {rep.Name}, "form": {"params"}, "element": {"Direction"}, "sources": {sources}}
			r := httptest.NewRequest("GET", "/?"+q.Encode(), nil)
			if actor != nil {
				r = r.WithContext(auth.ContextWithUser(r.Context(), actor))
			}
			return s.resolveReportChoiceRequest(r, direction)
		}
		resolved, err := request("{}", user)
		if err != nil || len(resolved.Predicates) != 1 || resolved.Predicates[0].Value != b1 {
			t.Fatalf("remote picker lost typed server context: %+v %v", resolved, err)
		}
		if _, err := request(`{"Branch":"`+b2.String()+`"}`, user); err == nil {
			t.Fatal("browser overrode trusted leading parameter")
		}
		if _, err := request("{}", nil); err == nil {
			t.Fatal("missing actor broadened dependent choice")
		}
		if opts := s.buildReportParams(ctx, "ru", rep.Params, nil, rep); len(opts) != 1 || len(opts[0].Opts) != 0 {
			t.Fatal("initial missing context broadened choice")
		}
		rep.External = true
		if _, err := request("{}", user); err == nil || !strings.Contains(err.Error(), "context") {
			t.Fatal("external report obtained configuration-only context")
		}
	})
}

func TestReportParameterChoiceNativeMarkedHistory(t *testing.T) {
	dbtest.ForEachDialect(t, func(t *testing.T, db *storage.DB) {
		ctx := context.Background()
		branch := &metadata.Entity{Name: "ChoiceMarkedBranch", Kind: metadata.KindCatalog, Fields: []metadata.Field{{Name: "Name", Type: metadata.FieldTypeString}}}
		direction := &metadata.Entity{Name: "ChoiceMarkedDirection", Kind: metadata.KindCatalog, Fields: []metadata.Field{{Name: "Name", Type: metadata.FieldTypeString}, {Name: "Branch", Type: "reference:ChoiceMarkedBranch", RefEntity: branch.Name}}}
		entities := []*metadata.Entity{branch, direction}
		if err := db.Migrate(ctx, entities); err != nil {
			t.Fatal(err)
		}
		if err := db.EnsureDeletionMark(ctx, entities); err != nil {
			t.Fatal(err)
		}
		b, d := uuid.New(), uuid.New()
		if err := db.Upsert(ctx, branch.Name, b, map[string]any{"Name": "History branch"}, branch); err != nil {
			t.Fatal(err)
		}
		if err := db.Upsert(ctx, direction.Name, d, map[string]any{"Name": "History direction", "Branch": b}, direction); err != nil {
			t.Fatal(err)
		}
		if err := db.MarkForDeletion(ctx, direction.Name, d, true); err != nil {
			t.Fatal(err)
		}
		rep := &reportpkg.Report{Name: "MarkedChoice", Params: []reportpkg.Param{
			{Name: "Branch", Type: "reference:ChoiceMarkedBranch"},
			{Name: "Direction", Type: "reference:ChoiceMarkedDirection", ChoiceFilter: []reportpkg.ParamChoiceFilter{{Field: "Branch", Param: "Branch"}}},
		}}
		reg := runtime.NewRegistry()
		reg.Load(runtime.LoadOptions{Entities: entities, Reports: []*reportpkg.Report{rep}})
		s := &Server{store: db, reg: reg}
		for _, show := range []bool{true, false} {
			rep.Params[1].ChoiceShowDeleted = &show
			values := map[string]any{"Branch": b.String()}
			options := s.buildReportParams(ctx, "ru", rep.Params, values, rep)
			want := 0
			if show {
				want = 1
			}
			if len(options[1].Opts) != want {
				t.Fatalf("initial marked=%v: %+v", show, options[1].Opts)
			}
			q := url.Values{"form_kind": {"report"}, "form_entity": {rep.Name}, "form": {"params"}, "element": {"Direction"}, "sources": {`{"Branch":"` + b.String() + `"}`}, "selected_id": {d.String()}, "search": {"0"}, "marked": {"0"}}
			w := serveRefOptions(t, s, direction.Name, q.Encode(), nil)
			var result map[string]any
			if err := json.Unmarshal(w.Body.Bytes(), &result); err != nil || w.Code != 200 || result["total"] != float64(want) || result["selected_allowed"] != true {
				t.Fatalf("remote marked=%v: %d %s %v", show, w.Code, w.Body.String(), err)
			}
			values["Direction"] = d.String()
			if err := s.validateReportChoiceValues(ctx, rep, values); err != nil {
				t.Fatalf("historical selected value became a write prohibition: %v", err)
			}
		}
	})
}
