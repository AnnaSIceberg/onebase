package ui

import (
	"context"
	"encoding/json"
	"fmt"
	"io"
	"strings"

	"github.com/google/uuid"
	"github.com/ivantit66/onebase/internal/auth"
	"github.com/ivantit66/onebase/internal/metadata"
	processorpkg "github.com/ivantit66/onebase/internal/processor"
	"github.com/ivantit66/onebase/internal/storage"
)

type kpiChoiceOwnerKey struct{}

// The initial page supplies declared blank defaults itself, but must not turn
// a nonempty whitespace value into the opt-in ALL sentinel.
func kpiChoiceInitialSource(values any, name string) (string, bool) {
	switch values := values.(type) {
	case map[string]string:
		return values[name], true
	case map[string]any:
		value, present := values[name]
		if !present {
			return "", true // server-supplied declared blank default
		}
		if text, ok := value.(string); ok {
			return text, true
		}
		if id, ok := value.(uuid.UUID); ok && id != uuid.Nil {
			return id.String(), true
		}
		return "", false
	default:
		return "", false
	}
}

// The ordinary decoder is deliberately unchanged. For this opt-in, JSON null
// must not be converted to an empty string and duplicate keys are ambiguous.
func validateKPIChoiceSources(raw string) error {
	if strings.TrimSpace(raw) == "" {
		return nil // absent sources remain absent, never an explicit empty value
	}
	d := json.NewDecoder(strings.NewReader(raw))
	start, err := d.Token()
	if err != nil || start != json.Delim('{') {
		return fmt.Errorf("invalid KPI choice sources")
	}
	seen := map[string]bool{}
	for d.More() {
		key, err := d.Token()
		name, ok := key.(string)
		if err != nil || !ok || seen[name] {
			return fmt.Errorf("invalid or duplicate KPI choice source")
		}
		seen[name] = true
		value, err := d.Token()
		if _, ok := value.(string); err != nil || !ok {
			return fmt.Errorf("KPI choice source must be a string")
		}
	}
	if end, err := d.Token(); err != nil || end != json.Delim('}') {
		return fmt.Errorf("invalid KPI choice sources")
	}
	if _, err := d.Token(); err != io.EOF {
		return fmt.Errorf("invalid trailing KPI choice sources")
	}
	return nil
}

func withKPIChoiceOwner(ctx context.Context, proc *processorpkg.Processor) context.Context {
	return context.WithValue(ctx, kpiChoiceOwnerKey{}, proc)
}

// This extension is activated by server-owned processor/form metadata, never
// by a browser mode flag. A missing actor cannot request the ALL case.
func (s *Server) validateKPIChoiceScope(ctx context.Context, owner *metadata.Entity, form *metadata.FormModule, target *metadata.Entity, el *metadata.FormElement) error {
	proc, _ := ctx.Value(kpiChoiceOwnerKey{}).(*processorpkg.Processor)
	if proc == nil || proc.Kind != "kpi" || proc.External || owner == nil || proc.Name != owner.Name || auth.UserFromContext(ctx) == nil ||
		!s.canCtx(ctx, "processor", proc.Name, "run") || pickProcessorManagedFormCtx(ctx, proc) != form {
		return fmt.Errorf("kpi_choice requires an authenticated internal KPI processor form")
	}
	return metadata.ValidateKPIChoice(owner, form, el, target)
}

// Only the native primary label field is read for the suffix. No fallback to
// unrelated text/phone/address fields; pii is rejected even for administrators.
func (s *Server) kpiChoiceLabelField(ctx context.Context, target *metadata.Entity) (metadata.Field, bool) {
	if target == nil || !s.canCtx(ctx, string(target.Kind), target.Name, "read") {
		return metadata.Field{}, false
	}
	fields := metadata.LabelFields(target)
	if len(fields) == 0 || fields[0].PII || choiceAttrMasked(s.fieldDecisions(ctx, target), fields[0].Name) {
		return metadata.Field{}, false
	}
	return fields[0], true
}

func (s *Server) kpiChoiceReadable(ctx context.Context, target *metadata.Entity, el *metadata.FormElement) bool {
	if _, ok := s.kpiChoiceLabelField(ctx, target); !ok {
		return false
	}
	field, ok := entityFieldByName(target, el.KPIChoice.LabelSuffix)
	if !ok || field.PII || choiceAttrMasked(s.fieldDecisions(ctx, target), field.Name) {
		return false
	}
	_, ok = s.kpiChoiceLabelField(ctx, s.reg.GetEntity(field.RefEntity))
	return ok
}

func (s *Server) kpiChoiceLabels(ctx context.Context, target *metadata.Entity, spec *metadata.FormKPIChoice, rows []map[string]any) error {
	if spec == nil || len(rows) == 0 {
		return nil
	}
	fk, ok := entityFieldByName(target, spec.LabelSuffix)
	if !ok {
		return fmt.Errorf("invalid kpi_choice suffix")
	}
	ref := s.reg.GetEntity(fk.RefEntity)
	labelField, ok := s.kpiChoiceLabelField(ctx, ref)
	if !ok {
		return fmt.Errorf("kpi_choice suffix is unavailable")
	}
	params, err := s.rowFilterFor(ctx, ref, "read", storage.ListParams{})
	if err != nil {
		return err
	}
	cache := map[uuid.UUID]string{}
	for _, row := range rows {
		id, err := uuid.Parse(refValueString(recordValueFold(row, fk.Name)))
		if err != nil || id == uuid.Nil {
			continue
		}
		label, found := cache[id]
		if !found {
			cache[id] = ""
			visible, err := s.choiceSelectedAllowed(ctx, ref, id, nil, true)
			if err != nil {
				return err
			}
			if !visible {
				continue
			}
			records, err := s.store.GetFieldsByIDsFiltered(ctx, ref, []uuid.UUID{id}, []metadata.Field{labelField}, params.RowFilter)
			if err != nil {
				return err
			}
			if record := records[id.String()]; record != nil {
				label, _ = record[labelField.Name].(string)
				cache[id] = label
			}
		}
		if label != "" {
			row["_label"] = fmt.Sprint(row["_label"]) + " (" + label + ")"
		}
	}
	return nil
}
