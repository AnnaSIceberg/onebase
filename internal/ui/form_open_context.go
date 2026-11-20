package ui

import (
	"fmt"
	"net/http"
	"strings"
	"sync"
	"time"

	"github.com/google/uuid"
	"github.com/ivantit66/onebase/internal/auth"
	"github.com/ivantit66/onebase/internal/dsl/interpreter"
	"github.com/ivantit66/onebase/internal/metadata"
)

const (
	formOpenContextQuery = "__ob_open"
	formOpenTicketTTL    = 2 * time.Minute
	formOpenSessionLimit = 32
	formOpenServerLimit  = 2048
)

type formOpenPrincipal struct {
	owner         [32]byte
	authorization [32]byte
	generation    uint64
}

type formOpeningContext struct {
	principal formOpenPrincipal
	entity    string
	record    string
	values    map[string]string
	expires   time.Time
	pending   bool
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

func (s *Server) bindFormOpenContext(r *http.Request, entity *metadata.Entity, id, nonce string, now time.Time) {
	if r.Method != http.MethodGet || entity == nil || entity.Kind != metadata.KindDocument || id == "" {
		return
	}
	query := r.URL.Query()[formOpenContextQuery]
	if len(query) != 1 || len(query[0]) != 36 {
		return
	}
	principal, valid := s.formOpenPrincipal(r)
	if !valid {
		return
	}
	store := &s.formOpenContexts
	store.mu.Lock()
	defer store.mu.Unlock()
	entry, found := store.entries[query[0]]
	if !found || !entry.pending || entry.principal != principal || entry.entity != entity.Name || entry.record != id ||
		!now.Before(entry.expires) {
		return
	}
	delete(store.entries, query[0])
	entry.pending = false
	entry.expires = now.Add(managedFormInstanceTTL)
	store.entries[nonce] = entry
}

func (s *Server) formOpenContextBuiltin(r *http.Request, entity *metadata.Entity, form *metadata.FormModule, id string, schema func() string) interpreter.BuiltinFunc {
	return func(args []any, _ string, _ int) (any, error) {
		if len(args) != 1 {
			return nil, fmt.Errorf("КонтекстОткрытияФормы: ожидается имя параметра")
		}
		if form == nil || !form.InstanceContext || entity.Kind != metadata.KindDocument || id == "" {
			return "", nil
		}
		nonce, _ := s.validManagedFormInstance(r, id, schema())
		if nonce == "" {
			return "", nil
		}
		store := &s.formOpenContexts
		store.mu.Lock()
		defer store.mu.Unlock()
		entry, found := store.entries[nonce]
		if !found || entry.pending || entry.entity != entity.Name || entry.record != id || !s.concealNow().Before(entry.expires) {
			return "", nil
		}
		name, _ := args[0].(string)
		return entry.values[strings.ToLower(name)], nil
	}
}
