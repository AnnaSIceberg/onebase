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
	"github.com/ivantit66/onebase/internal/report/compose"
)

// resolveReportViewReferences preserves identity only for the active, opted-in
// groupings. The ordinary reference resolver still owns all label reads.
func (s *Server) resolveReportViewReferences(ctx context.Context, rows []map[string]any, compiled query.Result, spec *reportpkg.Composition, skipCols ...string) {
	if spec == nil || len(spec.ReferenceGroupings) == 0 {
		s.resolveUUIDsInReport(ctx, rows, compiled, skipCols...)
		return
	}
	var fields []string
	for _, field := range spec.Groupings {
		if reportColumnSkipped(field, spec.ReferenceGroupings) {
			fields = append(fields, field)
		}
	}
	type groupReference struct {
		row   map[string]any
		field string
		raw   any
		id    uuid.UUID
	}
	var references []groupReference
	for _, row := range rows {
		for _, field := range fields {
			key, ok := rowKeyByName(row, field)
			if !ok {
				continue
			}
			raw := row[key]
			parsed := raw
			if bs, ok := raw.([]byte); ok {
				parsed = string(bs)
			}
			if _, id, ok := uuidFromValue(parsed); ok {
				references = append(references, groupReference{row: row, field: key, raw: raw, id: id})
			}
		}
	}

	// The detail link already retains its ID and resolves a separate caption.
	// Resolve any other configured group caption with the same authorization,
	// restoring the original drilldown input afterward.
	resolveSkip := make([]string, 0, len(skipCols))
	for i, field := range skipCols {
		if i == 0 || !reportColumnSkipped(field, fields) {
			resolveSkip = append(resolveSkip, field)
		}
	}
	s.resolveUUIDsInReport(ctx, rows, compiled, resolveSkip...)
	for _, ref := range references {
		label := fmtVal(ref.row[ref.field])
		isDetail := len(skipCols) > 0 && strings.EqualFold(ref.field, skipCols[0])
		if isDetail {
			label = fmtVal(ref.row[reportDetailLinkLabelKey])
		}
		if label == fmtVal(ref.raw) || strings.EqualFold(label, ref.id.String()) {
			label = ""
		}
		if reportColumnSkipped(ref.field, skipCols) {
			ref.row[ref.field] = ref.raw
		} else {
			ref.row[ref.field] = label
		}
		if isDetail {
			ref.row[reportDetailLinkLabelKey] = label
		}
		compose.SetReferenceGroup(ref.row, ref.field, ref.id, label)
	}
}

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
		// Транслятор доказывает только «Ссылка»/реквизит главного источника;
		// ВЫБОР по ссылкам одной сущности и реквизит табличной части
		// доказывает Lineage. Доказанное транслятором не переопределяется.
		if lineage := s.reportLineageRefColumns(compiled); len(lineage) > 0 {
			merged := make(map[string]string, len(refColumns)+len(lineage))
			for column, target := range lineage {
				merged[column] = target
			}
			for column, target := range refColumns {
				merged[column] = target
			}
			refColumns = merged
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
	if depth > 4 || len(line.Paths) == 0 {
		return ""
	}
	if !line.Plain {
		// ВЫБОР КОГДА … ТОГДА З.Направление ИНАЧЕ Б.Направление КОНЕЦ,
		// ЕСТЬNULL(А, Б), МИНИМУМ/МАКСИМУМ(А): значение — одна из ссылок. Все
		// значения обязаны быть ссылками на одну сущность, без голых имён.
		if !refPassThroughCall(line.Call) || line.Star {
			return ""
		}
		inPath := map[string]bool{}
		for _, path := range line.Paths {
			for _, segment := range path {
				inPath[strings.ToLower(segment)] = true
			}
		}
		for _, name := range line.Uses {
			if !inPath[strings.ToLower(name)] && !strings.EqualFold(name, line.Output) {
				return ""
			}
		}
		target := ""
		for _, path := range line.Paths {
			one := query.ProjectionLineage{Plain: true, Paths: [][]string{path}, Uses: path}
			entity := s.lineageRefTarget(res, one, depth+1)
			if entity == "" || (target != "" && entity != target) {
				return ""
			}
			target = entity
		}
		return target
	}
	if len(line.Paths) != 1 {
		return ""
	}
	path := line.Paths[0]
	if refs, ok := res.Projection.SourceAliases[strings.ToLower(path[0])]; ok {
		target := ""
		for i, src := range refs {
			entity := s.referencePathTarget(s.lineageSourceMeta(res, src), path[1:])
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

// refPassThroughCall — выражение, значение которого — одна из его ссылок:
// ВЫБОР (Call пусто), ЕСТЬNULL/COALESCE, МИНИМУМ/МАКСИМУМ.
func refPassThroughCall(call string) bool {
	switch call {
	case "", "естьnull", "isnull", "coalesce", "минимум", "максимум", "min", "max":
		return true
	}
	return false
}

// lineageSourceMeta — метаданные источника по алиасу. Табличная часть
// («Документ.А_ОтчетМастера.Работы КАК Р») приходит в запрос синтетической
// таблицей; её поля берутся у владельца по Sources.
func (s *Server) lineageSourceMeta(res query.Result, ref query.SourceRef) *metadata.Entity {
	if meta := s.sourceMeta(ref.Kind, ref.Name); meta != nil {
		return meta
	}
	for _, src := range res.Sources {
		if src.TablePart == "" || !strings.EqualFold(metadata.TablePartTableName(src.Name, src.TablePart), ref.Name) {
			continue
		}
		owner := s.reg.GetEntity(src.Name)
		if owner == nil {
			return nil
		}
		for i := range owner.TableParts {
			if strings.EqualFold(owner.TableParts[i].Name, src.TablePart) {
				return metadata.TablePartQueryEntity(owner, &owner.TableParts[i], owner.Name)
			}
		}
	}
	return nil
}
