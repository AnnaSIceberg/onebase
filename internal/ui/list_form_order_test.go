package ui

import (
	"bytes"
	"encoding/json"
	"html"
	"net/http"
	"net/http/httptest"
	"net/url"
	"os"
	"path/filepath"
	"reflect"
	"regexp"
	"strings"
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
			Name: "MasterOrder", Kind: metadata.KindCatalog, Hierarchical: true, Forms: []*metadata.FormModule{form},
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
		router.Get("/ui/_tree-children/{entity}", s.treeChildrenJSON)
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
		excelNames := func(user *auth.User, path string, query url.Values) []string {
			t.Helper()
			book, err := excelize.OpenReader(bytes.NewReader(request(user, path, query).Body.Bytes()))
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
		opening := request(user, list, page).Body.String()
		assertDefaultSortRows(t, opening, byRank[:2])
		excelLink := regexp.MustCompile(`href="([^"]+/excel[^"]*)"`).FindStringSubmatch(opening)
		if len(excelLink) != 2 {
			t.Fatal("missing Excel action")
		}
		excelURL, err := url.Parse(html.UnescapeString(excelLink[1]))
		if err != nil {
			t.Fatal(err)
		}
		assertDefaultSortRows(t, request(user, list, pinnedSecond).Body.String(), byRank[2:])
		checkOrder(excelNames(user, list+"/excel", url.Values{"limit": {"4"}}), byRank)

		s.cfg.UIEnhancements = true
		assertDefaultSortRows(t, request(user, list, page).Body.String(), byRank[:2])
		chunk := url.Values{"_list_chunk": {"1"}, "limit": {"2"}, "page": {"2"}, "sort_default": {"1"}}
		assertDefaultSortRows(t, request(user, list, chunk).Body.String(), byRank[2:])
		sortDesc := url.Values{"sort": {"Наименование"}, "dir": {"desc"}, "limit": {"2"}}
		assertDefaultSortRows(t, request(user, list, sortDesc).Body.String(), byNameDesc[:2])
		assertDefaultSortRows(t, request(user, list, page).Body.String(), byNameDesc[:2])
		checkOrder(excelNames(user, list+"/excel", url.Values{"limit": {"4"}}), byNameDesc)
		checkOrder(excelNames(user, excelURL.Path, excelURL.Query()), byRank)
		checkOrder(excelNames(user, list+"/excel", url.Values{"limit": {"4"}, "sort": {"Наименование"}, "dir": {"asc"}}), byName)
		assertDefaultSortRows(t, request(user, list, page).Body.String(), byNameDesc[:2])

		for _, role := range s.cfg.UIExceptRoles {
			mixed := leader(role)
			mixed.Roles = append(mixed.Roles, &auth.Role{Name: role})
			assertDefaultSortRows(t, request(mixed, list, page).Body.String(), byName[:2])
			checkOrder(excelNames(mixed, list+"/excel", url.Values{"limit": {"4"}}), byName)
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

		folder := uuid.New()
		root := map[string]any{"Наименование": "GroupOne", "is_folder": true}
		if err := db.Upsert(t.Context(), entity.Name, folder, root, entity); err != nil {
			t.Fatal(err)
		}
		for i, name := range []string{"Alpha", "Beta", "Gamma", "Omega"} {
			row := map[string]any{"Наименование": name, "Ранг": []int{2, 1, 2, 1}[i], "parent_id": folder}
			if err := db.Upsert(t.Context(), entity.Name, uuid.New(), row, entity); err != nil {
				t.Fatal(err)
			}
		}
		checkTree := func(query url.Values, rootNames, childNames []string) {
			t.Helper()
			body := request(user, list, query).Body.String()
			assertDefaultSortRows(t, body, rootNames)
			match := regexp.MustCompile(`<script type="application/json" id="ob-list-config">([^<]+)</script>`).FindStringSubmatch(body)
			if len(match) != 2 {
				t.Fatal("missing list configuration")
			}
			var cfg struct {
				ListOrderQuery string
			}
			if err := json.Unmarshal([]byte(match[1]), &cfg); err != nil {
				t.Fatal(err)
			}
			childQuery, err := url.ParseQuery(cfg.ListOrderQuery)
			if err != nil {
				t.Fatal(err)
			}
			childQuery.Set("parent", folder.String())
			if err := db.SaveListSortUserSettings(t.Context(), entity.Name, user.Login, "Наименование", "asc"); err != nil {
				t.Fatal(err)
			}
			var response treeChildrenResponse
			if err := json.Unmarshal(request(user, "/ui/_tree-children/masterorder", childQuery).Body.Bytes(), &response); err != nil {
				t.Fatal(err)
			}
			names := make([]string, 0, len(response.Rows))
			for _, row := range response.Rows {
				names = append(names, strings.TrimSpace(row.Cells[row.TreeCell]))
			}
			checkOrder(names, childNames)
		}
		checkTree(url.Values{"view": {"tree"}, "sort": {"Наименование"}, "dir": {"desc"}}, byNameDesc, byNameDesc)
		checkTree(url.Values{"view": {"tree"}, "sort_default": {"1"}}, byRank, byRank)
		if err := db.SaveListSortUserSettings(t.Context(), entity.Name, user.Login, "Наименование", "desc"); err != nil {
			t.Fatal(err)
		}
		checkTree(url.Values{"view": {"tree"}}, byNameDesc, byNameDesc)
	})
}

func TestManagedDocumentListOrderKeepsLastPageHTTP(t *testing.T) {
	dbtest.ForEachDialect(t, func(t *testing.T, db *storage.DB) {
		doc := &metadata.Entity{
			Name: "OrderedDocument", Kind: metadata.KindDocument,
			Fields: []metadata.Field{
				{Name: "Наименование", Type: metadata.FieldTypeString},
				{Name: "Дата", Type: metadata.FieldTypeDate},
				{Name: "Ранг", Type: metadata.FieldTypeNumber},
			},
		}
		doc.Forms = []*metadata.FormModule{{
			Name: "List", Kind: "list", EntityName: doc.Name, LayoutKind: metadata.FormLayoutManaged,
			ListOrderBy: []string{"Ранг", "Наименование"},
		}}
		s := newSubmitTestServerOn(t, db, []*metadata.Entity{doc})
		for i, name := range []string{"Alpha", "Beta", "Gamma", "Omega"} {
			row := map[string]any{
				"Наименование": name, "Дата": "2026-11-21", "Ранг": []int{2, 1, 2, 1}[i],
			}
			if err := db.Upsert(t.Context(), doc.Name, uuid.New(), row, doc); err != nil {
				t.Fatal(err)
			}
		}
		router := chi.NewRouter()
		router.Get("/ui/{kind}/{entity}", s.list)
		request := func(query string) string {
			t.Helper()
			rec := httptest.NewRecorder()
			router.ServeHTTP(rec, httptest.NewRequest(http.MethodGet, "/ui/document/ordereddocument?limit=2"+query, nil))
			if rec.Code != http.StatusOK {
				t.Fatalf("HTTP %d %s", rec.Code, rec.Body.String())
			}
			return rec.Body.String()
		}
		assertDefaultSortRows(t, request(""), []string{"Alpha", "Gamma"})
		assertDefaultSortRows(t, request("&page=last"), []string{"Alpha", "Gamma"})
		assertDefaultSortRows(t, request("&page=last&sort=Наименование"), []string{"Gamma", "Omega"})
		s.cfg.UIEnhancements = true
		body := request("")
		assertDefaultSortRows(t, body, []string{"Alpha", "Gamma"})
		if !strings.Contains(body, `data-ob-stream="up"`) {
			t.Fatal("documents no longer load earlier rows on upward scroll")
		}
		assertDefaultSortRows(t, request("&_list_chunk=1&page=1"), []string{"Beta", "Omega"})
	})
}
