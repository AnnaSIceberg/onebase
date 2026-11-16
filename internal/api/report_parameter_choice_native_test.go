package api

import (
	"net/http/httptest"
	"net/url"
	"testing"

	"github.com/google/uuid"
	"github.com/ivantit66/onebase/internal/auth"
	"github.com/ivantit66/onebase/internal/metadata"
	reportpkg "github.com/ivantit66/onebase/internal/report"
)

func TestAPIV2ReportChoiceInactiveHistory(t *testing.T) {
	branch := &metadata.Entity{Name: "ApiHistoryBranch", Kind: metadata.KindCatalog, Activity: &metadata.ActivityConfig{Field: "Active", HideFromChoice: true}, Fields: []metadata.Field{{Name: "Name", Type: metadata.FieldTypeString}, {Name: "Active", Type: metadata.FieldTypeBool}}}
	direction := &metadata.Entity{Name: "ApiHistoryDirection", Kind: metadata.KindCatalog, Activity: &metadata.ActivityConfig{Field: "Active", HideFromChoice: true}, Fields: []metadata.Field{{Name: "Name", Type: metadata.FieldTypeString}, {Name: "Active", Type: metadata.FieldTypeBool}, {Name: "Branch", Type: "reference:ApiHistoryBranch", RefEntity: branch.Name}}}
	rep := &reportpkg.Report{Name: "ApiHistoryChoice", Query: "ВЫБРАТЬ Name ИЗ Справочник.ApiHistoryDirection", Params: []reportpkg.Param{{Name: "Branch", Type: "reference:ApiHistoryBranch"}, {Name: "Direction", Type: "reference:ApiHistoryDirection", ChoiceFilter: []reportpkg.ParamChoiceFilter{{Field: "Branch", Param: "Branch"}}}}}
	h, ctx := newAPITestHandlerWithReports(t, []*metadata.Entity{branch, direction}, []*reportpkg.Report{rep}, nil)
	b1, b2, d := uuid.New(), uuid.New(), uuid.New()
	for _, b := range []uuid.UUID{b1, b2} {
		if err := h.store.Upsert(ctx, branch.Name, b, map[string]any{"Name": "Inactive branch", "Active": false}, branch); err != nil {
			t.Fatal(err)
		}
	}
	if err := h.store.Upsert(ctx, direction.Name, d, map[string]any{"Name": "Inactive direction", "Active": false, "Branch": b1}, direction); err != nil {
		t.Fatal(err)
	}
	for _, tc := range []struct {
		name, branch string
		denied       bool
		want         int
	}{
		{name: "matching inactive history", branch: b1.String(), want: 200},
		{name: "inactive mismatched pair", branch: b2.String(), want: 400},
		{name: "empty branch includes history", want: 200},
		{name: "read permission retained", branch: b1.String(), denied: true, want: 400},
	} {
		t.Run(tc.name, func(t *testing.T) {
			q := url.Values{"Branch": {tc.branch}, "Direction": {d.String()}}
			r := reqWithEntity("GET", "/api/v2/report/ApiHistoryChoice?"+q.Encode(), nil, map[string]string{"name": rep.Name}, nil)
			if tc.denied {
				r = withUser(r, apiUser("history-runner", auth.Permission{Reports: map[string][]string{rep.Name: {"run"}}, Catalogs: map[string][]string{direction.Name: {"read"}}}))
			}
			w := httptest.NewRecorder()
			h.runReportV2().ServeHTTP(w, r)
			if w.Code != tc.want {
				t.Fatalf("%d want %d: %s", w.Code, tc.want, w.Body.String())
			}
		})
	}
}
