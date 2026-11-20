package ui

import (
	"fmt"
	"net/http"
	"strings"
	"time"

	"github.com/ivantit66/onebase/internal/metadata"
)

func (s *Server) objectFormHeading(r *http.Request, entity *metadata.Entity, values any, isNew bool) string {
	lang := s.resolveLang(r)
	name := entity.DisplayName(lang)
	if isNew {
		return fmt.Sprintf(s.tr(lang, "%s (создание)"), name)
	}
	if entity.Kind == metadata.KindCatalog {
		caption := objectFormFieldText(values, "Наименование", "Name")
		if caption == "" {
			caption = recordCardTitle(entity, values)
		}
		if caption == "" {
			return name
		}
		return caption + " (" + name + ")"
	}

	number := objectFormFieldText(values, metadata.StandardNumberField, "Number")
	if number != "" {
		name += " " + number
	}
	if field := entity.DocumentDateField(); field != nil {
		if date, ok := canonicalFormDate(objectFormFieldText(values, field.Name)); ok {
			name = fmt.Sprintf(s.tr(lang, "%s от %s"), name, date.In(time.Local).Format("02.01.2006"))
		}
	}
	return name
}

// The current form values already carry the field visibility and masking rules.
func objectFormFieldText(values any, names ...string) string {
	for _, name := range names {
		switch fields := values.(type) {
		case map[string]string:
			for key, val := range fields {
				if strings.EqualFold(key, name) {
					if val = strings.TrimSpace(val); val != "" {
						return val
					}
				}
			}
		case map[string]any:
			for key, val := range fields {
				if strings.EqualFold(key, name) {
					if text := strings.TrimSpace(formatUntypedValueForInput(val)); text != "" {
						return text
					}
				}
			}
		}
	}
	return ""
}
