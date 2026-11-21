package ui

import (
	"net/http"
	"testing"

	"github.com/google/uuid"
	"github.com/stretchr/testify/require"

	"github.com/ivantit66/onebase/internal/auth"
	"github.com/ivantit66/onebase/internal/metadata"
	"github.com/ivantit66/onebase/internal/processor"
)

func TestManagedFormHeadingOmitsTechnicalBadgeHTTP(t *testing.T) {
	articleForm := &metadata.FormModule{
		Name: "ФормаНоменклатуры", EntityName: "Номенклатура", Kind: "object", LayoutKind: metadata.FormLayoutManaged,
		Elements: []*metadata.FormElement{{Kind: metadata.FormElementField, Name: "Название", DataPath: "Объект.Наименование"}},
	}
	ent := &metadata.Entity{
		Name: "Номенклатура", Kind: metadata.KindCatalog, Forms: []*metadata.FormModule{articleForm},
		Fields: []metadata.Field{{Name: "Наименование", Type: metadata.FieldTypeString}},
	}
	s, ctx := newSubmitTestServer(t, []*metadata.Entity{ent})
	s.cfg.UIExceptRoles = []string{"ОператорКЦ", "ОператорМессенджера"}
	id := uuid.New()
	require.NoError(t, s.store.Upsert(ctx, ent.Name, id, map[string]any{"Наименование": "Артикул 42"}, ent))
	proc := &processor.Processor{
		Name: "PrintLabels", Title: "Печать этикеток",
		Forms: []*metadata.FormModule{processorExecutionForm(&metadata.FormElement{
			Kind: metadata.FormElementGroupBox, Name: "Этикетки",
		})},
	}
	s.reg.LoadProcessors([]*processor.Processor{proc})
	user := func(role string) *auth.User {
		return &auth.User{Login: role, Roles: []*auth.Role{{Name: role, Permissions: auth.Permission{
			Catalogs:   map[string][]string{ent.Name: {"read", "write"}},
			Processors: map[string][]string{proc.Name: {"run"}},
		}}}}
	}
	get := func(actor *auth.User, path string) string {
		t.Helper()
		w := tabularHTTP(s, actor, "managed-heading", http.MethodGet, path, nil)
		require.Equal(t, http.StatusOK, w.Code)
		return w.Body.String()
	}
	warehouse := user("Кладовщик")
	listPath := "/ui/catalog/" + ent.Name
	require.Contains(t, get(warehouse, listPath), id.String())
	cardPath := listPath + "/" + id.String()
	procPath := "/ui/processor/" + proc.Name
	expect := func(page, title string, operator bool) {
		t.Helper()
		require.Contains(t, page, "data-ob-form-title>")
		require.Contains(t, page, title)
		require.Contains(t, page, `id="ob-managed-config"`)
		require.Contains(t, page, "data-ob-close-tab")
		if operator {
			require.Contains(t, page, "data-ob-legacy-entity-ui")
			require.Contains(t, page, "◇ managed")
		} else {
			require.NotContains(t, page, "◇ managed")
		}
	}
	expect(get(warehouse, cardPath), "Артикул 42", false)
	expect(get(warehouse, procPath), proc.Title, false)
	for _, role := range s.cfg.UIExceptRoles {
		expect(get(user(role), cardPath), "Артикул 42", true)
		expect(get(user(role), procPath), proc.Title, true)
	}
}
