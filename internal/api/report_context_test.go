package api

import (
	"context"
	"encoding/json"
	"fmt"
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
	reportpkg "github.com/ivantit66/onebase/internal/report"
	"github.com/ivantit66/onebase/internal/runtime"
	"github.com/ivantit66/onebase/internal/storage"
)

func TestAPIReportTrustedContextMatrix(t *testing.T) {
	dbtest.ForEachDialect(t, func(t *testing.T, db *storage.DB) {
		ctx := context.Background()
		repo := auth.NewRepo(db)
		if err := repo.EnsureSchema(ctx); err != nil {
			t.Fatal(err)
		}
		user, err := repo.Create(ctx, "context-api-alice", "test-context-password", "Alice", true)
		if err != nil {
			t.Fatal(err)
		}
		department := &metadata.Entity{Name: "APIContextDepartment", Kind: metadata.KindCatalog, Fields: []metadata.Field{{Name: "Name", Type: metadata.FieldTypeString}}}
		unitID, otherUnitID := uuid.New(), uuid.New()
		card := &metadata.Entity{Name: "APIContextStaff", Kind: metadata.KindCatalog, Fields: []metadata.Field{
			{Name: "Account", Type: "reference:_users", RefEntity: "_users"}, {Name: "Department", Type: "reference:APIContextDepartment", RefEntity: "APIContextDepartment"},
		}}
		entity := &metadata.Entity{Name: "APIContextEntry", Kind: metadata.KindCatalog, Fields: []metadata.Field{
			{Name: "Name", Type: metadata.FieldTypeString}, {Name: "Owner", Type: metadata.FieldTypeString}, {Name: "Department", Type: "reference:APIContextDepartment", RefEntity: "APIContextDepartment"},
			{Name: "OwnerAccount", Type: "reference:_users", RefEntity: "_users"},
		}}
		if err := db.Migrate(ctx, []*metadata.Entity{department, card, entity}); err != nil {
			t.Fatal(err)
		}
		for _, id := range []uuid.UUID{unitID, otherUnitID} {
			if err := db.Upsert(ctx, department.Name, id, map[string]any{"Name": "synthetic unit"}, department); err != nil {
				t.Fatal(err)
			}
		}
		if err := db.Upsert(ctx, card.Name, uuid.New(), map[string]any{"Account": user.ID, "Department": unitID.String()}, card); err != nil {
			t.Fatal(err)
		}
		for _, row := range []map[string]any{
			{"Name": "VISIBLE_API_CONTEXT", "Owner": user.Login, "Department": unitID.String()},
			{"Name": "FOREIGN_API_USER", "Owner": "bob", "Department": unitID.String()},
			{"Name": "FOREIGN_API_UNIT", "Owner": user.Login, "Department": otherUnitID.String()},
		} {
			row["OwnerAccount"] = user.ID
			if err := db.Upsert(ctx, entity.Name, uuid.New(), row, entity); err != nil {
				t.Fatal(err)
			}
		}
		rep := &reportpkg.Report{Name: "APITrustedContext", Query: `ВЫБРАТЬ Name ИЗ Справочник.APIContextEntry ГДЕ Owner=&Actor И Department=&Unit И OwnerAccount=&ActorID`, Params: []reportpkg.Param{
			{Name: "Actor", Type: "string", Context: &reportpkg.ParamContext{User: "login"}},
			{Name: "ActorID", Type: "reference:_users", Context: &reportpkg.ParamContext{User: "id"}},
			{Name: "Unit", Type: "reference:APIContextDepartment", Context: &reportpkg.ParamContext{UserCard: &auth.UserCardSource{Catalog: card.Name, AccountField: "Account", Field: "Department"}}},
		}}
		user.IsAdmin = false
		user.Roles = []*auth.Role{{Name: "ContextRunner", Permissions: auth.Permission{Reports: map[string][]string{rep.Name: {"run"}}, Catalogs: map[string][]string{entity.Name: {"read"}, department.Name: {"read"}}}}}
		reg := runtime.NewRegistry()
		reg.Load(runtime.LoadOptions{Entities: []*metadata.Entity{department, card, entity}, Reports: []*reportpkg.Report{rep}})
		bound, err := reportpkg.ResolveContextParams(auth.ContextWithUser(ctx, user), rep, map[string]any{"Unit": otherUnitID.String()}, reg, db)
		if err != nil {
			t.Fatal(err)
		}
		if actual, ok := bound["Unit"].(uuid.UUID); !ok || actual != unitID {
			t.Fatalf("native card reference was reparsed/lost: %T", bound["Unit"])
		}
		h := &handler{reg: reg, store: db, interp: interpreter.New()}
		router := chi.NewRouter()
		h.mountV2(router)
		rep.Variants = []reportpkg.ReportVariant{{Name: "scoped", Query: rep.Query, Filters: []reportpkg.VariantFilter{{Field: "name", Op: "eq", Value: "VISIBLE_API_CONTEXT"}}}}
		for i, q := range []url.Values{{}, {"Actor": {"bob"}, "ActorID": {"malformed-client-value"}, "Unit": {otherUnitID.String()}}, {"Actor": {""}, "Unit": {""}, "actor": {"bob"}}, {"variant": {"scoped"}, "Unit": {otherUnitID.String()}, "Actor": {"bob"}}} {
			t.Run(fmt.Sprintf("spoof-%d", i), func(t *testing.T) {
				r := httptest.NewRequest("GET", "/api/v2/report/"+rep.Name+"?"+q.Encode(), nil)
				r = r.WithContext(auth.ContextWithUser(r.Context(), user))
				w := httptest.NewRecorder()
				router.ServeHTTP(w, r)
				if w.Code != 200 {
					t.Fatalf("status%d %s", w.Code, w.Body.String())
				}
				var body struct {
					Data []map[string]any `json:"data"`
				}
				if err := json.Unmarshal(w.Body.Bytes(), &body); err != nil {
					t.Fatal(err)
				}
				if len(body.Data) != 1 || body.Data[0]["name"] != "VISIBLE_API_CONTEXT" {
					t.Fatalf("wrong actor rows: %#v", body.Data)
				}
			})
		}
		if b, err := json.Marshal(reportOpenAPISchema(rep)); err != nil || strings.Contains(string(b), "Actor") || strings.Contains(string(b), "Unit") {
			t.Fatal("trusted bindings advertised as API inputs")
		}
		t.Run("reference-join-read-denied", func(t *testing.T) {
			delete(user.Roles[0].Permissions.Catalogs, department.Name)
			defer func() { user.Roles[0].Permissions.Catalogs[department.Name] = []string{"read"} }()
			w := httptest.NewRecorder()
			r := httptest.NewRequest("GET", "/api/v2/report/"+rep.Name, nil)
			r = r.WithContext(auth.ContextWithUser(r.Context(), user))
			router.ServeHTTP(w, r)
			if w.Code != http.StatusForbidden {
				t.Fatalf("reference target read was bypassed:%d", w.Code)
			}
		})
		user.ID = uuid.NewString()
		w := httptest.NewRecorder()
		r := httptest.NewRequest("GET", "/api/v2/report/"+rep.Name+"?Actor=bob&Unit=OTHER_UNIT", nil)
		r = r.WithContext(auth.ContextWithUser(r.Context(), user))
		router.ServeHTTP(w, r)
		if w.Code != http.StatusForbidden {
			t.Fatalf("missing user card did not fail closed: %d", w.Code)
		}
	})
}
