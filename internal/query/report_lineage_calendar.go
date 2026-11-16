package query

import "github.com/ivantit66/onebase/internal/metadata"

// These functions have explicit rewrites in both ordinary compiler dialects.
// A known result type does not erase the input's data/control dependencies.
func lineageCalendarFunction(name string) string {
	switch upperFast(name) {
	case "ГОД", "YEAR":
		return "YEAR"
	case "МЕСЯЦ", "MONTH":
		return "MONTH"
	case "ДЕНЬ", "DAY":
		return "DAY"
	}
	return ""
}

func lineageCalendarType(args []ReportExpression) (metadata.FieldType, bool) {
	if len(args) != 1 || args[0].Type != metadata.FieldTypeDate {
		return "", false
	}
	return metadata.FieldTypeNumber, true
}
