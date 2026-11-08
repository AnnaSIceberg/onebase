package ui

import (
	"context"
	"net/url"
	"strings"
	"testing"

	"github.com/ivantit66/onebase/internal/metadata"
	"github.com/ivantit66/onebase/internal/processor"
	"github.com/ivantit66/onebase/internal/runtime"
)

func TestManagedDeferredNewFormCannotDuplicateOrdinarySave(t *testing.T) {
	for _, deferredFirst := range []bool{false, true} {
		t.Run(map[bool]string{false: "ordinary-first", true: "deferred-first"}[deferredFirst], func(t *testing.T) {
			srv, ent := setupManagedEventsServer(t, `
Процедура ЗагрузитьИсторию()
 Объект.Записать();
КонецПроцедуры
Процедура Сохранить()
 Объект.Записать();
КонецПроцедуры`, nil, []*metadata.FormElement{
				{Kind: metadata.FormElementField, Name: "Имя", DataPath: "Объект.Наименование"},
				{Kind: metadata.FormElementGroupBox, Name: "История", Handlers: map[metadata.FormEventType]string{metadata.FormEventAfterOpen: "ЗагрузитьИсторию"}},
				{Kind: metadata.FormElementButton, Name: "Сохранить", Handlers: map[metadata.FormEventType]string{metadata.FormEventOnClick: "Сохранить"}},
			})
			// Both requests have already captured the same new form with no _id.
			// Omitting _deferred must not bypass the server's read-only contract.
			background := url.Values{"Наименование": {"new form"}, "_element": {"История"}, "_event": {formEventAfterOpen}}
			ordinary := url.Values{"Наименование": {"new form"}, "_element": {"Сохранить"}, "_event": {string(metadata.FormEventOnClick)}}
			var load, save formEventResponse
			runLoad := func() { load = decodeFormEventResponse(t, executeFormEvent(t, srv, ent, background).Body.Bytes()) }
			runSave := func() { save = decodeFormEventResponse(t, executeFormEvent(t, srv, ent, ordinary).Body.Bytes()) }
			if deferredFirst {
				runLoad()
				runSave()
			} else {
				runSave()
				runLoad()
			}
			if load.OK || !load.Deferred || load.SavedID != "" || load.Version != 0 || !strings.Contains(load.Error, "контексте только чтения") {
				t.Fatalf("background write was not rejected: %+v", load)
			}
			if !save.OK || save.SavedID == "" {
				t.Fatalf("ordinary save failed: %+v", save)
			}
			var count int
			if err := srv.store.QueryRow(context.Background(), `SELECT COUNT(*) FROM контрагент`).Scan(&count); err != nil {
				t.Fatal(err)
			}
			if count != 1 {
				t.Fatalf("one form created %d rows", count)
			}
		})
	}
}

func TestManagedDeferredTableUsesMetadataContext(t *testing.T) {
	for _, readOnly := range []bool{false, true} {
		t.Run(map[bool]string{false: "editable", true: "readonly"}[readOnly], func(t *testing.T) {
			srv, ent := setupManagedEventsServer(t, `Процедура ЗагрузитьИсторию()
 Сообщить(ИмяТабличнойЧасти);
 Объект.История.Добавить().Текст = "loaded";
КонецПроцедуры`, nil, []*metadata.FormElement{
				{Kind: metadata.FormElementField, Name: "Имя", DataPath: "Объект.Наименование"},
				{Kind: metadata.FormElementTablePart, Name: "ТаблицаИстория", DataPath: "История", ReadOnly: readOnly,
					Handlers: map[metadata.FormEventType]string{metadata.FormEventAfterOpen: "ЗагрузитьИсторию", metadata.FormEventOnChange: "ЗагрузитьИсторию"}},
			})
			ent.Forms[0].Attributes = []*metadata.FormAttribute{{Name: "История", TypeRef: "ValueTable", Columns: []*metadata.FormAttributeColumn{{Name: "Текст", TypeRef: "string"}}}}
			id := insertContragent(t, srv, ent, "Original")
			if html := executeFormEditGET(t, srv, ent, id).Body.String(); !strings.Contains(html, `data-ob-after-open="ТаблицаИстория"`) {
				t.Fatal("missing rendered loader")
			}
			body := url.Values{"_id": {id.String()}, "Наименование": {"Original"}, "_element": {"ТаблицаИстория"}, "_event": {formEventAfterOpen}, "_deferred": {"1"}}
			resp := decodeFormEventResponse(t, executeFormEvent(t, srv, ent, body).Body.Bytes())
			if !resp.OK || !resp.Deferred || len(resp.Messages) != 1 || resp.Messages[0] != "История" || len(resp.FormTables["История"]) != 1 {
				t.Fatalf("table loader failed: %+v", resp)
			}
			body.Set("_tp_row", "0")
			forged := decodeFormEventResponse(t, executeFormEvent(t, srv, ent, body).Body.Bytes())
			if forged.OK || !strings.Contains(forged.Error, "контекст строки") {
				t.Fatalf("loader accepted row context: %+v", forged)
			}
			if readOnly {
				body.Del("_tp_row")
				body.Set("_tp", "История")
				body.Set("_event", string(metadata.FormEventOnChange))
				ordinary := decodeFormEventResponse(t, executeFormEvent(t, srv, ent, body).Body.Bytes())
				if ordinary.OK {
					t.Fatalf("readonly table accepted ordinary mutation: %+v", ordinary)
				}
			}
		})
	}
}

func TestManagedDeferredProcessorServiceFieldCollisions(t *testing.T) {
	for _, kind := range []metadata.FormElementType{metadata.FormElementGroupBox, metadata.FormElementTablePart} {
		t.Run(string(kind), func(t *testing.T) {
			form := processorExecutionForm(
				&metadata.FormElement{Kind: metadata.FormElementField, Name: "Имя", DataPath: "Объект.Имя"},
				&metadata.FormElement{Kind: metadata.FormElementField, Name: "EventParam", DataPath: "Объект._event"},
				&metadata.FormElement{Kind: metadata.FormElementField, Name: "DeferredParam", DataPath: "Объект._deferred"},
				&metadata.FormElement{Kind: kind, Name: "Загрузка", DataPath: "История", ReadOnly: true, Handlers: map[metadata.FormEventType]string{metadata.FormEventAfterOpen: "Загрузить"}},
			)
			form.Attributes = []*metadata.FormAttribute{{Name: "История", TypeRef: "ValueTable", Columns: []*metadata.FormAttributeColumn{{Name: "Текст", TypeRef: "string"}}}}
			form.ProgramAST = mustParse(t, `Процедура Загрузить()
 Объект.Имя = "Loaded";
 Сообщить("loaded");
КонецПроцедуры`)
			proc := &processor.Processor{Name: "DeferredCollision", Params: []processor.Param{{Name: "Имя", Type: "string"}, {Name: "_event", Type: "string"}, {Name: "_deferred", Type: "string"}, {Name: "_ob_service_deferred", Type: "string"}}, Forms: []*metadata.FormModule{form}}
			srv, _ := newProcessorFormEventExecutionServer(t, proc, nil)
			names := processorServiceFieldNames(proc.Params)
			// Match serviceField's fallback in the client when testing the old HEAD.
			deferredKey := names["_deferred"]
			if deferredKey == "" {
				deferredKey = "_deferred"
			}
			body := url.Values{names["_element"]: {"Загрузка"}, names["_event"]: {formEventAfterOpen}, "_event": {"user event"}, "_deferred": {"user deferred"}, "Имя": {"Original"}}
			body.Set(deferredKey, "1")
			resp := decodeFormEventResponse(t, postProcessorFormEventExecution(t, srv, proc.Name, "application/x-www-form-urlencoded; charset=utf-8", strings.NewReader(body.Encode())).Body.Bytes())
			if !resp.OK || !resp.Deferred || len(resp.Values) != 1 || valueCIDeferred(resp.Values, "Имя") != "Loaded" || len(resp.Messages) != 1 || resp.Messages[0] != "loaded" {
				t.Fatalf("mapped loader returned full/stale response: %+v", resp)
			}
		})
	}
}

// The background restriction must follow manager calls through explicit DSL
// transactions, including processor forms which do not persist their own object.
func TestManagedDeferredCannotWriteThroughManagers(t *testing.T) {
	for _, transaction := range []bool{false, true} {
		t.Run(map[bool]string{false: "direct", true: "transaction"}[transaction], func(t *testing.T) {
			statement := `Зап = Справочники.Проба.Создать(); Зап.Наименование = "background"; Зап.Записать();`
			if transaction {
				statement = "НачатьТранзакцию(); " + statement + " ЗафиксироватьТранзакцию();"
			}
			form := processorExecutionForm(&metadata.FormElement{Kind: metadata.FormElementGroupBox, Name: "Загрузка", Handlers: map[metadata.FormEventType]string{metadata.FormEventAfterOpen: "Загрузить"}})
			form.ProgramAST = mustParse(t, "Процедура Загрузить()\n"+statement+"\nКонецПроцедуры")
			proc := &processor.Processor{Name: "ReadOnlyLoader", Forms: []*metadata.FormModule{form}}
			srv, db := newProcessorFormEventExecutionServer(t, proc, nil)
			ent := &metadata.Entity{Name: "Проба", Kind: metadata.KindCatalog, Fields: []metadata.Field{{Name: "Наименование", Type: metadata.FieldTypeString}}}
			if err := db.Migrate(context.Background(), []*metadata.Entity{ent}); err != nil {
				t.Fatal(err)
			}
			srv.reg.Load(runtime.LoadOptions{Entities: []*metadata.Entity{ent}})
			body := url.Values{"_element": {"Загрузка"}, "_event": {formEventAfterOpen}, "_deferred": {"1"}}
			resp := decodeFormEventResponse(t, postProcessorFormEventExecution(t, srv, proc.Name, "application/x-www-form-urlencoded; charset=utf-8", strings.NewReader(body.Encode())).Body.Bytes())
			if resp.OK || !resp.Deferred || !strings.Contains(resp.Error, "контексте только чтения") {
				t.Fatalf("manager wrote from background: %+v", resp)
			}
			var count int
			if err := db.QueryRow(context.Background(), `SELECT COUNT(*) FROM проба`).Scan(&count); err != nil {
				t.Fatal(err)
			}
			if count != 0 {
				t.Fatalf("background created %d rows", count)
			}
		})
	}
}
