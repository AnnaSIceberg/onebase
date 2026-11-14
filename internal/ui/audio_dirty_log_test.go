package ui

import (
	"strings"
	"testing"

	"github.com/ivantit66/onebase/internal/metadata"
)

func TestAudioDirtyLogSafe(t *testing.T) {
	ent := &metadata.Entity{Fields: []metadata.Field{{Name: "Phone", Type: metadata.FieldTypeString}}}
	form := &metadata.FormModule{Attributes: []*metadata.FormAttribute{{Name: "Source", TypeRef: "string"}}}
	for _, reason := range []string{
		"entity-field Phone type=string live=secret-uuid stored=len18:deadbeef",
		"form-attr Source before=secret-uuid after=len18:deadbeef",
		"record-read-error: secret-uuid private-dsn deadbeef",
		"other secret-uuid deadbeef",
		"form-attr secret-uuid before=deadbeef",
	} {
		class, name, typ := audioDirtyLogFields(ent, form, reason)
		out := class + name + typ
		for _, secret := range []string{"secret-uuid", "deadbeef", "private-dsn", "len18"} {
			if strings.Contains(out, secret) {
				t.Fatal("raw reason leaked")
			}
		}
	}
}
