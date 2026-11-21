package ui

import (
	"bytes"
	"net/http"
	"net/http/httptest"
	"net/url"
	"strings"
	"testing"

	"github.com/go-chi/chi/v5"
	"github.com/google/uuid"
	"github.com/ivantit66/onebase/internal/auth"
	"github.com/ivantit66/onebase/internal/dbtest"
	"github.com/ivantit66/onebase/internal/metadata"
	reportpkg "github.com/ivantit66/onebase/internal/report"
	"github.com/ivantit66/onebase/internal/runtime"
	"github.com/ivantit66/onebase/internal/storage"
	"github.com/stretchr/testify/require"
	"github.com/xuri/excelize/v2"
	"gopkg.in/yaml.v3"
)

func TestReportCatalogNameContextHTTP(t *testing.T) {
	var rep reportpkg.Report
	require.NoError(t, yaml.Unmarshal([]byte(`
name: StockPrice
params:
  - name: PriceType
    type: reference:PriceTypeCatalog
    context:
      catalog_name:
        catalog: PriceTypeCatalog
        name: Цена мастера
query: |
  ВЫБРАТЬ С.Наименование ИЗ Справочник.StockOwner КАК М
  ВНУТРЕННЕЕ СОЕДИНЕНИЕ Справочник.StockEntry КАК С ПО С.Владелец = М.Ссылка
  ГДЕ С.ТипЦены = &PriceType
`), &rep))
	price := &metadata.Entity{Name: "PriceTypeCatalog", Kind: metadata.KindCatalog, Hierarchical: true, Fields: []metadata.Field{
		{Name: "Наименование", Type: metadata.FieldTypeString},
	}}
	owner := &metadata.Entity{Name: "StockOwner", Kind: metadata.KindCatalog}
	row := &metadata.Entity{Name: "StockEntry", Kind: metadata.KindCatalog, Fields: []metadata.Field{
		{Name: "Наименование", Type: metadata.FieldTypeString},
		{Name: "ТипЦены", Type: metadata.FieldTypeString, RefEntity: price.Name},
		{Name: "Владелец", Type: metadata.FieldTypeString, RefEntity: owner.Name},
	}}
	entities := []*metadata.Entity{price, owner, row}
	s, ctx := newSubmitTestServer(t, entities)
	s.ops = newOperationLimiter()
	s.reg.Load(runtime.LoadOptions{Entities: entities, Reports: []*reportpkg.Report{&rep}})
	master, alternate := uuid.New(), uuid.New()
	putPrice := func(id uuid.UUID, name string) {
		t.Helper()
		require.NoError(t, s.store.Upsert(ctx, price.Name, id, map[string]any{"Наименование": name}, price))
	}
	putPrice(master, "Цена мастера")
	putPrice(alternate, "Розничная цена")
	for _, flag := range []string{"deletion_mark", "is_folder"} {
		id := uuid.New()
		putPrice(id, "Цена мастера")
		_, err := s.store.Exec(ctx, "UPDATE "+metadata.TableName(price.Name)+" SET "+flag+" = 1 WHERE id = ?", id.String())
		require.NoError(t, err)
	}
	ownerID := uuid.New()
	require.NoError(t, s.store.Upsert(ctx, owner.Name, ownerID, nil, owner))
	require.NoError(t, s.store.Upsert(ctx, row.Name, uuid.New(), map[string]any{
		"Наименование": "STOCK_MASTER_PRICE", "ТипЦены": master.String(), "Владелец": ownerID.String(),
	}, row))
	require.NoError(t, s.store.Upsert(ctx, row.Name, uuid.New(), map[string]any{
		"Наименование": "STOCK_ALTERNATE_PRICE", "ТипЦены": alternate.String(), "Владелец": ownerID.String(),
	}, row))
	user := &auth.User{ID: uuid.NewString(), Login: "stock-otk", Roles: []*auth.Role{{
		Name: "ОТКСотрудник", Permissions: auth.Permission{
			Reports:  map[string][]string{rep.Name: {"run"}},
			Catalogs: map[string][]string{row.Name: {"read"}, owner.Name: {"read"}},
		},
	}}}
	router := chi.NewRouter()
	s.Mount(router)
	get := func(path string) *httptest.ResponseRecorder {
		req := httptest.NewRequest(http.MethodGet, path, nil)
		req = req.WithContext(auth.ContextWithUser(req.Context(), user))
		res := httptest.NewRecorder()
		router.ServeHTTP(res, req)
		return res
	}
	require.Equal(t, http.StatusForbidden, get("/ui/catalog/"+price.Name).Code)
	require.Equal(t, http.StatusForbidden, get("/ui/catalog/"+price.Name+"/"+master.String()).Code)
	base := "/ui/report/" + rep.Name
	spoof := url.Values{"__run": {"1"}, "PriceType": {alternate.String()}, "pricetype": {alternate.String()}}
	actorCtx := auth.ContextWithUser(ctx, user)
	bound, err := reportpkg.ResolveContextParams(actorCtx, &rep, nil, s.reg, s.store)
	require.NoError(t, err)
	require.Equal(t, master.String(), bound["PriceType"])
	compiled, err := s.compileQueryWithRowAccess(actorCtx, rep.Query, bound)
	require.NoError(t, err)
	require.Empty(t, s.deniedQuerySource(actorCtx, compiled.Sources), "sources=%v", compiled.Sources)
	html := get(base + "?" + spoof.Encode())
	require.Equal(t, http.StatusOK, html.Code)
	require.Contains(t, html.Body.String(), "STOCK_MASTER_PRICE")
	require.NotContains(t, html.Body.String(), "STOCK_ALTERNATE_PRICE")
	require.NotContains(t, html.Body.String(), master.String())
	require.NotContains(t, html.Body.String(), `name="PriceType"`)
	xlsx := get(base + "/excel?" + spoof.Encode())
	require.Equal(t, http.StatusOK, xlsx.Code)
	book, err := excelize.OpenReader(bytes.NewReader(xlsx.Body.Bytes()))
	require.NoError(t, err)
	rows, err := book.GetRows(book.GetSheetList()[0])
	require.NoError(t, err)
	require.NoError(t, book.Close())
	require.Contains(t, rows, []string{"STOCK_MASTER_PRICE"})
	require.NotContains(t, rows, []string{"STOCK_ALTERNATE_PRICE"})
	pdf := get(base + "/pdf?" + spoof.Encode())
	require.Equal(t, http.StatusOK, pdf.Code)
	require.True(t, strings.HasPrefix(pdf.Body.String(), "%PDF-"))

	putPrice(master, "Другая цена")
	missing := get(base + "?" + spoof.Encode())
	require.Equal(t, http.StatusForbidden, missing.Code)
	require.Contains(t, missing.Body.String(), "запись для параметра PriceType не найдена")
	putPrice(master, "Цена мастера")
	putPrice(uuid.New(), "Цена мастера")
	for _, path := range []string{base, base + "/excel"} {
		res := get(path + "?" + spoof.Encode())
		require.Equal(t, http.StatusForbidden, res.Code)
		require.Contains(t, res.Body.String(), "для параметра PriceType найдено несколько записей")
	}
}

func TestReportCatalogNameAllowMissingHTTP(t *testing.T) {
	var rep reportpkg.Report
	require.NoError(t, yaml.Unmarshal([]byte(`
name: PartiesWithOptionalPrice
params:
  - name: PriceType
    type: reference:OptionalPriceType
    context:
      catalog_name:
        catalog: OptionalPriceType
        name: Цена мастера
        allow_missing: true
query: |
  ВЫБРАТЬ П.Наименование, П.Количество, П.Себестоимость, Ц.Цена
  ИЗ Справочник.OptionalPriceParty КАК П
  ЛЕВОЕ СОЕДИНЕНИЕ Справочник.OptionalPriceEntry КАК Ц
    ПО Ц.Партия = П.Ссылка И Ц.ТипЦены = &PriceType
`), &rep))
	dbtest.ForEachDialect(t, func(t *testing.T, db *storage.DB) {
		ctx := t.Context()
		priceType := &metadata.Entity{Name: "OptionalPriceType", Kind: metadata.KindCatalog, Fields: []metadata.Field{
			{Name: "Наименование", Type: metadata.FieldTypeString},
		}}
		party := &metadata.Entity{Name: "OptionalPriceParty", Kind: metadata.KindCatalog, Fields: []metadata.Field{
			{Name: "Наименование", Type: metadata.FieldTypeString},
			{Name: "Количество", Type: metadata.FieldTypeNumber},
			{Name: "Себестоимость", Type: metadata.FieldTypeNumber},
		}}
		price := &metadata.Entity{Name: "OptionalPriceEntry", Kind: metadata.KindCatalog, Fields: []metadata.Field{
			{Name: "Партия", Type: metadata.FieldTypeString, RefEntity: party.Name},
			{Name: "ТипЦены", Type: metadata.FieldTypeString, RefEntity: priceType.Name},
			{Name: "Цена", Type: metadata.FieldTypeNumber},
		}}
		entities := []*metadata.Entity{priceType, party, price}
		s := newSubmitTestServerOn(t, db, entities)
		s.ops = newOperationLimiter()
		s.reg.Load(runtime.LoadOptions{Entities: entities, Reports: []*reportpkg.Report{&rep}})
		partyID, retailID, masterID := uuid.New(), uuid.New(), uuid.New()
		putType := func(id uuid.UUID, name string) {
			t.Helper()
			require.NoError(t, db.Upsert(ctx, priceType.Name, id, map[string]any{"Наименование": name}, priceType))
		}
		putPrice := func(typeID any, amount int) {
			t.Helper()
			require.NoError(t, db.Upsert(ctx, price.Name, uuid.New(), map[string]any{
				"Партия": partyID.String(), "ТипЦены": typeID, "Цена": amount,
			}, price))
		}
		putType(retailID, "Розничная цена")
		require.NoError(t, db.Upsert(ctx, party.Name, partyID, map[string]any{
			"Наименование": "PARTY_FIXTURE", "Количество": 3, "Себестоимость": 30,
		}, party))
		putPrice(retailID.String(), 800)
		putPrice(nil, 900)

		router := chi.NewRouter()
		s.Mount(router)
		const base = "/ui/report/PartiesWithOptionalPrice"
		spoof := "?" + url.Values{
			"__run": {"1"}, "PriceType": {retailID.String()}, "pricetype": {retailID.String()},
		}.Encode()
		check := func(wantPrice string, status int) {
			t.Helper()
			for _, roleName := range []string{"ОТКСотрудник", "ОТКРуководитель"} {
				user := &auth.User{ID: uuid.NewString(), Login: roleName, Roles: []*auth.Role{{
					Name: roleName, Permissions: auth.Permission{
						Reports:  map[string][]string{rep.Name: {"run"}},
						Catalogs: map[string][]string{party.Name: {"read"}, price.Name: {"read"}},
					},
				}}}
				get := func(path string) *httptest.ResponseRecorder {
					t.Helper()
					req := httptest.NewRequest(http.MethodGet, path, nil)
					req = req.WithContext(auth.ContextWithUser(req.Context(), user))
					rec := httptest.NewRecorder()
					router.ServeHTTP(rec, req)
					return rec
				}
				require.Equal(t, http.StatusForbidden, get("/ui/catalog/"+priceType.Name).Code, roleName)
				list := get(base + spoof)
				xlsx := get(base + "/excel" + spoof)
				if status != http.StatusOK {
					for _, rec := range []*httptest.ResponseRecorder{list, xlsx} {
						require.Equal(t, status, rec.Code, roleName)
						require.Contains(t, rec.Body.String(), "для параметра PriceType найдено несколько записей")
					}
					continue
				}
				require.Equal(t, status, list.Code, list.Body.String())
				require.Contains(t, list.Body.String(), "PARTY_FIXTURE")
				require.NotContains(t, list.Body.String(), `name="PriceType"`)
				require.Equal(t, status, xlsx.Code, xlsx.Body.String())
				book, err := excelize.OpenReader(bytes.NewReader(xlsx.Body.Bytes()))
				require.NoError(t, err)
				sheet := book.GetSheetName(0)
				rows, err := book.GetRows(sheet)
				require.NoError(t, err)
				matchedRows := 0
				for i, record := range rows {
					if len(record) == 0 || record[0] != "PARTY_FIXTURE" {
						continue
					}
					matchedRows++
					require.GreaterOrEqual(t, len(record), 3)
					require.Equal(t, []string{"3", "30"}, record[1:3])
					cell, err := excelize.CoordinatesToCellName(4, i+1)
					require.NoError(t, err)
					value, err := book.GetCellValue(sheet, cell)
					require.NoError(t, err)
					require.Equal(t, wantPrice, value, roleName)
				}
				require.NoError(t, book.Close())
				require.Equal(t, 1, matchedRows, rows)
			}
		}

		check("", http.StatusOK)
		putType(masterID, "Цена мастера")
		putPrice(masterID.String(), 7)
		check("7", http.StatusOK)
		putType(uuid.New(), "Цена мастера")
		check("", http.StatusForbidden)
	})
}
