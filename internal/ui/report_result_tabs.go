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
		if param.Context == nil && (param.Type == "date" || param.Type == "datetime" || (param.Type == "string" && param.Enhanced.DateOnly)) {
			values = append(values, params[param.Name])
		}
	}
	return displayedReportPeriod(values...)
}

func displayedReportPeriod(values ...any) string {
	var dates []string
	for _, value := range values {
		date, err := time.Parse(time.DateOnly, parseDateInputValue(fmt.Sprint(value)))
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

func parseDateInputValue(value string) string {
	if len(value) >= len(time.DateOnly) {
		for _, layout := range []string{time.DateOnly, "02.01.2006"} {
			if date, err := time.Parse(layout, value[:len(time.DateOnly)]); err == nil {
				return date.Format(time.DateOnly)
			}
		}
	}
	return ""
}
