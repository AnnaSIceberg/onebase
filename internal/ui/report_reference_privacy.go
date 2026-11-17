package ui

import (
	"context"
	"fmt"
	"strings"

	"github.com/google/uuid"
	"github.com/ivantit66/onebase/internal/access"
	"github.com/ivantit66/onebase/internal/auth"
	"github.com/ivantit66/onebase/internal/metadata"
	"github.com/ivantit66/onebase/internal/query"
	reportpkg "github.com/ivantit66/onebase/internal/report"
)

// Resolve only columns proven to contain entity references. Unknown UUID-like
// strings never trigger a search across unrelated catalogs/documents.
func (s *Server) resolveUUIDsInReport(ctx context.Context, rows []map[string]any, compiled query.Result, skipCols ...string) {
	refColumns := compiled.RefColumns
	outputCounts := map[string]int{}
	count := func(name string) {
		if logical, ok := compiled.LongIdents[name]; ok {
			name = logical
		}
		outputCounts[strings.ToLower(name)]++
	}
	if compiled.Projection.Simple {
		for _, col := range compiled.Projection.Columns {
			if col.Star {
				return // a wildcard may introduce a second column with the same name
			}
			count(col.Output)
		}
	} else {
		// Подзапрос/ОБЪЕДИНИТЬ: транслятор ссылки не доказывает, колонку
		// результата прослеживает Lineage (reportLineageRefColumns).
		refColumns = s.reportLineageRefColumns(compiled)
		for _, line := range compiled.Projection.Lineage {
			if !line.Result {
				continue
			}
			if line.Star {
				return
			}
			count(line.Output)
		}
	}
	ctx = access.WithReportOutput(ctx)
	cache := map[string]string{}
	for column, target := range refColumns {
		if logical, ok := compiled.LongIdents[column]; ok {
			column = logical
		}
		if outputCounts[strings.ToLower(column)] != 1 {
			continue
		}
		isDetail := len(skipCols) > 0 && skipCols[0] != "" && strings.EqualFold(column, skipCols[0])
		if reportColumnSkipped(column, skipCols) && !isDetail {
			continue
		}
		entity := s.reg.GetEntity(target)
		if entity == nil || !s.canCtx(ctx, string(entity.Kind), entity.Name, "read") {
			continue
		}
		for _, row := range rows {
			key, ok := rowKeyByName(row, column)
			if !ok {
				continue
			}
			var text string
			switch value := row[key].(type) {
			case string:
				text = value
			case []byte:
				text = string(value)
			case uuid.UUID:
				text = value.String()
			default:
				continue
			}
			id, err := uuid.Parse(text)
			if err != nil || id == uuid.Nil {
				continue
			}
			cacheKey := string(entity.Kind) + ":" + entity.Name + ":" + id.String()
			label, found := cache[cacheKey]
			if !found {
				cache[cacheKey] = ""
				record, err := s.store.GetByID(ctx, entity.Name, id, entity)
				if err == nil && s.rowAllowedContext(ctx, entity, "read", record) {
					label = s.reportOutputRecordLabel(ctx, entity, record)
					cache[cacheKey] = label
				}
			}
			if label == "" {
				continue
			}
			if isDetail {
				row[reportDetailLinkLabelKey] = label
			} else {
				row[key] = label
			}
		}
	}
}

// Validate native dependent choices before any List/selected-membership read.
// The scope is the report helper, not the ordinary card/picker endpoints.
func (s *Server) validateReportReferenceChoice(ctx context.Context, p reportpkg.Param, target *metadata.Entity) error {
	allowed := func(name string) bool {
		if !access.ReportOutputFilterAllowed(target, name) {
			return false
		}
		decisions := access.FieldDecisions(auth.UserFromContext(ctx), string(target.Kind), target.Name, target)
		decision, ok := fieldDecisionByName(decisions, name)
		return !ok || !decision.Masked()
	}
	for _, filter := range p.ChoiceFilter {
		field, ok := entityFieldByName(target, filter.Field)
		if !ok || field.RefEntity == "" || !allowed(field.Name) {
			return fmt.Errorf("report reference filter is unavailable")
		}
	}
	if p.ChoiceLabelSuffix != "" {
		field, ok := entityFieldByName(target, p.ChoiceLabelSuffix)
		if !ok || field.RefEntity == "" || !allowed(field.Name) {
			return fmt.Errorf("report reference suffix is unavailable")
		}
		ref := s.reg.GetEntity(field.RefEntity)
		if ref == nil || access.ReportOutputLabelProtected(ref) || !s.canCtx(ctx, string(ref.Kind), ref.Name, "read") {
			return fmt.Errorf("report reference suffix is unavailable")
		}
	}
	return nil
}

// reportLineageRefColumns доказывает ссылочные колонки результата запроса с
// подзапросом или ОБЪЕДИНИТЬ: элемент выборки — простая ссылка
// «Алиас.Реквизит[.Ссылка]» или «Алиас.Ссылка» на источник по алиасу, либо
// колонка подзапроса, которая сама так доказана. Для ОБЪЕДИНИТЬ все ветви
// обязаны дать одну сущность; имя колонки — из первой ветви. Неоднозначное
// не доказывается и остаётся UUID.
func (s *Server) reportLineageRefColumns(res query.Result) map[string]string {
	type column struct {
		name, entity string
		conflict     bool
	}
	byPos := map[int]*column{}
	for _, line := range res.Projection.Lineage {
		if !line.Result {
			continue
		}
		if line.Star {
			return nil
		}
		entity := s.lineageRefTarget(res, line, 0)
		col := byPos[line.Position]
		if col == nil {
			col = &column{entity: entity}
			byPos[line.Position] = col
		} else if col.entity != entity {
			col.conflict = true
		}
		if line.Branch == 0 {
			col.name = line.Output
		}
		if entity == "" {
			col.conflict = true
		}
	}
	out := map[string]string{}
	for _, col := range byPos {
		if !col.conflict && col.name != "" && col.entity != "" {
			out[col.name] = col.entity
		}
	}
	return out
}

func (s *Server) lineageRefTarget(res query.Result, line query.ProjectionLineage, depth int) string {
	if depth > 4 || !line.Plain || len(line.Paths) != 1 {
		return ""
	}
	path := line.Paths[0]
	if refs, ok := res.Projection.SourceAliases[strings.ToLower(path[0])]; ok {
		target := ""
		for i, src := range refs {
			entity := s.referencePathTarget(s.sourceMeta(src.Kind, src.Name), path[1:])
			if entity == "" || (i > 0 && entity != target) {
				return ""
			}
			target = entity
		}
		return target
	}
	// Алиас подзапроса: «Т.Колонка» — колонка вложенного SELECT.
	if len(path) != 2 {
		return ""
	}
	target := ""
	for _, inner := range res.Projection.Lineage {
		if inner.Result || !strings.EqualFold(inner.Output, path[1]) {
			continue
		}
		entity := s.lineageRefTarget(res, inner, depth+1)
		if entity == "" || (target != "" && entity != target) {
			return ""
		}
		target = entity
	}
	return target
}

// referencePathTarget — сущность, на которую ссылается путь полей от meta:
// «Ссылка» (только последним) — сама meta, ссылочный реквизит — его сущность.
func (s *Server) referencePathTarget(meta *metadata.Entity, fields []string) string {
	if meta == nil || len(fields) == 0 {
		return ""
	}
	for i, name := range fields {
		if isReportReferenceName(name) {
			if i != len(fields)-1 || meta.Kind == "" {
				return ""
			}
			return meta.Name
		}
		var ref string
		for _, f := range meta.Fields {
			if strings.EqualFold(f.Name, name) {
				ref = f.RefEntity
				break
			}
		}
		if ref == "" {
			return ""
		}
		meta = s.reg.GetEntity(ref)
		if meta == nil {
			return ""
		}
	}
	return meta.Name
}

func isReportReferenceName(name string) bool {
	return strings.EqualFold(name, "Ссылка") || strings.EqualFold(name, "Ref")
}
