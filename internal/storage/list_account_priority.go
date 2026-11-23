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
		priority.NameField.Type != metadata.FieldTypeString || len(priority.Roles) == 0 {
		return "", nil, fmt.Errorf("invalid account priority for catalog %s", entity.Name)
	}
	table := metadata.TableName(entity.Name)
	account := table + "." + metadata.ColumnName(priority.AccountField)
	name := db.listSearchFold(table+"."+metadata.ColumnName(priority.NameField), priority)
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
