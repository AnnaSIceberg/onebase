package ui

import (
	"context"
	"encoding/json"
	"fmt"
	"net/http"
	"strings"

	"github.com/google/uuid"
	"github.com/ivantit66/onebase/internal/metadata"
	reportpkg "github.com/ivantit66/onebase/internal/report"
	"github.com/ivantit66/onebase/internal/storage"
)

const reportChoiceKind = "report"

func (p reportParamUI) ChoiceContext(reportName string, allParams ...[]reportpkg.Param) string {
	if len(p.Choice.ChoiceFilter) == 0 && p.Choice.ChoiceLabelSuffix == "" {
		return ""
	}
	sources := map[string]string{}
	for _, f := range p.Choice.ChoiceFilter {
		trusted := false
		for _, params := range allParams {
			for _, source := range params {
				if source.Name == f.Param && source.Context != nil {
					trusted = true
				}
			}
		}
		if !trusted {
			sources[f.Param] = f.Param
		}
	}
	b, _ := json.Marshal(managedChoiceContext{FormEntity: reportName, FormKind: reportChoiceKind, Form: "params", Element: p.Name, Sources: sources})
	return string(b)
}

func reportChoiceParam(rep *reportpkg.Report, name string) *reportpkg.Param {
	for i := range rep.Params {
		if rep.Params[i].Name == name {
			return &rep.Params[i]
		}
	}
	return nil
}

// Choice metadata belongs to the report. Only values travel from the browser.
func (s *Server) reportChoicePredicates(ctx context.Context, p reportpkg.Param, target *metadata.Entity, values map[string]any) ([]storage.ChoicePredicate, error) {
	var out []storage.ChoicePredicate
	for _, f := range p.ChoiceFilter {
		field, ok := entityFieldByName(target, f.Field)
		if !ok || field.RefEntity == "" {
			return nil, fmt.Errorf("invalid report choice field: %s", f.Field)
		}
		value := refValueString(values[f.Param])
		if value == "" {
			continue
		}
		id, err := uuid.Parse(value)
		if err != nil || id == uuid.Nil {
			return nil, fmt.Errorf("invalid report choice parameter: %s", f.Param)
		}
		ref := s.reg.GetEntity(field.RefEntity)
		if ref == nil || !s.canCtx(ctx, string(ref.Kind), ref.Name, "read") {
			return nil, fmt.Errorf("unknown report choice source")
		}
		visible, err := s.reportChoiceSelectedAllowedWithParams(ctx, ref, id, storage.ListParams{})
		if err != nil || !visible {
			return nil, fmt.Errorf("report choice source is unavailable: %s", f.Param)
		}
		out = append(out, storage.ChoicePredicate{Field: f.Field, Op: metadata.FormChoiceOpEqual, Value: id})
	}
	return out, nil
}

func (s *Server) reportChoiceLabels(ctx context.Context, p reportpkg.Param, target *metadata.Entity, rows []map[string]any) {
	if p.ChoiceLabelSuffix == "" {
		return
	}
	f, ok := entityFieldByName(target, p.ChoiceLabelSuffix)
	if !ok || f.RefEntity == "" {
		return
	}
	ref := s.reg.GetEntity(f.RefEntity)
	if ref == nil || !s.canCtx(ctx, string(ref.Kind), ref.Name, "read") {
		return
	}
	cache := map[string]string{}
	for _, row := range rows {
		idStr := refValueString(recordValueFold(row, f.Name))
		if idStr == "" {
			continue
		}
		label, found := cache[idStr]
		if !found {
			cache[idStr] = ""
			id, err := uuid.Parse(idStr)
			if err != nil {
				continue
			}
			visible, err := s.reportChoiceSelectedAllowedWithParams(ctx, ref, id, storage.ListParams{})
			if err != nil || !visible {
				continue
			}
			if record, err := s.store.GetByID(ctx, ref.Name, id, ref); err == nil {
				label = s.maskedRecordLabel(ctx, ref, record)
				cache[idStr] = label
			}
		}
		if label != "" {
			row["_label"] = fmt.Sprint(row["_label"]) + " (" + label + ")"
		}
	}
}

func recordValueFold(row map[string]any, name string) any {
	for key, value := range row {
		if strings.EqualFold(key, name) {
			return value
		}
	}
	return nil
}

func (s *Server) reportReferenceOptions(ctx context.Context, p reportpkg.Param, values map[string]any, target *metadata.Entity) ([]map[string]any, error) {
	if !s.canCtx(ctx, string(target.Kind), target.Name, "read") {
		return nil, fmt.Errorf("report choice is unavailable")
	}
	if len(p.ChoiceFilter) == 0 && p.ChoiceLabelSuffix == "" {
		return s.initialReferenceOptions(ctx, target, refOptionsFilter.withMarked(p.ChoiceShowDeleted), []string{refValueString(values[p.Name])})
	}
	predicates, err := s.reportChoicePredicates(ctx, p, target, values)
	if err != nil {
		return nil, err
	}
	rows, err := s.referenceOptionsWithParams(ctx, target, refOptionsFilter.withMarked(p.ChoiceShowDeleted), storage.ListParams{Limit: refPickerDefaultLimit, ChoicePredicates: predicates})
	if err != nil {
		return nil, err
	}
	selected := refValueString(values[p.Name])
	if selected != "" {
		id, err := uuid.Parse(selected)
		if err == nil {
			allowed, err := s.reportChoiceSelectedAllowedWithParams(ctx, target, id, storage.ListParams{ChoicePredicates: predicates})
			if err == nil && allowed {
				rows = s.appendSelectedRefOptions(ctx, rows, target, []string{selected})
			}
		}
	}
	s.reportChoiceLabels(ctx, p, target, rows)
	return rows, nil
}

func (s *Server) resolveReportChoiceRequest(r *http.Request, target *metadata.Entity) (*resolvedChoiceRequest, error) {
	q := r.URL.Query()
	name, err := oneQueryValue(q, "form_entity", true)
	if err != nil {
		return nil, err
	}
	rep := s.reg.GetReport(name)
	if rep == nil || !s.can(r, "report", rep.Name, "run") {
		return nil, fmt.Errorf("report is unavailable")
	}
	form, err := oneQueryValue(q, "form", true)
	if err != nil || form != "params" {
		return nil, fmt.Errorf("invalid report choice form")
	}
	element, err := oneQueryValue(q, "element", true)
	if err != nil {
		return nil, err
	}
	p := reportChoiceParam(rep, element)
	if p == nil || p.Context != nil || !strings.EqualFold(p.Type, "reference:"+target.Name) || (len(p.ChoiceFilter) == 0 && p.ChoiceLabelSuffix == "") {
		return nil, fmt.Errorf("invalid report choice target")
	}
	if _, ok := q["row_id"]; ok {
		return nil, fmt.Errorf("unexpected row_id")
	}
	raw, err := oneQueryValue(q, "sources", false)
	if err != nil {
		return nil, err
	}
	sources, err := decodeChoiceSources(raw)
	if err != nil {
		return nil, err
	}
	values := map[string]any{}
	expected := map[string]bool{}
	for _, f := range p.ChoiceFilter {
		source := reportChoiceParam(rep, f.Param)
		if source == nil {
			return nil, fmt.Errorf("unknown report choice source")
		}
		if source.Context == nil {
			expected[f.Param] = true
		}
	}
	for key, value := range sources {
		if !expected[key] {
			return nil, fmt.Errorf("unexpected report choice source")
		}
		values[key] = value
	}
	for key := range expected {
		if _, ok := sources[key]; !ok {
			return nil, fmt.Errorf("missing report choice source")
		}
	}
	values, err = reportpkg.ResolveContextParams(r.Context(), rep, values, s.reg, s.store)
	if err != nil {
		return nil, fmt.Errorf("report context unavailable")
	}
	predicates, err := s.reportChoicePredicates(r.Context(), *p, target, values)
	if err != nil {
		return nil, err
	}
	out := &resolvedChoiceRequest{Predicates: predicates, ReportParam: p, ShowDeleted: p.ChoiceShowDeleted}
	selected, err := oneQueryValue(q, "selected_id", false)
	if err != nil {
		return nil, err
	}
	if selected != "" {
		id, err := uuid.Parse(selected)
		if err != nil || id == uuid.Nil {
			return nil, fmt.Errorf("invalid selected_id")
		}
		out.Selected = &id
	}
	return out, nil
}

func (s *Server) validateReportChoiceValues(ctx context.Context, rep *reportpkg.Report, values map[string]any) error {
	for _, p := range rep.Params {
		if len(p.ChoiceFilter) == 0 {
			continue
		}
		target := s.reg.GetEntity(strings.TrimPrefix(p.Type, "reference:"))
		if target == nil {
			return fmt.Errorf("unknown report choice target: %s", p.Name)
		}
		predicates, err := s.reportChoicePredicates(ctx, p, target, values)
		if err != nil {
			return err
		}
		selected := refValueString(values[p.Name])
		if selected == "" {
			continue
		}
		id, err := uuid.Parse(selected)
		if err != nil {
			return fmt.Errorf("invalid report choice: %s", p.Name)
		}
		allowed, err := s.reportChoiceSelectedAllowedWithParams(ctx, target, id, storage.ListParams{ChoicePredicates: predicates})
		if err != nil {
			return err
		}
		if !allowed {
			return fmt.Errorf("report parameter does not match its choice filter: %s", p.Name)
		}
	}
	return nil
}

// Reports search history, including inactive records. Selected/source/suffix
// checks use the same activity scope as the report picker. Read permission,
// row access, folder exclusion and dependent predicates still apply. Marked
// visibility controls the offered list, not an already selected historical ref.
func (s *Server) reportChoiceSelectedAllowedWithParams(ctx context.Context, target *metadata.Entity, id uuid.UUID, params storage.ListParams) (bool, error) {
	if target == nil || !s.canCtx(ctx, string(target.Kind), target.Name, "read") {
		return false, fmt.Errorf("report choice is unavailable")
	}
	base := s.refListParamsForMode(target, refOptionsFilter)
	base.Filters = params.Filters
	base.ChoicePredicates = params.ChoicePredicates
	base, err := s.rowFilterFor(ctx, target, "read", base)
	if err != nil {
		return false, err
	}
	return s.store.ListContainsID(ctx, target.Name, target, id, base)
}

// Resolve hidden leading parameters only for the picker. Do not put their values
// into the public form defaults or accept browser values for trusted context.
func (s *Server) reportInitialChoiceOptions(ctx context.Context, p reportpkg.Param, params []reportpkg.Param, values map[string]any, target *metadata.Entity, reports ...*reportpkg.Report) ([]map[string]any, error) {
	needsContext := false
	for _, f := range p.ChoiceFilter {
		for _, source := range params {
			if source.Name == f.Param && source.Context != nil {
				needsContext = true
			}
		}
	}
	if needsContext {
		if len(reports) != 1 || reports[0] == nil {
			return nil, fmt.Errorf("report context unavailable")
		}
		var err error
		values, err = reportpkg.ResolveContextParams(ctx, reports[0], values, s.reg, s.store)
		if err != nil {
			return nil, fmt.Errorf("report context unavailable")
		}
	}
	return s.reportReferenceOptions(ctx, p, values, target)
}
