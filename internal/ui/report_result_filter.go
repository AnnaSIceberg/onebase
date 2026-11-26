package ui

import (
	"context"
	"net/http"
	"slices"
	"strings"

	"github.com/ivantit66/onebase/internal/auth"
	"github.com/ivantit66/onebase/internal/query"
	reportpkg "github.com/ivantit66/onebase/internal/report"
	"github.com/ivantit66/onebase/internal/report/resultfilter"
)

func (s *Server) reportResultFilterFields(ctx context.Context, res query.Result) []resultfilter.Field {
	fields := resultfilter.Schema(res, s.sourceMeta, resultfilter.ReportAllowed(auth.UserFromContext(ctx)))
	for i := range fields {
		if fields[i].EnumName != "" {
			fields[i].Values = resultfilter.RegistryEnums(s.reg)(fields[i].EnumName)
		}
	}
	return fields
}
func (s *Server) reportResultFilterPlan(ctx context.Context, res query.Result, settings *reportpkg.UserReportSettings) (*resultfilter.Plan, error) {
	var filters []reportpkg.Filter
	if settings != nil {
		filters = settings.Filters
	}
	return resultfilter.New(s.reportResultFilterFields(ctx, res), filters, resultfilter.RegistryEnums(s.reg))
}
func (s *Server) runReportResultRows(ctx context.Context, res *query.Result, settings *reportpkg.UserReportSettings, maxRows int) ([]map[string]any, []string, bool, error) {
	plan, err := s.reportResultFilterPlan(ctx, *res, settings)
	if err != nil {
		return nil, nil, false, err
	}
	if plan.Empty() {
		return query.RunLimit(ctx, s.store, res, maxRows)
	}
	result, err := resultfilter.Run(ctx, s.store, res, plan, resultfilter.Options{MaxRows: maxRows})
	return result.Rows, result.Columns, result.Truncated, err
}

// addReportResultFilterData only publishes proven and currently permitted
// descriptors. Error responses must not echo obsolete protected preset values.
func (s *Server) addReportResultFilterData(r *http.Request, data map[string]any) {
	rep, ok := data["Report"].(*reportpkg.Report)
	if !ok || rep == nil {
		return
	}
	settings, _ := data["UserSettings"].(*reportpkg.UserReportSettings)
	variant, _ := data["ActiveVariant"].(string)
	if settings != nil {
		variant = settings.Variant
	} else {
		settings = &reportpkg.UserReportSettings{Variant: variant}
	}
	values, _ := data["ParamValues"].(map[string]any)
	values, err := reportpkg.ResolveContextParams(r.Context(), rep, values, s.reg, s.store)
	if err != nil {
		return
	}
	text, _, err := rep.QueryForVariant(variant, values)
	if err != nil || strings.TrimSpace(text) == "" {
		return
	}
	res, err := s.compileReportVariantQuery(r.Context(), rep, variant, values)
	if err != nil {
		return
	}
	if s.deniedQuerySource(r.Context(), res.Sources) != "" {
		return
	}
	comp := effectiveComposition(rep, settings)
	fields := s.reportResultFilterFields(r.Context(), res)
	fields = slices.DeleteFunc(fields, func(field resultfilter.Field) bool {
		return comp.InternalField(field.Name)
	})
	data["ReportFilterFields"] = fields
	if columns, ok := data["ReportCols"].([]string); ok && comp != nil && len(comp.InternalFields) > 0 {
		data["ReportCols"] = slices.DeleteFunc(slices.Clone(columns), comp.InternalField)
	} else if data["ReportCols"] == nil {
		columns := make([]string, len(fields))
		for i, field := range fields {
			columns[i] = field.Name
		}
		data["ReportCols"] = columns
	}
	data["ReportHasComposition"] = comp != nil
	if data["ReportSettingsJSON"] == nil {
		data["ReportSettingsJSON"] = reportSettingsPanelJSON(rep, settings)
	}
}
func (s *Server) validateReportResultSettings(r *http.Request, rep *reportpkg.Report, settings *reportpkg.UserReportSettings) error {
	if settings == nil || len(settings.Filters) == 0 {
		return nil
	}
	values, err := reportParamValuesFromRequest(r, rep, s.constantResolver(r.Context()))
	if err != nil {
		return err
	}
	values, err = reportpkg.ResolveContextParams(r.Context(), rep, values, s.reg, s.store)
	if err != nil {
		return err
	}
	res, err := s.compileReportVariantQuery(r.Context(), rep, settings.Variant, values)
	if err != nil {
		return err
	}
	_, err = s.reportResultFilterPlan(r.Context(), res, settings)
	return err
}
