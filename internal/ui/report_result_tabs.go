package ui

import (
	"fmt"
	"strings"
	"time"

	"github.com/ivantit66/onebase/internal/report"
)

func reportPeriodCaption(rep *report.Report, params map[string]any) string {
	var values []any
	for _, param := range rep.Params {
		if param.Context == nil && (param.Type == "date" || param.Type == "datetime") {
			values = append(values, params[param.Name])
		}
	}
	return displayedReportPeriod(values...)
}

func displayedReportPeriod(values ...any) string {
	var dates []string
	for _, value := range values {
		raw := fmt.Sprint(value)
		if len(raw) < len(time.DateOnly) {
			continue
		}
		date, err := time.Parse(time.DateOnly, raw[:len(time.DateOnly)])
		if err == nil {
			dates = append(dates, date.Format("02.01.2006"))
			if len(dates) == 2 {
				break
			}
		}
	}
	if len(dates) == 2 && dates[0] == dates[1] {
		dates = dates[:1]
	}
	return strings.Join(dates, " – ")
}
