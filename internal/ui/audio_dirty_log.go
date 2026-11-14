package ui

import (
	"context"
	"log"
	"os"
	"strings"

	"github.com/ivantit66/onebase/internal/metadata"
	"github.com/ivantit66/onebase/internal/storage"
)

// Diagnostic candidate only. Never print the raw reason, values, UUIDs,
// digests or database errors. Names/types are resolved from server metadata.
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

func audioDirtyDiagnosticEvent(event metadata.FormEventType) bool {
	return event == metadata.FormEventOnPageChange || event == metadata.FormEventOnRowActivated
}

func logAudioDirtyReason(ctx context.Context, entity *metadata.Entity, form *metadata.FormModule, event metadata.FormEventType, reason string) {
	if os.Getenv("ONEBASE_AUDIO_DIRTY_DIAGNOSTIC") != "1" || reason == "" || !audioDirtyDiagnosticEvent(event) {
		return
	}
	class, name, typ := audioDirtyLogFields(entity, form, reason)
	formName := ""
	if form != nil {
		formName = form.Name
	}
	log.Printf("audio_dirty_diagnostic event=%q class=%s field=%q type=%q form=%q context_alive=%t tx_present=%t", event, class, name, typ, formName, ctx.Err() == nil, storage.HasTx(ctx))
}
