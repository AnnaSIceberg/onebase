package ui

import (
	"context"
	"slices"
	"strings"
	"sync/atomic"

	"github.com/google/uuid"
	"github.com/ivantit66/onebase/internal/auth"
	"github.com/ivantit66/onebase/internal/i18n/i18nerr"
	"github.com/ivantit66/onebase/internal/metadata"
	"github.com/ivantit66/onebase/internal/query"
	"github.com/ivantit66/onebase/internal/storage"
)

type savedPhoneMatchContextKey struct{}

type savedPhoneMatchBinding struct {
	source *metadata.Entity
	id     uuid.UUID
	spec   *metadata.SavedPhoneMatchSpec
	match  atomic.Pointer[query.SavedPhoneMatch]
}

func withSavedPhoneMatch(ctx context.Context, source *metadata.Entity, form *metadata.FormModule, id uuid.UUID) context.Context {
	if form.SavedPhoneMatch == nil || source.Kind != metadata.KindDocument ||
		!strings.EqualFold(form.Kind, "object") || id == uuid.Nil || auth.UserFromContext(ctx) == nil {
		return ctx
	}
	return context.WithValue(ctx, savedPhoneMatchContextKey{}, &savedPhoneMatchBinding{source: source, id: id, spec: form.SavedPhoneMatch})
}

func savedPhoneMatchRequested(ctx context.Context, text string) bool {
	binding, _ := ctx.Value(savedPhoneMatchContextKey{}).(*savedPhoneMatchBinding)
	return binding != nil && query.UsesSavedPhoneMatch(text)
}

func (s *Server) savedPhoneMatchForQuery(ctx context.Context, text string) (*query.SavedPhoneMatch, error) {
	binding, _ := ctx.Value(savedPhoneMatchContextKey{}).(*savedPhoneMatchBinding)
	if binding == nil || !query.UsesSavedPhoneMatch(text) {
		return nil, nil
	}

	source, spec := binding.source, binding.spec
	target := s.reg.GetEntity(strings.TrimSpace(spec.TargetDocument))
	if target == nil || target.Kind != metadata.KindDocument ||
		len(spec.SourceFields) < 1 || len(spec.SourceFields) > 2 ||
		len(spec.TargetFields) < 1 || len(spec.TargetFields) > 3 {
		return nil, i18nerr.New("Некорректные настройки поиска по сохранённому телефону")
	}
	if !s.canCtx(ctx, string(source.Kind), source.Name, "read") ||
		!s.canCtx(ctx, string(target.Kind), target.Name, "read") {
		return nil, i18nerr.New("Недостаточно прав для поиска по сохранённому телефону")
	}

	match := &query.SavedPhoneMatch{Target: target.Name}
	for _, field := range spec.TargetFields {
		dest, found := entityFieldByName(target, strings.TrimSpace(field))
		if !found || dest.Type != metadata.FieldTypeString || !hasPhoneMatchIndex(target, dest.Name) {
			return nil, i18nerr.New("Некорректные настройки поиска по сохранённому телефону")
		}
		if !slices.Contains(match.Fields, dest.Name) {
			match.Fields = append(match.Fields, dest.Name)
		}
	}

	// Reload the durable row, independent of the values in the form event.
	// A read hook does not inherit the lookup capability.
	ctx = context.WithValue(ctx, savedPhoneMatchContextKey{}, (*savedPhoneMatchBinding)(nil))
	obj, row, err := s.loadRuntimeObjectRow(ctx, source, binding.id)
	if err != nil || asBool(row["deletion_mark"]) || !s.rowAllowsSelected(ctx, source, row) {
		return nil, i18nerr.New("Недостаточно прав для поиска по сохранённому телефону")
	}
	if readForm := pickObjectFormWithReadHook(source); readForm != nil {
		if err := s.runFormReadHookOnObject(ctx, source, readForm, obj); err != nil {
			return nil, i18nerr.New("Недостаточно прав для поиска по сохранённому телефону")
		}
	}

	for _, field := range spec.SourceFields {
		src, found := entityFieldByName(source, strings.TrimSpace(field))
		if !found || src.Type != metadata.FieldTypeString {
			return nil, i18nerr.New("Некорректные настройки поиска по сохранённому телефону")
		}
		raw, _ := maskCIKeyValue(row, src.Name)
		number, _ := raw.(string)
		if key := storage.NormalizePhoneLast10(number); key != "" && !slices.Contains(match.Keys, key) {
			match.Keys = append(match.Keys, key)
		}
	}
	binding.match.Store(match)
	return match, nil
}

func hasPhoneMatchIndex(entity *metadata.Entity, name string) bool {
	for _, idx := range entity.Indexes {
		if idx.Normalizer == metadata.IndexNormalizerPhoneLast10 && len(idx.Fields) == 1 &&
			!idx.Unique && strings.EqualFold(idx.Fields[0], name) {
			return true
		}
	}
	return false
}
