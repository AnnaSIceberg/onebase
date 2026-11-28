package ui

import (
	"context"
	"strings"

	"github.com/google/uuid"
	"github.com/ivantit66/onebase/internal/access"
	"github.com/ivantit66/onebase/internal/auth"
	"github.com/ivantit66/onebase/internal/metadata"
)

// Apply only to option copies whose fields have already been masked.
func (s *Server) applyCatalogChoiceLabels(ctx context.Context, entity *metadata.Entity, rows []map[string]any) {
	if entity == nil || entity.ChoiceLabel == nil || len(rows) == 0 {
		return
	}
	spec := entity.ChoiceLabel
	if !auth.RoleScopeMatches(auth.UserFromContext(ctx), nil, spec.ExceptRoles, false) {
		return
	}
	field, ok := entityFieldByName(entity, strings.TrimSpace(spec.SuffixField))
	if !ok {
		return
	}
	related := s.reg.GetEntity(field.RefEntity)
	if related == nil || access.ReportOutputLabelProtected(related) {
		return
	}
	ids := make(map[string]uuid.UUID)
	for _, row := range rows {
		id, err := uuid.Parse(refValueString(recordValueFold(row, field.Name)))
		if err == nil && id != uuid.Nil {
			ids[id.String()] = id
		}
	}
	labels := make(map[string]string, len(ids))
	s.batchReadableLabels(ctx, related, ids, labels)
	for _, row := range rows {
		caption, _ := row["_label"].(string)
		id, err := uuid.Parse(refValueString(recordValueFold(row, field.Name)))
		if err != nil || caption == "" {
			continue
		}
		if suffix := labels[id.String()]; suffix != "" {
			row["_label"] = caption + " (" + suffix + ")"
		}
	}
}
