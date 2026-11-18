package ui

import (
	"context"
	"log"
	"net/http"
	"os"
	"reflect"
	"strconv"
	"strings"
	"time"

	"github.com/google/uuid"
	"github.com/ivantit66/onebase/internal/metadata"
	"github.com/shopspring/decimal"
)

// Own diagnostic candidate only. It never serializes raw values/reasons/errors,
// UUIDs, digests or labels. Default off; only the native PageChange is eligible.
type managedDirtyAllDiagnostic struct {
	event               metadata.FormEventType
	form                string
	submitted, readonly map[string]bool
	count, logged       int
	complete, truncated bool
}

func newManagedDirtyAllDiagnostic(r *http.Request, form *metadata.FormModule, event metadata.FormEventType) *managedDirtyAllDiagnostic {
	if os.Getenv("ONEBASE_DIRTY_ALL_DIAGNOSTIC") != "1" || event != metadata.FormEventOnPageChange {
		return nil
	}
	name := ""
	if form != nil {
		name = form.Name
	}
	submitted := map[string]bool{}
	if r != nil {
		submitted = submittedFormKeys(r)
	}
	d := &managedDirtyAllDiagnostic{event: event, form: name, submitted: submitted, readonly: readOnlyFormFields(form), complete: true}
	log.Printf("managed_dirty_alldiff phase=begin event=%q form=%q", event, name)
	return d
}

func diagnosticGoType(v any) string {
	if v == nil {
		return "nil"
	}
	t := reflect.TypeOf(v)
	prefix := ""
	for t.Kind() == reflect.Pointer {
		prefix += "*"
		t = t.Elem()
	}
	if t.Name() != "" {
		return prefix + t.String()
	}
	// Anonymous structs/type tags can embed arbitrary text. Report their kind,
	// not a reflection-generated declaration or any dynamic value.
	return prefix + t.Kind().String()
}

func diagnosticNil(v any) bool {
	if v == nil {
		return true
	}
	r := reflect.ValueOf(v)
	switch r.Kind() {
	case reflect.Chan, reflect.Func, reflect.Interface, reflect.Map, reflect.Pointer, reflect.Slice:
		return r.IsNil()
	}
	return false
}

func diagnosticReference(v any) (format string, id uuid.UUID, known bool) {
	if diagnosticNil(v) {
		return "nil", uuid.Nil, true
	}
	raw, provider := snapshotReferenceUUID(v)
	if !provider {
		switch value := v.(type) {
		case string:
			raw = value
		case uuid.UUID:
			raw = value.String()
		case [16]byte:
			raw = uuid.UUID(value).String()
		default:
			return "unsupported_wrapper", uuid.Nil, false
		}
	}
	if raw == "" {
		return "empty", uuid.Nil, true
	}
	parsed, err := uuid.Parse(strings.TrimSpace(raw))
	if err != nil {
		return "invalid_text", uuid.Nil, false
	}
	if parsed == uuid.Nil {
		return "zero_uuid", parsed, true
	}
	canonical := parsed.String()
	switch {
	case raw == canonical:
		return "uuid_canonical", parsed, true
	case strings.EqualFold(raw, canonical):
		return "uuid_case_variant", parsed, true
	case strings.HasPrefix(strings.ToLower(strings.TrimSpace(raw)), "urn:uuid:"):
		return "uuid_urn", parsed, true
	case strings.HasPrefix(strings.TrimSpace(raw), "{"):
		return "uuid_braces", parsed, true
	case len(strings.TrimSpace(raw)) == 32:
		return "uuid_compact", parsed, true
	default:
		return "uuid_noncanonical", parsed, true
	}
}

func diagnosticEmpty(f metadata.Field, v any) bool {
	if diagnosticNil(v) {
		return true
	}
	if f.RefEntity != "" || metadata.IsReference(f.Type) {
		_, id, known := diagnosticReference(v)
		return known && id == uuid.Nil
	}
	switch x := v.(type) {
	case string:
		return strings.TrimSpace(x) == ""
	case bool:
		return !x
	case time.Time:
		return x.IsZero()
	case decimal.Decimal:
		return x.IsZero()
	}
	r := reflect.ValueOf(v)
	switch r.Kind() {
	case reflect.Int, reflect.Int8, reflect.Int16, reflect.Int32, reflect.Int64:
		return r.Int() == 0
	case reflect.Uint, reflect.Uint8, reflect.Uint16, reflect.Uint32, reflect.Uint64:
		return r.Uint() == 0
	case reflect.Float32, reflect.Float64:
		return r.Float() == 0
	case reflect.Map, reflect.Slice, reflect.Array:
		return r.Len() == 0
	}
	return false
}

func (d *managedDirtyAllDiagnostic) cell(scope, path string, row int, f metadata.Field, live, stored any, storedTypeOverride string) {
	if d == nil {
		return
	}
	d.count++
	if d.logged >= 4000 {
		d.truncated = true
		d.complete = false
		return
	}
	d.logged++
	lt, st := diagnosticGoType(live), diagnosticGoType(stored)
	if storedTypeOverride != "" {
		st = storedTypeOverride
	}
	lf, sf, eq := "not_reference", "not_reference", "not_applicable"
	if f.RefEntity != "" || metadata.IsReference(f.Type) {
		var li, si uuid.UUID
		var lk, sk bool
		lf, li, lk = diagnosticReference(live)
		sf, si, sk = diagnosticReference(stored)
		eq = "unknown"
		if lk && sk {
			eq = strconv.FormatBool(li == si)
		}
	}
	log.Printf("managed_dirty_alldiff phase=diff event=%q form=%q scope=%q field=%q meta_type=%q row=%d live_type=%q stored_type=%q live_empty=%t stored_empty=%t live_nil=%t stored_nil=%t live_ref_format=%q stored_ref_format=%q ref_identity_equal=%s submitted_key_present=%t readonly_static=%t", d.event, d.form, scope, path, f.Type, row, lt, st, diagnosticEmpty(f, live), diagnosticEmpty(f, stored), diagnosticNil(live), diagnosticNil(stored), lf, sf, eq, formKeySubmitted(d.submitted, f.Name), d.readonly[strings.ToLower(f.Name)])
}

func (d *managedDirtyAllDiagnostic) failure(scope, name, typ string) {
	if d == nil {
		return
	}
	d.complete = false
	d.count++
	if d.logged >= 4000 {
		d.truncated = true
		return
	}
	d.logged++
	log.Printf("managed_dirty_alldiff phase=diff event=%q form=%q scope=%q field=%q meta_type=%q comparison=unavailable", d.event, d.form, scope, name, typ)
}

func (d *managedDirtyAllDiagnostic) tables(scope string, tp metadata.TablePart, live, stored []map[string]any, dialect string) {
	if d == nil {
		return
	}
	if len(live) != len(stored) {
		d.cell(scope+"_row_count", tp.Name, -1, metadata.Field{Name: tp.Name, Type: "table"}, live, stored, "")
	}
	n := len(live)
	if len(stored) > n {
		n = len(stored)
	}
	for i := 0; i < n; i++ {
		for _, f := range tp.Fields {
			var a, b any
			if i < len(live) {
				a = tpCellValue(live[i], f.Name)
			}
			if i < len(stored) {
				b = tpCellValue(stored[i], f.Name)
			}
			if i >= len(live) || i >= len(stored) || tpCellNormStored(f, a, dialect) != tpCellNormStored(f, b, dialect) {
				d.cell(scope, tp.Name+"."+f.Name, i, f, a, b, "")
			}
		}
	}
}

func diagnosticSnapshot(v string, present bool) (any, string) {
	if !present {
		return nil, "snapshot:absent"
	}
	kind, value, ok := strings.Cut(v, ":")
	if !ok {
		return nil, "snapshot:opaque"
	}
	switch kind {
	case "nil":
		return nil, "snapshot:nil"
	case "string", "uuid", "bytes":
		return value, "snapshot:" + kind
	case "bool":
		return value == "true", "snapshot:bool"
	case "number":
		return value, "snapshot:number"
	case "time":
		if t, err := time.Parse(time.RFC3339Nano, value); err == nil {
			return t, "snapshot:time"
		}
	}
	return nil, "snapshot:opaque"
}

func (d *managedDirtyAllDiagnostic) finish(ctx context.Context, dirty bool) {
	if d == nil {
		return
	}
	if ctx == nil || ctx.Err() != nil {
		d.complete = false
	}
	log.Printf("managed_dirty_alldiff phase=end event=%q form=%q dirty=%t differences=%d logged=%d finish_complete=%t truncated=%t context_alive=%t", d.event, d.form, dirty, d.count, d.logged, d.complete, d.truncated, ctx != nil && ctx.Err() == nil)
}
