package ui

import (
	"fmt"
	"strings"
	"time"

	"github.com/ivantit66/onebase/internal/report"
)

func reportPeriodCaption(rep *report.Report, params map[string]any) string {
	var dates []string
	for _, param := range rep.Params {
		if param.Context != nil || (param.Type != "date" && param.Type != "datetime") {
			continue
		}
		value := fmt.Sprint(params[param.Name])
		if len(value) < len(time.DateOnly) {
			continue
		}
		date, err := time.Parse(time.DateOnly, value[:len(time.DateOnly)])
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

func validReportPeriodCaption(value string) bool {
	if len(value) != len("01.01.2006") && len(value) != len("01.01.2006 – 02.01.2006") {
		return false
	}
	for _, part := range strings.Split(value, " – ") {
		if _, err := time.Parse("02.01.2006", part); err != nil {
			return false
		}
	}
	return true
}
