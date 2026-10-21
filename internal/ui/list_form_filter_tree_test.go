package ui

import (
	"bytes"
	"encoding/json"
	"net/http"
	"net/http/httptest"
	"net/url"
	"strings"
	"testing"

	"github.com/google/uuid"
	"github.com/ivantit66/onebase/internal/auth"
	"github.com/ivantit66/onebase/internal/dbtest"
	"github.com/ivantit66/onebase/internal/metadata"
	"github.com/ivantit66/onebase/internal/storage"
	"github.com/xuri/excelize/v2"
)

// Один и тот же отбор и его переключатель действуют в списке, Excel и дереве.
func TestUI_ListFormFilterEntryPointsMatrix(t *testing.T) {
	dbtest.ForEachDialect(t, func(t *testing.T, db *storage.DB) {
		entity := &metadata.Entity{Name: "Заявка", Kind: metadata.KindCatalog, Hierarchical: true,
			Fields: []metadata.Field{{Name: "Наименование", Type: metadata.FieldTypeString}, {Name: "Автор", Type: metadata.FieldTypeString}}}
		form := &metadata.FormModule{Name: "ФормаСписка", Kind: "list", EntityName: entity.Name, LayoutKind: metadata.FormLayoutManaged}
		entity.Forms = []*metadata.FormModule{form}
		s, ctx := newSubmitTestServer(t, []*metadata.Entity{entity})
		if err := db.Migrate(ctx, []*metadata.Entity{entity}); err != nil {
			t.Fatal(err)
		}
		s.store = db
		s.entitySvc.Store = db
		parent := uuid.New()
		if err := db.Upsert(ctx, entity.Name, parent, map[string]any{"Наименование": "Folder", "Автор": "u-op", "is_folder": true}, entity); err != nil {
			t.Fatal(err)
		}
		for _, row := range []map[string]any{
			{"Наименование": "first-child", "Автор": "u-op", "parent_id": parent.String(), "is_folder": true},
			{"Наименование": "second-child", "Автор": "u-other", "parent_id": parent.String()},
		} {
			if err := db.Upsert(ctx, entity.Name, uuid.New(), row, entity); err != nil {
				t.Fatal(err)
			}
		}
		for _, tc := range []struct {
			name      string
			removable bool
			off       string
			admin     bool
			wantOther bool
		}{
			{name: "permanent"},
			{name: "permanent_cannot_be_removed", off: "0"},
			{name: "default_on", removable: true},
			{name: "default_off", removable: true, off: "0", wantOther: true},
			{name: "administrator", admin: true, wantOther: true},
		} {
			t.Run(tc.name, func(t *testing.T) {
				form.ListFilter = []metadata.FormListCondition{{Query: listQueryMine, Removable: tc.removable}}
				u := listQueryUser("u-op")
				u.IsAdmin = tc.admin
				qs := url.Values{"parent": {parent.String()}}
				if tc.off != "" {
					qs.Set(listFormFilterOffParam, tc.off)
				}
				check := func(body string) {
					t.Helper()
					if !strings.Contains(body, "first-child") || strings.Contains(body, "second-child") != tc.wantOther {
						t.Fatal("list entries differ from the selected form filter")
					}
				}
				for _, export := range []bool{false, true} {
					path := "/ui/catalog/" + entity.Name
					if export {
						path += "/excel"
					}
					r := reqWithChi(http.MethodGet, path+"?"+qs.Encode(), nil, map[string]string{"kind": "catalog", "entity": entity.Name})
					r = r.WithContext(auth.ContextWithUser(r.Context(), u))
					w := httptest.NewRecorder()
					if export {
						s.listExcel(w, r)
					} else {
						s.list(w, r)
					}
					if w.Code != http.StatusOK {
						t.Fatalf("export=%v: HTTP %d", export, w.Code)
					}
					if !export {
						check(w.Body.String())
						if !strings.Contains(w.Body.String(), `"listFormOff":"`+tc.off+`"`) {
							t.Fatal("list configuration lost the filter toggle")
						}
						continue
					}
					book, err := excelize.OpenReader(bytes.NewReader(w.Body.Bytes()))
					if err != nil {
						t.Fatal(err)
					}
					rows, err := book.GetRows(book.GetSheetName(0))
					if closeErr := book.Close(); closeErr != nil {
						t.Fatal(closeErr)
					}
					if err != nil {
						t.Fatal(err)
					}
					var text strings.Builder
					for _, row := range rows {
						text.WriteString(strings.Join(row, " "))
						text.WriteByte('\n')
					}
					check(text.String())
				}
				w := serveTreeChildren(t, s, entity.Name, qs.Encode()+"&depth=0", u)
				if w.Code != http.StatusOK {
					t.Fatalf("tree: HTTP %d", w.Code)
				}
				var body treeChildrenResponse
				if err := json.Unmarshal(w.Body.Bytes(), &body); err != nil {
					t.Fatal(err)
				}
				var text strings.Builder
				for _, row := range body.Rows {
					text.WriteString(strings.Join(row.Cells, " "))
					folder, err := url.Parse(row.FolderURL)
					if err != nil || folder.Query().Get(listFormFilterOffParam) != tc.off {
						t.Fatal("tree folder link lost the filter toggle")
					}
				}
				check(text.String())
			})
		}
	})
}
