package ui

import (
	"net/url"
	"testing"

	"github.com/google/uuid"
	"github.com/ivantit66/onebase/internal/metadata"
)

// Обработчик ставит дату «сейчас» и записывает объект — после записи форма не
// «изменена». ТекущаяДатаВремя() несёт доли секунды, а хранилище пишет дату с
// точностью до секунды (как дата 1С): сравнение до наносекунд видело разницу,
// и закрытие только что записанной формы спрашивало «Данные были изменены.
// Сохранить?» (конфигурация cc2, «Распределить» заявки).
func TestManagedFormEvent_WrittenCurrentDateIsNotDirty(t *testing.T) {
	ent := &metadata.Entity{
		Name: "ЗаявкаСДатой", Kind: metadata.KindCatalog,
		Fields: []metadata.Field{
			{Name: "Наименование", Type: metadata.FieldTypeString},
			{Name: "ДатаРаспределения", Type: metadata.FieldTypeDate},
		},
	}
	form := managedObjectForm(&metadata.FormElement{
		Kind: metadata.FormElementButton, Name: "Команда",
		Handlers: map[metadata.FormEventType]string{metadata.FormEventOnClick: "Распределить"},
	})
	form.EntityName = ent.Name
	form.ProgramAST = mustParse(t, `
Процедура Распределить()
	Объект.ДатаРаспределения = ТекущаяДатаВремя();
	Объект.Записать();
КонецПроцедуры
`)
	ent.Forms = []*metadata.FormModule{form}
	srv, ctx := newSubmitTestServer(t, []*metadata.Entity{ent})
	id := uuid.New()
	if err := srv.store.Upsert(ctx, ent.Name, id, map[string]any{"Наименование": "заявка"}, ent); err != nil {
		t.Fatal(err)
	}
	body := url.Values{
		"_element": {"Команда"}, "_event": {string(metadata.FormEventOnClick)}, "_kind": {"object"},
		"_id": {id.String()}, "_version": {"1"}, "Наименование": {"заявка"},
	}
	resp := decodeFormEventResponse(t, executeFormEvent(t, srv, ent, body).Body.Bytes())
	if !resp.OK || resp.Version == 0 {
		t.Fatalf("обработчик не записал объект: %+v", resp)
	}
	if resp.Dirty == nil || *resp.Dirty {
		t.Fatalf("записанная форма с датой «сейчас» помечена изменённой: %+v", resp)
	}
}
