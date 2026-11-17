package api

import (
	"encoding/json"
	"net/http"
	"net/http/httptest"
	"net/url"
	"testing"

	"github.com/google/uuid"
	"github.com/ivantit66/onebase/internal/metadata"
	reportpkg "github.com/ivantit66/onebase/internal/report"
	"github.com/ivantit66/onebase/internal/storage"
)

func TestReportResultFilterAPIPaginationAndPresetGate(t *testing.T) {
	entity := &metadata.Entity{Name: "FilterProbe", Kind: metadata.KindCatalog, Presentation: []string{"Label"}, Fields: []metadata.Field{{Name: "Label", Type: "string"}, {Name: "Bucket", Type: "string"}, {Name: "Amount", Type: "number"}, {Name: "Secret", Type: "string", PII: true}}}
	rep := &reportpkg.Report{Name: "FilterReport", Query: "ВЫБРАТЬ П.Bucket КАК bucket, П.Amount КАК amount, П.Secret КАК contact ИЗ Справочник.FilterProbe КАК П УПОРЯДОЧИТЬ ПО П.Bucket"}
	h, ctx := newAPITestHandlerWithReports(t, []*metadata.Entity{entity}, []*reportpkg.Report{rep}, nil)
	if err := h.store.EnsureServiceSchema(ctx); err != nil {
		t.Fatal(err)
	}
	for i, name := range []string{"DROP_A", "DROP_B", "KEEP_C", "KEEP_D"} {
		if err := h.store.Upsert(ctx, entity.Name, uuid.New(), map[string]any{"Bucket": name, "Amount": i + 1, "Secret": "private"}, entity); err != nil {
			t.Fatal(err)
		}
	}
	request := func(q url.Values) *httptest.ResponseRecorder {
		r := reqWithEntity("GET", "/api/v2/report/FilterReport?"+q.Encode(), nil, map[string]string{"name": rep.Name}, nil)
		w := httptest.NewRecorder()
		h.runReportV2().ServeHTTP(w, r)
		return w
	}
	settings := `{"filters":[{"field":"bucket","op":"contains","value":"KEEP"},{"field":"amount","op":"ge","value":"3"}]}`
	for _, tc := range []struct{ page, want string }{{"1", "KEEP_C"}, {"2", "KEEP_D"}} {
		w := request(url.Values{"__settings": {settings}, "limit": {"1"}, "page": {tc.page}})
		if w.Code != http.StatusOK {
			t.Fatalf("api %d %s", w.Code, w.Body.String())
		}
		var got struct {
			Data []map[string]any `json:"data"`
			Meta restV2Meta       `json:"meta"`
		}
		if err := json.Unmarshal(w.Body.Bytes(), &got); err != nil {
			t.Fatal(err)
		}
		if len(got.Data) != 1 || got.Data[0]["bucket"] != tc.want || got.Meta.Total != 2 || got.Meta.TotalPages != 2 || got.Data[0]["contact"] != nil {
			t.Fatalf("pagination/PII %#v %+v", got.Data, got.Meta)
		}
	}
	for _, raw := range []string{"{", `{"filters":[{"field":"contact","op":"eq","value":"private"}]}`, `{"filters":[{"field":"amount","op":"contains","value":"1"}]}`} {
		if w := request(url.Values{"__settings": {raw}}); w.Code != 400 {
			t.Fatalf("invalid filter %d %s", w.Code, w.Body.String())
		}
	}
	if w := request(url.Values{"__preset": {"not-owned"}}); w.Code != 400 {
		t.Fatal("unknown preset silently ignored", w.Code)
	}
}

func TestReportResultFilterAPIVariantAndPresetSchema(t *testing.T) {
	entity := &metadata.Entity{Name: "VariantFilterRow", Kind: metadata.KindCatalog, Fields: []metadata.Field{{Name: "Bucket", Type: "string"}, {Name: "Amount", Type: "number"}}}
	rep := &reportpkg.Report{Name: "VariantFilter", Query: "ВЫБРАТЬ Bucket КАК value ИЗ Справочник.VariantFilterRow", Variants: []reportpkg.ReportVariant{{Name: "numeric", Query: "ВЫБРАТЬ Amount КАК value ИЗ Справочник.VariantFilterRow"}}}
	h, ctx := newAPITestHandlerWithReports(t, []*metadata.Entity{entity}, []*reportpkg.Report{rep}, nil)
	if err := h.store.EnsureServiceSchema(ctx); err != nil {
		t.Fatal(err)
	}
	if err := h.store.Upsert(ctx, entity.Name, uuid.New(), map[string]any{"Bucket": "KEEP", "Amount": 7}, entity); err != nil {
		t.Fatal(err)
	}
	preset, err := h.store.SaveReportPreset(ctx, storage.ReportPreset{Report: rep.Name, User: "", Name: "Numeric", IsDefault: true, SettingsJSON: `{"variant":"numeric","filters":[{"field":"value","op":"ge","value":"7"}]}`})
	if err != nil {
		t.Fatal(err)
	}
	for _, tc := range []struct {
		name   string
		query  url.Values
		status int
		want   any
	}{
		{"default-preset", nil, 200, "7"},
		{"owned-preset", url.Values{"__preset": {preset}}, 200, "7"},
		// The same output alias is not a type proof for another variant.
		{"main-invalid-numeric-filter", url.Values{"__variant": {""}}, 400, nil},
		{"explicit-main", url.Values{"__variant": {""}, "__settings": {`{"variant":"numeric","filters":[{"field":"value","op":"contains","value":"KEEP"}]}`}}, 200, "KEEP"},
		{"numeric-invalid-string-filter", url.Values{"variant": {"numeric"}, "__settings": {`{"filters":[{"field":"value","op":"contains","value":"KEEP"}]}`}}, 400, nil},
		{"unknown-variant", url.Values{"variant": {"missing"}}, 400, nil},
	} {
		t.Run(tc.name, func(t *testing.T) {
			r := reqWithEntity("GET", "/api/v2/report/"+rep.Name+"?"+tc.query.Encode(), nil, map[string]string{"name": rep.Name}, nil)
			w := httptest.NewRecorder()
			h.runReportV2().ServeHTTP(w, r)
			if w.Code != tc.status {
				t.Fatalf("HTTP %d want %d: %s", w.Code, tc.status, w.Body.String())
			}
			if tc.status != 200 {
				return
			}
			var got struct {
				Data []map[string]any `json:"data"`
			}
			if err := json.Unmarshal(w.Body.Bytes(), &got); err != nil {
				t.Fatal(err)
			}
			if len(got.Data) != 1 || got.Data[0]["value"] != tc.want {
				t.Fatalf("wrong variant result: %s", w.Body.String())
			}
		})
	}
}

func TestReportResultFilterAPICompositionBeforeTotals(t *testing.T) {
	entity := &metadata.Entity{Name: "FilterTotals", Kind: metadata.KindCatalog, Fields: []metadata.Field{{Name: "Bucket", Type: "string"}, {Name: "Amount", Type: "number"}}}
	rep := &reportpkg.Report{Name: "FilterTotals", Query: "ВЫБРАТЬ Bucket, Amount ИЗ Справочник.FilterTotals", Composition: &reportpkg.Composition{Groupings: []string{"Bucket"}, Measures: []reportpkg.Measure{{Field: "Amount", Agg: "sum"}}}}
	h, ctx := newAPITestHandlerWithReports(t, []*metadata.Entity{entity}, []*reportpkg.Report{rep}, nil)
	if err := h.store.EnsureServiceSchema(ctx); err != nil {
		t.Fatal(err)
	}
	for _, row := range []map[string]any{{"Bucket": "KEEP", "Amount": 2}, {"Bucket": "KEEP", "Amount": 4}, {"Bucket": "DROP", "Amount": 100}} {
		if err := h.store.Upsert(ctx, entity.Name, uuid.New(), row, entity); err != nil {
			t.Fatal(err)
		}
	}
	q := url.Values{"composition": {"1"}, "limit": {"1"}, "__settings": {`{"filters":[{"field":"Bucket","op":"eq","value":"KEEP"},{"field":"Amount","op":"gt","value":"2"}]}`}}
	r := reqWithEntity("GET", "/api/v2/report/"+rep.Name+"?"+q.Encode(), nil, map[string]string{"name": rep.Name}, nil)
	w := httptest.NewRecorder()
	h.runReportV2().ServeHTTP(w, r)
	if w.Code != 200 {
		t.Fatalf("composition: %d %s", w.Code, w.Body.String())
	}
	var got struct {
		Data struct {
			Result struct {
				Grand    map[string]string `json:"grand"`
				RowCount int               `json:"row_count"`
			} `json:"result"`
		} `json:"data"`
		Meta restV2Meta `json:"meta"`
	}
	if err := json.Unmarshal(w.Body.Bytes(), &got); err != nil {
		t.Fatal(err)
	}
	if got.Data.Result.RowCount != 1 || got.Data.Result.Grand["Amount"] != "4" || got.Meta.Total != 1 {
		t.Fatalf("unfiltered totals: %s", w.Body.String())
	}
}
