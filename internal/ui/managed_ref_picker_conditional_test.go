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
	"github.com/ivantit66/onebase/internal/project"
)

// У ссылочного поля с условным запретом (readonly_when) кнопка подбора должна
// быть в разметке всегда — просто выключенной, пока условие истинно. Клиент
// снимает запрет по ответу события формы, но создать отсутствующую кнопку он не
// может: поле осталось бы невыбираемым до перезагрузки страницы.
//
// Постоянный запрет (readonly: true) кнопку по-прежнему не рисует: там выбирать
// нечего никогда.
func TestManagedRefPickerStaysForConditionalReadOnly(t *testing.T) {
	for _, tc := range []struct {
		name       string
		настройка  string
		wantPicker bool
		wantOff    bool
	}{
		{name: "условный запрет: кнопка есть, выключена", настройка: "    readonly_when: \"НЕ ЗначениеЗаполнено(Филиал)\"\n", wantPicker: true, wantOff: true},
		{name: "постоянный запрет: кнопки нет", настройка: "    readonly: true\n", wantPicker: false},
		{name: "без запрета: кнопка есть, включена", wantPicker: true},
	} {
		t.Run(tc.name, func(t *testing.T) {
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
  - name: Филиал
    type: string
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
`+tc.настройка), 0o644); err != nil {
				t.Fatalf("запись формы: %v", err)
			}

			proj, err := project.Load(dir)
			if err != nil {
				t.Fatalf("project.Load: %v", err)
			}
			defer proj.Close()

			s, _ := newSubmitTestServer(t, proj.Entities)
			router := chi.NewRouter()
			s.Mount(router)
			response := httptest.NewRecorder()
			router.ServeHTTP(response, httptest.NewRequest(http.MethodGet, "/ui/document/"+url.PathEscape("Заказ")+"/new", nil))
			if response.Code != http.StatusOK {
				t.Fatalf("GET формы → %d: %.800s", response.Code, response.Body.String())
			}

			html := response.Body.String()
			начало := strings.Index(html, `data-ob-ref-picker="ref-Клиент"`)
			если := начало >= 0
			if если != tc.wantPicker {
				t.Fatalf("кнопка подбора есть = %v, ожидалось %v:\n%.1500s", если, tc.wantPicker, html)
			}
			if !tc.wantPicker {
				return
			}
			кнопка := html[начало:]
			if конец := strings.Index(кнопка, ">"); конец >= 0 {
				кнопка = кнопка[:конец]
			}
			выключена := strings.Contains(кнопка, "disabled")
			if выключена != tc.wantOff {
				t.Fatalf("кнопка выключена = %v, ожидалось %v: %s", выключена, tc.wantOff, кнопка)
			}
		})
	}
}
