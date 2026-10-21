package ui

import (
	"net/url"
	"strings"
	"testing"

	"github.com/ivantit66/onebase/internal/metadata"
)

// Поля диалога ПоказатьВопрос: фаза 1 отдаёт их в question.fields, фаза 2
// возвращает введённое обработчику Ответ в ДиалогПоля. Через публичный
// form-event, как из браузера.
func TestShowQuestion_FieldsRoundTrip(t *testing.T) {
	srv, ent := setupManagedEventsServer(t, `
Процедура КомандаНажатие()
	Поля = Новый Массив;
	Поля.Добавить(Новый Структура("заголовок, значение, выделить", "Заявка", "ЗК-17", Истина));
	Поля.Добавить(Новый Структура("имя, заголовок, правка, многострочное", "Ошибка", "Текст ошибки", Истина, Истина));
	ПоказатьВопрос("Уведомить тех. отдел?", "ДаНет", "Ошибка", Поля);
КонецПроцедуры

Процедура КомандаОтвет()
	Сообщить("Ответ: " + ВопросОтвет + "; Ошибка=" + ДиалогПоля.Ошибка);
КонецПроцедуры
`, nil, []*metadata.FormElement{{
		Kind: metadata.FormElementButton,
		Name: "Команда",
		Handlers: map[metadata.FormEventType]string{
			metadata.FormEventOnClick:  "КомандаНажатие",
			metadata.FormEventOnAnswer: "КомандаОтвет",
		},
	}})

	click := url.Values{}
	click.Set("_element", "Команда")
	click.Set("_event", string(metadata.FormEventOnClick))
	resp := decodeFormEventResponse(t, executeFormEvent(t, srv, ent, click).Body.Bytes())
	if !resp.OK || resp.Question == nil {
		t.Fatalf("фаза 1: ok=%v error=%q question=%+v", resp.OK, resp.Error, resp.Question)
	}
	f := resp.Question.Fields
	if len(f) != 2 {
		t.Fatalf("fields=%+v", f)
	}
	if f[0].Value != "ЗК-17" || !f[0].Strong || f[0].Editable || f[0].Name == "" {
		t.Fatalf("строка для показа: %+v", f[0])
	}
	if f[1].Name != "Ошибка" || !f[1].Editable || !f[1].Multiline {
		t.Fatalf("многострочное поле ввода: %+v", f[1])
	}

	answer := url.Values{}
	answer.Set("_element", "Команда")
	answer.Set("_event", string(metadata.FormEventOnAnswer))
	answer.Set("_question_answer", "Да")
	answer.Set("_question_fields", `{"Ошибка":"не открывается карточка"}`)
	resp = decodeFormEventResponse(t, executeFormEvent(t, srv, ent, answer).Body.Bytes())
	if !resp.OK {
		t.Fatalf("фаза 2: %q", resp.Error)
	}
	if got := strings.Join(resp.Messages, "\n"); !strings.Contains(got, "Ответ: Да; Ошибка=не открывается карточка") {
		t.Fatalf("обработчик Ответ не получил ДиалогПоля: %v", resp.Messages)
	}
}

func TestParseQuestionFields_DropsNonStrings(t *testing.T) {
	if parseQuestionFields(`{"a": 1, "": "x"}`) != nil {
		t.Fatal("нестроковые значения и пустые имена отбрасываются")
	}
	if parseQuestionFields(`не json`) != nil || parseQuestionFields("") != nil {
		t.Fatal("мусор не даёт структуры")
	}
}
