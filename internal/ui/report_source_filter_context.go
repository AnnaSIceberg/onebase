package ui

import (
	"context"
	"fmt"
	"strconv"
	"strings"

	"github.com/ivantit66/onebase/internal/dsl/interpreter"
	"github.com/ivantit66/onebase/internal/query"
	reportpkg "github.com/ivantit66/onebase/internal/report"
	"github.com/ivantit66/onebase/internal/report/resultfilter"
)

type reportSourceFilterKey struct{}
type reportSourceFilterState struct {
	fields  []resultfilter.Field
	filters []reportpkg.Filter
}

func (s *Server) withReportSourceFilters(ctx context.Context, compiled query.Result, settings *reportpkg.UserReportSettings) context.Context {
	if settings == nil || len(settings.Filters) == 0 {
		return ctx
	}
	state := &reportSourceFilterState{fields: s.reportResultFilterFields(ctx, compiled), filters: append([]reportpkg.Filter(nil), settings.Filters...)}
	return context.WithValue(ctx, reportSourceFilterKey{}, state)
}

// A filtered result_proc receives the already filtered source and a guarded
// query factory. Other data managers cannot silently create unfiltered rows;
// their broader lineage contract remains a separate capability.
func (s *Server) reportFilteredVars(ctx context.Context, tx *interpreter.TxState) (map[string]any, bool) {
	state, ok := ctx.Value(reportSourceFilterKey{}).(*reportSourceFilterState)
	if !ok {
		return nil, false
	}
	execute := func(ctx context.Context, res query.Result) ([]map[string]any, error) {
		mask := s.reportOutputMaskPlan(ctx, res)
		if mask.Denied != "" {
			return nil, fmt.Errorf("дополнительный запрос использует защищённое поле: %s", mask.Denied)
		}
		fields := s.reportResultFilterFields(ctx, res)
		filters, err := resultfilter.Rebind(state.fields, fields, state.filters)
		if err != nil {
			return nil, err
		}
		plan, err := resultfilter.New(fields, filters, resultfilter.RegistryEnums(s.reg))
		if err != nil {
			return nil, err
		}
		result, err := resultfilter.Run(ctx, s.store, &res, plan, resultfilter.Options{MaxRows: s.cfg.Limits.ReportMaxRows})
		if err != nil {
			return nil, err
		}
		if result.Truncated {
			return nil, fmt.Errorf("отобранный вход дополнительного запроса превысил report_max_rows")
		}
		return result.Rows, nil
	}
	factory := interpreter.NewQueryFactoryGuardedRunnerSource(tx, s.store, s.reg, s.compileDSLQueryWithRowAccess, s.dslQueryGuard, execute)
	readonly := &reportEffectiveFilters{filters: state.filters}
	return map[string]any{"__factory_Запрос": factory, "__factory_Query": factory, "ЭффективныеОтборы": readonly, "effectiveFilters": readonly}, true
}

type reportEffectiveFilters struct{ filters []reportpkg.Filter }

func (*reportEffectiveFilters) Get(string) any { return nil }
func (*reportEffectiveFilters) Set(string, any) {
	interpreter.RaiseUserError("эффективные отборы доступны только для чтения")
}
func (v *reportEffectiveFilters) IterateThis() []interpreter.This {
	items := make([]interpreter.This, len(v.filters))
	for i, f := range v.filters {
		items[i] = &reportEffectiveFilter{filter: f}
	}
	return items
}
func (v *reportEffectiveFilters) CallMethod(name string, args []any) any {
	switch strings.ToLower(name) {
	case "количество", "count":
		return len(v.filters)
	case "получить", "get":
		if len(args) == 1 {
			index, err := strconv.Atoi(fmt.Sprint(args[0]))
			if err == nil && index >= 0 && index < len(v.filters) {
				return &reportEffectiveFilter{filter: v.filters[index]}
			}
		}
	}
	interpreter.RaiseUserError("эффективные отборы: недопустимая операция")
	return nil
}

type reportEffectiveFilter struct{ filter reportpkg.Filter }

func (v *reportEffectiveFilter) Get(name string) any {
	switch strings.ToLower(name) {
	case "поле", "field":
		return v.filter.Field
	case "оператор", "op":
		return v.filter.Op
	case "значение", "value":
		return v.filter.Value
	}
	return nil
}
func (*reportEffectiveFilter) Set(string, any) {
	interpreter.RaiseUserError("эффективные отборы доступны только для чтения")
}
