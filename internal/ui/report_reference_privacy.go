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
	if !compiled.Projection.Simple {
		return
	}
	outputCounts := map[string]int{}
	for _, col := range compiled.Projection.Columns {
		if col.Star {
			return // a wildcard may introduce a second column with the same name
		}
		name := col.Output
		if logical, ok := compiled.LongIdents[name]; ok {
			name = logical
		}
		outputCounts[strings.ToLower(name)]++
	}
	ctx = access.WithReportOutput(ctx)
	cache := map[string]string{}
	for column, target := range compiled.RefColumns {
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
		if ref == nil || len(access.ReportOutputDecisions(ref)) > 0 || !s.canCtx(ctx, string(ref.Kind), ref.Name, "read") {
			return fmt.Errorf("report reference suffix is unavailable")
		}
	}
	return nil
}
