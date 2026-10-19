package ui

import (
	"context"
	"net/http"
	"net/http/httptest"
	"net/url"
	"strings"
	"testing"

	"github.com/go-chi/chi/v5"

	"github.com/ivantit66/onebase/internal/metadata"
)

// Объект.Провести() в обработчике кнопки формы проводит документ так же, как
// кнопка «Провести»: ОбработкаПроведения исполняется, документ отмечен
// проведённым. Раньше вызов уходил в runtime.Object и молча ничего не делал:
// документ даже не записывался.
func TestFormObject_ПровестиИзОбработчикаКнопки(t *testing.T) {
	doc := &metadata.Entity{
		Name: "Расчёт", Kind: metadata.KindDocument, Posting: true,
		Fields: []metadata.Field{
			{Name: "Номер", Type: metadata.FieldTypeString},
			{Name: "Отметка", Type: metadata.FieldTypeString},
		},
	}
	form := managedObjectForm(fieldEl("ПолеНомер", "Объект.Номер"), writeButton("КнПровести", "КнПровестиНажатие"))
	form.EntityName = doc.Name
	form.ProgramAST = mustParse(t, `
Процедура КнПровестиНажатие()
	Объект.Провести();
КонецПроцедуры
`)
	doc.Forms = []*metadata.FormModule{form}
	s, ctx := newSubmitTestServerWithPrograms(t, []*metadata.Entity{doc}, map[string]string{doc.Name: `
Процедура ОбработкаПроведения()
	this.Отметка = "проведён";
КонецПроцедуры
`})

	rec := postFormEvent(t, s, doc, "КнПровести")
	if !strings.Contains(rec.Body.String(), `"ok":true`) {
		t.Fatalf("обработчик не провёл документ: %d %s", rec.Code, rec.Body.String())
	}
	row := onlyDefaultsDoc(t, s, ctx, doc)
	if posted, _ := row["posted"].(bool); !posted {
		t.Fatalf("документ не проведён: %#v", row)
	}
	if got := row["Отметка"]; got != "проведён" {
		t.Fatalf("ОбработкаПроведения не исполнена: Отметка = %v", got)
	}
}

// Документ без проведения: Провести() сообщает об этом, а не молчит.
func TestFormObject_ПровестиНепроводимого(t *testing.T) {
	cat := &metadata.Entity{
		Name: "Справка", Kind: metadata.KindDocument,
		Fields: []metadata.Field{{Name: "Номер", Type: metadata.FieldTypeString}},
	}
	form := managedObjectForm(fieldEl("ПолеНомер", "Объект.Номер"), writeButton("КнПровести", "КнПровестиНажатие"))
	form.EntityName = cat.Name
	form.ProgramAST = mustParse(t, `
Процедура КнПровестиНажатие()
	Объект.Провести();
КонецПроцедуры
`)
	cat.Forms = []*metadata.FormModule{form}
	s, _ := newSubmitTestServerWithPrograms(t, []*metadata.Entity{cat}, nil)
	rec := postFormEvent(t, s, cat, "КнПровести")
	if !strings.Contains(rec.Body.String(), "объект не проводится") {
		t.Fatalf("ждали «объект не проводится»: %d %s", rec.Code, rec.Body.String())
	}
}

func postFormEvent(t *testing.T, s *Server, doc *metadata.Entity, element string) *httptest.ResponseRecorder {
	t.Helper()
	body := url.Values{"_element": {element}, "_event": {"Нажатие"}, "Номер": {"0001"}}
	req := httptest.NewRequest(http.MethodPost, "/ui/document/"+doc.Name+"/form-event", strings.NewReader(body.Encode()))
	req.Header.Set("Content-Type", "application/x-www-form-urlencoded; charset=utf-8")
	rctx := chi.NewRouteContext()
	rctx.URLParams.Add("kind", "document")
	rctx.URLParams.Add("entity", doc.Name)
	req = req.WithContext(context.WithValue(req.Context(), chi.RouteCtxKey, rctx))
	rec := httptest.NewRecorder()
	s.handleManagedFormEvent(rec, req)
	return rec
}
