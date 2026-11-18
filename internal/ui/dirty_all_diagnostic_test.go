package ui

import (
	"bytes"
	"context"
	"log"
	"net/http/httptest"
	"net/url"
	"reflect"
	"strings"
	"testing"

	"github.com/google/uuid"
	"github.com/ivantit66/onebase/internal/auth"
	"github.com/ivantit66/onebase/internal/dbtest"
	"github.com/ivantit66/onebase/internal/metadata"
	"github.com/ivantit66/onebase/internal/runtime"
	"github.com/ivantit66/onebase/internal/storage"
)

func TestManagedDirtyAllDiagnosticPrivacy(t *testing.T) {
	var buf bytes.Buffer
	old := log.Writer()
	log.SetOutput(&buf)
	t.Cleanup(func() { log.SetOutput(old) })
	form := &metadata.FormModule{Name: "OwnForm", Elements: []*metadata.FormElement{{Kind: metadata.FormElementField, Name: "BranchView", DataPath: "Объект.Branch", ReadOnly: true}}}
	req := httptest.NewRequest("POST", "/own", nil)
	req.PostForm = url.Values{"Branch": {"secret-client-value"}}
	t.Setenv("ONEBASE_DIRTY_ALL_DIAGNOSTIC", "")
	if newManagedDirtyAllDiagnostic(req, form, metadata.FormEventOnPageChange) != nil {
		t.Fatal("default must be off")
	}
	t.Setenv("ONEBASE_DIRTY_ALL_DIAGNOSTIC", "1")
	if newManagedDirtyAllDiagnostic(req, form, metadata.FormEventAudioPlay) != nil {
		t.Fatal("Play outside scope")
	}
	d := newManagedDirtyAllDiagnostic(req, form, metadata.FormEventOnPageChange)
	a, b := uuid.New(), uuid.New()
	d.cell("entity", "Branch", -1, metadata.Field{Name: "Branch", Type: "reference:OwnTarget", RefEntity: "OwnTarget"}, a.String(), b.String(), "")
	d.cell("entity", "Note", -1, metadata.Field{Name: "Note", Type: metadata.FieldTypeString}, "secret-live-value", "secret-stored-value", "")
	d.finish(context.Background(), true)
	out := buf.String()
	for _, secret := range []string{a.String(), b.String(), "secret-client-value", "secret-live-value", "secret-stored-value", "live=len", "stored=len"} {
		if strings.Contains(out, secret) {
			t.Fatal("sensitive output")
		}
	}
	for _, expected := range []string{`field="Branch"`, `submitted_key_present=true`, `readonly_static=true`, `ref_identity_equal=false`, `finish_complete=true`, `truncated=false`} {
		if !strings.Contains(out, expected) {
			t.Fatal("missing safe diagnostic flag")
		}
	}
	canceled, cancel := context.WithCancel(context.Background())
	cancel()
	buf.Reset()
	d.finish(canceled, true)
	if !strings.Contains(buf.String(), "finish_complete=false") {
		t.Fatal("closed context reported complete")
	}
}

func TestManagedDirtyAllDiagnosticMatrix(t *testing.T) {
	dbtest.ForEachDialect(t, func(t *testing.T, db *storage.DB) {
		ctx := auth.ContextWithUser(context.Background(), &auth.User{ID: "own-diag", Login: "own-diag", IsAdmin: true})
		target := &metadata.Entity{Name: "AllDiffTargets", Kind: metadata.KindCatalog, Fields: []metadata.Field{{Name: "Name", Type: metadata.FieldTypeString}}}
		entity := &metadata.Entity{Name: "AllDiffEntity", Kind: metadata.KindCatalog, Fields: []metadata.Field{
			{Name: "First", Type: metadata.FieldTypeString}, {Name: "Flag", Type: metadata.FieldTypeBool}, {Name: "Branch", Type: "reference:AllDiffTargets", RefEntity: target.Name},
		}, TableParts: []metadata.TablePart{{Name: "Lines", Fields: []metadata.Field{{Name: "Text", Type: metadata.FieldTypeString}}}}}
		if err := db.Migrate(ctx, []*metadata.Entity{target, entity}); err != nil {
			t.Fatal(err)
		}
		a, b, id := uuid.New(), uuid.New(), uuid.New()
		for _, key := range []uuid.UUID{a, b} {
			if err := db.Upsert(ctx, target.Name, key, map[string]any{"Name": "same label"}, target); err != nil {
				t.Fatal(err)
			}
		}
		if err := db.Upsert(ctx, entity.Name, id, map[string]any{"First": "old", "Flag": false, "Branch": a.String()}, entity); err != nil {
			t.Fatal(err)
		}
		tp := entity.TableParts[0]
		if err := db.UpsertTablePartRows(ctx, entity.Name, tp.Name, id, []map[string]any{{"Text": "old"}}, tp); err != nil {
			t.Fatal(err)
		}
		reg := runtime.NewRegistry()
		reg.Load(runtime.LoadOptions{Entities: []*metadata.Entity{target, entity}})
		s := &Server{store: db, reg: reg}
		form := &metadata.FormModule{Name: "OwnForm", Attributes: []*metadata.FormAttribute{{Name: "Note", TypeRef: "string", Save: true}, {Name: "View", TypeRef: "string", Save: false}}}
		initial := map[string]any{"First": "old", "Flag": false, "Branch": a.String(), "Note": "", "View": "old-view"}
		before := snapshotFieldValues(initial)
		obj := &runtime.Object{ID: id, Type: entity.Name, Kind: entity.Kind, Fields: map[string]any{"First": "secret-first-value", "Flag": true, "Branch": &runtime.Object{ID: b}, "Note": "secret-note-value", "View": "new-view"}, TablePartRows: map[string][]map[string]any{"Lines": {{"Text": "secret-line-value"}}}}
		fieldsCopy := cloneRecord(obj.Fields)
		tablesCopy := cloneTablePartRowMap(obj.TablePartRows)
		storedBefore, err := db.GetByID(ctx, entity.Name, id, entity)
		if err != nil {
			t.Fatal(err)
		}
		normal := s.managedCloseStateDirtyReason(ctx, entity, form, obj, before, nil, nil)
		var buf bytes.Buffer
		old := log.Writer()
		log.SetOutput(&buf)
		t.Cleanup(func() { log.SetOutput(old) })
		t.Setenv("ONEBASE_DIRTY_ALL_DIAGNOSTIC", "1")
		diag := newManagedDirtyAllDiagnostic(nil, form, metadata.FormEventOnPageChange)
		observed := s.managedCloseStateDirtyReason(ctx, entity, form, obj, before, nil, nil, diag)
		diag.finish(ctx, observed != "")
		if normal != observed || !strings.HasPrefix(observed, "entity-field First ") {
			t.Fatal("original first dirty decision changed")
		}
		out := buf.String()
		for _, name := range []string{"First", "Flag", "Branch", "Lines.Text", "Note"} {
			if !strings.Contains(out, `field="`+name+`"`) {
				t.Fatalf("missing field %s", name)
			}
		}
		if strings.Contains(out, `field="View"`) {
			t.Fatal("save:false incorrectly diagnosed as dirty")
		}
		for _, secret := range []string{"secret-first-value", "secret-note-value", "secret-line-value", a.String(), b.String(), id.String()} {
			if strings.Contains(out, secret) {
				t.Fatal("raw value leaked")
			}
		}
		if !strings.Contains(out, "finish_complete=true") || diag.count != 5 {
			t.Fatal("incomplete all-field traversal")
		}
		if !reflect.DeepEqual(obj.Fields, fieldsCopy) || !reflect.DeepEqual(obj.TablePartRows, tablesCopy) {
			t.Fatal("observer changed live object")
		}
		storedAfter, err := db.GetByID(ctx, entity.Name, id, entity)
		if err != nil || !reflect.DeepEqual(storedBefore, storedAfter) {
			t.Fatal("observer wrote durable state")
		}
		buf.Reset()
		closed, cancel := context.WithCancel(ctx)
		cancel()
		failed := newManagedDirtyAllDiagnostic(nil, form, metadata.FormEventOnPageChange)
		_ = s.managedCloseStateDirtyReason(closed, entity, form, obj, before, nil, nil, failed)
		failed.finish(closed, true)
		if !strings.Contains(buf.String(), "finish_complete=false") {
			t.Fatal("read/context failure reported complete")
		}
	})
}
