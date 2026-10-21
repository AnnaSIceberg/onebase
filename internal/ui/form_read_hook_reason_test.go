package ui

import (
	"net/http"
	"strings"
	"testing"

	"github.com/ivantit66/onebase/internal/metadata"
)

// Отказ ПриЧтенииНаСервере показывает пользователю текст ВызватьИсключение —
// «почему нельзя», как Предупреждение в 1С, — а не только общее «нет прав».
// Данные записи при этом по-прежнему не раскрываются.
func TestFormEdit_OnReadAtServerDenialShowsExceptionText(t *testing.T) {
	srv, ent := setupManagedEventsServer(t, `
Процедура ПроверитьДоступ()
	ВызватьИсключение("Запрещен просмотр карточки для должности: Менеджер направления колл центр");
КонецПроцедуры
`, map[metadata.FormEventType]string{
		metadata.FormEventType("ПриЧтенииНаСервере"): "ПроверитьДоступ",
	}, []*metadata.FormElement{
		{Kind: metadata.FormElementField, Name: "Наименование", DataPath: "Объект.Наименование"},
	})
	id := insertContragent(t, srv, ent, "СЕКРЕТНЫЙ-КОНТРАГЕНТ")
	rec := executeFormEditGET(t, srv, ent, id)
	if rec.Code != http.StatusForbidden {
		t.Fatalf("ожидался 403, получен %d", rec.Code)
	}
	body := rec.Body.String()
	if !strings.Contains(body, "Запрещен просмотр карточки для должности: Менеджер направления колл центр") {
		t.Errorf("страница отказа не показала текст исключения:\n%s", body)
	}
	if strings.Contains(body, "СЕКРЕТНЫЙ-КОНТРАГЕНТ") {
		t.Error("при отказе данные записи раскрыты")
	}
	if strings.Contains(body, ".os:") {
		t.Error("пользователю показан путь модуля и номер строки")
	}
}
