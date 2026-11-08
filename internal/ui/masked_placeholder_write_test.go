package ui

import (
	"context"
	"strings"
	"testing"

	"github.com/google/uuid"
	"github.com/ivantit66/onebase/internal/auth"
	"github.com/ivantit66/onebase/internal/dbtest"
	"github.com/ivantit66/onebase/internal/dsl/interpreter"
	"github.com/ivantit66/onebase/internal/dsl/lexer"
	"github.com/ivantit66/onebase/internal/dsl/parser"
	"github.com/ivantit66/onebase/internal/metadata"
	"github.com/ivantit66/onebase/internal/runtime"
	"github.com/ivantit66/onebase/internal/storage"
)

// Модуль под ролью с маской ПДн читает телефон клиента как «••••••». Раньше
// ничто не мешало присвоить заглушку реквизиту нового документа и записать —
// так в базу оператора КЦ попали телефоны «••••••». Путь пользовательский:
// DSL-окружение запроса, Документы.Заявка.Создать() / Записать().
func TestDSL_MaskedPlaceholderCannotBeWrittenToAnotherObject(t *testing.T) {
	client, order := dslMaskEntities()
	s, ctx := newSubmitTestServer(t, []*metadata.Entity{client, order})
	clientID := uuid.New()
	if err := s.store.Upsert(ctx, "Клиент", clientID, map[string]any{
		"Наименование": "Иванов", "Телефон": "+79161234455",
	}, client); err != nil {
		t.Fatal(err)
	}
	user := &auth.User{ID: "op", Login: "operator", Roles: []*auth.Role{{Name: "Оператор", Permissions: auth.Permission{
		Catalogs:    map[string][]string{"Клиент": {"read"}},
		Documents:   map[string][]string{"Заявка": {"read", "write"}},
		FieldAccess: auth.FieldAccess{Catalogs: map[string]auth.FieldPolicies{"Клиент": {"Телефон": {Read: "mask_all"}}}},
	}}}}
	uctx := auth.ContextWithUser(ctx, user)

	run := func(phone string) error {
		t.Helper()
		var msgs []string
		vars, tx := s.buildDSLVarsWithMessagesTx(uctx, runtime.NewMovementsCollector("test", uuid.Nil), &msgs)
		defer rollbackDSLExecution(tx)
		vars["Клиент"] = &interpreter.Ref{UUID: clientID.String(), Type: "Клиент", AttrResolver: s.newDSLRefAttrResolver(uctx)}
		prog, err := parser.New(lexer.New(`Процедура Скопировать()
	З = Документы.Заявка.Создать();
	З.Номер = "1";
	З.Телефон = `+phone+`;
	З.Записать();
КонецПроцедуры`, "copy.os")).ParseProgram()
		if err != nil {
			t.Fatal(err)
		}
		return finishDSLExecution(tx, interpreter.New().Run(prog.Procedures[0], nil, vars))
	}

	err := run("Клиент.Телефон")
	if err == nil || !strings.Contains(err.Error(), "скрытое значение нельзя записать") {
		t.Fatalf("заглушка маски записана в другой объект: err=%v", err)
	}
	if rows, _ := s.store.List(ctx, "Заявка", order, storage.ListParams{}); len(rows) != 0 {
		t.Fatalf("документ с заглушкой сохранён: %v", rows)
	}
	// Обычная запись не затронута.
	if err := run(`"+79990001122"`); err != nil {
		t.Fatalf("обычная запись отвергнута: %v", err)
	}
}

// Одиночная точка mask_tail перед пробельным хвостом похожа на пункт списка.
// Проверяем реальное чтение под ролью и попытку скопировать маску через DSL,
// затем обычную запись списка, которую storage по-прежнему должен принимать.
func TestDSL_MaskTailWhitespaceCannotBeWrittenToAnotherObject(t *testing.T) {
	for _, tc := range []struct {
		name, phone, shown string
	}{
		{"space", "1 234", storage.MaskFixed},
		{"tab", "1\t234", storage.MaskFixed},
		{"newline", "1\n234", storage.MaskFixed},
		{"nonbreaking space", "1\u00a0234", storage.MaskFixed},
		{"em space", "1\u2003234", storage.MaskFixed},
		{"two masked runes", "12 234", "•• 234"},
	} {
		t.Run(tc.name, func(t *testing.T) {
			dbtest.ForEachDialect(t, func(t *testing.T, db *storage.DB) {
				ctx := context.Background()
				client, order := dslMaskEntities()
				entities := []*metadata.Entity{client, order}
				if err := db.Migrate(ctx, entities); err != nil {
					t.Fatal(err)
				}
				reg := runtime.NewRegistry()
				reg.Load(runtime.LoadOptions{Entities: entities})
				interp := interpreter.New()
				interp.LookupProc = reg.GetModuleProc
				s := &Server{store: db, reg: reg, interp: interp,
					lockMgr: runtime.NewLockManager(), messages: NewMessageStore()}
				s.entitySvc = s.newEntityService(nil)
				clientID := uuid.New()
				if err := db.Upsert(ctx, client.Name, clientID, map[string]any{
					"Наименование": "Клиент", "Телефон": tc.phone,
				}, client); err != nil {
					t.Fatal(err)
				}
				user := &auth.User{ID: "op", Login: "operator", Roles: []*auth.Role{{
					Name: "Оператор", Permissions: auth.Permission{
						Catalogs:  map[string][]string{"Клиент": {"read"}},
						Documents: map[string][]string{"Заявка": {"read", "write"}},
						FieldAccess: auth.FieldAccess{Catalogs: map[string]auth.FieldPolicies{
							"Клиент": {"Телефон": {Read: "mask_tail", Keep: 4}},
						}},
					},
				}}}
				uctx := auth.ContextWithUser(ctx, user)
				ref := &interpreter.Ref{UUID: clientID.String(), Type: client.Name,
					AttrResolver: s.newDSLRefAttrResolver(uctx)}
				read, err := parser.New(lexer.New(`Функция Прочитать()
	Возврат Клиент.Телефон;
КонецФункции`, "read-mask.os")).ParseProgram()
				if err != nil {
					t.Fatal(err)
				}
				var shown any
				if err := interpreter.New().RunWithResult(read.Procedures[0], nil, &shown, map[string]any{"Клиент": ref}); err != nil {
					t.Fatal(err)
				}
				if shown != tc.shown {
					t.Errorf("DSL mask = %q, want %q", shown, tc.shown)
				}
				run := func(value string) error {
					t.Helper()
					var msgs []string
					vars, tx := s.buildDSLVarsWithMessagesTx(uctx, runtime.NewMovementsCollector("test", uuid.Nil), &msgs)
					defer rollbackDSLExecution(tx)
					vars["Клиент"] = ref
					prog, err := parser.New(lexer.New(`Процедура Скопировать()
	З = Документы.Заявка.Создать();
	З.Номер = "1";
	З.Телефон = `+value+`;
	З.Записать();
КонецПроцедуры`, "copy-mask.os")).ParseProgram()
					if err != nil {
						t.Fatal(err)
					}
					return finishDSLExecution(tx, interpreter.New().Run(prog.Procedures[0], nil, vars))
				}
				if err := run("Клиент.Телефон"); err == nil || !strings.Contains(err.Error(), "скрытое значение нельзя записать") {
					t.Errorf("copying mask must fail with masked-value rejection, got %v", err)
				}
				rows, err := db.List(ctx, order.Name, order, storage.ListParams{})
				if err != nil {
					t.Fatal(err)
				}
				if len(rows) != 0 {
					t.Fatalf("document containing mask committed: %v", rows)
				}
				if err := run(`"• Позвонить клиенту"`); err != nil {
					t.Fatalf("ordinary bullet text rejected: %v", err)
				}
				rows, err = db.List(ctx, order.Name, order, storage.ListParams{})
				if err != nil {
					t.Fatal(err)
				}
				if len(rows) != 1 || rows[0]["Телефон"] != "• Позвонить клиенту" {
					t.Fatalf("ordinary bullet text not preserved: %v", rows)
				}
				stored, err := db.GetByID(ctx, client.Name, clientID, client)
				if err != nil {
					t.Fatal(err)
				}
				if stored["Телефон"] != tc.phone {
					t.Fatalf("source changed: %v", stored)
				}
			})
		})
	}
}
