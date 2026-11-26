package ui

import (
	"context"
	"encoding/base64"
	"fmt"
	"net/http"
	"net/url"
	"strconv"
	"strings"
	"time"

	"github.com/google/uuid"
	"github.com/shopspring/decimal"

	"github.com/ivantit66/onebase/internal/access"
	"github.com/ivantit66/onebase/internal/dsl/interpreter"
	reportpkg "github.com/ivantit66/onebase/internal/report"
)

func (s *Server) reportWorkbookBuiltin(tx *interpreter.TxState) interpreter.BuiltinFunc {
	return func(args []any, _ string, _ int) (any, error) {
		const function = "СформироватьОтчетXLSX"
		if len(args) != 3 {
			return nil, fmt.Errorf("%s: нужны имя отчёта, вариант и структура параметров", function)
		}
		name, nameOK := args[0].(string)
		variant, variantOK := args[1].(string)
		params, paramsOK := args[2].(*interpreter.Struct)
		if !nameOK || !variantOK || !paramsOK {
			return nil, fmt.Errorf("%s: ожидаются строка, строка и структура", function)
		}
		ctx := tx.Ctx()
		if s.store == nil {
			return nil, fmt.Errorf("%s: база данных не подключена", function)
		}
		rep := s.reg.GetReport(name)
		if rep == nil || !s.canCtx(ctx, "report", rep.Name, "run") {
			return nil, fmt.Errorf("%s: отчёт недоступен: %q", function, name)
		}
		if variant != "" {
			found := false
			for _, v := range rep.Variants {
				if v.Name == variant {
					found = true
					break
				}
			}
			if !found {
				return nil, fmt.Errorf("%s: вариант %q не найден в отчёте %q", function, variant, rep.Name)
			}
		}
		values, err := reportWorkbookParams(rep.Params, params)
		if err != nil {
			return nil, fmt.Errorf("%s: %w", function, err)
		}
		// Run the same export pipeline as Excel in the report form. An explicit
		// standard preset keeps a scheduled workbook independent of personal
		// settings. The request is local to this invocation.
		values.Set("__preset", standardReportPresetID)
		values.Set("__variant", variant)
		req := (&http.Request{
			Method: http.MethodGet,
			URL:    &url.URL{RawQuery: values.Encode()},
		}).WithContext(ctx)
		rep = rep.WithVariantRenderer(variant)
		if rep.ResultProc != "" {
			data, err := s.reportTabularWorkbook(ctx, req, rep)
			if err != nil {
				return nil, fmt.Errorf("%s %q: %w", function, rep.Name, err)
			}
			return base64.StdEncoding.EncodeToString(data), nil
		}
		file, err := s.buildReportExportFile(ctx, req, rep, "excel", &reportExportStats{})
		if err != nil {
			return nil, fmt.Errorf("%s %q: %w", function, rep.Name, err)
		}
		return base64.StdEncoding.EncodeToString(file.Data), nil
	}
}

// reportTabularWorkbook builds the same tabular result as the report form,
// using the standard preset selected by the caller. No browser tab is published.
func (s *Server) reportTabularWorkbook(ctx context.Context, r *http.Request, rep *reportpkg.Report) ([]byte, error) {
	ctx = access.WithReportOutput(ctx)
	r = r.WithContext(ctx)
	rs := s.reportSettingsForRequest(r, rep)
	if rs.Err != nil {
		return nil, rs.Err
	}
	params, err := s.reportExportParamValues(ctx, r, rep)
	if err != nil {
		return nil, err
	}
	variant := r.FormValue("__variant")
	if rs.Settings != nil {
		variant = rs.Settings.Variant
		if len(rs.Settings.Filters) > 0 {
			return nil, errReportFilterProducerUnproven
		}
	}
	text, _, err := rep.QueryForVariant(variant, params)
	if err != nil {
		return nil, err
	}
	var rows []map[string]any
	if strings.TrimSpace(text) != "" {
		compiled, err := s.compileReportVariantQuery(ctx, rep, variant, params)
		if err != nil {
			return nil, err
		}
		if denied := s.deniedQuerySource(ctx, compiled.Sources); denied != "" {
			return nil, fmt.Errorf("нет доступа к объекту: %s", denied)
		}
		mask := s.reportOutputMaskPlan(ctx, compiled)
		if mask.Denied != "" {
			return nil, fmt.Errorf("отчёт содержит защищённое поле: %s", mask.Denied)
		}
		var truncated bool
		rows, _, truncated, err = s.runReportResultRows(ctx, &compiled, rs.Settings, s.cfg.Limits.ExportMaxRows)
		if err != nil {
			return nil, err
		}
		if truncated {
			return nil, fmt.Errorf("результат выгрузки превышает export_max_rows")
		}
		if err := mask.Apply(rows); err != nil {
			return nil, err
		}
	}
	doc, err := s.buildTabularReportDocument(r, rep, params, rows)
	if err != nil {
		return nil, err
	}
	return doc.XLSX()
}

func reportWorkbookParams(defs []reportpkg.Param, params *interpreter.Struct) (url.Values, error) {
	definitions := make(map[string]reportpkg.Param, len(defs))
	for _, p := range defs {
		definitions[strings.ToLower(p.Name)] = p
	}
	values := make(url.Values, len(params.Fields())+2)
	for _, key := range params.Fields() {
		p, ok := definitions[strings.ToLower(key)]
		if !ok || strings.HasPrefix(p.Name, "__") {
			return nil, fmt.Errorf("неизвестный параметр отчёта %q", key)
		}
		raw, err := reportWorkbookParam(p, params.Get(key))
		if err != nil {
			return nil, fmt.Errorf("параметр %q: %w", p.Name, err)
		}
		values.Set(p.Name, raw)
	}
	return values, nil
}

func reportWorkbookParam(p reportpkg.Param, value any) (string, error) {
	var raw string
	switch v := value.(type) {
	case nil:
	case string:
		raw = v
	case time.Time:
		if p.Type == "datetime" {
			raw = v.Format("2006-01-02T15:04:05")
		} else {
			raw = v.Format("2006-01-02")
		}
	case bool:
		raw = strconv.FormatBool(v)
	case float64:
		raw = strconv.FormatFloat(v, 'g', -1, 64)
	case decimal.Decimal:
		raw = v.String()
	case *interpreter.Ref:
		raw = v.UUID
	case uuid.UUID:
		raw = v.String()
	default:
		return "", fmt.Errorf("неподдерживаемое значение %T", value)
	}
	if _, err := reportpkg.ParseParamValue(raw, p, reportpkg.ParamParseAPI); err != nil {
		return "", err
	}
	return raw, nil
}
