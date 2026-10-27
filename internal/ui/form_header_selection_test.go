package ui

import (
	"context"
	"encoding/json"
	"fmt"
	"io"
	"net/http"
	"net/http/httptest"
	"net/url"
	"os"
	"os/exec"
	"path/filepath"
	"strings"
	"testing"
	"time"

	"github.com/go-chi/chi/v5"
	"github.com/google/uuid"
	"github.com/ivantit66/onebase/internal/auth"
	"github.com/ivantit66/onebase/internal/dbtest"
	"github.com/ivantit66/onebase/internal/dsl/interpreter"
	"github.com/ivantit66/onebase/internal/metadata"
	"github.com/ivantit66/onebase/internal/runtime"
	"github.com/ivantit66/onebase/internal/storage"
	"github.com/ivantit66/onebase/internal/testutil/browseracceptance"
)

// The mounted GET is always tested. With browser acceptance enabled the same
// fixture also exercises actual clicks, FormData, form-event POST and DSL on
// both databases, including rows rebuilt by the production JavaScript.
func TestManagedHeaderSelectionHTTPMatrix(t *testing.T) {
	dbtest.ForEachDialect(t, func(t *testing.T, db *storage.DB) {
		var entities []*metadata.Entity
		paths := map[string]string{}
		for _, mode := range []string{"header", "local", "mixed", "grid", "readonly", "hidden", "none"} {
			command := &metadata.FormElement{Kind: metadata.FormElementButton, Name: "SelectRows", Placement: metadata.FormPlacementHeader,
				Handlers: map[metadata.FormEventType]string{metadata.FormEventOnClick: "SelectRows"}}
			if mode == "local" {
				command.Placement = ""
			}
			table := &metadata.FormElement{Kind: metadata.FormElementTablePart, Name: "Table", DataPath: "Объект.Lines",
				NoGrid: mode != "grid", ReadOnly: mode == "readonly", Children: []*metadata.FormElement{command}}
			if mode == "none" {
				table.Children = nil
			}
			if mode == "mixed" {
				table.Children = append(table.Children, &metadata.FormElement{Kind: metadata.FormElementButton, Name: "LocalRows",
					Handlers: map[metadata.FormEventType]string{metadata.FormEventOnClick: "SelectRows"}})
			}
			group := &metadata.FormElement{Kind: metadata.FormElementGroupBox, Name: "Group", Children: []*metadata.FormElement{table}}
			if mode == "hidden" {
				group.HiddenWhen = `Наименование = "hidden"`
			}
			form := &metadata.FormModule{Name: "Object", Kind: "object", LayoutKind: metadata.FormLayoutManaged,
				Elements: []*metadata.FormElement{
					{Kind: metadata.FormElementField, Name: "Name", DataPath: "Объект.Наименование"}, group,
					{Kind: metadata.FormElementTablePart, Name: "OtherTable", DataPath: "Объект.Other", NoGrid: true},
				}}
			form.ProgramAST = mustParse(t, `Процедура SelectRows()
 Сообщить(ИмяТабличнойЧасти);
 Если ВыделенныеСтроки = Неопределено Тогда
  Сообщить(0);
  Возврат;
 КонецЕсли;
 Сообщить(ВыделенныеСтроки.Количество());
 Для Каждого СтрокаТЧ Из ВыделенныеСтроки Цикл
  Сообщить(СтрокаТЧ.Note);
 КонецЦикла;
КонецПроцедуры`)
			entity := &metadata.Entity{Name: "HeaderSelection" + mode, Kind: metadata.KindCatalog,
				Fields: []metadata.Field{{Name: "Наименование", Type: metadata.FieldTypeString}},
				TableParts: []metadata.TablePart{
					{Name: "Lines", Fields: []metadata.Field{{Name: "Note", Type: metadata.FieldTypeString}}},
					{Name: "Other", Fields: []metadata.Field{{Name: "Note", Type: metadata.FieldTypeString}}},
				}, Forms: []*metadata.FormModule{form}}
			form.EntityName = entity.Name
			entities = append(entities, entity)
		}
		if err := db.Migrate(t.Context(), entities); err != nil {
			t.Fatal(err)
		}
		for _, entity := range entities {
			id := uuid.New()
			mode := strings.TrimPrefix(entity.Name, "HeaderSelection")
			if err := db.Upsert(t.Context(), entity.Name, id, map[string]any{"Наименование": mode}, entity); err != nil {
				t.Fatal(err)
			}
			for _, tp := range entity.TableParts {
				if err := db.UpsertTablePartRows(t.Context(), entity.Name, tp.Name, id,
					[]map[string]any{{"Note": "A"}, {"Note": "B"}}, tp); err != nil {
					t.Fatal(err)
				}
			}
			paths[mode] = "/ui/catalog/" + entity.Name + "/" + id.String()
		}
		reg := runtime.NewRegistry()
		reg.Load(runtime.LoadOptions{Entities: entities})
		interp := interpreter.New()
		interp.LookupProc = reg.GetModuleProc
		s := New(reg, db, interp, nil, Config{}, nil)
		router := chi.NewRouter()
		router.Use(func(next http.Handler) http.Handler {
			return http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
				if r.URL.Query().Get("read_only") == "1" {
					permissions := map[string][]string{}
					for _, entity := range entities {
						permissions[entity.Name] = []string{"read"}
					}
					u := &auth.User{Roles: []*auth.Role{{Permissions: auth.Permission{Catalogs: permissions}}}}
					r = r.WithContext(auth.ContextWithUser(r.Context(), u))
				}
				next.ServeHTTP(w, r)
			})
		})
		s.MountStatic(router)
		s.Mount(router)
		server := httptest.NewServer(router)
		defer func() { s.CloseEventStreams(); server.Close(); s.Close() }()
		paths["popup"] = paths["header"] + "?_popup=1"
		paths["readuser"] = paths["header"] + "?read_only=1"
		for mode, path := range paths {
			t.Run(mode, func(t *testing.T) {
				resp, err := server.Client().Get(server.URL + path)
				if err != nil {
					t.Fatal(err)
				}
				defer func() {
					if err := resp.Body.Close(); err != nil {
						t.Error(err)
					}
				}()
				body, err := io.ReadAll(resp.Body)
				if err != nil || resp.StatusCode != http.StatusOK {
					t.Fatalf("GET: %d %v", resp.StatusCode, err)
				}
				html := string(body)
				if mode == "grid" {
					if !strings.Contains(html, `data-sg-cmd="1"`) {
						t.Fatal("header command lost SlickGrid selection marker")
					}
				} else {
					want := 2
					if mode == "none" || mode == "hidden" {
						want = 0
					}
					if got := strings.Count(html, `class="_tp-sel"`); got != want {
						t.Fatalf("row selectors=%d, want %d", got, want)
					}
					if strings.Contains(html, `data-tp-cmd="1"`) != (want > 0) {
						t.Fatal("row selection marker disagrees with commands")
					}
				}
				if mode != "none" && mode != "local" {
					if got := strings.Count(html, `data-ob-el="SelectRows"`); got != 1 {
						t.Fatalf("header command count=%d", got)
					}
				}
				// hidden_when controls presentation, while readonly and role
				// permissions also reject a forged event at the HTTP boundary.
				if mode == "none" || mode == "hidden" {
					return
				}
				u, err := url.Parse(path)
				if err != nil {
					t.Fatal(err)
				}
				parts := strings.Split(u.Path, "/")
				for _, selection := range []string{"1", "", "0,1"} {
					posted := url.Values{"_id": {parts[4]}, "_kind": {"object"}, "_element": {"SelectRows"},
						"_event": {"Нажатие"}, "_tp": {"Lines"}, "_tp_selected": {selection},
						"tp_json.Lines": {`[{"Note":"A"},{"Note":"B"}]`}}
					r, err := server.Client().PostForm(server.URL+"/ui/catalog/"+parts[3]+"/form-event?"+u.RawQuery, posted)
					if err != nil {
						t.Fatal(err)
					}
					data, err := io.ReadAll(r.Body)
					if closeErr := r.Body.Close(); closeErr != nil {
						t.Fatal(closeErr)
					}
					if err != nil {
						t.Fatal(err)
					}
					result := decodeFormEventResponse(t, data)
					if mode == "readonly" || mode == "readuser" {
						if result.OK || len(result.Messages) != 0 {
							t.Fatalf("restricted table command accepted: %+v", result)
						}
						continue
					}
					want := map[string]string{"1": "Lines|1|B", "": "Lines|0", "0,1": "Lines|2|A|B"}[selection]
					if !result.OK || strings.Join(result.Messages, "|") != want {
						t.Fatalf("selection %q: %+v, want %s", selection, result, want)
					}
				}
			})
		}
		if os.Getenv("ONEBASE_BROWSER_ACCEPTANCE") == "1" {
			data, err := json.Marshal(paths)
			if err != nil {
				t.Fatal(err)
			}
			ctx, cancel := context.WithTimeout(context.Background(), 3*time.Minute)
			defer cancel()
			cmd := exec.CommandContext(ctx, "node", "node_modules/@playwright/test/cli.js", "test", "--config", "acceptance.config.js", "header-selection.spec.js")
			cmd.Dir = filepath.Join(browseracceptance.Root(), "e2e")
			cmd.Env = append(os.Environ(), "OB_ACCEPTANCE_SUITE=header-selection", "OB_ACCEPTANCE_URL="+server.URL, "OB_HEADER_SELECTION_PATHS="+string(data))
			output, err := cmd.CombinedOutput()
			t.Logf("%s", output)
			if err != nil {
				t.Fatal(fmt.Errorf("header selection browser: %w", err))
			}
		}
	})
}
