package ui

import (
	"context"
	"net/http"
	"strings"
	"unicode/utf8"

	"github.com/google/uuid"
	"github.com/ivantit66/onebase/internal/access"
	"github.com/ivantit66/onebase/internal/auth"
	"github.com/ivantit66/onebase/internal/dsl/interpreter"
	"github.com/ivantit66/onebase/internal/metadata"
	"github.com/ivantit66/onebase/internal/storage"
)

// The application validates the answer, then requests a one-record disclosure.
// Field names come from the form declaration and a reading role's policy.
func (s *Server) managedFormDisclosureBuiltin(r *http.Request, entity *metadata.Entity, form *metadata.FormModule) interpreter.BuiltinFunc {
	return func(args []any, _ string, _ int) (any, error) {
		if len(args) != 2 {
			return false, nil
		}
		reason, validReason := args[0].(string)
		controller, validController := args[1].(string)
		reason = strings.TrimSpace(reason)
		if !validReason || !validController || reason == "" || utf8.RuneCountInString(reason) > 500 || strings.TrimSpace(controller) == "" {
			return false, nil
		}
		ctx := r.Context()
		cr, _ := ctx.Value(concealRequestKey{}).(*concealRequest)
		user := auth.UserFromContext(ctx)
		if cr == nil || cr.server != s || cr.close != nil || user == nil {
			return false, nil
		}
		e, _ := cr.instance()
		if e == nil || e.entity != entity || e.form != form {
			return false, nil
		}
		id, err := uuid.Parse(e.object)
		if err != nil || id == uuid.Nil {
			return false, nil
		}
		rules, err := metadata.FormConcealRules(entity, form)
		if err != nil {
			return false, nil
		}
		var fields []string
		for _, rule := range rules {
			if strings.EqualFold(rule.Controller, controller) && access.FieldFormDiscloseAllowed(user, string(entity.Kind), entity.Name, rule.Field) {
				fields = append(fields, rule.Field)
			}
		}
		if len(fields) == 0 {
			return false, nil
		}
		row, err := s.store.GetByID(ctx, entity.Name, id, entity)
		if err != nil || !s.rowAllowedContext(ctx, entity, "read", row) {
			return false, nil
		}
		auditCtx := storage.WithAuditUser(ctx, user.ID, user.Login)
		if err := s.store.LogDisclose(auditCtx, string(entity.Kind), entity.Name, id.String(), strings.Join(fields, ", "), reason); err != nil {
			return false, nil
		}
		cr.audited = make(map[string]bool, len(fields))
		for _, field := range fields {
			cr.audited[field] = true
		}
		return true, nil
	}
}

// Hydrate from the saved record, after normal masking and the session/form/record
// checks. No raw value enters script state, form input, or the disclosure cache.
func (s *Server) restoreAuditedFormFields(ctx context.Context, form *metadata.FormModule, entity *metadata.Entity, id uuid.UUID, values map[string]any, revealed []string) {
	if len(revealed) == 0 || id == uuid.Nil {
		return
	}
	cr, _ := ctx.Value(concealRequestKey{}).(*concealRequest)
	if cr == nil || cr.server != s || cr.close != nil {
		return
	}
	e, _ := cr.instance()
	if e == nil || e.form != form || e.entity != entity || e.object != id.String() {
		return
	}
	user := auth.UserFromContext(ctx)
	var row map[string]any
	for _, field := range revealed {
		if !access.FieldFormDiscloseAllowed(user, string(entity.Kind), entity.Name, field) {
			continue
		}
		if row == nil {
			var err error
			row, err = s.store.GetByID(ctx, entity.Name, id, entity)
			if err != nil || !s.rowAllowedContext(ctx, entity, "read", row) {
				return
			}
		}
		if value, ok := maskCIKeyValue(row, field); ok {
			values[field] = value
		}
	}
}
