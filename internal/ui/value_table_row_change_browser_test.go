package ui

import (
	"context"
	"encoding/json"
	"net/http"
	"net/http/httptest"
	"strings"
	"testing"
	"time"

	"github.com/go-chi/chi/v5"
	"github.com/ivantit66/onebase/internal/dsl/interpreter"
	"github.com/ivantit66/onebase/internal/metadata"
)

func TestValueTableRowChangeBrowser(t *testing.T) {
	browser := newMobileLayoutBrowser(t, managedLayoutTestBrowser(t))
	table := func(name, attribute, key string, ro bool) *metadata.FormElement {
		return &metadata.FormElement{
			Kind: metadata.FormElementTablePart, Name: name, DataPath: "Форма." + attribute,
			FixedRows: true, ReadOnly: ro, RowKey: key,
			Children: []*metadata.FormElement{
				{Kind: metadata.FormElementColumn, Name: "Отметка", DataPath: "Отмечен"},
				{Kind: metadata.FormElementColumn, Name: "Подпись", DataPath: "Имя", ReadOnly: true},
			},
			Handlers: map[metadata.FormEventType]string{
				metadata.FormEventOnRowActivated: "Активировать" + attribute,
				metadata.FormEventOnRowChanged:   "Обновить",
			},
		}
	}
	base, entity := setupManagedEventsServer(t, `
Процедура Заполнить()
 Результаты.Очистить();
 Для Индекс = 0 По 1 Цикл
  Запись = Результаты.Добавить();
  Запись.Имя = "Вариант";
  Запись.Номер = Индекс + 1;
  Запись.Отмечен = Ложь;
  СтрокаБезКлюча = БезКлюча.Добавить();
  СтрокаБезКлюча.Имя = "Описание" + Строка(Индекс);
  СтрокаБезКлюча.Отмечен = Ложь;
 КонецЦикла;
КонецПроцедуры

Процедура АктивироватьРезультаты()
 Если Объект.Наименование = "" Тогда
  Результаты.Очистить();
  Запись = Результаты.Добавить();
  Запись.Имя = "Вариант";
  Запись.Номер = 2;
  Запись.Отмечен = Ложь;
  Запись = Результаты.Добавить();
  Запись.Имя = "Уточнённый вариант";
  Запись.Номер = 1;
  Запись.Отмечен = Ложь;
 КонецЕсли;
КонецПроцедуры

Процедура АктивироватьБезКлюча()
 БезКлюча.Очистить();
 Для Индекс = 0 По 1 Цикл
  Запись = БезКлюча.Добавить();
  Запись.Имя = "Описание" + Строка(1 - Индекс);
  Запись.Отмечен = Ложь;
 КонецЦикла;
КонецПроцедуры

Процедура Обновить()
 Объект.Наименование = Строка(ТекущаяСтрока.Номер) + ":" + ИмяКолонки + ?(ТекущаяСтрока.Отмечен, ":да", ":нет");
КонецПроцедуры
`, map[metadata.FormEventType]string{metadata.FormEventOnOpen: "Заполнить"}, []*metadata.FormElement{
		{Kind: metadata.FormElementField, Name: "ПринятыйОтвет", DataPath: "Объект.Наименование"},
		{Kind: metadata.FormElementGroupBox, Name: "Варианты", Children: []*metadata.FormElement{
			table("РедактируемаяТаблица", "Результаты", "Номер", false),
			table("ТаблицаПросмотра", "Результаты", "Номер", true),
			table("ТаблицаБезКлюча", "БезКлюча", "", false),
		}},
	})
	entity.Forms[0].Attributes = []*metadata.FormAttribute{
		{Name: "Результаты", TypeRef: "ValueTable", Columns: []*metadata.FormAttributeColumn{
			{Name: "Имя", TypeRef: "string"}, {Name: "Отмечен", TypeRef: "bool"}, {Name: "Номер", TypeRef: "number"},
		}},
		{Name: "БезКлюча", TypeRef: "ValueTable", Columns: []*metadata.FormAttributeColumn{
			{Name: "Имя", TypeRef: "string"}, {Name: "Отмечен", TypeRef: "bool"},
		}},
	}
	s := New(base.reg, base.store, interpreter.New(), nil, Config{}, nil)
	t.Cleanup(s.Close)
	router := chi.NewRouter()
	s.MountStatic(router)
	router.Get("/ui/{kind}/{entity}/new", s.form)
	router.Post("/ui/{kind}/{entity}/form-event", s.handleManagedFormEvent)
	path := "/ui/catalog/контрагент/new"
	server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		if r.Method == http.MethodGet && r.URL.Path == path {
			rec := httptest.NewRecorder()
			router.ServeHTTP(rec, r)
			w.Header().Set("Content-Type", "text/html; charset=utf-8")
			w.WriteHeader(rec.Code)
			_, _ = w.Write([]byte(strings.Replace(rec.Body.String(), "</body>", valueTableRowChangeProbe+"</body>", 1)))
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
		t.Fatalf("form ValueTable: %s (%v)", data, err)
	}
}

const valueTableRowChangeProbe = `<script>
window.addEventListener('load', async function() {
  const report = error => {
    const marker = document.createElement('pre');
    marker.id = 'list-mobile-measure';
    marker.textContent = JSON.stringify({error: error ? String(error) : ''});
    document.body.appendChild(marker);
  };
  const check = (ok, message) => { if (!ok) throw new Error(message); };
  const wait = async (test, message) => {
    for (let count = 0; count < 160; count++) {
      if (test()) return;
      await new Promise(resolve => setTimeout(resolve, 20));
    }
    throw new Error(message);
  };
  try {
    const editable = '[data-ob-vt-rowchange="РедактируемаяТаблица"]';
    const locked = '[data-ob-vt-rowchange="ТаблицаПросмотра"]';
    const unkeyed = '[data-ob-vt-rowchange="ТаблицаБезКлюча"]';
    const answer = document.querySelector('[name="Наименование"]');
    const mark = (selector, index) => document.querySelectorAll(selector + ' input[type="checkbox"]')[index];
    const changeRequests = [];
    const fetchOriginal = window.fetch;
    window.fetch = function(url, options) {
      const body = options && options.body;
      if (String(url).includes('form-event') && body && body.get('_event') === 'ПриИзмененииСтроки') {
        changeRequests.push([body.get('_tp_row'), body.get('_tp_col'), body.get('vt.Результаты.1.Отмечен')]);
      }
      return fetchOriginal.apply(this, arguments);
    };
    await wait(() => mark(editable, 0), 'fixed_rows options did not load');
    mark(editable, 0).click();
    await wait(() => answer.value === '1:Отмечен:да', 'checkbox change did not reach the original row');
    check(!mark(editable, 0).checked && mark(editable, 1).checked, 'reordered row has the wrong mark: ' + JSON.stringify(changeRequests));
    check(document.querySelector('[name="vt.Результаты.1.Имя"]').value === 'Уточнённый вариант',
      'change overwrote the neighboring value returned by activation');
    await window.obFire('РедактируемаяТаблица', 'ПриАктивизацииСтроки', {_tp:'Результаты', _tp_row:'0'});
    check(!mark(editable, 0).checked && mark(editable, 1).checked, 'the next event overwrote the edited row');
    mark(editable, 1).click();
    await wait(() => answer.value === '1:Отмечен:нет', 'unchecked value did not reach the handler');
    check(!mark(editable, 1).checked, 'repaint rechecked the checkbox');

    const lockedMark = mark(locked, 0);
    check(lockedMark && lockedMark.disabled, 'readonly check is not disabled');
    lockedMark.click();
    lockedMark.dispatchEvent(new Event('change', {bubbles:true}));
    await window.obFire('РедактируемаяТаблица', 'ПриАктивизацииСтроки', {_tp:'Результаты', _tp_row:'1'});
    mark(unkeyed, 0).click();
    await wait(() => document.body.innerText.includes('Таблица обновилась. Повторите изменение строки.'),
      'ambiguous edit was not reported');
    check(document.querySelector('[name="vt.БезКлюча.1.Имя"]').value === 'Описание0',
      'the preceding event did not exchange the descriptions');
    check(!mark(unkeyed, 1).checked, 'an edit followed the old description to another row');
    check(JSON.stringify(changeRequests) === JSON.stringify([['1','Отмечен','true'], ['1','Отмечен','false']]),
      'unexpected row change requests: ' + JSON.stringify(changeRequests));
    report();
  } catch (error) {
    report(error);
  }
});
</script>`
