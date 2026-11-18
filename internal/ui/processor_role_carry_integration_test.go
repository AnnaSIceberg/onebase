package ui

import (
	"net/http"
	"net/http/httptest"
	"net/url"
	"strings"
	"testing"

	"github.com/ivantit66/onebase/internal/auth"
	"github.com/ivantit66/onebase/internal/dbtest"
	"github.com/ivantit66/onebase/internal/metadata"
	"github.com/ivantit66/onebase/internal/processor"
	"github.com/ivantit66/onebase/internal/storage"
)

// The selected form must allocate, seal and consume the same carry namespace
// on GET, events and run. A role-form attribute deliberately occupies the
// ordinary carry name; consulting the plain form loses the server state.
func TestProcessorRoleFormCarryThroughHTTP(t *testing.T) {
	dbtest.ForEachDialect(t, func(t *testing.T, db *storage.DB) {
		plain := processorExecutionForm(&metadata.FormElement{Kind: metadata.FormElementButton, Name: "Check", Handlers: map[metadata.FormEventType]string{metadata.FormEventOnClick: "Check"}})
		plain.Name = "Plain"
		compact := processorExecutionForm(&metadata.FormElement{Kind: metadata.FormElementButton, Name: "Check", Handlers: map[metadata.FormEventType]string{metadata.FormEventOnClick: "Check"}})
		compact.Name = "Compact"
		compact.Roles = []string{"OTK"}
		compact.ExceptRoles = []string{"Operator", "Messenger"}
		compact.IncludeAdmin = true
		compact.Attributes = []*metadata.FormAttribute{{Name: "_ob_carry", TypeRef: "string"}}
		for _, form := range []*metadata.FormModule{plain, compact} {
			form.ProgramAST = mustParse(t, "Процедура Check()\n Сообщить(Строка(Объект.Flag));\nКонецПроцедуры")
		}
		proc := &processor.Processor{Name: "RoleCarryProbe", Forms: []*metadata.FormModule{plain, compact}, Params: []processor.Param{{Name: "Flag", Type: "bool", Default: true}}}
		router := mountCarryScopeProcessor(t, db, proc)
		for _, profile := range []struct {
			name  string
			roles []string
			admin bool
		}{
			{"otk", []string{"OTK"}, false}, {"admin", nil, true},
			{"operator", []string{"Operator"}, false}, {"messenger", []string{"Messenger"}, false},
			{"mixed", []string{"OTK", "Messenger"}, false}, {"admin-operator", []string{"Operator"}, true},
		} {
			t.Run(profile.name, func(t *testing.T) {
				user := &auth.User{ID: profile.name, Login: profile.name, IsAdmin: profile.admin,
					Roles: []*auth.Role{{Name: "Access", Permissions: auth.Permission{Processors: map[string][]string{proc.Name: {"run"}}}}}}
				for _, role := range profile.roles {
					user.Roles = append(user.Roles, &auth.Role{Name: role})
				}
				handler := http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
					router.ServeHTTP(w, r.WithContext(auth.ContextWithUser(r.Context(), user)))
				})
				name, token := carryScopePage(t, handler, proc.Name)
				body := processorClickBody("Check")
				body.Set(name, token)
				response := carryScopePost(t, handler, proc.Name, body)
				if strings.Join(response.Messages, "|") != "true" {
					t.Fatalf("selected-form state lost in event: %v", response.Messages)
				}
				runBody := url.Values{name: {token}}
				request := httptest.NewRequest(http.MethodPost, "/ui/processor/"+proc.Name, strings.NewReader(runBody.Encode()))
				request.Header.Set("Content-Type", "application/x-www-form-urlencoded")
				recorder := httptest.NewRecorder()
				handler.ServeHTTP(recorder, request)
				if recorder.Code != http.StatusOK || !strings.Contains(recorder.Body.String(), "carry-run:true") {
					t.Fatalf("selected-form state lost on run: status=%d", recorder.Code)
				}
			})
		}
	})
}
