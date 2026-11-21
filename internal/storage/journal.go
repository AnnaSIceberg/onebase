package storage

import (
	"context"
	"fmt"
	"strings"
	"time"

	"github.com/google/uuid"
	"github.com/ivantit66/onebase/internal/metadata"
)

// colExprForDoc returns the SQL expression for a journal column in the given entity's SELECT.
// Returns ("NULL", nil) if the column has no matching field in the entity.
//
// приоритет резолва:
//  1. jcol.Map[entity.Name] — явный per-doc mapping (новый рекомендуемый путь).
//  2. Exact match — у документа есть поле с именем jcol.Field.
//  3. Fallback list — старый fallback (back compat); COALESCE если несколько.
func colExprForDoc(jcol metadata.JournalColumn, entity *metadata.Entity) (string, *metadata.Field) {
	if jcol.IsDeletionMark() {
		return metadata.JournalDeletionMarkField, nil
	}
	if jcol.ReferencePath != "" {
		if source := jcol.ReferenceSource(entity); source != nil {
			return metadata.ColumnName(*source), nil
		}
		return "NULL", nil
	}
	// 1. Explicit map для этого документа.
	if jcol.Map != nil {
		if mapped, ok := jcol.Map[entity.Name]; ok && mapped != "" {
			for i := range entity.Fields {
				f := &entity.Fields[i]
				if strings.EqualFold(f.Name, mapped) {
					return metadata.ColumnName(*f), f
				}
			}
			// Map ссылается на несуществующее поле — намеренно
			// возвращаем NULL, не падая на fallback'е молча.
			return "NULL", nil
		}
	}
	// 2. Exact match
	for i := range entity.Fields {
		f := &entity.Fields[i]
		if strings.EqualFold(f.Name, jcol.Field) {
			return metadata.ColumnName(*f), f
		}
	}
	// 3. Fallback match — build COALESCE of all matching columns
	type match struct {
		col string
		f   *metadata.Field
	}
	var found []match
	for _, fb := range jcol.Fallback {
		for i := range entity.Fields {
			f := &entity.Fields[i]
			if strings.EqualFold(f.Name, fb) {
				found = append(found, match{metadata.ColumnName(*f), f})
				break
			}
		}
	}
	if len(found) == 0 {
		return "NULL", nil
	}
	if len(found) == 1 {
		return found[0].col, found[0].f
	}
	cols := make([]string, len(found))
	for i, m := range found {
		cols[i] = m.col
	}
	return "COALESCE(" + strings.Join(cols, ", ") + ")", found[0].f
}

// ColRefMap maps journal column names (lowercase) to the ref entity name (if the column is a reference).
type ColRefMap map[string]string

// JournalQuery executes a UNION ALL across all documents in the journal.
// Returns rows, total count, and a ColRefMap for ref resolution.
func (db *DB) JournalQuery(
	ctx context.Context,
	j *metadata.Journal,
	docs map[string]*metadata.Entity,
	params ListParams,
	limit, offset int,
) ([]map[string]any, int, ColRefMap, error) {
	if limit <= 0 {
		limit = 100
	}

	colRefMap := make(ColRefMap)
	var args []any
	argIdx := 1
	d := db.dialect

	// The source type already belongs to every SELECT. Its presentation
	// position in the journal does not create another SQL column.
	queryColumns := make([]metadata.JournalColumn, 0, len(j.Columns))
	outCols := make([]string, 0, len(j.Columns))
	for _, jcol := range j.Columns {
		if !jcol.IsDocumentKind() {
			queryColumns = append(queryColumns, jcol)
			outCols = append(outCols, strings.ToLower(jcol.Field))
		}
	}

	// Build one SELECT per document
	var selects []string
	for _, docName := range j.Documents {
		entity, ok := docs[docName]
		if !ok {
			continue
		}
		table := metadata.TableName(docName)

		parts := []string{
			fmt.Sprintf("'%s' AS _doc_kind", docName),
			"id",
		}
		for i, jcol := range queryColumns {
			expr, f := colExprForDoc(jcol, entity)
			alias := outCols[i]
			parts = append(parts, fmt.Sprintf("%s AS %s", expr, alias))
			// Populate colRefMap from first doc that has a ref field for this column
			if f != nil && f.RefEntity != "" {
				if _, exists := colRefMap[alias]; !exists {
					colRefMap[alias] = f.RefEntity
				}
			}
		}
		selectSQL := fmt.Sprintf("SELECT %s FROM %s", strings.Join(parts, ", "), table)
		if params.JournalRowFilters != nil {
			if pred := params.JournalRowFilters[docName]; pred != nil {
				cond, condArgs, next, err := PredicateSQL(d, entity, pred, argIdx)
				if err != nil {
					return nil, 0, colRefMap, fmt.Errorf("journal %s row filter: %w", docName, err)
				}
				if cond != "" {
					selectSQL += " WHERE " + cond
					args = append(args, condArgs...)
					argIdx = next
				}
			}
		}
		selects = append(selects, selectSQL)
	}

	if len(selects) == 0 {
		return nil, 0, colRefMap, nil
	}

	union := strings.Join(selects, "\nUNION ALL\n")

	// Build WHERE from filters
	var whereParts []string

	for _, jf := range j.Filters {
		fv, ok := params.Filters[jf.Field]
		if !ok {
			continue
		}
		colName := strings.ToLower(jf.Field)
		switch {
		case jf.Type == "date_range":
			if fv.From != "" {
				condition, arg := journalDateCompare(d, colName, argIdx, fv.From, false)
				whereParts = append(whereParts, condition)
				args = append(args, arg)
				argIdx++
			}
			if fv.To != "" {
				condition, arg := journalDateCompare(d, colName, argIdx, fv.To, true)
				whereParts = append(whereParts, condition)
				args = append(args, arg)
				argIdx++
			}
		case strings.HasPrefix(jf.Type, "reference:"):
			if fv.Value != "" {
				whereParts = append(whereParts, fmt.Sprintf("%s = %s", colName, d.Placeholder(argIdx)))
				if id, err := uuid.Parse(fv.Value); err == nil {
					args = append(args, idArg(d, id))
				} else {
					args = append(args, fv.Value)
				}
				argIdx++
			}
		default:
			if fv.Value != "" {
				whereParts = append(whereParts, d.LowerLike(colName)+" LIKE "+d.LowerLike(d.Placeholder(argIdx)))
				args = append(args, "%"+fv.Value+"%")
				argIdx++
			}
		}
	}

	whereClause := ""
	if len(whereParts) > 0 {
		whereClause = " WHERE " + strings.Join(whereParts, " AND ")
	}

	// Determine ORDER BY: use first date column (by name heuristic) or id
	orderCol := "id"
	for _, jcol := range queryColumns {
		lf := strings.ToLower(jcol.Field)
		if lf == "дата" || lf == "date" || strings.Contains(lf, "дата") {
			orderCol = lf
			break
		}
	}

	// Count query
	countSQL := fmt.Sprintf("WITH j AS (%s)\nSELECT COUNT(*) FROM j%s", union, whereClause)
	var total int
	if err := db.QueryRow(ctx, countSQL, args...).Scan(&total); err != nil {
		return nil, 0, colRefMap, fmt.Errorf("journal count: %w", err)
	}

	// Data query. NULLS LAST is PG-specific; on SQLite ORDER BY DESC already
	// puts NULLs at the bottom, so we emit it only on PG.
	nullsClause := ""
	if d.Name() == "postgres" {
		nullsClause = " NULLS LAST"
	}
	orderDir := "DESC"
	tie := ""
	if params.ChronologicalDocuments {
		orderDir, nullsClause, tie = "ASC", " NULLS FIRST", ", id ASC, _doc_kind ASC"
		if offset < 0 {
			offset = max(total-1, 0) / limit * limit
		}
	}
	selectColumns := append([]string{metadata.JournalDocumentKindField, "id"}, outCols...)
	dataSQL := fmt.Sprintf(
		"WITH j AS (%s)\nSELECT %s FROM j%s ORDER BY %s %s%s%s LIMIT %s OFFSET %s",
		union,
		strings.Join(selectColumns, ", "),
		whereClause,
		orderCol, orderDir, nullsClause, tie,
		d.Placeholder(argIdx), d.Placeholder(argIdx+1),
	)
	dataArgs := append(args, limit, offset)

	rows2, err := db.Query(ctx, dataSQL, dataArgs...)
	if err != nil {
		return nil, 0, colRefMap, fmt.Errorf("journal query: %w", err)
	}
	defer rows2.Close()

	// Scan results: [_doc_kind, id, col0, col1, ...]
	var rows []map[string]any
	for rows2.Next() {
		n := len(selectColumns)
		dest := make([]any, n)
		ptrs := make([]any, n)
		for i := range dest {
			ptrs[i] = &dest[i]
		}
		if err := rows2.Scan(ptrs...); err != nil {
			return nil, 0, colRefMap, fmt.Errorf("journal scan: %w", err)
		}
		row := make(map[string]any, n)
		row[metadata.JournalDocumentKindField] = normalizeValue(dest[0])
		row["id"] = normalizeValue(dest[1])
		for i, jcol := range queryColumns {
			if jcol.IsDeletionMark() {
				row[jcol.Field] = normalizeBool(dest[i+2])
			} else {
				row[jcol.Field] = normalizeValue(dest[i+2])
			}
		}
		rows = append(rows, row)
	}
	return rows, total, colRefMap, rows2.Err()
}

// A calendar selection is a local interval from the start of its first day
// to the start of the day after its last. Explicit times keep their comparison.
func journalDateCompare(d Dialect, column string, position int, raw string, upper bool) (string, any) {
	op := ">="
	if upper {
		op = "<="
	}
	var arg any = raw
	if day, err := time.ParseInLocation(time.DateOnly, raw, time.Local); err == nil {
		if upper {
			day = day.AddDate(0, 0, 1)
			op = "<"
		}
		arg = day
		if d.Name() == "sqlite" {
			// Document dates may use different timestamp encodings in legacy rows.
			column = "ob_local_datetime(" + column + ")"
			arg = day.Format(time.DateTime)
		}
	}
	return fmt.Sprintf("%s %s %s", column, op, d.Placeholder(position)), arg
}
