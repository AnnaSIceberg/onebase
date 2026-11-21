package ui

import (
	"context"
	"strings"

	"github.com/google/uuid"
	"github.com/ivantit66/onebase/internal/access"
	"github.com/ivantit66/onebase/internal/metadata"
)

// The storage projection carries the source reference only as an intermediate
// value. Both journal outputs clear it before producing a visible cell.
func (s *Server) resolveJournalReferenceColumns(ctx context.Context, journal *metadata.Journal, docs map[string]*metadata.Entity, rows []map[string]any) {
	type refKey struct {
		entity string
		id     uuid.UUID
	}
	cache := make(map[refKey]map[string]any)
	read := func(entity *metadata.Entity, id uuid.UUID) map[string]any {
		key := refKey{entity: entity.Name, id: id}
		if row, ok := cache[key]; ok {
			return row
		}
		row := s.journalReferenceRecord(ctx, entity, id)
		cache[key] = row
		return row
	}

	for _, column := range journal.Columns {
		if column.ReferencePath == "" {
			continue
		}
		projections := make(map[string]metadata.JournalReferenceProjection, len(docs))
		for name, doc := range docs {
			if projection, ok := column.ResolveReference(doc, s.reg.GetEntity); ok {
				projections[name] = projection
			}
		}
		for _, row := range rows {
			raw := row[column.Field]
			delete(row, column.Field)
			projection, ok := projections[asString(row[metadata.JournalDocumentKindField])]
			if !ok {
				continue
			}
			_, sourceID, ok := uuidFromValue(raw)
			if !ok || sourceID == uuid.Nil {
				continue
			}
			related := read(projection.Related, sourceID)
			_, targetID, ok := uuidFromValue(related[projection.Field])
			if !ok || targetID == uuid.Nil {
				continue
			}
			if target := read(projection.Target, targetID); target != nil {
				label := firstStringField(target, projection.Target)
				if label != "<nil>" && !strings.EqualFold(label, targetID.String()) {
					row[column.Field] = label
				}
			}
		}
	}
}

func (s *Server) journalReferenceRecord(ctx context.Context, entity *metadata.Entity, id uuid.UUID) map[string]any {
	if !s.canCtx(ctx, string(entity.Kind), entity.Name, "read") {
		return nil
	}
	row, err := s.store.GetByID(ctx, entity.Name, id, entity)
	if err != nil || row == nil || !s.rowAllowedContext(ctx, entity, "read", row) {
		return nil
	}
	decisions := s.fieldDecisions(ctx, entity)
	for field, decision := range decisions {
		if decision.Masked() {
			// A masked value cannot be traversed as a reference or supply a label.
			decisions[field] = access.FieldDecision{Strategy: access.FieldHide}
		}
	}
	s.maskRecordWithDecisions(decisions, row)
	return row
}
