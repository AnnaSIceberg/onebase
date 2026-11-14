package ui

import (
	"bytes"
	"context"
	"log"
	"strings"
	"testing"

	"github.com/ivantit66/onebase/internal/metadata"
)

func TestAudioDirtyLogSafe(t *testing.T) {
	entity := &metadata.Entity{Fields: []metadata.Field{{Name: "Flag", Type: metadata.FieldTypeBool}}}
	form := &metadata.FormModule{Name: "OwnForm", Attributes: []*metadata.FormAttribute{{Name: "Note", TypeRef: "string", Save: true}}}
	for _, reason := range []string{"entity-field Flag live=secret-uuid stored=deadbeef", "form-attr Note before=private-dsn after=len18", "record-read-error: private-dsn secret-uuid", "entity-field secret-uuid type=private-dsn"} {
		class, name, typ := audioDirtyLogFields(entity, form, reason)
		got := class + name + typ
		for _, secret := range []string{"secret-uuid", "private-dsn", "deadbeef", "len18"} {
			if strings.Contains(got, secret) {
				t.Fatal("unsafe diagnostic field")
			}
		}
	}
	var buf bytes.Buffer
	old := log.Writer()
	log.SetOutput(&buf)
	t.Cleanup(func() { log.SetOutput(old) })
	t.Setenv("ONEBASE_AUDIO_DIRTY_DIAGNOSTIC", "")
	logAudioDirtyReason(context.Background(), entity, form, metadata.FormEventOnPageChange, "entity-field Flag secret-uuid")
	if buf.Len() != 0 {
		t.Fatal("diagnostic must default off")
	}
	t.Setenv("ONEBASE_AUDIO_DIRTY_DIAGNOSTIC", "1")
	for _, event := range []metadata.FormEventType{metadata.FormEventOnOpen, metadata.FormEventAudioPlay, "untrusted-secret-uuid"} {
		logAudioDirtyReason(context.Background(), entity, form, event, "entity-field Flag secret-uuid")
	}
	if buf.Len() != 0 {
		t.Fatal("non-whitelisted event logged")
	}
	for _, event := range []metadata.FormEventType{metadata.FormEventOnPageChange, metadata.FormEventOnRowActivated} {
		logAudioDirtyReason(context.Background(), entity, form, event, "entity-field Flag secret-uuid")
	}
	got := buf.String()
	if strings.Count(got, "audio_dirty_diagnostic") != 2 || strings.Contains(got, "secret-uuid") || !strings.Contains(got, `field="Flag" type="bool"`) {
		t.Fatal("unexpected diagnostic output")
	}
}
