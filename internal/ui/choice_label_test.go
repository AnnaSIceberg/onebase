package ui

import (
	"context"
	"net/http"
	"net/http/httptest"
	"net/url"
	"path/filepath"
	"testing"

	"github.com/go-chi/chi/v5"
	"github.com/google/uuid"
	"github.com/ivantit66/onebase/internal/auth"
	"github.com/ivantit66/onebase/internal/metadata"
	reportpkg "github.com/ivantit66/onebase/internal/report"
	"github.com/ivantit66/onebase/internal/runtime"
	"github.com/ivantit66/onebase/internal/storage"
)

func TestCatalogChoiceLabelHTTP(t *testing.T) {
	ctx := context.Background()
	branch := &metadata.Entity{
		Name: "Филиалы", Kind: metadata.KindCatalog,
		Fields: []metadata.Field{
			{Name: "Наименование", Type: metadata.FieldTypeString},
			{Name: "Видимость", Type: metadata.FieldTypeString},
		},
	}
	ads := &metadata.Entity{
		Name: "Реклама", Kind: metadata.KindCatalog,
		ChoiceLabel: &metadata.ChoiceLabel{SuffixField: "Филиал", ExceptRoles: []string{"ОператорКЦ", "ОператорМессенджера"}},
		Fields: []metadata.Field{
			{Name: "Наименование", Type: metadata.FieldTypeString},
			{Name: "Филиал", Type: metadata.FieldType("reference:Филиалы"), RefEntity: branch.Name},
		},
	}
	entities := []*metadata.Entity{branch, ads}
	db, err := storage.ConnectSQLite(ctx, filepath.Join(t.TempDir(), "choice-label.db"))
	if err != nil {
		t.Fatal(err)
	}
	t.Cleanup(func() { db.Close() })
	if err := db.Migrate(ctx, entities); err != nil {
		t.Fatal(err)
	}
	match := map[string]string{}
	for i, name := range []string{"МСК", "ЛПЦ", "Закрытый филиал", ""} {
		id := choiceHTTPUUID(0x83, i+1)
		ad := choiceHTTPUUID(0x84, i+1)
		values := map[string]any{"Наименование": "Один партнёр"}
		if name != "" {
			audience := "manager"
			if i == 2 {
				audience = "none"
			}
			if err := db.Upsert(ctx, branch.Name, id, map[string]any{"Наименование": name, "Видимость": audience}, branch); err != nil {
				t.Fatal(err)
			}
			values["Филиал"] = id.String()
		}
		if err := db.Upsert(ctx, ads.Name, ad, values, ads); err != nil {
			t.Fatal(err)
		}
		match[ad.String()] = "Один партнёр"
		if i < 2 {
			match[ad.String()] += " (" + name + ")"
		}
	}

	reg := runtime.NewRegistry()
	reg.Load(runtime.LoadOptions{Entities: entities})
	s := &Server{reg: reg, store: db}
	s.entitySvc = s.newEntityService(nil)
	router := chi.NewRouter()
	s.Mount(router)

	role := &auth.Role{Name: "Менеджер", Permissions: auth.Permission{
		Catalogs: map[string][]string{ads.Name: {"read"}, branch.Name: {"read"}},
		RowAccess: auth.RowAccess{Catalogs: map[string]auth.RowPolicies{
			branch.Name: {"read": {Field: "Видимость", Op: "eq", Value: auth.RowValue{User: "login"}}},
		}},
	}}
	for _, tc := range []struct {
		name   string
		roles  []*auth.Role
		suffix bool
	}{
		{"manager", []*auth.Role{role}, true},
		{"operator", []*auth.Role{{Name: "ОператорКЦ", Permissions: role.Permissions}}, false},
		{"mixed-messenger", []*auth.Role{role, {Name: "ОператорМессенджера", Permissions: role.Permissions}}, false},
		{"target-without-related-read", []*auth.Role{{Name: "Ограниченный", Permissions: auth.Permission{
			Catalogs: map[string][]string{ads.Name: {"read"}},
		}}}, false},
	} {
		t.Run(tc.name, func(t *testing.T) {
			user := &auth.User{Login: "manager", Roles: tc.roles}
			authCtx := auth.ContextWithUser(ctx, user)
			query := url.Values{"q": {"Один партнёр"}, "limit": {"10"}}
			request := httptest.NewRequest(http.MethodGet, "/ui/_ref-options/"+url.PathEscape(ads.Name)+"?"+query.Encode(), nil)
			request = request.WithContext(authCtx)
			rec := httptest.NewRecorder()
			router.ServeHTTP(rec, request)
			if rec.Code != http.StatusOK {
				t.Fatalf("ref options: %d %s", rec.Code, rec.Body.String())
			}
			got := decodeChoiceHTTP(t, rec)
			initial, err := s.reportReferenceOptions(authCtx, reportpkg.Param{Name: "Источник", Type: "reference:Реклама"}, nil, ads)
			if err != nil {
				t.Fatal(err)
			}
			for _, rows := range [][]map[string]any{got.Items, initial} {
				if len(rows) != 4 {
					t.Fatalf("got %d options, want 4", len(rows))
				}
				for _, row := range rows {
					id, err := uuid.Parse(refValueString(row["id"]))
					want, ok := match[id.String()]
					if err != nil || !ok {
						t.Fatalf("unexpected option identity %v", row["id"])
					}
					if !tc.suffix {
						want = "Один партнёр"
					}
					if row["_label"] != want {
						t.Errorf("label %v, want %s", row["_label"], want)
					}
				}
			}
			selected := s.appendSelectedRefOptions(authCtx, nil, ads, []string{choiceHTTPUUID(0x84, 1).String()})
			want := "Один партнёр"
			if tc.suffix {
				want += " (МСК)"
			}
			if len(selected) != 1 || selected[0]["_label"] != want {
				t.Fatalf("restored choice %v, want %s", selected, want)
			}
		})
	}
}
