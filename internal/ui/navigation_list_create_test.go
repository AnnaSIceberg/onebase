package ui

import (
	"context"
	"net/http"
	"net/url"
	"strings"
	"testing"

	"github.com/ivantit66/onebase/internal/auth"
	"golang.org/x/net/html"
)

func TestCreateFromListToolbarOnly(t *testing.T) {
	f := newNavigationHTTPFixture(t)
	f.server.entitySvc = f.server.newEntityService(nil)
	f.server.cfg.Lang = "ru"
	ctx := context.Background()
	reader := semanticTeacher().Roles[0]
	creator := &auth.Role{Name: "Creator", Permissions: auth.Permission{
		Catalogs:  map[string][]string{"Classes": {"read", "write", "create"}},
		Documents: map[string][]string{"Orders": {"read", "write", "create"}},
	}}
	writer := &auth.Role{Name: "Writer", Permissions: auth.Permission{
		Catalogs:  map[string][]string{"Classes": {"read", "write"}},
		Documents: map[string][]string{"Orders": {"read", "write"}},
	}}
	if err := f.server.authRepo.SyncRoles(ctx, []*auth.Role{reader, creator, writer}); err != nil {
		t.Fatal(err)
	}
	for _, role := range []*auth.Role{creator, writer} {
		login := strings.ToLower(role.Name)
		user, err := f.server.authRepo.Create(ctx, login, "test-password", login, false)
		if err != nil {
			t.Fatal(err)
		}
		if err := f.server.authRepo.AssignRole(ctx, user.ID, role.ID); err != nil {
			t.Fatal(err)
		}
		token, err := f.server.authRepo.CreateSession(ctx, user.ID, auth.SessionMeta{Kind: auth.SessionKindEnterprise})
		if err != nil {
			t.Fatal(err)
		}
		f.cookie[login] = &http.Cookie{Name: "onebase_session", Value: token}
	}
	if err := f.server.store.Migrate(ctx, f.server.reg.Entities()); err != nil {
		t.Fatal(err)
	}

	for _, tc := range []struct {
		login  string
		status int
	}{{"creator", http.StatusOK}, {"writer", http.StatusForbidden}, {"alice", http.StatusForbidden}} {
		t.Run(tc.login, func(t *testing.T) {
			r := f.request(t, "GET", "/ui/?subsystem=School", tc.login, nil)
			if r.Code != http.StatusOK {
				t.Fatalf("sidebar: HTTP %d", r.Code)
			}
			doc, err := html.Parse(strings.NewReader(r.Body.String()))
			if err != nil {
				t.Fatal(err)
			}
			for _, menu := range []struct{ id, path string }{
				{"classes", "/ui/catalog/Classes"}, {"orders", "/ui/document/Orders"},
			} {
				list := semanticFind(doc, "data-nav-id", "cfg:"+menu.id)
				if list == nil || semanticAttr(list, "href") != menu.path+"?subsystem=School" {
					t.Fatalf("%s list link missing or changed", menu.id)
				}
				page := f.request(t, "GET", menu.path+"?subsystem=School", tc.login, nil)
				if page.Code != http.StatusOK {
					t.Fatalf("%s list: HTTP %d", menu.id, page.Code)
				}
				tree, err := html.Parse(strings.NewReader(page.Body.String()))
				if err != nil {
					t.Fatal(err)
				}
				button := semanticFind(tree, "aria-label", "Создать")
				if button == nil || !strings.Contains(semanticAttr(button, "class"), "btn-command-create") ||
					semanticAttr(button.Parent.Parent, "class") != "entity-command-group" {
					t.Fatalf("%s: create button must be first in list toolbar", menu.id)
				}
				for sibling := button.Parent.PrevSibling; sibling != nil; sibling = sibling.PrevSibling {
					if sibling.Type == html.ElementNode {
						t.Fatalf("%s: create action is not the first toolbar control", menu.id)
					}
				}
				if tc.status == http.StatusOK {
					link, err := url.Parse(semanticAttr(button, "href"))
					if err != nil || !strings.EqualFold(link.Path, menu.path+"/new") ||
						link.Query().Get("subsystem") != "School" || link.Query().Get("_list_create") != "1" {
						t.Fatalf("bad form URL %v: %v", link, err)
					}
					if form := f.request(t, "GET", link.String(), tc.login, nil); form.Code != http.StatusOK {
						t.Fatalf("create form: HTTP %d", form.Code)
					}
				} else {
					if button.Data != "button" || !strings.Contains(semanticAttr(button.Parent, "title"), "Недостаточно прав") {
						t.Fatalf("%s: unavailable create action has no explanation", menu.id)
					}
					if _, disabled := keyboardHTMLAttr(button, "disabled"); !disabled {
						t.Fatalf("%s: create button is active without permission", menu.id)
					}
					form := f.request(t, "GET", menu.path+"/new?_list_create=1", tc.login, nil)
					if form.Code != tc.status || !strings.Contains(form.Body.String(), "Недостаточно прав") {
						t.Fatalf("unauthorized create form: HTTP %d", form.Code)
					}
				}
			}
			if strings.Contains(r.Body.String(), "data-nav-list-create") {
				t.Fatal("sidebar still contains a create shortcut")
			}
		})
	}
}
