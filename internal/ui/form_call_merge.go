package ui

import (
	"context"
	"crypto/hmac"
	"crypto/sha256"
	"encoding/hex"
	"encoding/json"
	"net/http"
	"reflect"
	"strconv"
	"strings"

	"github.com/google/uuid"
	"github.com/ivantit66/onebase/internal/auth"
	"github.com/ivantit66/onebase/internal/entityservice"
	"github.com/ivantit66/onebase/internal/metadata"
	"github.com/ivantit66/onebase/internal/runtime"
	"github.com/ivantit66/onebase/internal/storage"
)

const callFormProofField = "_ob_call_business"

func isCallServiceField(name string) bool {
	switch strings.ToLower(name) {
	case "состояние", "времязавершения", "длительность", "тарифныесекунды", "исход",
		"номерв1с", "ссылкав1с", "ошибкавыгрузкив1с", "идзаписи":
		return true
	default:
		return false
	}
}

// Only the operator call card has concurrent integrations which own these fields.
// Table parts and posting state are protected by the same baseline as the header.
func operatorCall(ctx context.Context, entity *metadata.Entity) bool {
	if entity == nil || entity.Kind != metadata.KindDocument || !strings.EqualFold(entity.Name, "А_Звонок") ||
		entity.Posting {
		return false
	}
	user := auth.UserFromContext(ctx)
	if user == nil || user.IsAdmin {
		return false
	}
	for _, role := range user.Roles {
		if strings.EqualFold(role.Name, "ОператорКЦ") || strings.EqualFold(role.Name, "ОператорМессенджера") {
			return true
		}
	}
	return false
}

func callBusinessDigest(entity *metadata.Entity, row map[string]any, tables map[string][]map[string]any) (string, bool) {
	if persistedEntityVersion(row) < 1 || len(tables) != len(entity.TableParts) {
		return "", false
	}
	values := map[string]any{"posted": asBool(row["posted"]), "deletion_mark": asBool(row["deletion_mark"])}
	index := newCIIndex(row, equalFoldKey)
	for _, field := range entity.Fields {
		if !isCallServiceField(field.Name) {
			_, value, _ := ciGet(row, index, field.Name)
			values[field.Name] = value
		}
	}
	data, err := json.Marshal([]any{values, tables})
	if err != nil {
		return "", false
	}
	digest := sha256.Sum256(data)
	return hex.EncodeToString(digest[:]), true
}

func (s *Server) storedCallDigest(ctx context.Context, entity *metadata.Entity, id uuid.UUID, row map[string]any) (string, bool) {
	tables := make(map[string][]map[string]any, len(entity.TableParts))
	for _, table := range entity.TableParts {
		rows, err := s.store.GetTablePartRows(ctx, entity.Name, table.Name, id, table)
		if err != nil {
			return "", false
		}
		tables[table.Name] = rows
	}
	return callBusinessDigest(entity, row, tables)
}

// The browser may carry the proof, but cannot select the baseline fields or
// reuse a proof from another document, revision, account or login session.
func (s *Server) callFormProof(r *http.Request, id uuid.UUID, version int64, digest string) string {
	owner, ok := concealOwner(r)
	if !ok {
		return ""
	}
	mac := hmac.New(sha256.New, s.formIdleKey())
	mac.Write([]byte("onebase.operator-call.v1"))
	mac.Write(owner[:])
	for _, value := range []string{id.String(), strconv.FormatInt(version, 10), digest} {
		writeFormCloseFingerprintString(mac, value)
	}
	return hex.EncodeToString(mac.Sum(nil))
}

type callFormMerge struct {
	server   *Server
	request  *http.Request
	entity   *metadata.Entity
	id       uuid.UUID
	proof    string
	expected int64

	savedDigest  string
	savedVersion int64
}

func (s *Server) prepareCallFormMerge(r *http.Request, entity *metadata.Entity, id uuid.UUID, version *int64) *callFormMerge {
	if !operatorCall(r.Context(), entity) {
		return nil
	}
	merge := &callFormMerge{server: s, request: r, entity: entity, id: id}
	if version != nil && *version > 0 {
		if proof := r.PostForm.Get(callFormProofField); len(proof) == sha256.Size*2 {
			merge.proof, merge.expected = proof, *version
		}
	}
	return merge
}

func mergeCallServiceFields(entity *metadata.Entity, current, fields map[string]any, keep func(string) bool) {
	storedKeys := newCIIndex(current, equalFoldKey)
	fieldKeys := newCIIndex(fields, equalFoldKey)
	for _, field := range entity.Fields {
		if !isCallServiceField(field.Name) || keep != nil && keep(field.Name) {
			continue
		}
		_, value, exists := ciGet(current, storedKeys, field.Name)
		if !exists {
			continue
		}
		key, _, exists := ciGet(fields, fieldKeys, field.Name)
		if !exists {
			key = field.Name
		}
		fields[key] = normalizeRestoredValue(field, value)
	}
}

// Reuse the transactional partial-update path. It reads under the recorder
// lock, sets CAS to that exact row and keeps the existing hooks and access checks.
func (m *callFormMerge) bind(req *entityservice.SaveRequest, keep func(string) bool) {
	if m == nil {
		return
	}
	if !req.IsNew && req.ExpectedVersion != nil {
		expected := *req.ExpectedVersion
		req.ExpectedVersion = nil
		req.MergeExistingFields = true
		req.PreparePartialUpdate = func(ctx context.Context, current, fields map[string]any) error {
			// Table parts can be updated without advancing the document revision.
			if len(m.entity.TableParts) != 0 || persistedEntityVersion(current) != expected {
				digest, ok := m.server.storedCallDigest(ctx, m.entity, m.id, current)
				if !ok || !m.accepts(expected, digest) {
					return storage.ErrVersionConflict
				}
			}
			mergeCallServiceFields(m.entity, current, fields, keep)
			if err := m.server.checkDSLRowAccess(ctx, m.entity, "write", m.id, fields); err != nil {
				return errSubmitRowAccessDenied
			}
			return nil
		}
	}
	final := req.FinalPreflight
	req.FinalPreflight = func(ctx context.Context, obj *runtime.Object) error {
		if final != nil {
			if err := final(ctx, obj); err != nil {
				return err
			}
		}
		row, err := m.server.store.GetByID(ctx, m.entity.Name, m.id, m.entity)
		if err != nil {
			return err
		}
		digest, _ := m.server.storedCallDigest(ctx, m.entity, m.id, row)
		version := persistedEntityVersion(row)
		storage.DeferUntilTxCommit(ctx, func() {
			m.savedDigest, m.savedVersion = digest, version
		})
		return nil
	}
}

func (m *callFormMerge) accepts(version int64, digest string) bool {
	if m == nil {
		return false
	}
	if m.savedDigest != "" && version == m.savedVersion {
		return digest == m.savedDigest
	}
	if version != m.expected {
		return false
	}
	proof := m.server.callFormProof(m.request, m.id, version, digest)
	return proof != "" && hmac.Equal([]byte(m.proof), []byte(proof))
}

func (m *callFormMerge) acceptsCommitted(ctx context.Context, row map[string]any) bool {
	if m == nil || m.savedDigest == "" {
		return false
	}
	digest, ok := m.server.storedCallDigest(ctx, m.entity, m.id, row)
	return ok && digest == m.savedDigest
}

func (m *callFormMerge) nextProof(version int64) string {
	if m == nil || m.savedDigest == "" || m.savedVersion != version {
		return ""
	}
	return m.server.callFormProof(m.request, m.id, version, m.savedDigest)
}

// A loaded DSL object in this same call form can also age while waiting for
// 1C. Its server-side snapshot supplies the baseline without a browser proof.
func (w *docWriter) rebaseLoadedCall(ctx context.Context) (*int64, error) {
	expected := w.expectedVersion
	if w.callBaseline == nil || expected == nil {
		return expected, nil
	}
	current, err := w.s.store.GetByID(ctx, w.entity.Name, w.obj.ID, w.entity)
	if err != nil {
		return nil, err
	}
	version := persistedEntityVersion(current)
	if version == *expected && len(w.entity.TableParts) == 0 {
		return expected, nil
	}
	digest, ok := w.s.storedCallDigest(ctx, w.entity, w.obj.ID, current)
	if !ok || w.callBaselineDigest != digest {
		return nil, storage.ErrVersionConflict
	}
	mergeCallServiceFields(w.entity, current, w.obj.Fields, func(name string) bool {
		before, _ := maskCIKeyValue(w.callBaseline, name)
		after, _ := maskCIKeyValue(current, name)
		return w.assigned[strings.ToLower(name)] && reflect.DeepEqual(before, after)
	})
	return &version, nil
}

type operatorCallFormKey struct{}
