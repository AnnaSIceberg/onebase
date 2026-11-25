package storage

import (
	"context"
	"fmt"
	"strings"

	"github.com/google/uuid"
	"github.com/ivantit66/onebase/internal/metadata"
)

type LinkedDocumentPath struct {
	TablePart *metadata.TablePart
	Field     metadata.Field
}

type LinkedDocument struct {
	Document *metadata.Entity
	Links    []LinkedDocumentPath
	Scope    *Predicate
	Match    func(uuid.UUID) (bool, error)
}

// List/query policies and individual object checks use the same visible links.
// Only resolved metadata supplies identifiers. User values are SQL parameters.
func (l *LinkedDocument) predicateSQL(d Dialect, reference string, args []any, nextArg int) (string, []any, int, error) {
	const owner = "ob_link_owner"
	const part = "ob_link_part"
	if l.Document == nil || len(l.Links) == 0 {
		return "", nil, nextArg, fmt.Errorf("invalid linked_document")
	}
	table := metadata.TableName(l.Document.Name)
	var sources []string
	for _, path := range l.Links {
		projection := owner + "." + metadata.ColumnName(path.Field)
		from := table + " " + owner
		if path.TablePart != nil {
			projection = part + "." + metadata.ColumnName(path.Field)
			from += " JOIN " + metadata.TablePartTableName(l.Document.Name, path.TablePart.Name) +
				" " + part + " ON " + part + ".parent_id = " + owner + ".id"
		}
		scope, params, next, err := PredicateSQLQualified(d, l.Document, l.Scope, nextArg, owner)
		if err != nil {
			return "", nil, nextArg, err
		}
		conditions := owner + ".deletion_mark = FALSE AND " + projection + " IS NOT NULL"
		if scope != "" {
			conditions += " AND (" + scope + ")"
		}
		sources = append(sources, "SELECT "+projection+" FROM "+from+" WHERE "+conditions)
		args = append(args, params...)
		nextArg = next
	}
	condition := reference + " IN (" + strings.Join(sources, " UNION ALL ") + ")"
	return "COALESCE((" + condition + "), " + "FALSE" + ")", args, nextArg, nil
}

func (db *DB) HasLinkedDocument(ctx context.Context, linked *LinkedDocument, id uuid.UUID) (bool, error) {
	d := db.dialect
	condition, args, _, err := linked.predicateSQL(d, d.Placeholder(1), []any{idArg(d, id)}, 2)
	if err != nil {
		return false, err
	}
	rows, err := db.Query(ctx, "SELECT 1 WHERE "+condition, args...)
	if err != nil {
		return false, err
	}
	defer rows.Close()
	return rows.Next(), rows.Err()
}
