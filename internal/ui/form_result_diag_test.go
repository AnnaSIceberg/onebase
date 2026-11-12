package ui

import (
	"bytes"
	"context"
	"log/slog"
	"net/url"
	"strings"
	"testing"

	"github.com/google/uuid"
)

// Отказ гарда ОткрытьФормуРезультата пишет в журнал сервера первую причину:
// «не записана в этом обработчике» или имя поля, расходящегося с базой, —
// без самого значения (ПДн). Оператор видит прежний текст, поведение то же.
func TestResultFormRefusalLogsFirstReason(t *testing.T) {
	for _, tc := range []struct {
		name, body, want string
	}{
		{"unsaved-field", "Объект.Записать();\n  Объект.Наименование = \"секрет-после-записи\";", "entity-field Наименование"},
		{"not-written", "", "saved=false"},
	} {
		t.Run(tc.name, func(t *testing.T) {
			var buf bytes.Buffer
			prev := slog.Default()
			slog.SetDefault(slog.New(slog.NewTextHandler(&buf, &slog.HandlerOptions{Level: slog.LevelWarn})))
			t.Cleanup(func() { slog.SetDefault(prev) })

			s, e, _ := resultTestServer(t)
			e.Forms[0].ProgramAST = mustParse(t, `
Процедура Завершить()
  `+tc.body+`
  ОткрытьФормуРезультата("Результат", Объект.Ссылка, "Документ");
КонецПроцедуры`)
			id := uuid.New()
			if err := s.store.Upsert(context.Background(), e.Name, id, map[string]any{"Наименование": "saved"}, e); err != nil {
				t.Fatal(err)
			}
			resp := resultSourceEvent(t, s, e, url.Values{"_event": {"Нажатие"}, "_element": {"Завершить"}, "_id": {id.String()}, "Наименование": {"saved"}})
			if resp.OK || resp.ResultForm != nil || !strings.Contains(resp.Error, "исходная карточка должна быть успешно записана") {
				t.Fatalf("guard must still refuse with the same text: %+v", resp)
			}
			log := buf.String()
			if !strings.Contains(log, "отказ гарда исходной карточки") || !strings.Contains(log, tc.want) || !strings.Contains(log, id.String()) {
				t.Fatalf("server log lacks the reason %q: %s", tc.want, log)
			}
			if strings.Contains(log, "секрет-после-записи") {
				t.Fatalf("server log leaks the field value: %s", log)
			}
		})
	}
}
