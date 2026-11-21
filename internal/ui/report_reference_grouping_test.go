package ui

import (
	"bytes"
	"net/http"
	"net/http/httptest"
	"regexp"
	"strings"
	"testing"

	"github.com/go-chi/chi/v5"
	"github.com/google/uuid"
	"github.com/stretchr/testify/require"
	"github.com/xuri/excelize/v2"

	"github.com/ivantit66/onebase/internal/auth"
	"github.com/ivantit66/onebase/internal/dbtest"
	"github.com/ivantit66/onebase/internal/metadata"
	"github.com/ivantit66/onebase/internal/report"
	"github.com/ivantit66/onebase/internal/runtime"
	"github.com/ivantit66/onebase/internal/storage"
)

func TestReportReferenceGroupingsHTTP(t *testing.T) {
	rep, err := report.ParseBytes([]byte(`name: GroupedReferences
query: |
  ВЫБРАТЬ Src.Объект КАК Заявка, Src.Оценка КАК Оценка, 'More details' КАК Caption
  ИЗ Справочник.GroupedReferenceInput КАК Src
drilldowns:
  - field: Заявка
    id_field: Заявка
    target: {kind: entity, entity: GroupedReferenceTarget}
  - field: Caption
    id_field: Заявка
    target: {kind: entity, entity: GroupedReferenceTarget}
  - title: Open row
    id_field: Заявка
    target: {kind: entity, entity: GroupedReferenceTarget}
composition:
  groupings: [Заявка]
  reference_groupings: [Заявка]
  detail: true
  detail_fields: [{field: Заявка}, {field: Caption}]
  detail_link: Заявка
  detail_entity: GroupedReferenceTarget
  measures: [{field: Оценка, agg: sum}]
  totals: {grand: true, subtotals: true}
`))
	require.NoError(t, err)
	dbtest.ForEachDialect(t, func(t *testing.T, db *storage.DB) {
		target := &metadata.Entity{Name: "GroupedReferenceTarget", Kind: metadata.KindCatalog, Fields: []metadata.Field{
			{Name: "Наименование", Type: metadata.FieldTypeString},
		}}
		input := &metadata.Entity{Name: "GroupedReferenceInput", Kind: metadata.KindCatalog, Fields: []metadata.Field{
			{Name: "Объект", Type: metadata.FieldType("reference:" + target.Name), RefEntity: target.Name},
			{Name: "Оценка", Type: metadata.FieldTypeNumber},
		}}
		entities := []*metadata.Entity{target, input}
		s := newSubmitTestServerOn(t, db, entities)
		s.ops = newOperationLimiter()
		s.reg.Load(runtime.LoadOptions{Entities: entities, Reports: []*report.Report{rep}})
		const shared = "SAME_REFERENCE_LABEL"
		ids := []uuid.UUID{uuid.New(), uuid.New()}
		for _, id := range ids {
			require.NoError(t, db.Upsert(t.Context(), target.Name, id, map[string]any{"Наименование": shared}, target))
		}
		for i, targetID := range []uuid.UUID{ids[0], ids[0], ids[1]} {
			require.NoError(t, db.Upsert(t.Context(), input.Name, uuid.New(), map[string]any{
				"Объект": targetID.String(), "Оценка": []int{1, 2, 4}[i],
			}, input))
		}
		router := chi.NewRouter()
		s.Mount(router)
		base := "/ui/report/" + rep.Name

		check := func(readTarget bool) {
			t.Helper()
			permissions := auth.Permission{
				Reports:  map[string][]string{rep.Name: {"run"}},
				Catalogs: map[string][]string{input.Name: {"read"}, target.Name: {"read"}},
			}
			if !readTarget {
				// The query can project the reference, but the label lookup
				// must not read the target records under this row policy.
				permissions.RowAccess.Catalogs = map[string]auth.RowPolicies{
					target.Name: {"read": {Field: "Наименование", Op: "eq", Value: auth.RowValue{User: "login"}}},
				}
			}
			user := &auth.User{ID: uuid.NewString(), Login: "group-reader", Roles: []*auth.Role{
				{Name: "Reader", Permissions: permissions},
			}}
			get := func(path string) *httptest.ResponseRecorder {
				t.Helper()
				req := httptest.NewRequest(http.MethodGet, path, nil)
				req = req.WithContext(auth.ContextWithUser(req.Context(), user))
				rec := httptest.NewRecorder()
				router.ServeHTTP(rec, req)
				require.Equal(t, http.StatusOK, rec.Code, "response for %s", path)
				return rec
			}
			body := get(base + "?__run=1").Body.String()
			groups := regexp.MustCompile(`class="grp" data-group="([^"]+)"[^>]*>[^<]*<td[^>]*>(.*?)</td>`).
				FindAllStringSubmatch(body, -1)
			if len(groups) != 2 {
				require.Len(t, groups, 2, "HTML result: %.3000s", body)
			}
			require.NotEqual(t, groups[0][1], groups[1][1])
			for _, group := range groups {
				require.Equal(t, readTarget, strings.Contains(group[2], shared))
			}
			visible := regexp.MustCompile(`<[^>]+>`).ReplaceAllString(body, "")
			require.Equal(t, readTarget, strings.Contains(body, `class="report-drilldown"`))
			if readTarget {
				require.Equal(t, 5, strings.Count(visible, shared))
			}
			for _, id := range ids {
				for _, group := range groups {
					require.NotContains(t, group[1], id.String())
				}
				require.NotContains(t, visible, id.String())
				for _, action := range []string{"0", "1", "2"} {
					require.Equal(t, readTarget, strings.Contains(body, "/drilldown?d="+action+"&amp;id="+id.String()))
				}
				require.Equal(t, readTarget, strings.Contains(body, "/ui/document/"+target.Name+"/"+id.String()))
				if !readTarget {
					require.NotContains(t, body, id.String())
				}
			}

			book, err := excelize.OpenReader(bytes.NewReader(get(base + "/excel").Body.Bytes()))
			require.NoError(t, err)
			rows, err := book.GetRows(book.GetSheetName(0))
			require.NoError(t, err)
			require.NoError(t, book.Close())
			var sums []string
			for _, cells := range rows {
				for _, cell := range cells {
					for _, id := range ids {
						require.NotContains(t, cell, id.String())
					}
					if !readTarget {
						require.NotContains(t, cell, shared)
					}
				}
				if len(cells) == 4 && cells[0] == shared {
					sums = append(sums, cells[3])
				}
			}
			if readTarget {
				require.ElementsMatch(t, []string{"3", "4"}, sums, rows)
			}
		}
		check(true)
		check(false)
	})
}
