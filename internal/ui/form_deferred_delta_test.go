package ui

import (
	"net/url"
	"strings"
	"testing"

	"github.com/ivantit66/onebase/internal/metadata"
	"github.com/ivantit66/onebase/internal/processor"
)

func deferredDeltaFixture(t *testing.T, code string) (*Server, *metadata.Entity, url.Values) {
	t.Helper()
	srv, ent := setupManagedEventsServer(t, code, nil, []*metadata.FormElement{
		{Kind: metadata.FormElementField, Name: "ПолеНаименование", DataPath: "Объект.Наименование", ReadOnlyWhen: "Ложь"},
		{Kind: metadata.FormElementCheckbox, Name: "ПолеФлаг", DataPath: "Флаг"},
		{Kind: metadata.FormElementTablePart, Name: "ДругаяТаблица", DataPath: "Список"},
		{Kind: metadata.FormElementGroupBox, Name: "ГруппаИстория", ReadOnly: true,
			Handlers: map[metadata.FormEventType]string{metadata.FormEventAfterOpen: "ЗагрузитьИсторию"},
			Children: []*metadata.FormElement{{Kind: metadata.FormElementField, Name: "ПолеИстория", DataPath: "История"}}},
	})
	ent.Forms[0].Attributes = []*metadata.FormAttribute{
		{Name: "История", TypeRef: "string"},
		{Name: "Флаг", TypeRef: "bool"},
		{Name: "Список", TypeRef: "ValueTable", Columns: []*metadata.FormAttributeColumn{
			{Name: "Текст", TypeRef: "string"}, {Name: "Активен", TypeRef: "bool"},
		}},
	}
	id := insertContragent(t, srv, ent, "Original")
	body := url.Values{
		"_id": {id.String()}, "_element": {"ГруппаИстория"},
		"_event": {string(metadata.FormEventAfterOpen)}, "_deferred": {"1"},
		"Наименование": {"Original"}, "История": {""}, "_ob_present_Флаг": {"1"},
		"vt.Список.0.Текст": {"old table cell"},
	}
	return srv, ent, body
}

func TestManagedDeferredSuccessOmitsUnchangedBoolAndTable(t *testing.T) {
	srv, ent, body := deferredDeltaFixture(t, `
Процедура ЗагрузитьИсторию()
    Объект.История = "history ready";
КонецПроцедуры
`)
	rec := executeFormEvent(t, srv, ent, body)
	resp := decodeFormEventResponse(t, rec.Body.Bytes())
	if !resp.OK || !resp.Deferred {
		t.Fatalf("event failed: %+v", resp)
	}
	if v, ok := lookupCI(resp.Values, "Флаг"); ok {
		t.Errorf("unchanged unchecked checkbox returned as a delta: %v", v)
	}
	if len(resp.FormTables) != 0 {
		t.Errorf("unchanged unrelated table returned as a delta: %v", resp.FormTables)
	}
	if valueCIDeferred(resp.Values, "История") != "history ready" {
		t.Fatalf("handler result missing: %+v", resp)
	}
}

func TestManagedDeferredErrorStillFiltersResponse(t *testing.T) {
	srv, ent, body := deferredDeltaFixture(t, `
Процедура ЗагрузитьИсторию()
    Объект.Наименование = "Loaded before error";
    ВызватьИсключение("external service failed");
КонецПроцедуры
`)
	body.Del("_id") // ошибка фоновой загрузки новой формы не должна создавать запись
	rec := executeFormEvent(t, srv, ent, body)
	resp := decodeFormEventResponse(t, rec.Body.Bytes())
	if resp.OK || resp.Error == "" {
		t.Fatalf("expected handler failure: %+v", resp)
	}
	if !resp.Deferred {
		t.Error("error response lost deferred marker")
	}
	if resp.SavedID != "" || resp.Version != 0 || resp.Dirty != nil || !strings.Contains(resp.Error, "external service failed") {
		t.Fatalf("deferred error returned durable save identity or dirty state: %+v", resp)
	}
	if len(resp.FormTables) != 0 || resp.ElementStates != nil || len(resp.TableParts) != 0 {
		t.Errorf("error response can repaint stale tables/states: %+v", resp)
	}
}

func TestManagedDeferredProcessorUsesDeferredResponse(t *testing.T) {
	for _, tc := range []struct {
		name, code string
		failed     bool
		changed    bool
	}{
		{name: "unchanged", code: `Сообщить("history ready");`},
		{name: "changed", code: `Объект.Имя = "Loaded";`, changed: true},
		{name: "error", code: `Объект.Имя = "Loaded"; ВызватьИсключение("external service failed");`, failed: true, changed: true},
	} {
		t.Run(tc.name, func(t *testing.T) {
			form := processorExecutionForm(
				&metadata.FormElement{Kind: metadata.FormElementField, Name: "ПолеИмя", DataPath: "Объект.Имя", ReadOnlyWhen: "Ложь"},
				&metadata.FormElement{Kind: metadata.FormElementCheckbox, Name: "ПолеФлаг", DataPath: "Объект.Флаг"},
				&metadata.FormElement{Kind: metadata.FormElementGroupBox, Name: "ГруппаИстория",
					Handlers: map[metadata.FormEventType]string{metadata.FormEventAfterOpen: "ЗагрузитьИсторию"}},
			)
			form.ProgramAST = mustParse(t, "Процедура ЗагрузитьИсторию()\n"+tc.code+"\nКонецПроцедуры")
			proc := &processor.Processor{Name: "DeferredProcessor",
				Params: []processor.Param{{Name: "Имя", Type: "string"}, {Name: "Флаг", Type: "bool"}},
				Forms:  []*metadata.FormModule{form}}
			srv, _ := newProcessorFormEventExecutionServer(t, proc, nil)
			body := url.Values{"_element": {"ГруппаИстория"}, "_event": {string(metadata.FormEventAfterOpen)},
				"_deferred": {"1"}, "Имя": {"Original"}, "_ob_present_Флаг": {"1"}}
			rec := postProcessorFormEventExecution(t, srv, proc.Name, "application/x-www-form-urlencoded; charset=utf-8", strings.NewReader(body.Encode()))
			resp := decodeFormEventResponse(t, rec.Body.Bytes())
			if resp.OK == tc.failed || (resp.Error != "") != tc.failed {
				t.Fatalf("unexpected handler result: %+v", resp)
			}
			if !resp.Deferred || resp.ElementStates != nil || resp.Dirty != nil || len(resp.TableParts) != 0 {
				t.Fatalf("processor returned full response for deferred event: %+v", resp)
			}
			if tc.changed {
				if len(resp.Values) != 1 || valueCIDeferred(resp.Values, "Имя") != "Loaded" {
					t.Fatalf("processor lost handler delta: %+v", resp)
				}
			} else if len(resp.Values) != 0 {
				t.Fatalf("unchanged processor values returned: %+v", resp)
			}
		})
	}
}

func TestManagedDeferredKeepsHandlerTableChanges(t *testing.T) {
	srv, ent, body := deferredDeltaFixture(t, `
Процедура ЗагрузитьИсторию()
 Стр = Объект.Список.Получить(0);
 Стр.Текст = "loaded table cell";
 Стр.Активен = Истина;
КонецПроцедуры
`)
	resp := decodeFormEventResponse(t, executeFormEvent(t, srv, ent, body).Body.Bytes())
	rows := resp.FormTables["Список"]
	if !resp.OK || !resp.Deferred || len(rows) != 1 || rows[0]["Текст"] != "loaded table cell" || rows[0]["Активен"] != true {
		t.Fatalf("in-place handler table mutation lost: %+v", resp)
	}
}
