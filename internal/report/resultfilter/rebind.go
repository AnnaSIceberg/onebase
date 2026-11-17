package resultfilter

import (
	"fmt"
	"strings"

	reportpkg "github.com/ivantit66/onebase/internal/report"
)

// Rebind carries a report's effective conditions into another host-compiled
// query only through the exact metadata source/field. Equal output aliases
// alone are not authority. Repeated occurrences need a stronger host trace.
func Rebind(base, target []Field, filters []reportpkg.Filter) ([]reportpkg.Filter, error) {
	out := make([]reportpkg.Filter, 0, len(filters))
	for _, filter := range filters {
		var source *Field
		for i := range base {
			if strings.EqualFold(base[i].Name, filter.Field) {
				if source != nil {
					return nil, fmt.Errorf("неоднозначное поле исходного отбора")
				}
				source = &base[i]
			}
		}
		if source == nil || source.SourceCount != 1 {
			return nil, fmt.Errorf("происхождение отбора не доказано для дополнительного запроса")
		}
		var column *Field
		for i := range target {
			field := &target[i]
			if field.SourceCount == 1 && field.Source == source.Source && strings.EqualFold(field.SourceField, source.SourceField) && field.Type == source.Type && field.RefEntity == source.RefEntity && field.EnumName == source.EnumName {
				if column != nil {
					return nil, fmt.Errorf("неоднозначное поле дополнительного запроса")
				}
				column = field
			}
		}
		if column == nil {
			return nil, fmt.Errorf("дополнительный запрос не доказывает поле эффективного отбора")
		}
		filter.Field = column.Name
		out = append(out, filter)
	}
	return out, nil
}
