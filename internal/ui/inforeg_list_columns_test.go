package ui

import (
	"html"
	"net/http"
	"net/http/httptest"
	"os"
	"path/filepath"
	"regexp"
	"strings"
	"testing"
	"time"

	"github.com/go-chi/chi/v5"
	"github.com/google/uuid"
	"github.com/stretchr/testify/require"

	"github.com/ivantit66/onebase/internal/auth"
	"github.com/ivantit66/onebase/internal/dbtest"
	"github.com/ivantit66/onebase/internal/metadata"
	"github.com/ivantit66/onebase/internal/runtime"
	"github.com/ivantit66/onebase/internal/storage"
)

func TestInfoRegListColumnsAndDetailPanelHTTP(t *testing.T) {
	path := filepath.Join(t.TempDir(), "register.yaml")
	require.NoError(t, os.WriteFile(path, []byte(`name: ScopedInfoList
periodic: true
dimensions:
  - {name: Код, type: string}
resources:
  - {name: Подпись, type: string}
  - {name: Показатель, type: number}
  - {name: Заметка, type: string}
  - {name: Цвет, type: string}
list:
  except_roles: [ОператорКЦ, ОператорМессенджера]
  columns:
    - {field: подпись, label: Имя}
    - {field: period, label: Начало работы}
    - {field: Показатель, admin_only: true}
    - {field: Заметка, roles: [Руководитель]}
`), 0o600))
	ir, err := metadata.LoadInfoRegisterFile(path)
	require.NoError(t, err)
	dbtest.ForEachDialect(t, func(t *testing.T, db *storage.DB) {
		s := newSubmitTestServerOn(t, db, nil)
		s.reg.Load(runtime.LoadOptions{InfoRegs: []*metadata.InfoRegister{ir}})
		ctx := t.Context()
		require.NoError(t, db.MigrateInfoRegisters(ctx, []*metadata.InfoRegister{ir}))
		period := time.Date(2026, time.September, 30, 10, 0, 0, 0, time.Local)
		for _, row := range []struct{ key, title string }{{"first", "SELECTED_ROW"}, {"second", "FILTERED_OUT_ROW"}} {
			require.NoError(t, db.InfoRegSet(ctx, ir, map[string]any{"Код": row.key}, map[string]any{
				"Подпись": row.title, "Показатель": 84,
				"Заметка": "CHIEF_NOTE", "Цвет": "LEGACY_COLOR_MARKER",
			}, &period))
		}
		router := chi.NewRouter()
		s.Mount(router)
		oldLayout := []string{"Период", "Код", "Подпись", "Показатель", "Заметка", "Цвет"}
		for _, tc := range []struct {
			name  string
			roles []string
			admin bool
			want  []string
		}{
			{"manager", []string{"Руководитель"}, false, []string{"Имя", "Начало работы", "Заметка"}},
			{"admin", nil, true, []string{"Имя", "Начало работы", "Показатель"}},
			{"unlisted-role", []string{"Другой"}, false, []string{"Имя", "Начало работы"}},
			{"operator", []string{"ОператорКЦ"}, false, oldLayout},
			{"messenger", []string{"ОператорМессенджера"}, false, oldLayout},
			{"mixed", []string{"Руководитель", "ОператорКЦ"}, false, oldLayout},
		} {
			t.Run(tc.name, func(t *testing.T) {
				user := &auth.User{ID: uuid.NewString(), Login: "list-layout-reader", IsAdmin: tc.admin}
				for _, name := range tc.roles {
					user.Roles = append(user.Roles, &auth.Role{Name: name, Permissions: auth.Permission{
						InfoRegs: map[string][]string{ir.Name: {"read"}},
					}})
				}
				req := httptest.NewRequest(http.MethodGet, "/ui/inforeg/ScopedInfoList?flt_%D0%9A%D0%BE%D0%B4=first&from=2026-09-30&to=2026-09-30", nil)
				req = req.WithContext(auth.ContextWithUser(req.Context(), user))
				rec := httptest.NewRecorder()
				router.ServeHTTP(rec, req)
				require.Equal(t, http.StatusOK, rec.Code)
				body := rec.Body.String()
				require.Contains(t, body, "SELECTED_ROW")
				require.NotContains(t, body, "FILTERED_OUT_ROW")
				require.Contains(t, body, `name="flt_Код"`)

				thead := regexp.MustCompile(`(?s)<thead><tr>(.*?)</tr></thead>`).FindStringSubmatch(body)
				require.Len(t, thead, 2)
				var headerLabels []string
				for _, match := range regexp.MustCompile(`<th>([^<]+)</th>`).FindAllStringSubmatch(thead[1], -1) {
					headerLabels = append(headerLabels, html.UnescapeString(match[1]))
				}
				require.Equal(t, tc.want, headerLabels)
				panel := firstDetailPanelData(t, body)
				var detailLabels []string
				for _, tab := range panel.Tabs {
					for _, field := range tab.Fields {
						detailLabels = append(detailLabels, field.Label)
					}
				}
				require.Equal(t, tc.want, detailLabels)
				require.Equal(t, tc.want[0] == "Период", strings.Contains(body, "LEGACY_COLOR_MARKER"))
			})
		}
	})
}
