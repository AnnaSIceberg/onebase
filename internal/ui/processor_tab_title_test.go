package ui

import (
	"net/http"
	"net/http/httptest"
	"testing"

	"github.com/go-chi/chi/v5"
	"github.com/ivantit66/onebase/internal/auth"
	"github.com/ivantit66/onebase/internal/metadata"
	"github.com/ivantit66/onebase/internal/processor"
	"github.com/stretchr/testify/require"
)

func TestManagedProcessorForm_TabTitle(t *testing.T) {
	const title = "Отчет за период (ОТК)"
	form := processorExecutionForm(&metadata.FormElement{
		Kind: metadata.FormElementField, Name: "Период", DataPath: "Объект.Период",
	})
	form.Title = map[string]string{"ru": title}
	proc := &processor.Processor{
		Name: "А_ФормаОтчетаОТК", Title: "Отчет для ОТК",
		Params: []processor.Param{{Name: "Период", Type: "string"}},
		Forms:  []*metadata.FormModule{form},
	}
	s, _ := newProcessorFormEventExecutionServer(t, proc, nil)
	s.cfg.UIExceptRoles = []string{"ОператорКЦ", "ОператорМессенджера"}
	router := chi.NewRouter()
	s.Mount(router)

	for _, role := range []string{"ОТКРуководитель", "ОператорКЦ", "ОператорМессенджера"} {
		t.Run(role, func(t *testing.T) {
			user := &auth.User{ID: "test-user", Login: "test-user", Roles: []*auth.Role{{
				Name:        role,
				Permissions: auth.Permission{Processors: map[string][]string{proc.Name: {"run"}}},
			}}}
			req := httptest.NewRequest(http.MethodGet, "/ui/processor/"+proc.Name+"?Период=2026-09", nil)
			rec := httptest.NewRecorder()
			router.ServeHTTP(rec, req.WithContext(auth.ContextWithUser(req.Context(), user)))
			require.Equal(t, http.StatusOK, rec.Code)
			body := rec.Body.String()
			require.Contains(t, body, `value="2026-09"`)
			if role == "ОТКРуководитель" {
				require.Contains(t, body, `<meta name="ob-tab-title" content="`+title+`">`)
			} else {
				require.NotContains(t, body, `<meta name="ob-tab-title"`)
			}
		})
	}
}
