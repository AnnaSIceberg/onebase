package navigation

import (
	"fmt"
	"net/url"
	"slices"
	"strings"

	"github.com/ivantit66/onebase/internal/metadata"
)

// ListFilterFields projects fields accepted by the standard list equality filter.
func ListFilterFields(entity *metadata.Entity) []string {
	var result []string
	for _, field := range entity.Fields {
		if field.Type == metadata.FieldTypeDate || entity.Activity != nil && field.Name == entity.Activity.Field {
			continue
		}
		result = append(result, field.Name)
	}
	return result
}

func listPresetQuery(object Object, filters map[string]string) (string, error) {
	if len(filters) == 0 {
		return "", nil
	}
	if object.Target.Kind != "document" && object.Target.Kind != "catalog" {
		return "", fmt.Errorf("list_filters применяется к спискам документов и справочников")
	}
	values := make(url.Values, len(filters))
	for field, value := range filters {
		if !slices.Contains(object.ListFilterFields, field) {
			return "", fmt.Errorf("неверное имя поля list_filters: %q", field)
		}
		values.Set("f."+field+".eq", value)
	}
	return values.Encode(), nil
}

func listPresetURL(path, query string) string {
	if query == "" {
		return path
	}
	if strings.Contains(path, "?") {
		return path + "&" + query
	}
	return path + "?" + query
}
