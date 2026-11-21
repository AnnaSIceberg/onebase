package ui

import (
	"bytes"
	"net/http"
	"testing"

	"github.com/google/uuid"
	"github.com/stretchr/testify/require"
	"github.com/xuri/excelize/v2"

	"github.com/ivantit66/onebase/internal/auth"
	"github.com/ivantit66/onebase/internal/dbtest"
	"github.com/ivantit66/onebase/internal/metadata"
	reportpkg "github.com/ivantit66/onebase/internal/report"
	"github.com/ivantit66/onebase/internal/runtime"
	"github.com/ivantit66/onebase/internal/storage"
)

func TestRequestPhoneJoinReportHTTP(t *testing.T) {
	const reportName = "А_ЗаявкиПоНомеруТелефона"
	const source = `ВЫБРАТЬ
		ВЫБОР КОГДА МИНИМУМ(П.Тег) < Л.Тег ТОГДА МИНИМУМ(П.Тег) ИНАЧЕ Л.Тег КОНЕЦ КАК ГруппаЗаявок,
		Л.Тег КАК Тег
		ИЗ Документ.А_Заявка КАК Л
		ВНУТРЕННЕЕ СОЕДИНЕНИЕ Документ.А_Заявка КАК П ПО П.Ссылка <> Л.Ссылка
			И ТЕЛЕФОНЫЗАЯВОКСОВПАДАЮТ(Л, П)
		СГРУППИРОВАТЬ ПО Л.Тег
		УПОРЯДОЧИТЬ ПО Тег`
	dbtest.ForEachDialect(t, func(t *testing.T, db *storage.DB) {
		doc := &metadata.Entity{Name: "А_Заявка", Kind: metadata.KindDocument, Fields: []metadata.Field{
			{Name: "Тег", Type: metadata.FieldTypeString},
			{Name: "Зона", Type: metadata.FieldTypeString},
			{Name: "ТелефонМобильный", Type: metadata.FieldTypeString, PII: true},
			{Name: "ТелефонДомашний", Type: metadata.FieldTypeString, PII: true},
			{Name: "ТелефонРабочий", Type: metadata.FieldTypeString, PII: true},
		}}
		s := newSubmitTestServerOn(t, db, []*metadata.Entity{doc})
		s.cfg.UIExceptRoles = []string{"ОператорКЦ", "ОператорМессенджера"}
		s.ops = newOperationLimiter()
		for _, fixture := range []struct {
			tag, zone, mobile, home, work string
		}{
			{"A100", "visible", "79000000101", "", ""},
			{"A200", "visible", "", "", "79000000101"},
			{"A000", "hidden", "", "79000000101", ""},
			{"B100", "visible", "79000000102", "", ""},
			{"B200", "hidden", "79000000102", "", ""},
			{"C100", "visible", "89000000173", "", ""},
			{"C200", "visible", "9000000173", "", ""},
			{"D100", "visible", "", "", ""},
			{"D200", "visible", "(   )   -  -", "", "(495)   -  -"},
			{"D300", "visible", "", "(   )   -  -", "(495)   -  -"},
			{"S100", "visible", "123", "", ""},
			{"S200", "visible", "", "123", ""},
		} {
			require.NoError(t, db.Upsert(t.Context(), doc.Name, uuid.New(), map[string]any{
				"Тег": fixture.tag, "Зона": fixture.zone,
				"ТелефонМобильный": fixture.mobile, "ТелефонДомашний": fixture.home, "ТелефонРабочий": fixture.work,
			}, doc))
		}
		enabled := &reportpkg.Report{Name: reportName, Query: source, CompareRequestPhones: true}
		other := &reportpkg.Report{Name: "ДругойОтчёт", Query: source, CompareRequestPhones: true}
		s.reg.Load(runtime.LoadOptions{Entities: []*metadata.Entity{doc}, Reports: []*reportpkg.Report{enabled, other}})

		perms := auth.Permission{
			Reports:   map[string][]string{reportName: {"run"}, other.Name: {"run"}},
			Documents: map[string][]string{doc.Name: {"read"}},
			RowAccess: auth.RowAccess{Documents: map[string]auth.RowPolicies{
				doc.Name: {"read": {Field: "Зона", Op: "eq", Value: auth.RowValue{Literal: "visible"}}},
			}},
		}
		get := func(actor *auth.User, path string) (int, []byte) {
			t.Helper()
			w := tabularHTTP(s, actor, "request-phone-join", http.MethodGet, path, nil)
			return w.Code, w.Body.Bytes()
		}
		path := "/ui/report/" + reportName
		for _, role := range []string{"РуководительКЦ", "ОператорКЦ"} {
			actor := &auth.User{Login: role, Roles: []*auth.Role{{Name: role, Permissions: perms}}}
			code, body := get(actor, path+"?__run=1")
			require.Equal(t, http.StatusOK, code)
			for _, tag := range []string{"A100", "A200", "S100", "S200"} {
				require.Contains(t, string(body), tag)
			}
			for _, hidden := range []string{"A000", "B100", "B200", "C100", "D300", "79000000101", "79000000102", "ТелефонМобильный", "ТелефонДомашний", "ТелефонРабочий"} {
				require.NotContains(t, string(body), hidden)
			}
			code, data := get(actor, path+"/excel")
			require.Equal(t, http.StatusOK, code, string(data))
			book, err := excelize.OpenReader(bytes.NewReader(data))
			require.NoError(t, err)
			rows, rowsErr := book.GetRows(book.GetSheetName(0))
			require.NoError(t, book.Close())
			require.NoError(t, rowsErr)
			require.Len(t, rows, 5)
			require.Len(t, rows[0], 2)
			require.Equal(t, [][]string{
				{"A100", "A100"}, {"A100", "A200"}, {"S100", "S100"}, {"S100", "S200"},
			}, rows[1:])
		}

		actor := &auth.User{Login: "negative", Roles: []*auth.Role{{Name: "РуководительКЦ", Permissions: perms}}}
		code, data := get(actor, "/ui/report/"+other.Name+"/excel?compare_request_phones=true")
		require.Equal(t, http.StatusBadRequest, code)
		require.Contains(t, string(data), "настроенном встроенном отчёте")

		noRead := perms
		noRead.Documents = nil
		code, _ = get(&auth.User{Login: "no-read", Roles: []*auth.Role{{Permissions: noRead}}}, path+"/excel")
		require.Equal(t, http.StatusForbidden, code)

		external := *enabled
		external.External = true
		s.reg.Load(runtime.LoadOptions{Entities: []*metadata.Entity{doc}, Reports: []*reportpkg.Report{&external}})
		code, data = get(actor, path+"/excel")
		require.Equal(t, http.StatusBadRequest, code)
		require.Contains(t, string(data), "настроенном встроенном отчёте")
	})
}
