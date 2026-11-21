package ui

import (
	"bytes"
	"net/http"
	"net/http/httptest"
	"os"
	"path/filepath"
	"strings"
	"testing"

	"github.com/go-chi/chi/v5"
	"github.com/google/uuid"
	"github.com/stretchr/testify/require"
	"github.com/xuri/excelize/v2"

	"github.com/ivantit66/onebase/internal/auth"
	"github.com/ivantit66/onebase/internal/dbtest"
	"github.com/ivantit66/onebase/internal/metadata"
	"github.com/ivantit66/onebase/internal/storage"
)

func TestJournalProjectedReferenceAndDeletionMarkHTTP(t *testing.T) {
	path := filepath.Join(t.TempDir(), "journal.yaml")
	require.NoError(t, os.WriteFile(path, []byte(`name: ProjectionJournal
documents: [ProjectionDocument]
columns:
  - {field: Номер, label: Номер}
  - {field: deletion_mark, label: Пометка удаления}
  - {field: Филиал, label: Филиал, reference_path: Мастер.ФилиалАйсберг}
`), 0o600))
	journal, err := metadata.LoadJournalFile(path)
	require.NoError(t, err)
	dbtest.ForEachDialect(t, func(t *testing.T, db *storage.DB) {
		branch := &metadata.Entity{Name: "ProjectionBranch", Kind: metadata.KindCatalog,
			Fields: []metadata.Field{{Name: "Наименование", Type: metadata.FieldTypeString}}}
		master := &metadata.Entity{Name: "ProjectionMaster", Kind: metadata.KindCatalog, Fields: []metadata.Field{
			{Name: "Наименование", Type: metadata.FieldTypeString},
			{Name: "ФилиалАйсберг", Type: "reference:ProjectionBranch", RefEntity: branch.Name},
		}}
		doc := &metadata.Entity{Name: "ProjectionDocument", Kind: metadata.KindDocument, Fields: []metadata.Field{
			{Name: "Номер", Type: metadata.FieldTypeString},
			{Name: "Мастер", Type: "reference:ProjectionMaster", RefEntity: master.Name},
		}}
		s := newSubmitTestServerOn(t, db, []*metadata.Entity{branch, master, doc})
		s.reg.LoadJournals([]*metadata.Journal{journal})
		branchID, masterID, docID := uuid.New(), uuid.New(), uuid.New()
		const label = "BRANCH_ALLOWED_LABEL"
		ctx := t.Context()
		require.NoError(t, db.Upsert(ctx, branch.Name, branchID, map[string]any{"Наименование": label}, branch))
		require.NoError(t, db.Upsert(ctx, master.Name, masterID, map[string]any{
			"Наименование": "Master", "ФилиалАйсберг": branchID.String(),
		}, master))
		require.NoError(t, db.Upsert(ctx, doc.Name, docID, map[string]any{"Номер": "T-1", "Мастер": masterID.String()}, doc))
		require.NoError(t, db.MarkForDeletion(ctx, doc.Name, docID, true))
		router := chi.NewRouter()
		s.Mount(router)
		denyRow := auth.RowPolicies{"read": {Field: "Наименование", Op: "eq", Value: auth.RowValue{User: "login"}}}
		for _, profile := range []struct {
			name   string
			change func(*auth.Permission)
		}{
			{name: "allowed"},
			{name: "master-object", change: func(p *auth.Permission) { delete(p.Catalogs, master.Name) }},
			{name: "branch-object", change: func(p *auth.Permission) { delete(p.Catalogs, branch.Name) }},
			{name: "master-row", change: func(p *auth.Permission) {
				p.RowAccess.Catalogs = map[string]auth.RowPolicies{master.Name: denyRow}
			}},
			{name: "branch-row", change: func(p *auth.Permission) {
				p.RowAccess.Catalogs = map[string]auth.RowPolicies{branch.Name: denyRow}
			}},
			{name: "document-field", change: func(p *auth.Permission) {
				p.FieldAccess.Documents = map[string]auth.FieldPolicies{doc.Name: {"Мастер": {Read: "mask_city"}}}
			}},
			{name: "master-field", change: func(p *auth.Permission) {
				p.FieldAccess.Catalogs = map[string]auth.FieldPolicies{master.Name: {"ФилиалАйсберг": {Read: "hide"}}}
			}},
			{name: "branch-label", change: func(p *auth.Permission) {
				p.FieldAccess.Catalogs = map[string]auth.FieldPolicies{branch.Name: {"Наименование": {Read: "hide"}}}
			}},
		} {
			t.Run(profile.name, func(t *testing.T) {
				permission := auth.Permission{
					Documents: map[string][]string{doc.Name: {"read"}},
					Catalogs:  map[string][]string{master.Name: {"read"}, branch.Name: {"read"}},
				}
				if profile.change != nil {
					profile.change(&permission)
				}
				user := &auth.User{ID: uuid.NewString(), Login: "projection-reader", Roles: []*auth.Role{
					{Name: "ProjectionReader", Permissions: permission},
				}}
				request := func(suffix string) *httptest.ResponseRecorder {
					t.Helper()
					r := httptest.NewRequest(http.MethodGet, "/ui/journal/ProjectionJournal"+suffix, nil)
					r = r.WithContext(auth.ContextWithUser(r.Context(), user))
					rec := httptest.NewRecorder()
					router.ServeHTTP(rec, r)
					require.Equal(t, http.StatusOK, rec.Code, "journal response for %s", suffix)
					return rec
				}

				allowed := profile.change == nil
				html := request("").Body.String()
				require.Contains(t, html, ">Пометка удаления</th>")
				require.Contains(t, html, ">Филиал</th>")
				require.Less(t, strings.Index(html, ">Пометка удаления</th>"), strings.Index(html, ">Филиал</th>"))
				require.Contains(t, html, ">true</td>")
				require.Equal(t, allowed, strings.Contains(html, label))
				require.NotContains(t, html, masterID.String())
				require.NotContains(t, html, branchID.String())

				book, err := excelize.OpenReader(bytes.NewReader(request("/excel").Body.Bytes()))
				require.NoError(t, err)
				defer func() { require.NoError(t, book.Close()) }()
				sheet := book.GetSheetName(0)
				rows, err := book.GetRows(sheet)
				require.NoError(t, err)
				require.Len(t, rows, 2)
				require.Equal(t, []string{"Номер", "Пометка удаления", "Филиал"}, rows[0][1:])
				require.Equal(t, "TRUE", strings.ToUpper(rows[1][2]))
				value, err := book.GetCellValue(sheet, "D2")
				require.NoError(t, err)
				if allowed {
					require.Equal(t, label, value)
				} else {
					require.Empty(t, value)
				}
			})
		}
	})
}
