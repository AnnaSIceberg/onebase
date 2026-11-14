package ui

import (
	"context"
	"log"
	"os"
	"strings"

	"github.com/ivantit66/onebase/internal/metadata"
	"github.com/ivantit66/onebase/internal/storage"
)

// Deliberately never return the raw reason: it may contain values/digests or a
// database error. Names/types below come only from the selected server metadata.
func audioDirtyLogFields(entity *metadata.Entity, form *metadata.FormModule, reason string) (class, name, typ string) {
	parts := strings.Fields(reason)
	if len(parts) == 0 {
		return "clean", "", ""
	}
	class = strings.TrimSuffix(parts[0], ":")
	switch class {
	case "no-record", "record-read-error", "record-missing":
		return class, "", ""
	case "entity-field", "service-field", "table-part", "form-table", "form-attr":
	default:
		return "other", "", ""
	}
	if len(parts) < 2 {
		return class, "", ""
	}
	if field := entityField(entity, parts[1]); field != nil {
		return class, field.Name, string(field.Type)
	}
	if entity != nil {
		for _, tp := range entity.TableParts {
			if tp.Name == parts[1] {
				return class, tp.Name, "table-part"
			}
		}
	}
	if form != nil {
		for _, attr := range form.Attributes {
			if attr != nil && attr.Name == parts[1] {
				return class, attr.Name, attr.TypeRef
			}
		}
	}
	if entityServiceFieldName(parts[1]) {
		return class, parts[1], "service"
	}
	return class, "", ""
}

func logAudioDirtyReason(ctx context.Context, entity *metadata.Entity, form *metadata.FormModule, reason string) {
	if os.Getenv("ONEBASE_AUDIO_DIRTY_DIAGNOSTIC") != "1" || reason == "" {
		return
	}
	class, name, typ := audioDirtyLogFields(entity, form, reason)
	players := 0
	walkBrowserFormElements(form, func(v browserFormElementVisit) {
		if v.element.Kind == metadata.FormElementAudio {
			players++
		}
	})
	view := audioViewStateFields(form, entity)
	formName := ""
	if form != nil {
		formName = form.Name
	}
	log.Printf("audio_dirty_diagnostic class=%s field=%q type=%q form=%q players=%d view_fields=%d excluded=%t context_alive=%t tx_present=%t", class, name, typ, formName, players, len(view), view[strings.ToLower(name)], ctx.Err() == nil, storage.HasTx(ctx))
}
