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
	// Выражение, меняющее значение ссылки, не доказывается.
	got = resolve(`ВЫБРАТЬ ПОДСТРОКА(З.Филиал, 1, 36) КАК Ф ИЗ Документ.ЗвонокВх КАК З
		ЛЕВОЕ СОЕДИНЕНИЕ (ВЫБРАТЬ Д.Ссылка КАК Ид ИЗ Справочник.ПодразделениеВх КАК Д) КАК Т ПО Т.Ид = З.Подразделение`, map[string]any{"ф": branchID.String()})
	if got["ф"] != branchID.String() {
		t.Fatalf("expression resolved without proof: %#v", got)
	}
}

// perenos-codex-3: ВЫБОР по ссылкам одной сущности и реквизит табличной части
// в простом запросе выводились UUID.
func TestReportReferenceLabelsCaseAndTablePart(t *testing.T) {
	work := &metadata.Entity{Name: "РаботаВх", Kind: metadata.KindCatalog, Fields: []metadata.Field{{Name: "Наименование", Type: metadata.FieldTypeString}}}
	dir := &metadata.Entity{Name: "НаправлениеВх", Kind: metadata.KindCatalog, Fields: []metadata.Field{{Name: "Наименование", Type: metadata.FieldTypeString}}}
	report := &metadata.Entity{Name: "ОтчетВх", Kind: metadata.KindDocument,
		Fields: []metadata.Field{{Name: "Направление", Type: "reference:НаправлениеВх", RefEntity: dir.Name}, {Name: "Резерв", Type: "reference:НаправлениеВх", RefEntity: dir.Name}},
		TableParts: []metadata.TablePart{{Name: "Работы", Fields: []metadata.Field{{Name: "Работа", Type: "reference:РаботаВх", RefEntity: work.Name}}}}}
	entities := []*metadata.Entity{work, dir, report}
	s, ctx := newSubmitTestServer(t, entities)
	workID, dirID := uuid.New(), uuid.New()
	if err := s.store.Upsert(ctx, work.Name, workID, map[string]any{"Наименование": "Замена ТЭНа"}, work); err != nil {
		t.Fatal(err)
	}
	if err := s.store.Upsert(ctx, dir.Name, dirID, map[string]any{"Наименование": "СМА"}, dir); err != nil {
		t.Fatal(err)
	}
	actx := auth.ContextWithUser(ctx, &auth.User{IsAdmin: true})
	resolve := func(text string, row map[string]any) map[string]any {
		t.Helper()
		compiled, err := query.Compile(text, query.CompileOpts{Entities: entities})
		if err != nil {
			t.Fatal(err)
		}
		rows := []map[string]any{row}
		s.resolveUUIDsInReport(actx, rows, compiled)
		return rows[0]
	}
	got := resolve(`ВЫБРАТЬ ВЫБОР КОГДА О.Направление ЕСТЬ NULL ТОГДА О.Резерв ИНАЧЕ О.Направление КОНЕЦ КАК НапрИтог,
		ЕСТЬNULL(О.Направление, О.Резерв) КАК Напр2 ИЗ Документ.ОтчетВх КАК О`,
		map[string]any{"напритог": dirID.String(), "напр2": dirID.String()})
	if got["напритог"] != "СМА" || got["напр2"] != "СМА" {
		t.Fatalf("ВЫБОР/ЕСТЬNULL по ссылкам: %#v", got)
	}
	got = resolve(`ВЫБРАТЬ Р.Работа КАК Работа ИЗ Документ.ОтчетВх.Работы КАК Р`, map[string]any{"работа": workID.String()})
	if got["работа"] != "Замена ТЭНа" {
		t.Fatalf("реквизит табличной части: %#v", got)
	}
	got = resolve(`ВЫБРАТЬ ВЫБОР КОГДА О.Направление ЕСТЬ NULL ТОГДА О.Резерв ИНАЧЕ Р.Работа КОНЕЦ КАК Смесь
		ИЗ Документ.ОтчетВх КАК О ЛЕВОЕ СОЕДИНЕНИЕ Документ.ОтчетВх.Работы КАК Р ПО Р.Ссылка = О.Ссылка`, map[string]any{"смесь": dirID.String()})
	if got["смесь"] != dirID.String() {
		t.Fatalf("ВЫБОР по ссылкам разных сущностей не доказан: %#v", got)
	}
}
