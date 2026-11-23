package ui

import (
	"context"
	"net/http"
	"slices"
	"strings"

	"github.com/ivantit66/onebase/internal/auth"
	"github.com/ivantit66/onebase/internal/metadata"
	"github.com/ivantit66/onebase/internal/storage"
)

func (s *Server) listFilterAccountPriority(ctx context.Context, source *metadata.Entity, filter *metadata.Field, target *metadata.Entity) *storage.ListAccountPriority {
	opt := s.cfg.UIListFilterAccountPriority
	if opt == nil || source == nil || filter == nil || target == nil ||
		source.Kind != metadata.KindDocument || target.Kind != metadata.KindCatalog ||
		!strings.EqualFold(filter.Name, opt.Field) || !strings.EqualFold(filter.RefEntity, target.Name) ||
		!slices.ContainsFunc(opt.Documents, func(name string) bool { return strings.EqualFold(name, source.Name) }) ||
		!auth.RoleScopeMatches(auth.UserFromContext(ctx), nil, s.cfg.UIExceptRoles, false) {
		return nil
	}
	account := listFilterField(target, opt.AccountField)
	name := listFilterField(target, opt.NameField)
	if account == nil || account.RefEntity != "_users" || name == nil || name.Type != metadata.FieldTypeString ||
		name.RefEntity != "" || name.EnumName != "" || len(opt.Roles) == 0 {
		return nil
	}
	return &storage.ListAccountPriority{AccountField: *account, NameField: *name, Roles: opt.Roles}
}

func (s *Server) listFilterPriorityFields(ctx context.Context, entity *metadata.Entity, fields []metadata.Field) map[string]string {
	if s.cfg.UIListFilterAccountPriority == nil {
		return nil
	}
	for _, filter := range fields {
		target := s.reg.GetEntity(filter.RefEntity)
		if s.listFilterAccountPriority(ctx, entity, &filter, target) != nil {
			return map[string]string{filter.Name: entity.Name + ":" + filter.Name}
		}
	}
	return nil
}

func (s *Server) listFilterPriorityFromRequest(r *http.Request, target *metadata.Entity) *storage.ListAccountPriority {
	sourceName, fieldName, ok := strings.Cut(r.URL.Query().Get("list_filter"), ":")
	if !ok || s.cfg.UIListFilterAccountPriority == nil {
		return nil
	}
	source := s.reg.GetEntity(sourceName)
	if source == nil || !s.can(r, string(source.Kind), source.Name, "read") {
		return nil
	}
	return s.listFilterAccountPriority(r.Context(), source, listFilterField(source, fieldName), target)
}
