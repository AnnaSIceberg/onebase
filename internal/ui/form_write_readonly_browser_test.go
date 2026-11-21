package ui

import (
	"context"
	"encoding/json"
	"net/http"
	"net/http/httptest"
	"os"
	"path/filepath"
	"strings"
	"testing"
	"time"

	"github.com/go-chi/chi/v5"
	"github.com/google/uuid"

	"github.com/ivantit66/onebase/internal/auth"
	"github.com/ivantit66/onebase/internal/dsl/interpreter"
	"github.com/ivantit66/onebase/internal/dsl/loader"
	"github.com/ivantit66/onebase/internal/metadata"
	"github.com/ivantit66/onebase/internal/runtime"
	"github.com/ivantit66/onebase/internal/storage"
)

func TestManagedWriteActionsReadOnlyBrowser(t *testing.T) {
	browser := newMobileLayoutBrowser(t, managedLayoutTestBrowser(t))
	dir := t.TempDir()
	yamlPath := filepath.Join(dir, "object.form.yaml")
	formYAML := `schema: onebase.form/v1
form:
  name: ФормаДокумента
  kind: object
  entity: Предоплата
  write_actions_readonly_when: ТолькоПросмотр
attributes:
  - name: ТолькоПросмотр
    type: boolean
events:
  ПриСозданииНаСервере: ПриСозданииНаСервере
elements:
  - kind: ПолеВвода
    name: Описание
    data_path: Объект.Описание
  - kind: Кнопка
    name: РазрешитьЗапись
    events: {Нажатие: РазрешитьЗапись}
actions:
  ok: {post: true}
`
	formOS := `Процедура ПриСозданииНаСервере()
	Объект.ТолькоПросмотр = Объект.Просмотр = Истина;
КонецПроцедуры

Процедура РазрешитьЗапись()
	Объект.ТолькоПросмотр = Ложь;
КонецПроцедуры
`
	if err := os.WriteFile(yamlPath, []byte(formYAML), 0o600); err != nil {
		t.Fatal(err)
	}
	if err := os.WriteFile(filepath.Join(dir, "object.form.os"), []byte(formOS), 0o600); err != nil {
		t.Fatal(err)
	}
	form, err := loader.NewManagedFormLoader().LoadFormFile(yamlPath, "Предоплата")
	if err != nil {
		t.Fatal(err)
	}
	doc := &metadata.Entity{
		Name: "Предоплата", Kind: metadata.KindDocument, Posting: true, Forms: []*metadata.FormModule{form},
		Fields: []metadata.Field{
			{Name: "Описание", Type: metadata.FieldTypeString},
			{Name: "Просмотр", Type: metadata.FieldTypeBool},
		},
	}
	db, err := storage.ConnectSQLite(t.Context(), filepath.Join(dir, "test.db"))
	if err != nil {
		t.Fatal(err)
	}
	t.Cleanup(func() { db.Close() })
	if err := db.Migrate(t.Context(), []*metadata.Entity{doc}); err != nil {
		t.Fatal(err)
	}
	id := uuid.New()
	if err := db.Upsert(t.Context(), doc.Name, id, map[string]any{"Описание": "Исходное", "Просмотр": true}, doc); err != nil {
		t.Fatal(err)
	}

	reg := runtime.NewRegistry()
	reg.Load(runtime.LoadOptions{Entities: []*metadata.Entity{doc}})
	s := New(reg, db, interpreter.New(), nil, Config{UIEnhancements: true}, nil)
	t.Cleanup(s.Close)
	router := chi.NewRouter()
	router.Use(func(next http.Handler) http.Handler {
		return http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
			next.ServeHTTP(w, r.WithContext(auth.ContextWithUser(r.Context(), &auth.User{Login: "admin", IsAdmin: true})))
		})
	})
	s.MountStatic(router)
	s.Mount(router)
	path := "/ui/document/предоплата/" + id.String()
	for _, tc := range []struct{ suffix, state string }{{id.String(), "true"}, {"new", "false"}} {
		rec := httptest.NewRecorder()
		router.ServeHTTP(rec, httptest.NewRequest(http.MethodGet, "/ui/document/предоплата/"+tc.suffix, nil))
		if rec.Code != http.StatusOK || !strings.Contains(rec.Body.String(), `data-ob-managed-write-readonly="`+tc.state+`"`) {
			t.Fatalf("GET %s: HTTP %d, want initial readonly %s", tc.suffix, rec.Code, tc.state)
		}
	}
	server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		if r.Method == http.MethodGet && r.URL.Path == path {
			rec := httptest.NewRecorder()
			router.ServeHTTP(rec, r)
			w.Header().Set("Content-Type", "text/html; charset=utf-8")
			w.WriteHeader(rec.Code)
			_, _ = w.Write([]byte(strings.Replace(rec.Body.String(), "</body>", managedWriteReadOnlyProbe+"</body>", 1)))
			return
		}
		router.ServeHTTP(w, r)
	}))
	t.Cleanup(func() { s.CloseEventStreams(); server.Close() })
	ctx, cancel := context.WithTimeout(t.Context(), 30*time.Second)
	defer cancel()
	data, err := browser.measure(ctx, server.URL+path)
	if err != nil {
		t.Fatalf("browser: %v: %s", err, browser.diagnostics())
	}
	var result struct {
		Error string
	}
	if err := json.Unmarshal(data, &result); err != nil || result.Error != "" {
		t.Fatalf("managed form: %s (%v)", data, err)
	}
	row, err := db.GetByID(ctx, doc.Name, id, doc)
	if err != nil || row["Описание"] != "Сохранено" {
		t.Fatalf("Ctrl+S after unlock did not save: %v (%v)", row["Описание"], err)
	}
}

const managedWriteReadOnlyProbe = `<script>
window.addEventListener('load', async function() {
  const report = error => {
    const probe = document.createElement('pre');
    probe.id = 'list-mobile-measure';
    probe.textContent = JSON.stringify({error: error ? String(error) : ''});
    document.body.appendChild(probe);
  };
  const check = (ok, message) => { if (!ok) throw new Error(message); };
  try {
    const form = document.getElementById('main-form');
    const field = form.querySelector('[name="Описание"]');
    const buttons = [
      form.querySelector('button[name="_action"][value=""]'),
      form.querySelector('button[data-ob-close-reason="ok"]'),
      document.querySelector('button[name="_action"][value="post"][form="main-form"]')
    ];
    const close = form.querySelector('button[data-ob-close-reason="close"]');
    check(buttons.every(button => button && button.disabled), 'initial save/OK/post buttons');
    check(close && !close.disabled, 'close must stay available');

    const result = document.createElement('iframe');
    result.name = 'save-target';
    document.body.append(result);
    form.target = result.name;
    let submits = 0;
    form.addEventListener('submit', () => { submits++; });
    const ctrlS = () => field.dispatchEvent(new KeyboardEvent('keydown', {key: 's', code: 'KeyS', ctrlKey: true, bubbles: true, cancelable: true}));
    field.value = 'Не сохранять';
    window.obSetManagedFormDirty(true);
    buttons.forEach(button => button.click());
    ctrlS();
    form.requestSubmit();
    check(submits === 0, 'read-only form submitted a write');
    const blocked = await window.obRequestFormClose({reason:'ok', mode:'post'});
    check(!blocked.allowed && blocked.error === 'readonly-form', 'OK close intent was accepted');
    const decision = await window.obRequestFormClose({reason:'close'});
    check(decision.allowed && decision.mode === 'discard', 'ordinary Close did not discard');
    check(!document.getElementById('ob-managed-close-confirm'), 'Close displayed a save question');

    await window.obFire('РазрешитьЗапись', 'Нажатие');
    check(buttons.every(button => !button.disabled), 'event did not re-enable the standard actions');
    field.value = 'Сохранено';
    const loaded = new Promise(resolve => result.addEventListener('load', resolve, {once: true}));
    ctrlS();
    check(submits === 1, 'Ctrl+S after unlock did not submit once');
    await loaded;
    report();
  } catch (error) {
    report(error);
  }
});
</script>`
