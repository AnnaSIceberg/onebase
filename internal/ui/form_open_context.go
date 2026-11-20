package ui

import (
	"context"
	"fmt"
	"net/http"
	"slices"
	"strings"
	"sync"
	"time"

	"github.com/google/uuid"
	"github.com/ivantit66/onebase/internal/auth"
	"github.com/ivantit66/onebase/internal/dsl/interpreter"
	"github.com/ivantit66/onebase/internal/metadata"
)

const (
	formOpenContextQuery            = "__ob_open"
	formOpenSavedPhoneQuery         = "__ob_phone_link"
	savedPhoneOpeningInstancePrefix = "p-"
	formOpenInstanceField           = "_opening_instance"
	formOpenTicketTTL               = 2 * time.Minute
	formOpenSessionLimit            = 32
	formOpenServerLimit             = 2048
)

type formOpenPrincipal struct {
	owner         [32]byte
	authorization [32]byte
	generation    uint64
}

type formOpeningContext struct {
	principal     formOpenPrincipal
	entity        string
	record        string
	values        map[string]string
	maskedFields  []string
	recordVersion int64
	expires       time.Time
	pending       bool
}

type formOpeningContexts struct {
	mu      sync.Mutex
	entries map[string]formOpeningContext
}

func (s *Server) formOpenPrincipal(r *http.Request) (formOpenPrincipal, bool) {
	if auth.UserFromContext(r.Context()) == nil {
		return formOpenPrincipal{}, false
	}
	owner, hasSession := concealOwner(r)
	authorization, valid := formCloseAuthorizationFingerprint(r.Context())
	return formOpenPrincipal{owner, authorization, s.reg.Generation()}, hasSession && valid
}

func formOpenUUID(value any) (uuid.UUID, error) {
	switch v := value.(type) {
	case string:
		return uuid.Parse(v)
	case uuid.UUID:
		return v, nil
	case *interpreter.Ref:
		if v != nil {
			return uuid.Parse(v.UUID)
		}
	}
	return uuid.Nil, fmt.Errorf("ожидается ссылка или UUID")
}

// Commands set a response only after the normal read check. A captured URL
// can open at most one context-bearing form in the same authenticated session.
func (s *Server) openDocumentWithContext(r *http.Request, sink **navigationPayload) interpreter.BuiltinFunc {
	return func(args []any, _ string, _ int) (any, error) {
		if len(args) != 3 {
			return nil, fmt.Errorf("ОткрытьДокументСКонтекстом: ожидались имя документа, ссылка и структура")
		}
		name, _ := args[0].(string)
		target := s.reg.GetEntity(name)
		id, err := formOpenUUID(args[1])
		if ref, ok := args[1].(*interpreter.Ref); ok && (ref == nil || !strings.EqualFold(ref.Type, name)) {
			return nil, fmt.Errorf("ОткрытьДокументСКонтекстом: тип ссылки не совпадает с документом")
		}
		if target == nil || target.Kind != metadata.KindDocument || err != nil || id == uuid.Nil ||
			!s.rowAllowsOwnerID(r, string(metadata.KindDocument), target.Name, "read", id) {
			return nil, fmt.Errorf("ОткрытьДокументСКонтекстом: запись недоступна")
		}
		principal, valid := s.formOpenPrincipal(r)
		if !valid {
			return nil, fmt.Errorf("ОткрытьДокументСКонтекстом: требуется сеанс пользователя")
		}
		params, ok := args[2].(*interpreter.Struct)
		if !ok || params == nil || len(params.Fields()) == 0 || len(params.Fields()) > 8 {
			return nil, fmt.Errorf("ОткрытьДокументСКонтекстом: ожидается структура с 1–8 идентификаторами")
		}
		values := make(map[string]string, len(params.Fields()))
		for _, key := range params.Fields() {
			value, err := formOpenUUID(params.Get(key))
			if key == "" || len(key) > 128 || err != nil || value == uuid.Nil {
				return nil, fmt.Errorf("ОткрытьДокументСКонтекстом: контекст должен содержать UUID")
			}
			values[key] = value.String()
		}
		now := s.concealNow()
		context := formOpeningContext{
			principal: principal, entity: target.Name, record: id.String(), values: values,
			expires: now.Add(formOpenTicketTTL), pending: true,
		}
		ticket := s.formOpenContexts.issue(context, now)
		if ticket == "" {
			return nil, fmt.Errorf("ОткрытьДокументСКонтекстом: слишком много открытых контекстов")
		}
		*sink = &navigationPayload{
			URL:     "/ui/document/" + target.Name + "/" + id.String() + "?" + formOpenContextQuery + "=" + ticket,
			OpenTab: true,
		}
		return nil, nil
	}
}

func (store *formOpeningContexts) issue(entry formOpeningContext, now time.Time) string {
	store.mu.Lock()
	defer store.mu.Unlock()
	if store.entries == nil {
		store.entries = make(map[string]formOpeningContext)
	}
	count := 0
	for key, existing := range store.entries {
		if !now.Before(existing.expires) {
			delete(store.entries, key)
		} else if existing.pending && entry.principal.owner == existing.principal.owner {
			count++
		}
	}
	if count >= formOpenSessionLimit || len(store.entries) >= formOpenServerLimit {
		return ""
	}
	key := uuid.NewString()
	store.entries[key] = entry
	return key
}

func (s *Server) bindFormOpenContext(r *http.Request, entity *metadata.Entity, id, nonce string, now time.Time) bool {
	if r.Method != http.MethodGet || entity == nil || entity.Kind != metadata.KindDocument || id == "" {
		return false
	}
	query := r.URL.Query()[formOpenContextQuery]
	protected := r.URL.Query().Has(formOpenSavedPhoneQuery)
	if protected {
		query = r.URL.Query()[formOpenSavedPhoneQuery]
	}
	if len(query) != 1 || len(query[0]) != 36 {
		return false
	}
	principal, valid := s.formOpenPrincipal(r)
	if !valid {
		return false
	}
	store := &s.formOpenContexts
	store.mu.Lock()
	defer store.mu.Unlock()
	entry, found := store.entries[query[0]]
	if !found || !entry.pending || entry.principal != principal || entry.entity != entity.Name || entry.record != id ||
		protected != (len(entry.maskedFields) > 0) || !now.Before(entry.expires) {
		return false
	}
	delete(store.entries, query[0])
	entry.pending = false
	entry.expires = now.Add(managedFormInstanceTTL)
	store.entries[nonce] = entry
	return true
}

// Native HTML submits cannot set a request header. The reserved POST field
// uses the same signed instance proof, and the query string is ignored.
func requestWithFormOpeningInstance(r *http.Request, entity *metadata.Entity, form *metadata.FormModule) *http.Request {
	if len(r.Header.Values(managedFormInstanceHeader)) != 0 || r.Method != http.MethodPost || len(r.PostForm) == 0 {
		return r
	}
	tokens := r.PostForm[entityFormServiceFieldNames(entity, form)[formOpenInstanceField]]
	if len(tokens) != 1 || tokens[0] == "" || len(tokens[0]) > 128 {
		return r
	}
	instanceRequest := r.Clone(r.Context())
	instanceRequest.Header.Set(managedFormInstanceHeader, tokens[0])
	return instanceRequest
}

// The returned entry is immutable after the ticket is bound to the form.
func (s *Server) formOpeningEntry(r *http.Request, entity *metadata.Entity, form *metadata.FormModule, id, schema string) (formOpeningContext, bool) {
	if form == nil || !form.InstanceContext || entity.Kind != metadata.KindDocument || id == "" {
		return formOpeningContext{}, false
	}
	nonce, _ := s.validManagedFormInstance(r, id, schema)
	if nonce == "" {
		return formOpeningContext{}, false
	}
	store := &s.formOpenContexts
	store.mu.Lock()
	defer store.mu.Unlock()
	entry, found := store.entries[nonce]
	if !found || entry.pending || entry.entity != entity.Name || entry.record != id || !s.concealNow().Before(entry.expires) {
		return formOpeningContext{}, false
	}
	return entry, true
}

func (s *Server) formOpenContextBuiltin(r *http.Request, entity *metadata.Entity, form *metadata.FormModule, id string, schema func() string) interpreter.BuiltinFunc {
	return func(args []any, _ string, _ int) (any, error) {
		if len(args) != 1 {
			return nil, fmt.Errorf("КонтекстОткрытияФормы: ожидается имя параметра")
		}
		name, _ := args[0].(string)
		if form == nil || !form.InstanceContext || r.Header.Get(managedFormInstanceHeader) == "" {
			return "", nil
		}
		entry, _ := s.formOpeningEntry(r, entity, form, id, schema())
		return entry.values[strings.ToLower(name)], nil
	}
}

type formOpenWriteKey struct{}

func (s *Server) formOpenWriteContext(r *http.Request, entity *metadata.Entity, form *metadata.FormModule, id uuid.UUID) context.Context {
	if form == nil || !form.InstanceContext || entity.Kind != metadata.KindDocument || id == uuid.Nil {
		return r.Context()
	}
	instanceRequest := requestWithFormOpeningInstance(r, entity, form)
	if instanceRequest.Header.Get(managedFormInstanceHeader) == "" {
		return r.Context()
	}
	reader := s.formOpenContextBuiltin(instanceRequest, entity, form, id.String(), func() string {
		return entityFormCloseSchema(entity, form)
	})
	return context.WithValue(r.Context(), formOpenWriteKey{}, reader)
}

// Bind the GET ticket before form initialization. A rejected native save can
// render the same instance again when the proof still validates.
func (s *Server) prepareFormOpeningContext(r *http.Request, entity *metadata.Entity, form *metadata.FormModule, data map[string]any) interpreter.BuiltinFunc {
	id, _ := data["ID"].(string)
	if id == "" {
		return nil
	}
	switch r.Method {
	case http.MethodGet:
		if !r.URL.Query().Has(formOpenContextQuery) && !r.URL.Query().Has(formOpenSavedPhoneQuery) {
			return nil
		}
		schema := entityFormCloseSchema(entity, form)
		data["FormCloseSchema"] = schema
		s.issueManagedFormInstance(r, data)
		if token, _ := data["FormInstance"].(string); token != "" {
			instanceRequest := r.Clone(r.Context())
			instanceRequest.Header.Set(managedFormInstanceHeader, token)
			return s.formOpenContextBuiltin(instanceRequest, entity, form, id, func() string { return schema })
		}
	case http.MethodPost:
		instanceRequest := requestWithFormOpeningInstance(r, entity, form)
		if instanceRequest.Header.Get(managedFormInstanceHeader) == "" {
			return nil
		}
		schema := entityFormCloseSchema(entity, form)
		if _, ok := s.formOpeningEntry(instanceRequest, entity, form, id, schema); ok {
			data["FormInstance"] = instanceRequest.Header.Get(managedFormInstanceHeader)
			data["FormOpeningInstanceField"] = entityFormServiceFieldNames(entity, form)[formOpenInstanceField]
			return s.formOpenContextBuiltin(instanceRequest, entity, form, id, func() string { return schema })
		}
	}
	return nil
}

type savedPhoneOpeningKey struct{}

func (s *Server) issueSavedPhoneOpening(r *http.Request, entity *metadata.Entity, id string, version int64, fields []string) string {
	principal, valid := s.formOpenPrincipal(r)
	if !valid {
		return ""
	}
	now := s.concealNow()
	entry := formOpeningContext{
		principal: principal, entity: entity.Name, record: id, recordVersion: version,
		maskedFields: slices.Clone(fields), pending: true, expires: now.Add(formOpenTicketTTL),
	}
	ticket := s.formOpenContexts.issue(entry, now)
	if ticket == "" {
		return ""
	}
	return "/ui/document/" + entity.Name + "/" + id + "?" + formOpenSavedPhoneQuery + "=" + ticket
}

func withSavedPhoneOpeningContext(ctx context.Context, entry formOpeningContext) context.Context {
	if len(entry.maskedFields) == 0 {
		return ctx
	}
	return context.WithValue(ctx, savedPhoneOpeningKey{}, entry)
}

func withSavedPhoneOpening(r *http.Request, entry formOpeningContext) *http.Request {
	return r.WithContext(withSavedPhoneOpeningContext(r.Context(), entry))
}

func savedPhoneOpeningMasks(ctx context.Context, entity *metadata.Entity, field string) bool {
	entry, _ := ctx.Value(savedPhoneOpeningKey{}).(formOpeningContext)
	if entity == nil || entity.Kind != metadata.KindDocument || entry.entity != entity.Name {
		return false
	}
	return slices.ContainsFunc(entry.maskedFields, func(name string) bool {
		return strings.EqualFold(name, field)
	})
}

// A protected instance must not fall back to normal disclosure after expiry.
func (s *Server) applySavedPhoneOpening(r *http.Request, entity *metadata.Entity, form *metadata.FormModule, id string) (*http.Request, bool) {
	instance := requestWithFormOpeningInstance(r, entity, form)
	token := instance.Header.Get(managedFormInstanceHeader)
	if !strings.HasPrefix(token, savedPhoneOpeningInstancePrefix) {
		return r, true
	}
	entry, ok := s.formOpeningEntry(instance, entity, form, id, entityFormCloseSchema(entity, form))
	if !ok || len(entry.maskedFields) == 0 {
		return r, false
	}
	return withSavedPhoneOpening(r, entry), true
}
