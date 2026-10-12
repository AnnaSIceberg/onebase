package storage

import (
	"context"
	"fmt"
	"log/slog"

	"github.com/ivantit66/onebase/internal/metadata"
)

// Переименование таблиц и колонок с длинными именами в короткие (#1946).
//
// До metadata.SQLIdent имя длиннее 63 байт уходило в базу как есть: SQLite
// хранила его полностью, PostgreSQL — обрезанным. Теперь у такого имени другое
// физическое имя, и без этого шага миграция завела бы рядом новую пустую
// таблицу или колонку, а данные остались бы в старой, невидимой приложению.
// Переименование — ALTER … RENAME: данные не копируются, индексы и внешние
// ключи следуют за таблицей. Для коротких имён (почти всех) шаг ничего не
// делает и в базу не ходит.

// LongIdentAction — что миграция сделает с одним длинным именем (#1946):
// переименует прежнее физическое имя (Old → New) или откажет (Conflict).
// Имена, которым ничего не нужно (уже короткое, ещё не созданное), в план не
// попадают. План печатает `onebase migrate --dry-run`, по нему же работает
// сама миграция — инвентаризация и действие не расходятся.
type LongIdentAction struct {
	Table    string // таблица колонки; у переименования таблицы — пусто
	Old, New string
	Conflict string // непусто — миграция откажет, ничего не трогая в этом имени
}

func (a LongIdentAction) String() string {
	what := "таблица " + a.Old
	if a.Table != "" {
		what = "колонка " + a.Table + "." + a.Old
	}
	if a.Conflict != "" {
		return what + ": КОНФЛИКТ — " + a.Conflict
	}
	return what + " → " + a.New
}

func (a LongIdentAction) err() error {
	return fmt.Errorf("длинное имя (#1946): %s. Миграция остановлена: данные не переносятся без доказательства сохранности — см. docs/long-identifiers.md", a)
}

// planLegacyTable решает судьбу таблицы с логическим именем logical. claims —
// сколько таблиц конфигурации претендуют на каждое прежнее имя (nil — одна).
// Неоднозначность — отказ, а не угадывание (решение владельца по #1946):
// прежнее имя, общее для двух таблиц (PostgreSQL обрезал их одинаково), и
// пара «новая таблица + прежняя с данными» не разрешаются автоматически.
func (db *DB) planLegacyTable(ctx context.Context, logical string, claims map[string]int) (*LongIdentAction, error) {
	legacy := metadata.LegacySQLIdents(logical)
	if legacy == nil {
		return nil, nil
	}
	target := metadata.SQLIdent(logical)
	exists, err := db.tableExists(ctx, target)
	if err != nil {
		return nil, fmt.Errorf("длинное имя %s: %w", logical, err)
	}
	for _, old := range uniqueStrings(legacy) {
		found, err := db.tableExists(ctx, old)
		if err != nil {
			return nil, fmt.Errorf("длинное имя %s: %w", logical, err)
		}
		if !found {
			continue
		}
		if claims[old] > 1 {
			return &LongIdentAction{Old: old, New: target, Conflict: fmt.Sprintf(
				"на прежнее имя претендуют %d таблицы конфигурации, чья это таблица — не установить", claims[old])}, nil
		}
		if exists {
			has, err := db.tableHasRows(ctx, old)
			if err != nil {
				return nil, fmt.Errorf("длинное имя %s: %w", logical, err)
			}
			if has {
				return &LongIdentAction{Old: old, New: target, Conflict: "есть и новая таблица " + target +
					", и прежняя с данными — какие данные верные, не установить"}, nil
			}
			continue // прежняя пуста — переносить нечего
		}
		return &LongIdentAction{Old: old, New: target}, nil
	}
	return nil, nil
}

// renameLegacyTable переименовывает таблицу с логическим именем logical из её
// прежнего физического имени в metadata.SQLIdent(logical); конфликт — отказ.
func (db *DB) renameLegacyTable(ctx context.Context, logical string) error {
	return db.renameLegacyTableClaimed(ctx, logical, nil)
}

func (db *DB) renameLegacyTableClaimed(ctx context.Context, logical string, claims map[string]int) error {
	a, err := db.planLegacyTable(ctx, logical, claims)
	if err != nil || a == nil {
		return err
	}
	if a.Conflict != "" {
		return a.err()
	}
	if _, err := db.Exec(ctx, fmt.Sprintf("ALTER TABLE %s RENAME TO %s", a.Old, a.New)); err != nil {
		return fmt.Errorf("переименование таблицы %s → %s: %w", a.Old, a.New, err)
	}
	if err := db.renameSchemaMapTable(ctx, a.Old, a.New); err != nil {
		return err
	}
	slog.Info("длинное имя таблицы сокращено", "было", a.Old, "стало", a.New)
	return nil
}

// legacyTableClaims — сколько логических таблиц претендуют на каждое прежнее
// имя: таблица сущности и её табличные части с общим началом PostgreSQL
// обрезал одинаково.
func legacyTableClaims(logicals []string) map[string]int {
	claims := map[string]int{}
	for _, l := range logicals {
		for _, old := range uniqueStrings(metadata.LegacySQLIdents(l)) {
			claims[old]++
		}
	}
	return claims
}

// planLegacyColumns — план для колонок таблицы table с логическими именами
// logical (lower(поле) или lower(поле)_id).
func (db *DB) planLegacyColumns(ctx context.Context, table string, logical []string) ([]LongIdentAction, error) {
	claims := map[string]int{}
	var long []string
	for _, l := range logical {
		legacy := metadata.LegacySQLIdents(l)
		if legacy == nil {
			continue
		}
		long = append(long, l)
		for _, old := range uniqueStrings(legacy) {
			claims[old]++
		}
	}
	if len(long) == 0 {
		return nil, nil
	}
	exists, err := db.tableExists(ctx, table)
	if err != nil || !exists {
		return nil, err
	}
	cols, err := db.tableColumns(ctx, table)
	if err != nil {
		return nil, err
	}
	var out []LongIdentAction
	for _, l := range long {
		target := metadata.SQLIdent(l)
		_, hasTarget := cols[target]
		for _, old := range uniqueStrings(metadata.LegacySQLIdents(l)) {
			if _, ok := cols[old]; !ok {
				continue
			}
			if claims[old] > 1 {
				out = append(out, LongIdentAction{Table: table, Old: old, New: target, Conflict: fmt.Sprintf(
					"на прежнее имя претендуют %d поля, чьё это значение — не установить", claims[old])})
				break
			}
			if hasTarget {
				has, err := db.columnHasData(ctx, table, old)
				if err != nil {
					return nil, err
				}
				if has {
					out = append(out, LongIdentAction{Table: table, Old: old, New: target, Conflict: "есть и новая колонка " +
						target + ", и прежняя с данными — какие данные верные, не установить"})
				}
				break
			}
			out = append(out, LongIdentAction{Table: table, Old: old, New: target})
			break
		}
	}
	return out, nil
}

// renameLegacyColumns переименовывает колонки таблицы table с длинными
// логическими именами из прежних физических имён в metadata.SQLIdent; конфликт
// — отказ до любого переименования в этой таблице.
func (db *DB) renameLegacyColumns(ctx context.Context, table string, logical []string) error {
	plan, err := db.planLegacyColumns(ctx, table, logical)
	if err != nil {
		return err
	}
	for _, a := range plan {
		if a.Conflict != "" {
			return a.err()
		}
	}
	for _, a := range plan {
		if _, err := db.Exec(ctx, fmt.Sprintf("ALTER TABLE %s RENAME COLUMN %s TO %s", table, a.Old, a.New)); err != nil {
			return fmt.Errorf("переименование колонки %s.%s → %s: %w", table, a.Old, a.New, err)
		}
		if err := db.renameSchemaMapColumn(ctx, table, a.Old, a.New); err != nil {
			return err
		}
		slog.Info("длинное имя колонки сокращено", "таблица", table, "было", a.Old, "стало", a.New)
	}
	return nil
}

func (db *DB) tableHasRows(ctx context.Context, table string) (bool, error) {
	rows, err := db.Query(ctx, "SELECT 1 FROM "+table+" LIMIT 1")
	if err != nil {
		return false, err
	}
	defer rows.Close()
	has := rows.Next()
	return has, rows.Err()
}

func (db *DB) columnHasData(ctx context.Context, table, column string) (bool, error) {
	rows, err := db.Query(ctx, "SELECT 1 FROM "+table+" WHERE "+column+" IS NOT NULL LIMIT 1")
	if err != nil {
		return false, err
	}
	defer rows.Close()
	has := rows.Next()
	return has, rows.Err()
}

// PlanLongIdents — инвентаризация длинных имён (#1946) для `migrate
// --dry-run`: что будет переименовано и где миграция откажет. Ничего не
// меняет. Охватывает таблицы и колонки сущностей, их табличных частей,
// регистров накопления и сведений.
func (db *DB) PlanLongIdents(ctx context.Context, entities []*metadata.Entity, registers []*metadata.Register, infoRegs []*metadata.InfoRegister) ([]LongIdentAction, error) {
	var out []LongIdentAction
	addTable := func(logical string, claims map[string]int) error {
		a, err := db.planLegacyTable(ctx, logical, claims)
		if err == nil && a != nil {
			out = append(out, *a)
		}
		return err
	}
	addCols := func(logicalTable string, cols []string) error {
		// Колонки смотрим там, где таблица будет после переименования:
		// под прежним именем, если её ещё не переименовали.
		table := metadata.SQLIdent(logicalTable)
		if ok, err := db.tableExists(ctx, table); err != nil {
			return err
		} else if !ok {
			for _, old := range uniqueStrings(metadata.LegacySQLIdents(logicalTable)) {
				if found, err := db.tableExists(ctx, old); err != nil {
					return err
				} else if found {
					table = old
					break
				}
			}
		}
		plan, err := db.planLegacyColumns(ctx, table, cols)
		out = append(out, plan...)
		return err
	}
	var entityTables []string
	for _, e := range entities {
		entityTables = append(entityTables, metadata.TableLogical(e.Name))
		for _, tp := range e.TableParts {
			entityTables = append(entityTables, metadata.TablePartTableLogical(e.Name, tp.Name))
		}
	}
	claims := legacyTableClaims(entityTables)
	for _, e := range entities {
		if err := addTable(metadata.TableLogical(e.Name), claims); err != nil {
			return nil, err
		}
		if err := addCols(metadata.TableLogical(e.Name), logicalColumnNames(e.Fields)); err != nil {
			return nil, err
		}
		for _, tp := range e.TableParts {
			if err := addTable(metadata.TablePartTableLogical(e.Name, tp.Name), claims); err != nil {
				return nil, err
			}
			if err := addCols(metadata.TablePartTableLogical(e.Name, tp.Name), logicalColumnNames(tp.Fields)); err != nil {
				return nil, err
			}
		}
	}
	for _, r := range registers {
		if err := addTable(metadata.RegisterTableLogical(r.Name), nil); err != nil {
			return nil, err
		}
		fields := append(append(append([]metadata.Field{}, r.Dimensions...), r.Resources...), r.Attributes...)
		if err := addCols(metadata.RegisterTableLogical(r.Name), logicalColumnNames(fields)); err != nil {
			return nil, err
		}
	}
	for _, ir := range infoRegs {
		if err := addTable(metadata.InfoRegTableLogical(ir.Name), nil); err != nil {
			return nil, err
		}
		fields := append(append([]metadata.Field{}, ir.Dimensions...), ir.Resources...)
		if err := addCols(metadata.InfoRegTableLogical(ir.Name), logicalColumnNames(fields)); err != nil {
			return nil, err
		}
	}
	return out, nil
}

// indexName — физическое имя индекса с логическим именем logical на таблице
// table. Для длинного имени удаляет индекс под прежним именем, если он есть и
// стоит на той же таблице: следом создаётся индекс с новым именем, и без
// удаления в базе оставались бы два одинаковых. Индекс чужой таблицы с тем же
// обрезанным именем (так PostgreSQL сводил разные имена в одно) не трогается.
func (db *DB) indexName(ctx context.Context, table, logical string) (string, error) {
	target := metadata.SQLIdent(logical)
	for _, old := range uniqueStrings(metadata.LegacySQLIdents(logical)) {
		owner, err := db.indexTable(ctx, old)
		if err != nil {
			return "", fmt.Errorf("индекс %s: %w", old, err)
		}
		if owner == "" || owner != table {
			continue
		}
		if _, err := db.Exec(ctx, "DROP INDEX IF EXISTS "+old); err != nil {
			return "", fmt.Errorf("удаление индекса %s: %w", old, err)
		}
	}
	return target, nil
}

// indexTable — таблица индекса name ("" — индекса нет).
func (db *DB) indexTable(ctx context.Context, name string) (string, error) {
	var q string
	if db.IsSQLite() {
		q = `SELECT tbl_name FROM sqlite_master WHERE type = 'index' AND name = ?`
	} else {
		q = `SELECT tablename FROM pg_indexes WHERE schemaname = current_schema() AND indexname = $1`
	}
	rows, err := db.Query(ctx, q, name)
	if err != nil {
		return "", err
	}
	defer rows.Close()
	var table string
	if rows.Next() {
		if err := rows.Scan(&table); err != nil {
			return "", err
		}
	}
	return table, rows.Err()
}

// renameSchemaMapTable / renameSchemaMapColumn держат карту полей плана 81 в
// согласии с переименованием: иначе планировщик искал бы поле под старым
// именем таблицы или колонки.
func (db *DB) renameSchemaMapTable(ctx context.Context, old, target string) error {
	if !tableExistsIn(ctx, db, "_schema_fields") {
		return nil
	}
	d := db.dialect
	if _, err := db.Exec(ctx, `UPDATE _schema_fields SET table_name = `+d.Placeholder(1)+` WHERE table_name = `+d.Placeholder(2), target, old); err != nil {
		return fmt.Errorf("schema map: переименование таблицы %s: %w", old, err)
	}
	return nil
}

func (db *DB) renameSchemaMapColumn(ctx context.Context, table, old, target string) error {
	if !tableExistsIn(ctx, db, "_schema_fields") {
		return nil
	}
	d := db.dialect
	if _, err := db.Exec(ctx, `UPDATE _schema_fields SET column_name = `+d.Placeholder(1)+` WHERE table_name = `+d.Placeholder(2)+` AND column_name = `+d.Placeholder(3), target, table, old); err != nil {
		return fmt.Errorf("schema map: переименование колонки %s.%s: %w", table, old, err)
	}
	return nil
}

// logicalColumnNames — логические имена колонок полей (до SQLIdent).
func logicalColumnNames(fields []metadata.Field) []string {
	out := make([]string, 0, len(fields))
	for _, f := range fields {
		out = append(out, metadata.LogicalColumnName(f))
	}
	return out
}

func uniqueStrings(in []string) []string {
	seen := make(map[string]bool, len(in))
	out := in[:0:0]
	for _, s := range in {
		if !seen[s] {
			seen[s] = true
			out = append(out, s)
		}
	}
	return out
}
