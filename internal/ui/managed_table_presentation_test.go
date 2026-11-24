package ui

import (
	"net/http"
	"net/http/httptest"
	"strings"
	"testing"

	"github.com/go-chi/chi/v5"
	"github.com/ivantit66/onebase/internal/auth"
	"github.com/ivantit66/onebase/internal/metadata"
)

func TestManagedTablePresentationHTTP(t *testing.T) {
	for _, mode := range []string{"enhanced", "ОператорКЦ", "ОператорМессенджера"} {
		t.Run(mode, func(t *testing.T) {
			ent := layoutTestEntity(
				&metadata.FormElement{
					Kind: metadata.FormElementField, Name: "Notes", DataPath: "Object.Notes",
					Multiline: true, Rows: 4, Enhanced: &metadata.FormPresentation{Rows: 2},
				},
			)
			form := ent.Forms[0]
			form.Elements = append(form.Elements,
				&metadata.FormElement{
					Kind: metadata.FormElementTablePart, Name: "Lines", DataPath: "Object.Lines",
					NoGrid: true, Enhanced: &metadata.FormPresentation{FitWindow: true},
				},
				&metadata.FormElement{
					Kind: metadata.FormElementTablePart, Name: "Values", DataPath: "Form.Values",
					Rows: 3, Enhanced: &metadata.FormPresentation{FitWindow: true},
				},
			)
			form.Attributes = []*metadata.FormAttribute{{
				Name: "Values", TypeRef: "ValueTable",
				Columns: []*metadata.FormAttributeColumn{{Name: "Comment", TypeRef: "string"}},
			}}
			ent.Fields = append(ent.Fields, metadata.Field{Name: "Notes", Type: metadata.FieldTypeString})
			ent.TableParts = []metadata.TablePart{
				{Name: "Lines", Fields: []metadata.Field{{Name: "Item", Type: metadata.FieldTypeString}}},
			}
			ent.Forms[0].HeaderInline = true
			srv, _ := newSubmitTestServer(t, []*metadata.Entity{ent})
			srv.cfg.UIExceptRoles = []string{"ОператорКЦ", "ОператорМессенджера"}
			router := chi.NewRouter()
			srv.MountStatic(router)
			srv.Mount(router)
			req := httptest.NewRequest(http.MethodGet, "/ui/catalog/"+strings.ToLower(ent.Name)+"/new", nil)
			if mode != "enhanced" {
				u := &auth.User{Login: "operator", Roles: []*auth.Role{{Name: mode, Permissions: auth.Permission{
					Catalogs: map[string][]string{ent.Name: {"read", "write"}},
				}}}}
				req = req.WithContext(auth.ContextWithUser(req.Context(), u))
			}
			rec := httptest.NewRecorder()
			router.ServeHTTP(rec, req)
			if rec.Code != http.StatusOK {
				t.Fatalf("GET form: %d", rec.Code)
			}
			body := rec.Body.String()
			want := mode == "enhanced"
			for _, marker := range []string{
				`rows="2"`, `class="ob-object-inline-header"`,
				`data-ob-fit-grid="1"`, `src="/static/managed-table-paper.js"`,
			} {
				if strings.Contains(body, marker) != want {
					t.Errorf("%s presence must be %t", marker, want)
				}
			}
			if want && strings.Count(body, `data-ob-fit-grid="1"`) != 2 {
				t.Error("both the simple table part and ValueTable need a fitted viewport")
			}
			if !want && !strings.Contains(body, `rows="4"`) {
				t.Error("operator`s configured textarea rows changed")
			}
		})
	}
}
