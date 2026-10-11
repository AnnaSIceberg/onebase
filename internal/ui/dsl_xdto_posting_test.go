package ui

import (
	"context"
	"encoding/xml"
	"fmt"
	"os"
	"path/filepath"
	"testing"

	"github.com/google/uuid"
	"github.com/ivantit66/onebase/internal/dbtest"
	"github.com/ivantit66/onebase/internal/project"
	"github.com/ivantit66/onebase/internal/storage"
)

// Выполнение обработки, как в procrun: штатные DSL-менеджеры, проведение,
// транзакции и сериализатор должны видеть одно состояние документа.
func TestDSL_XDTOPostingStateMatrix(t *testing.T) {
	dbtest.ForEachDialect(t, func(t *testing.T, db *storage.DB) {
		for _, tc := range []struct {
			name, setup, operation string
			states                 []bool
			exists, posted         bool
		}{
			{"new_autocommit", "", "Док.Провести(); Снимок(Док, Истина);", []bool{false, true}, true, true},
			{"saved_autocommit", "Док.Записать();", "Док.Провести(); Снимок(Док, Истина);", []bool{false, true}, true, true},
			{"loaded_autocommit", "Док.Записать(); Док = Док.Ссылка.ПолучитьОбъект();", "Док.Провести(); Снимок(Док, Истина);", []bool{false, true}, true, true},
			{"commit", "Док.Записать();", `
				НачатьТранзакцию(); Док.Провести(); Снимок(Док, Истина);
				ЗафиксироватьТранзакцию(); Снимок(Док, Истина);`, []bool{false, true, true}, true, true},
			{"rollback_saved", "Док.Записать();", `
				НачатьТранзакцию(); Док.Провести(); Снимок(Док, Истина);
				ОтменитьТранзакцию(); Снимок(Док, Ложь);`, []bool{false, true, false}, true, false},
			{"rollback_new", "", `
				НачатьТранзакцию(); Док.Провести(); Снимок(Док, Истина);
				ОтменитьТранзакцию(); Снимок(Док, Ложь);`, []bool{false, true, false}, false, false},
			{"rollback_repost", "Док.Провести();", `
				НачатьТранзакцию(); Док.Провести(); Снимок(Док, Истина);
				ОтменитьТранзакцию(); Снимок(Док, Истина);`, []bool{true, true, true}, true, true},
			{"rollback_repeated", "Док.Записать();", `
				НачатьТранзакцию(); Док.Провести(); Снимок(Док, Истина);
				Док.Провести(); Снимок(Док, Истина);
				ОтменитьТранзакцию(); Снимок(Док, Ложь);`, []bool{false, true, true, false}, true, false},
			{"rejected", "Док.Записать();", `
				Док.Отказать = Истина; БылОтказ = Ложь;
				Попытка Док.Провести(); Исключение БылОтказ = Истина; КонецПопытки;
				Если Не БылОтказ Тогда ВызватьИсключение "ожидался отказ"; КонецЕсли;
				Снимок(Док, Ложь);`, []bool{false, false}, true, false},
		} {
			t.Run(tc.name, func(t *testing.T) {
				dir := t.TempDir()
				files := map[string]string{
					"documents/док.yaml":    "name: Док\nposting: true\nfields:\n  - {name: Номер, type: string}\n  - {name: Дата, type: date}\n  - {name: Отказать, type: bool}\n",
					"processors/проба.yaml": "name: Проба\n",
					"src/док.posting.os": `Процедура ОбработкаПроведения()
						Если this.Отказать Тогда ВызватьИсключение "проведение отклонено"; КонецЕсли;
					КонецПроцедуры`,
					"src/проба.proc.os": fmt.Sprintf(`Процедура Снимок(Док, Ожидаемое)
						Если Док.Проведен <> Ожидаемое Тогда ВызватьИсключение "неверный признак объекта"; КонецЕсли;
						Сообщить(СериализаторXDTO.ЗаписатьXML(Док));
					КонецПроцедуры
					Процедура Выполнить()
						Док = Документы.Док.Создать(); Док.Дата = ТекущаяДата(); Док.Номер = "%s";
						%s
						Снимок(Док, %s);
						%s
					КонецПроцедуры`, tc.name, tc.setup, map[bool]string{false: "Ложь", true: "Истина"}[tc.states[0]], tc.operation),
				}
				for name, body := range files {
					path := filepath.Join(dir, name)
					if err := os.MkdirAll(filepath.Dir(path), 0o755); err != nil {
						t.Fatal(err)
					}
					if err := os.WriteFile(path, []byte(body), 0o644); err != nil {
						t.Fatal(err)
					}
				}
				proj, err := project.Load(dir)
				if err != nil {
					t.Fatal(err)
				}
				defer proj.Close()
				ctx := context.Background()
				if err := db.Migrate(ctx, proj.Entities); err != nil {
					t.Fatal(err)
				}
				messages, runErr, err := RunProcessorOffline(ctx, proj, db, "Проба", nil, nil)
				if err != nil || runErr != nil {
					t.Fatalf("RunProcessorOffline: %v / %v; messages: %q", err, runErr, messages)
				}
				if len(messages) != len(tc.states) {
					t.Fatalf("got %d snapshots, want %d: %q", len(messages), len(tc.states), messages)
				}
				var id uuid.UUID
				for i, message := range messages {
					var snapshot struct {
						Ref    string
						Posted *bool
					}
					if err := xml.Unmarshal([]byte(message), &snapshot); err != nil {
						t.Fatal(err)
					}
					if snapshot.Posted == nil || *snapshot.Posted != tc.states[i] {
						t.Fatalf("snapshot %d: want Posted=%t: %s", i, tc.states[i], message)
					}
					ref, err := uuid.Parse(snapshot.Ref)
					if err != nil || ref == uuid.Nil || (i > 0 && ref != id) {
						t.Fatalf("snapshot %d changed Ref: %s", i, snapshot.Ref)
					}
					id = ref
				}
				rows, err := db.List(ctx, "Док", proj.Entities[0], storage.ListParams{})
				if err != nil {
					t.Fatal(err)
				}
				var row map[string]any
				for _, candidate := range rows {
					if fmt.Sprint(candidate["id"]) == id.String() {
						row = candidate
						break
					}
				}
				if (row != nil) != tc.exists || (row != nil && asBool(row["posted"]) != tc.posted) {
					t.Fatalf("stored state: want exists=%t, posted=%t; got %v", tc.exists, tc.posted, row)
				}
			})
		}
	})
}
