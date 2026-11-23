package ui

import (
	"crypto/sha256"
	"encoding/json"
	"fmt"
	"net/http"
	"slices"
	"strings"
	"time"

	"github.com/google/uuid"
	"github.com/ivantit66/onebase/internal/metadata"
	processorpkg "github.com/ivantit66/onebase/internal/processor"
)

const (
	workbookAnswerTokenField = "_workbook_answer_token"
	workbookAnswerLifetime   = 5 * time.Minute
	maxWorkbookAnswers       = 128
)

type workbookAnswerTicket struct {
	owner    [32]byte
	scope    [32]byte
	variants []string
	expires  time.Time
}

func workbookAnswerScope(proc *processorpkg.Processor, form *metadata.FormModule, element string, fields map[string]any) [32]byte {
	state := processorCarryValues(proc, form, fields)
	values := make(map[string]string, len(proc.Params))
	for _, param := range proc.Params {
		if processorCarryParam(param) {
			// Absent and explicitly cleared controls carry the same empty value.
			// A populated default or reference still has its own value.
			values[param.Name] = state[param.Name]
		}
	}
	// JSON serialization of strings and string maps cannot fail. The ticket
	// stores only a digest, never the submitted form values.
	encoded, _ := json.Marshal(values)
	return sha256.Sum256(append([]byte(proc.Name+"\x00"+form.Name+"\x00"+element+"\x00"), encoded...))
}

func (s *Server) issueWorkbookAnswer(w http.ResponseWriter, r *http.Request, scope [32]byte, variants []string) (string, error) {
	owner, err := ensureTabularOwner(w, r)
	if err != nil {
		return "", err
	}
	now := time.Now()
	store := &s.tabularResults
	store.mu.Lock()
	defer store.mu.Unlock()
	for id, ticket := range store.workbookAnswers {
		if !ticket.expires.After(now) {
			delete(store.workbookAnswers, id)
		}
	}
	if len(store.workbookAnswers) >= maxWorkbookAnswers {
		return "", fmt.Errorf("слишком много открытых подтверждений изменения книг")
	}
	if store.workbookAnswers == nil {
		store.workbookAnswers = make(map[string]workbookAnswerTicket)
	}
	id := uuid.NewString()
	store.workbookAnswers[id] = workbookAnswerTicket{owner: owner, scope: scope, variants: slices.Clone(variants), expires: now.Add(workbookAnswerLifetime)}
	return id, nil
}

func (s *Server) consumeWorkbookAnswer(r *http.Request, params []processorpkg.Param, scope [32]byte) bool {
	id, _ := processorPostFormText(r, processorServiceFieldName(params, workbookAnswerTokenField))
	if len(id) != 36 {
		return false
	}
	owner, err := tabularOwner(r)
	if err != nil {
		return false
	}
	answer, _ := processorPostFormText(r, processorServiceFieldName(params, "_question_answer"))
	store := &s.tabularResults
	store.mu.Lock()
	defer store.mu.Unlock()
	ticket, ok := store.workbookAnswers[id]
	if !ok || ticket.owner != owner || ticket.scope != scope || !ticket.expires.After(time.Now()) ||
		!slices.Contains(ticket.variants, strings.TrimSpace(answer)) {
		return false
	}
	// Either answer consumes the token. Replaying or racing a response cannot
	// write an already confirmed or cancelled update.
	delete(store.workbookAnswers, id)
	return true
}
