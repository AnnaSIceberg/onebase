package ui

import (
	"context"
	"net/url"
	"reflect"
	"strings"
	"testing"

	"github.com/ivantit66/onebase/internal/dbtest"
	"github.com/ivantit66/onebase/internal/dsl/interpreter"
	"github.com/ivantit66/onebase/internal/metadata"
	"github.com/ivantit66/onebase/internal/processor"
	"github.com/ivantit66/onebase/internal/runtime"
	"github.com/ivantit66/onebase/internal/storage"
)

// Exercise both HTTP entry points with typed browser input, including omitted
// false checkboxes. PostgreSQL uses dbtest's isolated ephemeral schema.
func TestDeferredTableColumnsHTTPMatrix(t *testing.T) {
	row := func(status, text string, amount any, active bool) map[string]any {
		return map[string]any{"_status": status, "Текст": text, "_amount": amount, "Активен": active}
	}
	original := row("old", "old cell", float64(7), false)
	for _, tc := range []struct {
		name, code                 string
		second, failed, formatting bool
		want                       []map[string]any // nil means the table must be absent, not cleared
	}{
		{name: "underscore-string", code: `Объект.Список.Получить(0)._status = "loaded";`, want: []map[string]any{row("loaded", "old cell", float64(7), false)}},
		{name: "underscore-number", code: `Объект.Список.Получить(0)._amount = 8;`, want: []map[string]any{row("old", "old cell", "8", false)}},
		{name: "unchanged", code: `Сообщить("unchanged");`},
		{name: "error-after-change", code: `Объект.Список.Получить(0)._status = "loaded"; ВызватьИсключение("loader failed");`, failed: true, want: []map[string]any{row("loaded", "old cell", float64(7), false)}},
		{name: "ordinary-column", code: `Объект.Список.Получить(0).Текст = "loaded";`, want: []map[string]any{row("old", "loaded", float64(7), false)}},
		{name: "checkbox", code: `Объект.Список.Получить(0).Активен = Истина;`, want: []map[string]any{row("old", "old cell", float64(7), true)}},
		{name: "append", code: `Стр = Объект.Список.Добавить(); Стр._status = "new"; Стр.Текст = "new cell"; Стр._amount = 9; Стр.Активен = Истина;`, want: []map[string]any{original, row("new", "new cell", "9", true)}},
		{name: "remove", second: true, code: `Объект.Список.Удалить(0);`, want: []map[string]any{row("second", "second cell", float64(9), true)}},
		{name: "reorder", second: true, code: `Объект.Список.Удалить(0); Стр = Объект.Список.Добавить(); Стр._status = "old"; Стр.Текст = "old cell"; Стр._amount = 7; Стр.Активен = Ложь;`, want: []map[string]any{row("second", "second cell", float64(9), true), row("old", "old cell", "7", false)}},
		{name: "clear", code: `Объект.Список.Очистить();`, want: []map[string]any{}},
		{name: "formatting-only", formatting: true, code: `ДобавитьПравилоОформления("Список", "Истина", Новый Структура("ЦветФона", "#ffeeee")); ДобавитьПравилоОформления("Список", "Истина", Новый Структура("Поле,Цвет", "Текст", "#ff0000"));`},
	} {
		t.Run(tc.name, func(t *testing.T) {
			for _, kind := range []string{"catalog", "processor"} {
				t.Run(kind, func(t *testing.T) {
					dbtest.ForEachDialect(t, func(t *testing.T, db *storage.DB) {
						form := processorExecutionForm(
							&metadata.FormElement{Kind: metadata.FormElementGroupBox, Name: "Загрузка", Handlers: map[metadata.FormEventType]string{metadata.FormEventAfterOpen: "Загрузить"}},
							&metadata.FormElement{Kind: metadata.FormElementTablePart, Name: "Таблица", DataPath: "Список"},
						)
						form.Handlers = map[metadata.FormEventType]string{metadata.FormEventOnOpen: "Загрузить"}
						form.Attributes = []*metadata.FormAttribute{{Name: "Список", TypeRef: "ValueTable", Columns: []*metadata.FormAttributeColumn{
							{Name: "_status", TypeRef: "string"}, {Name: "_amount", TypeRef: "number"}, {Name: "Текст", TypeRef: "string"}, {Name: "Активен", TypeRef: "bool"},
						}}}
						form.ProgramAST = mustParse(t, "Процедура Загрузить()\n"+tc.code+"\nКонецПроцедуры")
						ent := &metadata.Entity{Name: "Контрагент", Kind: metadata.KindCatalog, Forms: []*metadata.FormModule{form}}
						reg := runtime.NewRegistry()
						if kind == "catalog" {
							form.Name, form.EntityName = "ФормаОбъекта", ent.Name
							if err := db.Migrate(context.Background(), []*metadata.Entity{ent}); err != nil {
								t.Fatal(err)
							}
							reg.Load(runtime.LoadOptions{Entities: []*metadata.Entity{ent}})
						} else {
							reg.LoadProcessors([]*processor.Processor{{Name: "DeferredColumns", Forms: []*metadata.FormModule{form}}})
						}
						interp := interpreter.New()
						interp.LookupProc = reg.GetModuleProc
						srv := &Server{store: db, reg: reg, interp: interp, lockMgr: runtime.NewLockManager(), messages: NewMessageStore(), ops: newOperationLimiter()}
						srv.entitySvc = srv.newEntityService(nil)
						body := url.Values{"_element": {"Загрузка"}, "_event": {formEventAfterOpen}, "_deferred": {"1"},
							"vt.Список.0._status": {"old"}, "vt.Список.0._amount": {"7.00"}, "vt.Список.0.Текст": {"old cell"}}
						if tc.second {
							body.Set("vt.Список.1._status", "second")
							body.Set("vt.Список.1._amount", "9")
							body.Set("vt.Список.1.Текст", "second cell")
							body.Set("vt.Список.1.Активен", "true")
						}
						post := func() formEventResponse {
							t.Helper()
							if kind == "catalog" {
								return decodeFormEventResponse(t, executeFormEvent(t, srv, ent, body).Body.Bytes())
							}
							rec := postProcessorFormEventExecution(t, srv, "DeferredColumns", "application/x-www-form-urlencoded; charset=utf-8", strings.NewReader(body.Encode()))
							if rec.Code != 200 {
								t.Fatalf("HTTP %d: %s", rec.Code, rec.Body.String())
							}
							return decodeFormEventResponse(t, rec.Body.Bytes())
						}
						resp := post()
						if resp.OK == tc.failed || (resp.Error != "") != tc.failed || !resp.Deferred {
							t.Fatalf("unexpected event result: %+v", resp)
						}
						if tc.failed && !strings.Contains(resp.Error, "loader failed") {
							t.Fatalf("lost loader error: %+v", resp)
						}
						rows, present := resp.FormTables["Список"]
						if present != (tc.want != nil) {
							t.Fatalf("table presence=%v, want %v: %+v", present, tc.want != nil, resp)
						}
						if present && !reflect.DeepEqual(rows, tc.want) {
							t.Fatalf("table=%#v, want %#v", rows, tc.want)
						}
						if len(resp.TableParts) != 0 || resp.ElementStates != nil || resp.ConditionalCSS != "" || resp.Dirty != nil {
							t.Fatalf("deferred response returned stale state: %+v", resp)
						}
						// Prove the formatting handler really changed both reserved keys; the
						// background response must still omit this unchanged table.
						if tc.formatting {
							body.Set("_event", string(metadata.FormEventOnOpen))
							body.Set("_element", "")
							body.Del("_deferred")
							ordinary := post()
							rows := ordinary.FormTables["Список"]
							if !ordinary.OK || ordinary.Deferred || len(rows) != 1 || formRowClass(rows[0]) == "" || formCellClass(rows[0], "Текст") == "" {
								t.Fatalf("formatting control did not change reserved keys: %+v", ordinary)
							}
						}
					})
				})
			}
		})
	}
}
