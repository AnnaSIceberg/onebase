package ui

import (
	"bytes"
	"encoding/json"
	"net/http"
	"net/http/httptest"
	"net/url"
	"os"
	"path/filepath"
	"reflect"
	"testing"

	"github.com/go-chi/chi/v5"
	"github.com/google/uuid"
	"github.com/xuri/excelize/v2"

	"github.com/ivantit66/onebase/internal/auth"
	"github.com/ivantit66/onebase/internal/dbtest"
	"github.com/ivantit66/onebase/internal/dsl/loader"
	"github.com/ivantit66/onebase/internal/metadata"
	"github.com/ivantit66/onebase/internal/storage"
)

func TestManagedListOrderByRoleHTTP(t *testing.T) {
	yamlPath := filepath.Join(t.TempDir(), "masterorder.form.yaml")
	formYAML := `schema: onebase.form/v1
form:
  name: СписокМастеров
  kind: list
  entity: MasterOrder
  except_roles: [ОператорКЦ, ОператорМессенджера]
  order_by: [Ранг, Наименование]
elements:
  - kind: ПолеВвода
    name: Имя
    data_path: Объект.Наименование
`
	if err := os.WriteFile(yamlPath, []byte(formYAML), 0o600); err != nil {
		t.Fatal(err)
	}
	form, err := loader.NewManagedFormLoader().LoadFormFile(yamlPath, "MasterOrder")
	if err != nil {
		t.Fatal(err)
	}
	dbtest.ForEachDialect(t, func(t *testing.T, db *storage.DB) {
		entity := &metadata.Entity{
			Name: "MasterOrder", Kind: metadata.KindCatalog, Forms: []*metadata.FormModule{form},
			OrderBy: []string{"Наименование"},
			Fields: []metadata.Field{
				{Name: "Наименование", Type: metadata.FieldTypeString}, {Name: "Ранг", Type: metadata.FieldTypeNumber},
			},
		}
		s := newSubmitTestServerOn(t, db, []*metadata.Entity{entity})
		s.cfg.UIExceptRoles = []string{"ОператорКЦ", "ОператорМессенджера"}
		for i, name := range []string{"Alpha", "Beta", "Gamma", "Omega"} {
			row := map[string]any{"Наименование": name, "Ранг": []int{2, 1, 2, 1}[i]}
			if err := db.Upsert(t.Context(), entity.Name, uuid.New(), row, entity); err != nil {
				t.Fatal(err)
			}
		}
		router := chi.NewRouter()
		router.Get("/ui/{kind}/{entity}", s.list)
		router.Get("/ui/{kind}/{entity}/excel", s.listExcel)
		router.Get("/ui/_ref-options/{entity}", s.refOptionsJSON)
		leader := func(login string) *auth.User {
			return &auth.User{Login: login, Roles: []*auth.Role{{Name: "РуководительКЦ", Permissions: auth.Permission{
				Catalogs: map[string][]string{entity.Name: {"read"}},
			}}}}
		}
		user := leader("manager")
		request := func(user *auth.User, path string, query url.Values) *httptest.ResponseRecorder {
			t.Helper()
			rec := httptest.NewRecorder()
			req := httptest.NewRequest(http.MethodGet, path+"?"+query.Encode(), nil)
			router.ServeHTTP(rec, req.WithContext(auth.ContextWithUser(req.Context(), user)))
			if rec.Code != http.StatusOK {
				t.Fatalf("GET %s: HTTP %d %s", path, rec.Code, rec.Body.String())
			}
			return rec
		}
		const list = "/ui/catalog/masterorder"
		excelNames := func(user *auth.User, query url.Values) []string {
			t.Helper()
			book, err := excelize.OpenReader(bytes.NewReader(request(user, list+"/excel", query).Body.Bytes()))
			if err != nil {
				t.Fatal(err)
			}
			defer book.Close()
			rows, err := book.GetRows(book.GetSheetName(0))
			if err != nil || len(rows) == 0 {
				t.Fatalf("XLSX rows %v: %v", rows, err)
			}
			names := make([]string, 0, len(rows)-1)
			for _, row := range rows[1:] {
				names = append(names, row[0])
			}
			return names
		}
		checkOrder := func(got, want []string) {
			t.Helper()
			if !reflect.DeepEqual(got, want) {
				t.Fatalf("order %v, want %v", got, want)
			}
		}
		byName := []string{"Alpha", "Beta", "Gamma", "Omega"}
		byRank := []string{"Beta", "Omega", "Alpha", "Gamma"}
		byNameDesc := []string{"Omega", "Gamma", "Beta", "Alpha"}
		page := url.Values{"limit": {"2"}}
		pinnedSecond := url.Values{"limit": {"2"}, "page": {"2"}, "sort_default": {"1"}}
		assertDefaultSortRows(t, request(user, list, page).Body.String(), byRank[:2])
		assertDefaultSortRows(t, request(user, list, pinnedSecond).Body.String(), byRank[2:])
		checkOrder(excelNames(user, url.Values{"limit": {"4"}}), byRank)

		s.cfg.UIEnhancements = true
		assertDefaultSortRows(t, request(user, list, page).Body.String(), byRank[:2])
		chunk := url.Values{"_list_chunk": {"1"}, "limit": {"2"}, "page": {"2"}, "sort_default": {"1"}}
		assertDefaultSortRows(t, request(user, list, chunk).Body.String(), byRank[2:])
		sortDesc := url.Values{"sort": {"Наименование"}, "dir": {"desc"}, "limit": {"2"}}
		assertDefaultSortRows(t, request(user, list, sortDesc).Body.String(), byNameDesc[:2])
		assertDefaultSortRows(t, request(user, list, page).Body.String(), byNameDesc[:2])
		checkOrder(excelNames(user, url.Values{"limit": {"4"}}), byNameDesc)
		checkOrder(excelNames(user, url.Values{"limit": {"4"}, "sort_default": {"1"}}), byRank)
		checkOrder(excelNames(user, url.Values{"limit": {"4"}, "sort": {"Наименование"}, "dir": {"asc"}}), byName)
		assertDefaultSortRows(t, request(user, list, page).Body.String(), byNameDesc[:2])

		for _, role := range s.cfg.UIExceptRoles {
			mixed := leader(role)
			mixed.Roles = append(mixed.Roles, &auth.Role{Name: role})
			assertDefaultSortRows(t, request(mixed, list, page).Body.String(), byName[:2])
			checkOrder(excelNames(mixed, url.Values{"limit": {"4"}}), byName)
		}
		options := request(user, "/ui/_ref-options/masterorder", nil)
		var payload struct {
			Items []struct {
				Name string `json:"Наименование"`
			}
		}
		if err := json.Unmarshal(options.Body.Bytes(), &payload); err != nil {
			t.Fatal(err)
		}
		names := make([]string, 0, len(payload.Items))
		for _, row := range payload.Items {
			names = append(names, row.Name)
		}
		checkOrder(names, byName)
	})
}
