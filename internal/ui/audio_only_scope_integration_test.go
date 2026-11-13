package ui

import (
	"github.com/go-chi/chi/v5"
	"github.com/google/uuid"
	"github.com/ivantit66/onebase/internal/auth"
	"github.com/ivantit66/onebase/internal/metadata"
	"net/http"
	"net/http/httptest"
	"strings"
	"testing"
)

func TestAudioOnlyHostFormScopeThroughHTTP(t *testing.T) {
	e := &metadata.Entity{Name: "RecordingScopeProbe", Kind: metadata.KindCatalog, AudioPlayer: true, Fields: []metadata.Field{{Name: "Наименование", Type: metadata.FieldTypeString}}}
	s, ctx := newSubmitTestServer(t, []*metadata.Entity{e})
	id := uuid.New()
	if err := s.store.Upsert(ctx, e.Name, id, map[string]any{"Наименование": "Synthetic call"}, e); err != nil {
		t.Fatal(err)
	}
	s.cfg.UIEnhancements = true
	s.cfg.UIRoles = []string{"OTK", "Chief"}
	s.cfg.UIExceptRoles = []string{"Operator", "Messenger"}
	s.cfg.UIFormRoles = []string{}
	s.cfg.UIFormExceptRoles = []string{"Operator", "Messenger"}
	router := chi.NewRouter()
	s.Mount(router)
	for _, p := range []struct {
		name        string
		roles       []string
		admin, only bool
	}{
		{"otk", []string{"OTK"}, false, true}, {"chief", []string{"Chief"}, false, true}, {"other", []string{"Warehouse"}, false, true}, {"admin", nil, true, true},
		{"operator", []string{"Operator"}, false, false}, {"messenger", []string{"Messenger"}, false, false}, {"pair", []string{"Operator", "Messenger"}, false, false},
		{"mixed", []string{"OTK", "Messenger"}, false, false}, {"admin-operator", []string{"Operator"}, true, false},
	} {
		t.Run(p.name, func(t *testing.T) {
			u := &auth.User{ID: p.name, Login: p.name, IsAdmin: p.admin}
			for _, name := range p.roles {
				u.Roles = append(u.Roles, &auth.Role{Name: name, Permissions: auth.Permission{Catalogs: map[string][]string{e.Name: {"read", "write"}}}})
			}
			get := func(hide bool) string {
				s.cfg.HideAttachments = hide
				r := httptest.NewRequest(http.MethodGet, "/ui/catalog/"+e.Name+"/"+id.String(), nil)
				r = r.WithContext(auth.ContextWithUser(r.Context(), u))
				rec := httptest.NewRecorder()
				router.ServeHTTP(rec, r)
				if rec.Code != 200 {
					t.Fatalf("form: %d %.120s", rec.Code, rec.Body.String())
				}
				return rec.Body.String()
			}
			before, after := get(false), get(true)
			audio := strings.Contains(after, "<div data-ob-audio ")
			if audio != p.only || strings.Contains(after, "/static/audio.js") != p.only {
				t.Fatal("audio-only scope mismatch")
			}
			if strings.Contains(after, "data-ob-attachments") == p.only {
				t.Fatal("generic attachment panel scope mismatch")
			}
			if !p.only && before != after {
				t.Fatal("excluded operator recording HTML changed")
			}
			if p.only && (!strings.Contains(after, `preload="none"`) || strings.Contains(after, `data-ob-audio-media controls preload="none" src=`)) {
				t.Fatal("audio-only is not lazy")
			}
		})
	}
}
