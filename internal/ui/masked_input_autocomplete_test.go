package ui

import (
	"context"
	"net/http"
	"net/http/httptest"
	"regexp"
	"testing"

	"github.com/go-chi/chi/v5"
	"github.com/google/uuid"

	"github.com/ivantit66/onebase/internal/metadata"
	"github.com/ivantit66/onebase/internal/processor"
)

// Поле с маской ввода — почти всегда телефон. Автозаполнение Яндекс Браузера
// на autocomplete="off" (#595) не смотрит и подсказывает сохранённые номера из
// своего профиля; на поле "one-time-code" (по стандарту — код подтверждения)
// эти подсказки в Яндекс Браузере пропадают, проверено вручную. Поле без маски
// остаётся "off". Тест проверяет выдаваемый атрибут в разметке формы, отданной
// обычным GET (и у поля объекта, и у реквизита формы), а не поведение браузеров.
func TestManagedForm_MaskedInputSuppressesBrowserAutofill(t *testing.T) {
	srv, ent := setupManagedEventsServer(t, ``, nil, []*metadata.FormElement{
		{Kind: metadata.FormElementField, Name: "ПолеНаименование", DataPath: "Объект.Наименование", InputMask: "(000)000-00-00"},
		{Kind: metadata.FormElementField, Name: "ПолеТелефон", DataPath: "Телефон", InputMask: "(000)000-00-00"},
		{Kind: metadata.FormElementField, Name: "ПолеКомментарий", DataPath: "Комментарий"},
	})
	ent.Forms[0].Attributes = append(ent.Forms[0].Attributes,
		&metadata.FormAttribute{Name: "Телефон", TypeRef: "string"},
		&metadata.FormAttribute{Name: "Комментарий", TypeRef: "string"})
	id := uuid.New()
	if err := srv.store.Upsert(context.Background(), ent.Name, id,
		map[string]any{"Наименование": "(495)111-22-33"}, ent); err != nil {
		t.Fatal(err)
	}
	body := renderObjectFormGET(t, srv, ent, id)

	for name, want := range map[string]string{
		"Наименование": "one-time-code", // поле объекта с маской
		"Телефон":      "one-time-code", // реквизит формы с маской
		"Комментарий":  "off",           // без маски — как прежде (#595)
	} {
		re := regexp.MustCompile(`<input type="text" autocomplete="([^"]*)" name="` + regexp.QuoteMeta(name) + `"`)
		m := re.FindStringSubmatch(body)
		if m == nil {
			t.Fatalf("нет текстового поля %s в разметке формы", name)
		}
		if m[1] != want {
			t.Errorf("%s: autocomplete=%q, ожидалось %q", name, m[1], want)
		}
	}
}

// Форма обработки проходит через Server.Mount и публичный GET: и параметры
// обработки, и локальные реквизиты должны сохранить autocomplete из managed-
// шаблона. Проверяем HTML-атрибут, а не поведение браузерного автозаполнения.
func TestManagedProcessorForm_MaskedInputAutocompleteThroughHTTP(t *testing.T) {
	srv, _ := newSubmitTestServer(t, nil)
	proc := &processor.Processor{
		Name: "ТелефоннаяОбработка",
		Params: []processor.Param{
			{Name: "Телефон", Type: "string"},
			{Name: "Комментарий", Type: "string"},
		},
		Forms: []*metadata.FormModule{{
			Name:       "ФормаОбработки",
			Kind:       "object",
			LayoutKind: metadata.FormLayoutManaged,
			Attributes: []*metadata.FormAttribute{
				{Name: "ЛокальныйТелефон", TypeRef: "string"},
				{Name: "ЛокальныйКомментарий", TypeRef: "string"},
			},
			Elements: []*metadata.FormElement{
				{Kind: metadata.FormElementField, Name: "ПолеТелефон", DataPath: "Объект.Телефон", InputMask: "(000)000-00-00"},
				{Kind: metadata.FormElementField, Name: "ПолеКомментарий", DataPath: "Объект.Комментарий"},
				{Kind: metadata.FormElementField, Name: "ПолеЛокальныйТелефон", DataPath: "ЛокальныйТелефон", InputMask: "(000)000-00-00"},
				{Kind: metadata.FormElementField, Name: "ПолеЛокальныйКомментарий", DataPath: "ЛокальныйКомментарий"},
			},
		}},
	}
	srv.reg.LoadProcessors([]*processor.Processor{proc})
	router := chi.NewRouter()
	srv.Mount(router)
	rec := httptest.NewRecorder()
	router.ServeHTTP(rec, httptest.NewRequest(http.MethodGet, "/ui/processor/"+proc.Name, nil))
	if rec.Code != http.StatusOK {
		t.Fatalf("GET формы обработки: статус %d, ожидался %d: %s", rec.Code, http.StatusOK, rec.Body.String())
	}

	for _, tc := range []struct{ name, want string }{
		{"Телефон", "one-time-code"},
		{"Комментарий", "off"},
		{"ЛокальныйТелефон", "one-time-code"},
		{"ЛокальныйКомментарий", "off"},
	} {
		t.Run(tc.name, func(t *testing.T) {
			re := regexp.MustCompile(`<input type="text" autocomplete="([^"]*)" name="` + regexp.QuoteMeta(tc.name) + `"`)
			matches := re.FindAllStringSubmatch(rec.Body.String(), -1)
			if len(matches) != 1 {
				t.Fatalf("ожидалось одно текстовое поле %s, получено %d", tc.name, len(matches))
			}
			if got := matches[0][1]; got != tc.want {
				t.Errorf("autocomplete=%q, ожидалось %q", got, tc.want)
			}
		})
	}
}
