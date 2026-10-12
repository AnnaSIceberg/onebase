package storage_test

import (
	"context"
	"strings"
	"testing"

	"github.com/google/uuid"

	"github.com/ivantit66/onebase/internal/dbtest"
	"github.com/ivantit66/onebase/internal/metadata"
	"github.com/ivantit66/onebase/internal/storage"
)

// Решение владельца по #1946: неоднозначное прежнее имя — отказ, а не
// угадывание. Миграция не переименовывает ничего, чью принадлежность или
// сохранность данных нельзя доказать, а `migrate --dry-run` показывает это
// заранее (PlanLongIdents).

// Два реквизита с общими первыми 31 буквами PostgreSQL обрезал в одно имя.
// Колонка под таким именем — чьё в ней значение, не установить.
func TestMigrateRefusesAmbiguousLegacyColumn_Matrix(t *testing.T) {
	cat := &metadata.Entity{
		Name: "ФилиалКонфликт",
		Kind: metadata.KindCatalog,
		Fields: []metadata.Field{
			{Name: "ИспользоватьКоэффициентИзмененияЦенМастера", Type: metadata.FieldTypeString},
			{Name: "ИспользоватьКоэффициентИзмененияЦенВРемонт", Type: metadata.FieldTypeString},
		},
	}
	a, b := metadata.LogicalColumnName(cat.Fields[0]), metadata.LogicalColumnName(cat.Fields[1])
	truncated := metadata.LegacySQLIdents(a)[1]
	if truncated != metadata.LegacySQLIdents(b)[1] {
		t.Fatal("реквизиты должны обрезаться PostgreSQL в одно имя — иначе тест ничего не проверяет")
	}
	dbtest.ForEachDialect(t, func(t *testing.T, db *storage.DB) {
		ctx := context.Background()
		d := db.Dialect()
		table := metadata.TableName(cat.Name)
		if _, err := db.Exec(ctx, "CREATE TABLE "+table+" (id "+d.TypeUUID()+" PRIMARY KEY, "+truncated+" TEXT)"); err != nil {
			t.Fatal(err)
		}
		if _, err := db.Exec(ctx, "INSERT INTO "+table+" (id, "+truncated+") VALUES ("+d.Placeholder(1)+", 'да')", uuid.New().String()); err != nil {
			t.Fatal(err)
		}
		plan, err := db.PlanLongIdents(ctx, []*metadata.Entity{cat}, nil, nil)
		if err != nil {
			t.Fatal(err)
		}
		if len(plan) != 2 || plan[0].Conflict == "" || plan[1].Conflict == "" {
			t.Fatalf("dry-run обязан показать конфликт по обоим реквизитам: %v", plan)
		}
		err = db.Migrate(ctx, []*metadata.Entity{cat})
		if err == nil || !strings.Contains(err.Error(), "КОНФЛИКТ") || !strings.Contains(err.Error(), truncated) {
			t.Fatalf("миграция обязана отказать с конфликтом по %s, получили: %v", truncated, err)
		}
		// Ничего не переименовано: прежняя колонка с данными на месте.
		var v string
		if err := db.QueryRow(ctx, "SELECT "+truncated+" FROM "+table).Scan(&v); err != nil || v != "да" {
			t.Fatalf("прежняя колонка тронута: %q, %v", v, err)
		}
	})
}

// И новая колонка, и прежняя с данными — какие данные верные, не установить:
// отказ. Прежняя без данных — переносить нечего, миграция проходит.
func TestMigrateRefusesLegacyColumnBesideNew_Matrix(t *testing.T) {
	cat := &metadata.Entity{
		Name:   "КонфликтНовойИПрежней",
		Kind:   metadata.KindCatalog,
		Fields: []metadata.Field{{Name: "РеквизитСОченьДлиннымИменемКоторыйПришёлИзОдинЭс", Type: metadata.FieldTypeString}},
	}
	logical := metadata.LogicalColumnName(cat.Fields[0])
	full, target := metadata.LegacySQLIdents(logical)[0], metadata.SQLIdent(logical)
	for _, withData := range []bool{true, false} {
		dbtest.ForEachDialect(t, func(t *testing.T, db *storage.DB) {
			ctx := context.Background()
			d := db.Dialect()
			table := metadata.TableName(cat.Name)
			legacy := full
			if !db.IsSQLite() {
				legacy = metadata.LegacySQLIdents(logical)[1] // так её хранил PostgreSQL
			}
			if _, err := db.Exec(ctx, "CREATE TABLE "+table+" (id "+d.TypeUUID()+" PRIMARY KEY, "+target+" TEXT, "+legacy+" TEXT)"); err != nil {
				t.Fatal(err)
			}
			if withData {
				if _, err := db.Exec(ctx, "INSERT INTO "+table+" (id, "+legacy+") VALUES ("+d.Placeholder(1)+", 'старое')", uuid.New().String()); err != nil {
					t.Fatal(err)
				}
			}
			err := db.Migrate(ctx, []*metadata.Entity{cat})
			if withData && (err == nil || !strings.Contains(err.Error(), "есть и новая колонка")) {
				t.Fatalf("с данными в прежней колонке — отказ, получили: %v", err)
			}
			if !withData && err != nil {
				t.Fatalf("прежняя колонка пуста — миграция проходит, получили: %v", err)
			}
		})
	}
}

// Таблица сущности и её табличная часть с общим началом: PostgreSQL обрезал
// их в одно имя. Чья таблица под этим именем — не установить: отказ.
func TestMigrateRefusesAmbiguousLegacyTable_Matrix(t *testing.T) {
	doc := &metadata.Entity{
		Name:   "ДокументСОченьДлиннымИменемИзКонфигурацииОдинЭс",
		Kind:   metadata.KindDocument,
		Fields: []metadata.Field{{Name: "Номер", Type: metadata.FieldTypeString}},
		TableParts: []metadata.TablePart{
			{Name: "Товары", Fields: []metadata.Field{{Name: "Строчка", Type: metadata.FieldTypeString}}},
		},
	}
	tableOld := metadata.LegacySQLIdents(metadata.TableLogical(doc.Name))
	tpOld := metadata.LegacySQLIdents(metadata.TablePartTableLogical(doc.Name, "Товары"))
	if tableOld == nil || tpOld == nil || tableOld[1] != tpOld[1] {
		t.Fatal("таблица и табличная часть должны обрезаться в одно имя — иначе тест ничего не проверяет")
	}
	dbtest.ForEachDialect(t, func(t *testing.T, db *storage.DB) {
		ctx := context.Background()
		if _, err := db.Exec(ctx, "CREATE TABLE "+tableOld[1]+" (id "+db.Dialect().TypeUUID()+" PRIMARY KEY)"); err != nil {
			t.Fatal(err)
		}
		err := db.Migrate(ctx, []*metadata.Entity{doc})
		if err == nil || !strings.Contains(err.Error(), "претендуют 2 таблицы") {
			t.Fatalf("общее прежнее имя таблицы и табличной части — отказ, получили: %v", err)
		}
	})
}

// dry-run показывает переименование до миграции, сама база не меняется.
func TestPlanLongIdentsListsRename_Matrix(t *testing.T) {
	cat := &metadata.Entity{
		Name:   "СправочникСОченьДлиннымИменемИзКонфигурации1С",
		Kind:   metadata.KindCatalog,
		Fields: []metadata.Field{{Name: "Наименование", Type: metadata.FieldTypeString}},
	}
	logical := metadata.TableLogical(cat.Name)
	dbtest.ForEachDialect(t, func(t *testing.T, db *storage.DB) {
		ctx := context.Background()
		old := metadata.LegacySQLIdents(logical)[0]
		if !db.IsSQLite() {
			old = metadata.LegacySQLIdents(logical)[1]
		}
		if _, err := db.Exec(ctx, "CREATE TABLE "+old+" (id "+db.Dialect().TypeUUID()+" PRIMARY KEY, наименование TEXT)"); err != nil {
			t.Fatal(err)
		}
		plan, err := db.PlanLongIdents(ctx, []*metadata.Entity{cat}, nil, nil)
		if err != nil {
			t.Fatal(err)
		}
		if len(plan) != 1 || plan[0].Old != old || plan[0].New != metadata.SQLIdent(logical) || plan[0].Conflict != "" {
			t.Fatalf("ждали одно переименование %s → %s, получили %v", old, metadata.SQLIdent(logical), plan)
		}
		if exists, err := db.TableExists(ctx, old); err != nil || !exists {
			t.Fatalf("dry-run не должен менять базу: %v", err)
		}
	})
}
