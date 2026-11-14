package ui

import (
	"context"
	"encoding/binary"
	"encoding/json"
	"fmt"
	"net/http"
	"net/http/httptest"
	"net/url"
	"os"
	"path/filepath"
	"sort"
	"strings"
	"testing"
	"time"

	"github.com/go-chi/chi/v5"
	"github.com/google/uuid"
	"github.com/ivantit66/onebase/internal/auth"
	"github.com/ivantit66/onebase/internal/dbtest"
	"github.com/ivantit66/onebase/internal/dsl/ast"
	"github.com/ivantit66/onebase/internal/dsl/interpreter"
	"github.com/ivantit66/onebase/internal/dsl/loader"
	"github.com/ivantit66/onebase/internal/metadata"
	"github.com/ivantit66/onebase/internal/runtime"
	"github.com/ivantit66/onebase/internal/storage"
)

// Diagnostic-only branch atop exact f49. All observation code stays in this
// test file. The real public handler and its boolean decisions are unchanged.
// Existing debugger callbacks only copy the own synthetic form's object;
// they never eval, assign, pause on a channel, or log variable values.
type audioF49Observer struct {
	entity        string
	live          *formObjectThis
	before, after *runtime.Object
	selected      *metadata.FormModule
	stack         []string
	active        bool
}

func (*audioF49Observer) HookCheckBreakpoint(string, int, func(string) (bool, error)) bool {
	return false
}
func (*audioF49Observer) HookShouldStep(string, int) bool    { return true }
func (h *audioF49Observer) HookPushFrame(name string, _ int) { h.stack = append(h.stack, name) }
func (h *audioF49Observer) HookPopFrame() {
	if h.active && h.live != nil {
		h.after = audioF49Clone(h.live.obj)
	}
	if len(h.stack) > 0 {
		h.stack = h.stack[:len(h.stack)-1]
	}
}
func (h *audioF49Observer) HookOnPause(_ string, _ int, vars map[string]any, _ func(string) (any, error), _ string) {
	if !h.active {
		return
	}
	for _, value := range vars {
		f, ok := value.(*formObjectThis)
		if !ok || f.entity == nil || f.entity.Name != h.entity {
			continue
		}
		h.live, h.selected = f, f.form
		if h.before == nil {
			h.before = audioF49Clone(f.obj)
		}
		h.after = audioF49Clone(f.obj)
		return
	}
}
func audioF49Clone(o *runtime.Object) *runtime.Object {
	if o == nil {
		return nil
	}
	c := *o
	c.Fields = cloneRecord(o.Fields)
	c.TablePartRows = cloneTablePartRowMap(o.TablePartRows)
	return &c
}

const audioF49YAML = `schema: onebase.form/v1
form:
  name: Прослушивание
  kind: object
  entity: ДиагЗвонок
  except_roles: [ОператорКЦ, ОператорМессенджера]
  include_admin: true
  density: dense
events:
  ПриОткрытии: Открытие
actions:
  attachments: {visible: false}
attributes:
  - {name: Объект, type: DocumentRef.ДиагЗвонок, save: true, main: true}
  - {name: ИсточникЗаписи, type: string, save: false}
  - {name: ПодписьЗаписи, type: string, save: false}
  - {name: ЕстьЗапись, type: bool, save: false}
  - {name: СкрытьПрослушку, type: bool, save: false}
  - {name: Note, type: string, save: false}
elements:
  - kind: ГруппаФормы
    name: Шапка
    children:
      - {kind: ПолеВвода, name: ПолеИмя, data_path: Объект.Наименование}
      - {kind: Флажок, name: ПолеФлаг, data_path: Объект.Флаг}
      - {kind: ПолеВвода, name: ПолеМомент, data_path: Объект.Момент}
      - {kind: ПолеВвода, name: ПолеЗапись, data_path: Объект.Запись}
      - {kind: ПолеВвода, name: ПолеNote, data_path: Объект.Note}
      - kind: Аудиоплеер
        name: ПлеерЗаписи
        placement: header_end
        title: {ru: Записи нет}
        data_path: Объект.ИсточникЗаписи
        title_from: Объект.ПодписьЗаписи
        hidden_when: СкрытьПрослушку
        readonly_when: НЕ ЕстьЗапись
        events:
          Воспроизведение: ПлеерЗаписиВоспроизведение
  - kind: ГруппаФормы
    name: Данные
    children:
      - kind: ТабличнаяЧасть
        name: ТаблицаСтрок
        data_path: Объект.Строки
        columns:
          - {name: Текст, data_path: Текст}
          - {name: Флаг, data_path: Флаг}
%s
`

// Closed allowlist: never export dirtyReason strings (they include digests),
// dynamic names, object values, errors, actor/record IDs or wire bodies.
func audioF49SafeReason(raw string, entity *metadata.Entity, form *metadata.FormModule) map[string]any {
	r := map[string]any{"class": "unknown"}
	if raw == "" {
		r["class"] = "clean"
		return r
	}
	for _, f := range entity.Fields {
		if strings.HasPrefix(raw, "entity-field "+f.Name+" ") {
			return map[string]any{"class": "entity_field", "field": f.Name, "type": string(f.Type)}
		}
	}
	for _, tp := range entity.TableParts {
		if strings.HasPrefix(raw, "table-part "+tp.Name+" ") {
			return map[string]any{"class": "table_part", "field": tp.Name}
		}
	}
	for _, a := range form.Attributes {
		if strings.HasPrefix(raw, "form-attr "+a.Name+" ") {
			return map[string]any{"class": "form_attr", "field": a.Name, "type": a.TypeRef}
		}
	}
	for prefix, class := range map[string]string{"no-record:": "no_record", "record-read-error:": "record_read_error", "record-missing:": "record_missing", "service-field ": "service_field"} {
		if strings.HasPrefix(raw, prefix) {
			r["class"] = class
			return r
		}
	}
	return r
}

func TestAudioDirtyDiagnosticF49HTTP(t *testing.T) {
	dbtest.ForEachDialect(t, func(t *testing.T, db *storage.DB) {
		for _, audit := range []bool{false, true} {
			mode := "no_audit"
			if audit {
				mode = "audit_save"
			}
			for _, scenario := range []string{"clean", "entity_edit", "bool_edit", "TP_edit", "Note_handler_edit", "shared_binding"} {
				t.Run(mode+"/"+scenario, func(t *testing.T) {
					ctx := context.Background()
					entity := &metadata.Entity{Name: "ДиагЗвонок", Kind: metadata.KindDocument, Fields: []metadata.Field{
						{Name: "Наименование", Type: metadata.FieldTypeString}, {Name: "Флаг", Type: metadata.FieldTypeBool},
						{Name: "Момент", Type: metadata.FieldTypeDate}, {Name: "Запись", Type: metadata.FieldType("reference:ДиагЗаписи"), RefEntity: "ДиагЗаписи"},
					}, TableParts: []metadata.TablePart{{Name: "Строки", Fields: []metadata.Field{{Name: "Текст", Type: metadata.FieldTypeString}, {Name: "Флаг", Type: metadata.FieldTypeBool}}}}}
					record := &metadata.Entity{Name: "ДиагЗаписи", Kind: metadata.KindCatalog, Fields: []metadata.Field{{Name: "Наименование", Type: metadata.FieldTypeString}}, TableParts: []metadata.TablePart{{Name: "Прослушивания", Fields: []metadata.Field{{Name: "Метка", Type: metadata.FieldTypeString}, {Name: "Момент", Type: metadata.FieldTypeDate}}}}}
					dir := t.TempDir()
					yamlPath := filepath.Join(dir, "прослушивание.form.yaml")
					extra := ""
					if scenario == "shared_binding" {
						extra = "  - {kind: ПолеВвода, name: ЧужойВвод, data_path: Объект.ИсточникЗаписи}\n"
					}
					if os.WriteFile(yamlPath, []byte(fmt.Sprintf(audioF49YAML, extra)), 0600) != nil {
						t.Fatal("fixture YAML write failed")
					}
					note := ""
					if scenario == "Note_handler_edit" {
						note = `Объект.Note = "own unsaved note";`
					}
					module := `Процедура Открытие()
Объект.ЕстьЗапись = Истина;
Объект.СкрытьПрослушку = Ложь;
Объект.ПодписьЗаписи = "Own cached call";
КонецПроцедуры
Процедура ПлеерЗаписиВоспроизведение(Элемент)
Объект.ИсточникЗаписи = "";
Рез = ДиагТелефония.Подготовить(Объект.Запись);
` + note + `
Для Каждого Вложение Из СписокВложений(Рез) Цикл
Объект.ИсточникЗаписи = СокрЛП(Строка(Вложение.ИД));
Возврат;
КонецЦикла;
КонецПроцедуры`
					if os.WriteFile(strings.TrimSuffix(yamlPath, ".yaml")+".os", []byte(module), 0600) != nil {
						t.Fatal("fixture module write failed")
					}
					form, err := loader.NewManagedFormLoader().LoadFormFile(yamlPath, entity.Name)
					if err != nil {
						t.Fatal("native form loader failed")
					}
					base := *form
					base.Name = "Базовая"
					base.ExceptRoles = nil
					base.IncludeAdmin = false
					entity.Forms = []*metadata.FormModule{&base, form}
					s := newSubmitTestServerOn(t, db, []*metadata.Entity{entity, record})
					s.entitySvc = s.newEntityService(nil)
					actor := &auth.User{ID: "own-diag", Login: "own-diag", Roles: []*auth.Role{{Name: "ОТКСотрудник", Permissions: auth.Permission{Documents: map[string][]string{entity.Name: {"read", "write"}}, Catalogs: map[string][]string{record.Name: {"read", "write"}}}}}}
					actorCtx := auth.ContextWithUser(ctx, actor)
					if pickManagedFormCtx(actorCtx, entity, "object") != form {
						t.Fatal("selected form mismatch")
					}
					matches := 0
					walkBrowserFormElements(form, func(v browserFormElementVisit) {
						if v.element.Name == "ПлеерЗаписи" {
							matches++
							if v.element.Kind != metadata.FormElementAudio || v.element.Placement != metadata.FormPlacementHeaderEnd {
								t.Error("player kind/header placement mismatch")
							}
						}
					})
					if matches != 1 {
						t.Fatal("native player target is not unique")
					}
					doSave := ""
					if audit {
						doSave = `Об = Запись.ПолучитьОбъект(); Ряд = Об.Прослушивания.Добавить(); Ряд.Метка = "own audit"; Ряд.Момент = ТекущаяДатаВремя(); Об.Записать();`
					}
					s.reg.LoadModules(map[string]*ast.Program{"ДиагТелефония": mustParse(t, "Функция Подготовить(Запись) Экспорт\n"+doSave+"\nВозврат Запись;\nКонецФункции")})
					s.interp.LookupSiblingProc = s.reg.GetSiblingProc
					s.interp.LookupModuleProc = s.reg.GetModuleNamespacedProc
					obs := &audioF49Observer{entity: entity.Name}
					s.interp.DebugSource = func() interpreter.DebugHook { return obs }
					id, recordID := uuid.New(), uuid.New()
					date := time.Date(2026, 11, 14, 7, 0, 0, 0, time.UTC)
					if db.Upsert(ctx, record.Name, recordID, map[string]any{"Наименование": "own cached recording"}, record) != nil {
						t.Fatal("own record seed failed")
					}
					if db.Upsert(ctx, entity.Name, id, map[string]any{"Наименование": "own call", "Флаг": false, "Момент": date, "Запись": recordID.String()}, entity) != nil {
						t.Fatal("own entity seed failed")
					}
					storedRows := []map[string]any{{"Текст": "own row", "Флаг": true}}
					if db.UpsertTablePartRows(ctx, entity.Name, "Строки", id, storedRows, entity.TableParts[0]) != nil {
						t.Fatal("own TP seed failed")
					}
					s.store.SetFilesDir(t.TempDir())
					if db.EnsureAttachmentTable(ctx) != nil {
						t.Fatal("attachment schema failed")
					}
					wav := make([]byte, 16044)
					copy(wav, "RIFF")
					binary.LittleEndian.PutUint32(wav[4:], uint32(len(wav)-8))
					copy(wav[8:], "WAVEfmt ")
					binary.LittleEndian.PutUint32(wav[16:], 16)
					binary.LittleEndian.PutUint16(wav[20:], 1)
					binary.LittleEndian.PutUint16(wav[22:], 1)
					binary.LittleEndian.PutUint32(wav[24:], 8000)
					binary.LittleEndian.PutUint32(wav[28:], 16000)
					binary.LittleEndian.PutUint16(wav[32:], 2)
					binary.LittleEndian.PutUint16(wav[34:], 16)
					copy(wav[36:], "data")
					binary.LittleEndian.PutUint32(wav[40:], 16000)
					att, err := db.UploadAttachment(ctx, "catalog", record.Name, recordID, "own.wav", "audio/wav", "", strings.NewReader(string(wav)), 1<<20)
					if err != nil {
						t.Fatal("own cached WAV seed failed")
					}
					router := chi.NewRouter()
					s.Mount(router)
					invoke := func(method, path string, body url.Values) *httptest.ResponseRecorder {
						r := httptest.NewRequest(method, path, strings.NewReader(body.Encode()))
						r.Header.Set("Content-Type", "application/x-www-form-urlencoded")
						r = r.WithContext(auth.ContextWithUser(r.Context(), actor))
						w := httptest.NewRecorder()
						router.ServeHTTP(w, r)
						return w
					}
					get := invoke(http.MethodGet, "/ui/document/"+entity.Name+"/"+id.String(), nil)
					if get.Code != http.StatusOK || !strings.Contains(get.Body.String(), `data-audio-name="ПлеерЗаписи"`) {
						t.Fatal("public GET did not render selected native player")
					}
					payload, _ := json.Marshal(storedRows)
					body := url.Values{"_kind": {"object"}, "_id": {id.String()}, "_version": {"1"}, "Наименование": {"own call"}, "Флаг": {"false"}, "Момент": {date.In(time.Local).Format("2006-01-02T15:04:05")}, "Запись": {recordID.String()}, "tp_json.Строки": {string(payload)}, "ИсточникЗаписи": {""}, "ПодписьЗаписи": {""}, "ЕстьЗапись": {"false"}, "СкрытьПрослушку": {"false"}, "Note": {""}, "_event": {"ПриОткрытии"}}
					opened := invoke(http.MethodPost, "/ui/document/"+entity.Name+"/form-event", body)
					var open formEventResponse
					if json.Unmarshal(opened.Body.Bytes(), &open) != nil || !open.OK || open.Dirty == nil || *open.Dirty {
						t.Fatal("public OnOpen was not clean")
					}
					body.Set("ЕстьЗапись", "true")
					body.Set("ПодписьЗаписи", "Own cached call")
					if scenario == "entity_edit" {
						body.Set("Наименование", "own unsaved edit")
					}
					if scenario == "bool_edit" {
						body.Set("Флаг", "true")
					}
					if scenario == "TP_edit" {
						body.Set("tp_json.Строки", `[{"Текст":"own unsaved row","Флаг":true}]`)
					}
					body.Set("_event", "Воспроизведение")
					body.Set("_element", "ПлеерЗаписи")
					obs.active = true
					played := invoke(http.MethodPost, "/ui/document/"+entity.Name+"/form-event", body)
					obs.active = false
					var reply formEventResponse
					if json.Unmarshal(played.Body.Bytes(), &reply) != nil || !reply.OK || reply.Error != "" || reply.Dirty == nil {
						t.Fatal("public Play failed before dirty observation")
					}
					if obs.before == nil || obs.after == nil || obs.selected != form {
						t.Fatal("test-only readonly observer did not see selected live object")
					}
					projection, excluded := audioPlaybackComparison(form, entity, "ПлеерЗаписи", "Воспроизведение")
					names := []string{}
					for n := range excluded {
						names = append(names, n)
					}
					sort.Strings(names)
					fieldsBefore := snapshotFieldValues(obs.before.Fields)
					tpBefore := cloneTablePartRowMap(obs.before.TablePartRows)
					beforeReason := s.managedCloseStateDirtyReason(actorCtx, entity, form, obs.before, fieldsBefore, tpBefore, nil)
					originalReason := s.managedCloseStateDirtyReason(actorCtx, entity, form, obs.after, fieldsBefore, tpBefore, nil)
					projectedReason := s.managedCloseStateDirtyReason(actorCtx, entity, projection, obs.after, fieldsBefore, tpBefore, nil)
					auditRows, err := db.GetTablePartRows(ctx, record.Name, "Прослушивания", recordID, record.TableParts[0])
					if err != nil {
						t.Fatal("own audit read failed")
					}
					if (len(auditRows) == 1) != audit {
						t.Fatal("audit/no-audit shape did not execute as declared")
					}
					persisted, err := db.GetByID(ctx, entity.Name, id, entity)
					if err != nil || persistedEntityVersion(persisted) != 1 {
						t.Fatal("Play unexpectedly saved own form")
					}
					rows, err := db.GetTablePartRows(ctx, entity.Name, "Строки", id, entity.TableParts[0])
					if err != nil || !tpRowsEqualStored(rows, storedRows, entity.TableParts[0], db.Dialect().Name()) {
						t.Fatal("Play unexpectedly saved own TP")
					}
					if tpCellNorm(entity.Fields[0], objectGetIndexed(obs.after, newCIIndex(obs.after.Fields, strings.ToLower), "Наименование")) == "" {
						t.Fatal("vacuous live object")
					}
					flags := map[string]any{"selected_form": form.Name, "native_player_count": matches, "player_placement": "header_end", "excluded_metadata_names": names, "baseline": audioF49SafeReason(beforeReason, entity, form), "after_original": audioF49SafeReason(originalReason, entity, form), "after_projected": audioF49SafeReason(projectedReason, entity, form), "final_response_dirty": *reply.Dirty, "audit_saved": audit, "audit_row_present": len(auditRows) == 1, "response_refresh_live": reply.RefreshLive, "form_Save": false, "genuine_negative": scenario != "clean"}
					encoded, _ := json.Marshal(flags)
					t.Log("FLAGS_ONLY " + string(encoded))
					want := scenario != "clean"
					if *reply.Dirty != want {
						t.Errorf("PUBLIC_RED response_dirty=%t expected=%t", *reply.Dirty, want)
					}
					if (projectedReason != "") != *reply.Dirty {
						t.Error("readonly post-handler observation differs from actual response; stop without inferring cause")
					}
					if refValueString(reply.Values["ИсточникЗаписи"]) != att.ID.String() {
						t.Error("Play did not use own cached media")
					}
				})
			}
		}
	})
}
