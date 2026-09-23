package ui

import (
	"net/http"
	"net/http/httptest"
	"net/url"
	"os"
	"path/filepath"
	"strings"
	"testing"

	"github.com/go-chi/chi/v5"
	"github.com/google/uuid"
	"github.com/ivantit66/onebase/internal/project"
)

// Значение нередактируемого поля обязано уезжать с формой. Платформа рисует
// такое поле как <select disabled>, а отключённые элементы браузер не
// отправляет: проставленное при открытии значение молча терялось при записи —
// так пропадали автор документа и тип звонка.
func TestManagedReadOnlyValueIsSubmitted(t *testing.T) {
	dir := t.TempDir()
	for _, path := range []string{
		filepath.Join(dir, "catalogs"), filepath.Join(dir, "documents"), filepath.Join(dir, "forms", "заказ"),
	} {
		if err := os.MkdirAll(path, 0o755); err != nil {
			t.Fatalf("создание каталога %s: %v", path, err)
		}
	}
	if err := os.WriteFile(filepath.Join(dir, "catalogs", "клиент.yaml"), []byte(`name: Клиент
fields:
  - name: Наименование
    type: string
`), 0o644); err != nil {
		t.Fatalf("запись справочника: %v", err)
	}
	if err := os.WriteFile(filepath.Join(dir, "documents", "заказ.yaml"), []byte(`name: Заказ
fields:
  - name: Клиент
    type: reference:Клиент
`), 0o644); err != nil {
		t.Fatalf("запись документа: %v", err)
	}
	if err := os.WriteFile(filepath.Join(dir, "forms", "заказ", "объекта.form.yaml"), []byte(`schema: onebase.form/v1
form:
  name: ФормаОбъекта
  kind: object
  entity: Заказ
elements:
  - kind: ПолеВвода
    name: ПолеКлиент
    data_path: Объект.Клиент
    readonly: true
`), 0o644); err != nil {
		t.Fatalf("запись формы: %v", err)
	}

	proj, err := project.Load(dir)
	if err != nil {
		t.Fatalf("project.Load: %v", err)
	}
	defer proj.Close()

	s, ctx := newSubmitTestServer(t, proj.Entities)
	клиент := uuid.New()
	if err := s.store.Upsert(ctx, "Клиент", клиент, map[string]any{"Наименование": "Иванов"}, s.reg.GetEntity("Клиент")); err != nil {
		t.Fatalf("создание клиента: %v", err)
	}
	заказ := uuid.New()
	if err := s.store.Upsert(ctx, "Заказ", заказ, map[string]any{"Клиент": клиент.String()}, s.reg.GetEntity("Заказ")); err != nil {
		t.Fatalf("создание заказа: %v", err)
	}

	router := chi.NewRouter()
	s.Mount(router)
	response := httptest.NewRecorder()
	router.ServeHTTP(response, httptest.NewRequest(http.MethodGet,
		"/ui/document/"+url.PathEscape("Заказ")+"/"+заказ.String(), nil))
	if response.Code != http.StatusOK {
		t.Fatalf("GET формы → %d", response.Code)
	}

	html := response.Body.String()
	скрытое := `<input type="hidden" name="Клиент" value="` + клиент.String() + `">`
	if !strings.Contains(html, скрытое) {
		i := strings.Index(html, `id="ref-Клиент"`)
		фрагмент := html
		if i >= 0 {
			конец := i + 900
			if конец > len(html) {
				конец = len(html)
			}
			фрагмент = html[i:конец]
		}
		t.Fatalf("значение нередактируемого поля не уедет с формой — нет %s:\n%s", скрытое, фрагмент)
	}
}
