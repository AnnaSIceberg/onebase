package ui

import (
	"context"
	"crypto/hmac"
	"fmt"
	"net/http"
	"slices"
	"strconv"
	"strings"

	"github.com/google/uuid"
	"github.com/ivantit66/onebase/internal/dsl/interpreter"
	"github.com/ivantit66/onebase/internal/metadata"
	"github.com/ivantit66/onebase/internal/storage"
)

const savedPhoneLinkHeader = "X-OneBase-Saved-Phone-Link"

type savedPhoneLinkPayload struct {
	Label string `json:"label,omitempty"`
	Proof string `json:"proof,omitempty"`
}

type savedPhoneLinkEvent struct {
	sourceID string
	schema   string
	nonce    string
	issuedID string
	publish  bool
}

func (s *Server) bindSavedPhoneLinkEvent(r *http.Request, ctx context.Context, entity *metadata.Entity, form *metadata.FormModule,
	id uuid.UUID, event string, element *metadata.FormElement) (context.Context, savedPhoneLinkEvent) {
	spec := form.SavedPhoneMatch
	afterOpen := event == string(metadata.FormEventAfterOpen)
	configuredClick := event == string(metadata.FormEventOnClick) && form.IsSavedPhoneLink(element)
	legacyOpen := spec.LinkCommand == "" && event == string(metadata.FormEventOnOpen)
	if !afterOpen && !configuredClick && !legacyOpen {
		return ctx, savedPhoneLinkEvent{}
	}
	scope, schema := id.String(), entityFormCloseSchema(entity, form)
	nonce, _ := s.validManagedFormInstance(r, scope, schema)
	if nonce == "" {
		return ctx, savedPhoneLinkEvent{}
	}
	link := savedPhoneLinkEvent{sourceID: scope, schema: schema, nonce: nonce, publish: afterOpen && spec.LinkCommand != ""}
	if configuredClick {
		proofs := r.Header.Values(savedPhoneLinkHeader)
		if len(proofs) != 1 || len(proofs[0]) > 124 {
			return ctx, savedPhoneLinkEvent{}
		}
		targetID, rest, found := strings.Cut(proofs[0], ".")
		rowVersion, tag, signed := strings.Cut(rest, ".")
		if _, err := uuid.Parse(targetID); err != nil || !found || !signed ||
			!hmac.Equal([]byte(tag), []byte(link.signature(s, r, targetID+"."+rowVersion))) {
			return ctx, savedPhoneLinkEvent{}
		}
		link.issuedID = targetID + "." + rowVersion
	}
	return withSavedPhoneMatch(ctx, entity, form, id), link
}

func (link savedPhoneLinkEvent) signature(s *Server, r *http.Request, target string) string {
	return s.managedFormInstanceSignature(r, link.schema, link.sourceID, "saved-phone-link:"+link.nonce+":"+target)
}

func (link savedPhoneLinkEvent) install(s *Server, r *http.Request, ctx context.Context, vars map[string]any,
	payload **savedPhoneLinkPayload, nav **navigationPayload) {
	if link.publish {
		vars["ПоказатьСсылкуНайденнойЗаявки"] = interpreter.BuiltinFunc(func(args []any, _ string, _ int) (any, error) {
			if len(args) != 2 {
				return nil, fmt.Errorf("ПоказатьСсылкуНайденнойЗаявки: ожидаются ссылка и текст")
			}
			*payload = &savedPhoneLinkPayload{}
			label, _ := args[1].(string)
			label = strings.TrimSpace(label)
			if label == "" {
				return nil, nil
			}
			if _, id, v, ok := s.savedPhoneLinkTarget(ctx, args[0]); ok {
				identity := savedPhoneLinkIdentity(id, v)
				*payload = &savedPhoneLinkPayload{Label: label, Proof: identity + "." + link.signature(s, r, identity)}
			}
			return nil, nil
		})
	}
	if link.issuedID != "" {
		vars["ОткрытьНайденнуюЗаявку"] = interpreter.BuiltinFunc(func(args []any, _ string, _ int) (any, error) {
			if len(args) != 1 {
				return nil, fmt.Errorf("ОткрытьНайденнуюЗаявку: ожидается ссылка")
			}
			target, id, version, ok := s.savedPhoneLinkTarget(ctx, args[0])
			if !ok || link.issuedID != savedPhoneLinkIdentity(id, version) {
				return nil, fmt.Errorf("Заявка изменилась или недоступна. Откройте задачу заново")
			}
			form := pickManagedFormCtx(r.Context(), target, "object")
			if form == nil || !form.InstanceContext {
				return nil, fmt.Errorf("Для формы найденной заявки требуется instance_context")
			}
			binding := ctx.Value(savedPhoneMatchContextKey{}).(*savedPhoneMatchBinding)
			url := s.issueSavedPhoneOpening(r, target, id.String(), version, binding.match.Load().Fields)
			if url == "" {
				return nil, fmt.Errorf("Не удалось открыть форму найденной заявки")
			}
			*nav = &navigationPayload{URL: url, OpenTab: true}
			return nil, nil
		})
	}
}

func savedPhoneLinkIdentity(id uuid.UUID, version int64) string {
	return id.String() + "." + strconv.FormatInt(version, 10)
}

// A protected-field query must have run in this event. Verify the returned
// reference against durable, indexed fields and the ordinary target policy.
func (s *Server) savedPhoneLinkTarget(ctx context.Context, value any) (*metadata.Entity, uuid.UUID, int64, bool) {
	binding, _ := ctx.Value(savedPhoneMatchContextKey{}).(*savedPhoneMatchBinding)
	ref, _ := value.(*interpreter.Ref)
	if binding == nil || ref == nil {
		return nil, uuid.Nil, 0, false
	}
	match := binding.match.Load()
	if match == nil || !strings.EqualFold(ref.Type, match.Target) {
		return nil, uuid.Nil, 0, false
	}
	target := s.reg.GetEntity(match.Target)
	id, err := uuid.Parse(ref.UUID)
	if target == nil || err != nil || id == uuid.Nil {
		return nil, uuid.Nil, 0, false
	}
	ctx = context.WithValue(ctx, savedPhoneMatchContextKey{}, (*savedPhoneMatchBinding)(nil))
	obj, row, err := s.loadRuntimeObjectRow(ctx, target, id)
	if err != nil || asBool(row["deletion_mark"]) || !s.rowAllowsSelected(ctx, target, row) {
		return nil, uuid.Nil, 0, false
	}
	if hook := pickObjectFormWithReadHook(target); hook != nil && s.runFormReadHookOnObject(ctx, target, hook, obj) != nil {
		return nil, uuid.Nil, 0, false
	}
	version := persistedEntityVersion(row)
	for _, field := range match.Fields {
		raw, _ := maskCIKeyValue(row, field)
		number, _ := raw.(string)
		if version > 0 && slices.Contains(match.Keys, storage.NormalizePhoneLast10(number)) {
			return target, id, version, true
		}
	}
	return nil, uuid.Nil, 0, false
}
