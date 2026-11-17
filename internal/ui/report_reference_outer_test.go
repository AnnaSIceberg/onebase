package ui

import (
	"testing"

	"github.com/google/uuid"

	"github.com/ivantit66/onebase/internal/auth"
	"github.com/ivantit66/onebase/internal/metadata"
	"github.com/ivantit66/onebase/internal/query"
)

// Отчёт с подзапросом в СОЕДИНЕНИЕ (А_МИКО_ВходящиеЗвонкиКЦВыгрузка) выводил UUID
// вместо представлений ссылок: доказанные ссылочные колонки внешнего SELECT
// отбрасывались вместе с подзапросом. Ссылки внешнего SELECT однозначны — им
// нужно представление; ОБЪЕДИНИТЬ по-прежнему не доказывается.
func TestReportReferenceLabelsWithSubquery(t *testing.T) {
	branch := &metadata.Entity{Name: "ФилиалВх", Kind: metadata.KindCatalog, Fields: []metadata.Field{
		{Name: "Наименование", Type: metadata.FieldTypeString}, {Name: "Чат", Type: metadata.FieldTypeString, PII: true},
	}}
	dept := &metadata.Entity{Name: "ПодразделениеВх", Kind: metadata.KindCatalog, Fields: []metadata.Field{
		{Name: "Наименование", Type: metadata.FieldTypeString},
	}}
	call := &metadata.Entity{Name: "ЗвонокВх", Kind: metadata.KindDocument, Fields: []metadata.Field{
		{Name: "Филиал", Type: "reference:ФилиалВх", RefEntity: branch.Name},
		{Name: "Подразделение", Type: "reference:ПодразделениеВх", RefEntity: dept.Name},
	}}
	entities := []*metadata.Entity{branch, dept, call}
	s, ctx := newSubmitTestServer(t, entities)
	branchID, deptID := uuid.New(), uuid.New()
	if err := s.store.Upsert(ctx, branch.Name, branchID, map[string]any{"Наименование": "Север", "Чат": "79990009987"}, branch); err != nil {
		t.Fatal(err)
	}
	if err := s.store.Upsert(ctx, dept.Name, deptID, map[string]any{"Наименование": "КЦ-1"}, dept); err != nil {
		t.Fatal(err)
	}
	actx := auth.ContextWithUser(ctx, &auth.User{IsAdmin: true})
	compile := func(text string) query.Result {
		res, err := query.Compile(text, query.CompileOpts{Entities: entities})
		if err != nil {
			t.Fatal(err)
		}
		return res
	}
	resolve := func(text string, row map[string]any) map[string]any {
		t.Helper()
		rows := []map[string]any{row}
		s.resolveUUIDsInReport(actx, rows, compile(text))
		return rows[0]
	}
	// Как в А_МИКО: ссылка присоединённой таблицы, голый ссылочный реквизит и
	// колонка подзапроса, через алиас которого идёт ссылка.
	got := resolve(`ВЫБРАТЬ П.Ссылка КАК Подразделение, З.Филиал КАК Филиал, Т.Ид КАК ЧерезПодзапрос
		ИЗ Документ.ЗвонокВх КАК З
		ЛЕВОЕ СОЕДИНЕНИЕ (ВЫБРАТЬ Д.Ссылка КАК Ид ИЗ Справочник.ПодразделениеВх КАК Д) КАК Т ПО Т.Ид = З.Подразделение
		ЛЕВОЕ СОЕДИНЕНИЕ Справочник.ПодразделениеВх КАК П ПО П.Ссылка = Т.Ид`,
		map[string]any{"подразделение": deptID.String(), "филиал": branchID.String(), "черезподзапрос": deptID.String()})
	if got["подразделение"] != "КЦ-1" || got["филиал"] != "Север" || got["черезподзапрос"] != "КЦ-1" {
		t.Fatalf("subquery report kept UUIDs: %#v", got)
	}
	// ОБЪЕДИНИТЬ: одна сущность во всех ветвях — представление.
	got = resolve(`ВЫБРАТЬ П.Ссылка КАК Ид ИЗ Справочник.ПодразделениеВх КАК П
		ОБЪЕДИНИТЬ ВСЕ ВЫБРАТЬ З.Подразделение ИЗ Документ.ЗвонокВх КАК З`, map[string]any{"ид": deptID.String()})
	if got["ид"] != "КЦ-1" {
		t.Fatalf("UNION of one entity kept UUID: %#v", got)
	}
	// Разные сущности в ветвях — колонка не доказана, UUID остаётся.
	got = resolve(`ВЫБРАТЬ П.Ссылка КАК Ид ИЗ Справочник.ПодразделениеВх КАК П
		ОБЪЕДИНИТЬ ВСЕ ВЫБРАТЬ Ф.Ссылка ИЗ Справочник.ФилиалВх КАК Ф`, map[string]any{"ид": branchID.String()})
	if got["ид"] != branchID.String() {
		t.Fatalf("UNION column resolved against an unproven entity: %#v", got)
	}
	// Выражение не доказывается.
	got = resolve(`ВЫБРАТЬ ЕСТЬNULL(З.Филиал, З.Филиал) КАК Ф ИЗ Документ.ЗвонокВх КАК З
		ЛЕВОЕ СОЕДИНЕНИЕ (ВЫБРАТЬ Д.Ссылка КАК Ид ИЗ Справочник.ПодразделениеВх КАК Д) КАК Т ПО Т.Ид = З.Подразделение`, map[string]any{"ф": branchID.String()})
	if got["ф"] != branchID.String() {
		t.Fatalf("expression resolved without proof: %#v", got)
	}
}
