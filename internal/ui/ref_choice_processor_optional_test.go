package ui

import (
	"net/http"
	"strings"
	"testing"

	"github.com/google/uuid"
	"github.com/ivantit66/onebase/internal/auth"
	"github.com/ivantit66/onebase/internal/dbtest"
	"github.com/ivantit66/onebase/internal/metadata"
	"github.com/ivantit66/onebase/internal/storage"
	"golang.org/x/net/html"
)

func ordinaryOptionalChoice(f kpiChoiceFixture) {
	f.proc.Kind = ""
	f.field.KPIChoice = nil
	f.field.ChoiceFilter[0].SkipWhenEmpty = true
}

func TestProcessorOptionalChoiceHTTP(t *testing.T) {
	dbtest.ForEachDialect(t, func(t *testing.T, db *storage.DB) {
		f := newKPIChoiceFixture(t, db)
		ordinaryOptionalChoice(f)
		for _, tc := range []struct {
			name, source    string
			total           int
			selectedAllowed bool
		}{
			{"explicit empty", "", 3, true},
			{"selected branch", f.b1.String(), 2, true},
			{"changed branch", f.b2.String(), 1, false},
			{"unknown branch", uuid.NewString(), 0, false},
			{"hidden branch", f.hiddenBranch.String(), 0, false},
		} {
			t.Run(tc.name, func(t *testing.T) {
				q := f.query(tc.source)
				q.Set("selected_id", f.d1.String())
				data := decodeChoiceHTTP(t, f.picker(t, f.user, q))
				if data.Total != tc.total || len(data.Items) != tc.total ||
					data.SelectedAllowed == nil || *data.SelectedAllowed != tc.selectedAllowed {
					t.Fatalf("choice or current selection exceeded its scope: %+v", data)
				}
			})
		}
		for _, invalid := range []string{
			"{\"Объект.Branch\":null}", "{\"Объект.Branch\":\"\",\"Объект.Branch\":\"\"}",
			"{\"Объект.Branch\":\" \"}", "{\"Объект.Branch\":\"invalid\"}",
		} {
			q := f.query("")
			q.Set("sources", invalid)
			if w := f.picker(t, f.user, q); w.Code != http.StatusBadRequest {
				t.Fatalf("malformed source accepted: %d", w.Code)
			}
		}
		missing := f.query("")
		missing.Del("sources")
		if got := decodeChoiceHTTP(t, f.picker(t, f.user, missing)); got.Total != 0 {
			t.Fatal("missing input activated the empty case")
		}
		permissions := &f.user.Roles[0].Permissions
		permissions.Catalogs[f.branches.Name] = nil
		if got := decodeChoiceHTTP(t, f.picker(t, f.user, f.query(""))); got.Total != 0 {
			t.Fatal("source read denial activated the empty case")
		}
		permissions.Catalogs[f.branches.Name] = []string{"read"}
		permissions.Catalogs[f.directions.Name] = nil
		if w := f.picker(t, f.user, f.query("")); w.Code != http.StatusForbidden {
			t.Fatalf("target read denial bypassed: %d", w.Code)
		}
		permissions.Catalogs[f.directions.Name] = []string{"read"}
		permissions.FieldAccess = auth.FieldAccess{Catalogs: map[string]auth.FieldPolicies{
			f.directions.Name: {"Branch": {Read: "hide"}},
		}}
		if got := decodeChoiceHTTP(t, f.picker(t, f.user, f.query(""))); got.Total != 0 {
			t.Fatal("hidden target field activated the empty case")
		}
		permissions.FieldAccess = auth.FieldAccess{}

		f.user.Roles[0].Name = "Office"
		f.proc.Forms[0].Roles = []string{"Office"}
		f.proc.Forms = append(f.proc.Forms, &metadata.FormModule{
			Name: "Operator", LayoutKind: f.proc.Forms[0].LayoutKind, Elements: []*metadata.FormElement{
				{Kind: metadata.FormElementField, DataPath: "Объект.Direction"},
			},
		})
		f.user.Roles[0].Name = "Operator"
		if w := f.picker(t, f.user, f.query("")); w.Code != http.StatusBadRequest {
			t.Fatalf("another role's form accepted: %d", w.Code)
		}
		operator := f.serve(t, f.user, "/ui/processor/"+f.proc.Name)
		if operator.Code != 200 || strings.Contains(operator.Body.String(), "data-ref-choice-context=") {
			t.Fatalf("operator fallback received an office choice context: %d", operator.Code)
		}
		f.user.Roles[0].Name = "Office"
		initial := f.serve(t, f.user, "/ui/processor/"+f.proc.Name)
		doc, err := html.Parse(strings.NewReader(initial.Body.String()))
		if err != nil || initial.Code != 200 {
			t.Fatal("office form unavailable")
		}
		selectField := findSelectByName(doc, "Direction")
		if selectField == nil {
			t.Fatal("missing direction control")
		}
		if _, ok := htmlAttribute(selectField, "data-ref-choice-context"); !ok || !strings.Contains(initial.Body.String(), "Other") {
			t.Fatal("initial empty source did not render the office picker")
		}

		f.proc.Forms[0].Elements[0].HiddenWhen = "Истина"
		hidden := f.serve(t, f.user, "/ui/processor/"+f.proc.Name)
		if hidden.Code != http.StatusOK || strings.Contains(hidden.Body.String(), "Other") {
			t.Fatal("initial hidden source rendered choices for every branch")
		}
		permissions.Processors[f.proc.Name] = nil
		if w := f.picker(t, f.user, f.query("")); w.Code != http.StatusBadRequest {
			t.Fatalf("processor run denial bypassed: %d", w.Code)
		}
	})
}
