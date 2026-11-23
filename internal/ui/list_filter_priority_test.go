package ui

import (
	"context"
	"encoding/json"
	"net/http"
	"net/http/httptest"
	"net/url"
	"slices"
	"strings"
	"testing"

	"github.com/google/uuid"
	"github.com/ivantit66/onebase/internal/auth"
	"github.com/ivantit66/onebase/internal/dbtest"
	"github.com/ivantit66/onebase/internal/metadata"
	"github.com/ivantit66/onebase/internal/storage"
)

func TestListAuthorAccountPriorityHTTP(t *testing.T) {
	dbtest.ForEachDialect(t, func(t *testing.T, db *storage.DB) {
		ctx := context.Background()
		repo := auth.NewRepo(db)
		if err := repo.EnsureSchema(ctx); err != nil {
			t.Fatal(err)
		}
		if err := repo.EnsureRolesSchema(ctx); err != nil {
			t.Fatal(err)
		}
		op := &auth.Role{Name: "ОператорКЦ"}
		messenger := &auth.Role{Name: "ОператорМессенджера"}
		if err := repo.SyncRoles(ctx, []*auth.Role{op, messenger}); err != nil {
			t.Fatal(err)
		}
		authors := &metadata.Entity{Name: "Пользователи", Kind: metadata.KindCatalog, OrderBy: []string{"Наименование"}, Fields: []metadata.Field{
			{Name: "Наименование", Type: metadata.FieldTypeString},
			{Name: "УчётнаяЗапись", Type: "reference:_users", RefEntity: metadata.SystemUsersEntity},
		}}
		author := metadata.Field{Name: "АвторДокумента", Type: "reference:Пользователи", RefEntity: authors.Name}
		request := &metadata.Entity{Name: "А_Заявка", Kind: metadata.KindDocument, Fields: []metadata.Field{author}, ListFilters: []string{author.Name}}
		call := &metadata.Entity{Name: "А_Звонок", Kind: metadata.KindDocument, Fields: []metadata.Field{author}, ListFilters: []string{author.Name}}
		s := newSubmitTestServerOn(t, db, []*metadata.Entity{authors, request, call})
		s.cfg.UIExceptRoles = []string{op.Name, messenger.Name}
		s.cfg.UIListFilterAccountPriority = &metadata.ListFilterAccountPriority{
			Documents: []string{request.Name, call.Name}, Field: author.Name, AccountField: "УчётнаяЗапись",
			NameField: "Наименование", Roles: []string{op.Name, messenger.Name},
		}

		type choiceRow struct {
			login  string
			name   string
			role   *auth.Role
			marked bool
		}
		for _, row := range []choiceRow{
			{login: "zebra", name: "Яков", role: op},
			{login: "baker", name: "борис", role: messenger},
			{login: "andrew", name: "Андрей"},
			{name: "Архив"},
			{login: "archive", name: "Александр", role: op, marked: true},
		} {
			values := map[string]any{"Наименование": row.name}
			if row.login != "" {
				u, err := repo.Create(ctx, row.login, "auth-fixture-only-20", row.name, false)
				if err != nil {
					t.Fatal(err)
				}
				if row.role != nil {
					if err := repo.AssignRole(ctx, u.ID, row.role.ID); err != nil {
						t.Fatal(err)
					}
				}
				values["УчётнаяЗапись"] = u.ID
			}
			id := uuid.New()
			if err := db.Upsert(ctx, authors.Name, id, values, authors); err != nil {
				t.Fatal(err)
			}
			if row.marked {
				if err := db.MarkForDeletion(ctx, authors.Name, id, true); err != nil {
					t.Fatal(err)
				}
			}
		}
		supervisor := &auth.Role{Name: "РуководительКЦ", Permissions: auth.Permission{
			Catalogs:  map[string][]string{authors.Name: {"read"}},
			Documents: map[string][]string{request.Name: {"read"}, call.Name: {"read"}},
		}}
		viewer := &auth.User{Login: "supervisor", Roles: []*auth.Role{supervisor}}
		labels := func(doc *metadata.Entity, user *auth.User, query url.Values) ([]string, int) {
			t.Helper()
			query.Set("search", "1")
			query.Set("list_filter", doc.Name+":"+author.Name)
			rec := serveRefOptions(t, s, authors.Name, query.Encode(), user)
			if rec.Code != http.StatusOK {
				t.Fatalf("picker status=%d: %s", rec.Code, rec.Body.String())
			}
			var response struct {
				Items []struct {
					Label string `json:"_label"`
				} `json:"items"`
				Total int `json:"total"`
			}
			if err := json.Unmarshal(rec.Body.Bytes(), &response); err != nil {
				t.Fatal(err)
			}
			names := make([]string, 0, len(response.Items))
			for _, value := range response.Items {
				names = append(names, strings.Split(value.Label, " (")[0])
			}
			return names, response.Total
		}
		for _, doc := range []*metadata.Entity{request, call} {
			req := reqWithChi(http.MethodGet, "/ui/document/"+doc.Name, nil, map[string]string{"kind": "document", "entity": doc.Name})
			rec := httptest.NewRecorder()
			s.list(rec, req.WithContext(auth.ContextWithUser(req.Context(), viewer)))
			page := rec.Body.String()
			if rec.Code != http.StatusOK || !strings.Contains(page, "data-ref-list-filter=\""+doc.Name+":"+author.Name+"\"") ||
				strings.Index(page, "борис</option>") < 0 || strings.Index(page, "борис</option>") > strings.Index(page, "Андрей</option>") {
				t.Fatalf("initial list did not prioritize operators for %s, status %d", doc.Name, rec.Code)
			}
			first, total := labels(doc, viewer, url.Values{"limit": {"2"}})
			rest, totalRest := labels(doc, viewer, url.Values{"offset": {"2"}})
			if !slices.Equal(first, []string{"борис", "Яков"}) || !slices.Equal(rest, []string{"Андрей", "Александр", "Архив"}) ||
				total != 5 || totalRest != total {
				t.Fatalf("unexpected pages: %v then %v, totals %d/%d", first, rest, total, totalRest)
			}
			found, total := labels(doc, viewer, url.Values{"q": {"БОР"}})
			if !slices.Equal(found, []string{"борис"}) || total != 1 {
				t.Fatalf("typed search: %v, total %d", found, total)
			}
			for _, role := range []*auth.Role{op, messenger} {
				mixed := &auth.User{Login: "mixed", Roles: []*auth.Role{supervisor, role}}
				got, _ := labels(doc, mixed, url.Values{})
				if len(got) != 5 || got[0] == "борис" {
					t.Fatalf("operator profile unexpectedly prioritized: %v", got)
				}
			}
		}
	})
}
