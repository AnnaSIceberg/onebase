package storage

import (
	"fmt"
	"strings"

	"github.com/ivantit66/onebase/internal/metadata"
)

// ListAccountPriority applies to reference options, before pagination. A
// historical author remains selectable after the account or card is archived.
type ListAccountPriority struct {
	AccountField metadata.Field
	NameField    metadata.Field
	Roles        []string
	Staffing     *ListStaffingPriority
}

type ListStaffingPriority struct {
	Register          *metadata.InfoRegister
	DepartmentField   metadata.Field
	PositionField     metadata.Field
	DepartmentCatalog string
	PositionCatalog   string
	Department        string
	Position          string
	ArchiveField      *metadata.Field
}

// The default PostgreSQL C locale lowercases ASCII only. Keep the Cyrillic
// search and ordering of author choices consistent with SQLite.
func (db *DB) listSearchFold(expr string, priority *ListAccountPriority) string {
	fold := db.dialect.LowerLike(expr)
	if priority != nil && db.dialect.Name() == "postgres" {
		return "translate(" + fold + ", 'АБВГДЕЁЖЗИЙКЛМНОПРСТУФХЦЧШЩЪЫЬЭЮЯ', 'абвгдеёжзийклмнопрстуфхцчшщъыьэюя')"
	}
	return fold
}

func (db *DB) listAccountOrder(entity *metadata.Entity, priority *ListAccountPriority, argCount int) (string, []any, error) {
	if entity.Kind != metadata.KindCatalog || priority.AccountField.RefEntity != "_users" ||
		priority.NameField.Type != metadata.FieldTypeString || (priority.Staffing == nil && len(priority.Roles) == 0) {
		return "", nil, fmt.Errorf("invalid account priority for catalog %s", entity.Name)
	}
	table := metadata.TableName(entity.Name)
	account := table + "." + metadata.ColumnName(priority.AccountField)
	name := db.listSearchFold(table+"."+metadata.ColumnName(priority.NameField), priority)
	if priority.Staffing != nil {
		return db.listStaffingOrder(table, account, name, priority.Staffing, argCount)
	}
	args := make([]any, len(priority.Roles))
	placeholders := make([]string, len(args))
	for i, role := range priority.Roles {
		args[i] = role
		placeholders[i] = db.dialect.Placeholder(argCount + i + 1)
	}
	order := fmt.Sprintf(
		"CASE WHEN COALESCE(%s.deletion_mark, FALSE) OR NOT EXISTS (SELECT 1 FROM _users u WHERE u.id = %s) THEN 2 "+
			"WHEN EXISTS (SELECT 1 FROM _user_roles ur JOIN _roles rl ON rl.id = ur.role_id "+
			"WHERE ur.user_id = %s AND rl.name IN (%s)) THEN 0 ELSE 1 END, %s ASC NULLS LAST, %s.id ASC",
		table, account, account, strings.Join(placeholders, ", "), name, table)
	return order, args, nil
}

func (db *DB) listStaffingOrder(table, account, name string, staffing *ListStaffingPriority, argCount int) (string, []any, error) {
	if staffing.Register == nil || len(staffing.Register.Dimensions) != 1 {
		return "", nil, fmt.Errorf("invalid staffing priority for catalog %s", table)
	}
	d := db.dialect
	period, now := "staff.period", d.Now()
	if d.Name() == "sqlite" {
		// Imported ISO timestamps and SQLite driver timestamps have different spellings.
		period, now = "julianday(staff.period)", "julianday('now')"
	}
	inactive := "COALESCE(" + table + ".deletion_mark, FALSE)"
	if staffing.ArchiveField != nil {
		inactive += " OR COALESCE(" + table + "." + metadata.ColumnName(*staffing.ArchiveField) + ", FALSE)"
	}
	order := fmt.Sprintf(
		"CASE WHEN %[14]s THEN 2 "+
			"WHEN (SELECT dep._predefined_name = %[2]s AND job._predefined_name = %[3]s "+
			"AND dep._is_predefined AND job._is_predefined "+
			"AND NOT COALESCE(dep.deletion_mark, FALSE) AND NOT COALESCE(job.deletion_mark, FALSE) "+
			"FROM %[4]s staff "+
			"LEFT JOIN %[5]s dep ON dep.id = staff.%[6]s "+
			"LEFT JOIN %[7]s job ON job.id = staff.%[8]s "+
			"WHERE staff.%[9]s = %[1]s.id AND %[10]s <= %[11]s "+
			"ORDER BY %[10]s DESC LIMIT 1) THEN 0 "+
			"WHEN NOT EXISTS (SELECT 1 FROM _users u WHERE u.id = %[12]s) THEN 2 "+
			"ELSE 1 END, %[13]s ASC NULLS LAST, %[1]s.id ASC",
		table, d.Placeholder(argCount+1), d.Placeholder(argCount+2),
		metadata.InfoRegTableName(staffing.Register.Name),
		metadata.TableName(staffing.DepartmentCatalog), metadata.ColumnName(staffing.DepartmentField),
		metadata.TableName(staffing.PositionCatalog), metadata.ColumnName(staffing.PositionField),
		metadata.ColumnName(staffing.Register.Dimensions[0]), period, now, account, name, inactive)
	return order, []any{staffing.Department, staffing.Position}, nil
}
