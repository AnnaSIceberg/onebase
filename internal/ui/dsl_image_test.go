package ui

import (
	"context"
	"encoding/base64"
	"net/http"
	"net/http/httptest"
	"strings"
	"testing"

	"github.com/google/uuid"
	"github.com/ivantit66/onebase/internal/auth"
	"github.com/ivantit66/onebase/internal/dsl/ast"
	"github.com/ivantit66/onebase/internal/dsl/interpreter"
	"github.com/ivantit66/onebase/internal/dsl/lexer"
	"github.com/ivantit66/onebase/internal/dsl/parser"
	"github.com/ivantit66/onebase/internal/httpservice"
	"github.com/ivantit66/onebase/internal/metadata"
	"github.com/ivantit66/onebase/internal/processor"
	"github.com/ivantit66/onebase/internal/runtime"
	"github.com/ivantit66/onebase/internal/scheduler"
	"github.com/ivantit66/onebase/internal/storage"
)

func callDSLTestBuiltin(t *testing.T, vars map[string]any, name string, args ...any) (any, error) {
	t.Helper()
	fn, ok := vars[name].(interpreter.BuiltinFunc)
	if !ok {
		t.Fatalf("builtin %s не зарегистрирован", name)
	}
	return fn(args, "dsl-image-test.os", 1)
}

// Explicit Неопределено is how generated/application DSL commonly forwards an
// optional argument. Exercise the parser and interpreter, not a direct Go call:
// it must be indistinguishable from omitting owner and keep the legacy mode.
func TestSaveImageExplicitUndefinedKeepsLegacyOwnerlessMode(t *testing.T) {
	entity := ownerCatalog("Фотографии", metadata.Field{Name: "Картинка", Type: metadata.FieldTypeImage})
	s, ctx := newSubmitTestServer(t, []*metadata.Entity{entity})
	if err := s.store.EnsureBlobTable(ctx); err != nil {
		t.Fatal(err)
	}
	if err := s.store.SaveFileStorageMode(ctx, storage.FileStorageDB); err != nil {
		t.Fatal(err)
	}
	data := base64.StdEncoding.EncodeToString([]byte("\x89PNG\r\n\x1a\nlegacy"))
	src := `Функция Тест()
		Возврат СохранитьКартинку("` + data + `", "image/png", Неопределено);
	КонецФункции`
	prog, err := parser.New(lexer.New(src, "save-image-undefined.os")).ParseProgram()
	if err != nil {
		t.Fatalf("parse: %v", err)
	}
	if len(prog.Procedures) != 1 {
		t.Fatalf("процедур = %d, ожидалась 1", len(prog.Procedures))
	}
	vars, _ := s.buildDSLVarsTx(ctx, runtime.NewMovementsCollector("processor", [16]byte{}))
	var result any
	if err := s.interp.RunWithResult(prog.Procedures[0], runtime.NewObject("Test", metadata.KindDocument), &result, vars); err != nil {
		t.Fatalf("public DSL run: %v", err)
	}
	id, err := uuid.Parse(result.(string))
	if err != nil {
		t.Fatalf("UUID результата: %v", err)
	}
	blob, rc, err := s.store.OpenBlob(ctx, id)
	if err != nil {
		t.Fatalf("OpenBlob: %v", err)
	}
	_ = rc.Close()
	if blob.OwnerKind != "" || blob.OwnerEntity != "" {
		t.Fatalf("Неопределено неожиданно создало владельца %q/%q", blob.OwnerKind, blob.OwnerEntity)
	}
	var managed int
	if err := s.store.QueryRow(ctx, "SELECT dsl_managed FROM _blobs WHERE id=?", id.String()).Scan(&managed); err != nil {
		t.Fatalf("dsl_managed: %v", err)
	}
	if managed != 1 {
		t.Fatalf("dsl_managed=%d, ожидалось 1 для legacy owner-less режима", managed)
	}
}

// The MIME argument is optional independently of the owner argument. Exercise
// the public parser/interpreter path: Неопределено must keep the documented
// image/png default instead of being stringified as "<nil>".
func TestSaveImageUndefinedMIMEDefaultsWithOwner(t *testing.T) {
	entity := ownerCatalog("Фотографии", metadata.Field{Name: "Картинка", Type: metadata.FieldTypeImage})
	s, baseCtx := newSubmitTestServer(t, []*metadata.Entity{entity})
	if err := s.store.EnsureBlobTable(baseCtx); err != nil {
		t.Fatal(err)
	}
	if err := s.store.SaveFileStorageMode(baseCtx, storage.FileStorageDB); err != nil {
		t.Fatal(err)
	}
	ownerID := seedOwnerRow(t, baseCtx, s, entity, "u", map[string]any{"Картинка": ""})
	userCtx := auth.ContextWithUser(context.Background(), rowOwnerUser("u", entity.Name, "write"))
	data := base64.StdEncoding.EncodeToString([]byte("\x89PNG\r\n\x1a\nowner-default-mime"))
	src := `Функция Тест()
		НачатьТранзакцию();
		UUID = СохранитьКартинку("` + data + `", Неопределено, Реф);
		ЗафиксироватьТранзакцию();
		Возврат UUID;
	КонецФункции`
	prog, err := parser.New(lexer.New(src, "save-image-owner-default-mime.os")).ParseProgram()
	if err != nil {
		t.Fatalf("parse: %v", err)
	}
	vars, _ := s.buildDSLVarsTx(userCtx, runtime.NewMovementsCollector("processor", [16]byte{}))
	vars["Реф"] = &interpreter.Ref{UUID: ownerID.String(), Type: entity.Name}
	var result any
	if err := s.interp.RunWithResult(prog.Procedures[0], runtime.NewObject("Test", metadata.KindDocument), &result, vars); err != nil {
		t.Fatalf("public DSL run: %v", err)
	}
	id, err := uuid.Parse(result.(string))
	if err != nil {
		t.Fatalf("UUID результата: %v", err)
	}
	blob, rc, err := s.store.OpenBlob(baseCtx, id)
	if err != nil {
		t.Fatalf("OpenBlob: %v", err)
	}
	_ = rc.Close()
	if blob.Mime != "image/png" {
		t.Fatalf("MIME = %q, ожидался image/png", blob.Mime)
	}
	if blob.OwnerKind != string(entity.Kind) || blob.OwnerEntity != entity.Name {
		t.Fatalf("владелец blob = %q/%q, ожидался %q/%q",
			blob.OwnerKind, blob.OwnerEntity, entity.Kind, entity.Name)
	}
	var managed int
	if err := s.store.QueryRow(baseCtx, "SELECT dsl_managed FROM _blobs WHERE id=?", id.String()).Scan(&managed); err != nil {
		t.Fatalf("dsl_managed: %v", err)
	}
	if managed != 0 {
		t.Fatalf("dsl_managed=%d, ожидался 0 для owner-aware blob", managed)
	}
}

// Owner-aware СохранитьКартинку must not become a shortcut around entity/RLS
// checks. It also requires a transaction so the blob cannot be committed before
// the record that references it.
func TestSaveImageWithOwnerChecksAccessAndTransaction(t *testing.T) {
	entity := ownerCatalog("Фотографии", metadata.Field{Name: "Картинка", Type: metadata.FieldTypeImage})
	s, baseCtx := newSubmitTestServer(t, []*metadata.Entity{entity})
	if err := s.store.EnsureBlobTable(baseCtx); err != nil {
		t.Fatal(err)
	}
	if err := s.store.SaveFileStorageMode(baseCtx, storage.FileStorageDB); err != nil {
		t.Fatal(err)
	}
	visibleID := seedOwnerRow(t, baseCtx, s, entity, "u", map[string]any{"Картинка": ""})
	hiddenID := seedOwnerRow(t, baseCtx, s, entity, "other", map[string]any{"Картинка": ""})
	userCtx := auth.ContextWithUser(context.Background(), rowOwnerUser("u", entity.Name, "write"))
	vars, txState := s.buildDSLVarsTx(userCtx, runtime.NewMovementsCollector("processor", [16]byte{}))
	data := base64.StdEncoding.EncodeToString([]byte("\x89PNG\r\n\x1a\nowner-aware"))

	if _, err := callDSLTestBuiltin(t, vars, "СохранитьКартинку", data, "image/png",
		&interpreter.Ref{UUID: visibleID.String(), Type: entity.Name}); err == nil ||
		!strings.Contains(err.Error(), "внутри транзакции") {
		t.Fatalf("owner-aware вызов вне транзакции: %v", err)
	}

	if _, err := callDSLTestBuiltin(t, vars, "НачатьТранзакцию"); err != nil {
		t.Fatal(err)
	}
	if _, err := callDSLTestBuiltin(t, vars, "СохранитьКартинку", data, "image/png",
		&interpreter.Ref{UUID: hiddenID.String(), Type: entity.Name}); err == nil ||
		!strings.Contains(err.Error(), "нет доступа") {
		t.Fatalf("скрытая строка не была отклонена: %v", err)
	}
	if _, err := callDSLTestBuiltin(t, vars, "ОтменитьТранзакцию"); err != nil {
		t.Fatal(err)
	}

	if _, err := callDSLTestBuiltin(t, vars, "НачатьТранзакцию"); err != nil {
		t.Fatal(err)
	}
	got, err := callDSLTestBuiltin(t, vars, "СохранитьКартинку", data, "image/png",
		&interpreter.Ref{UUID: visibleID.String(), Type: entity.Name})
	if err != nil {
		t.Fatalf("видимая строка: %v", err)
	}
	id, err := uuid.Parse(got.(string))
	if err != nil {
		t.Fatalf("UUID результата: %v", err)
	}
	blob, rc, err := s.store.OpenBlob(txState.Ctx(), id)
	if err != nil {
		t.Fatalf("OpenBlob внутри транзакции: %v", err)
	}
	_ = rc.Close()
	if blob.OwnerKind != string(entity.Kind) || blob.OwnerEntity != entity.Name {
		t.Fatalf("владелец blob = %q/%q, ожидался %q/%q",
			blob.OwnerKind, blob.OwnerEntity, entity.Kind, entity.Name)
	}
	if _, err := callDSLTestBuiltin(t, vars, "ОтменитьТранзакцию"); err != nil {
		t.Fatal(err)
	}
	if _, _, err := s.store.OpenBlob(baseCtx, id); err == nil {
		t.Fatal("rollback оставил owner-aware blob")
	}
}

func TestReadImageBytesThroughDSL(t *testing.T) {
	entity := ownerCatalog("Фотографии", metadata.Field{Name: "Картинка", Type: metadata.FieldTypeImage})
	s, ctx := newSubmitTestServer(t, []*metadata.Entity{entity})
	if err := s.store.EnsureBlobTable(ctx); err != nil {
		t.Fatal(err)
	}
	if err := s.store.SaveFileStorageMode(ctx, storage.FileStorageDB); err != nil {
		t.Fatal(err)
	}
	data := "\x89PNG\r\n\x1a\n\x00\xff"
	blob, err := s.store.PutBlob(ctx, "image/png", strings.NewReader(data), 100,
		storage.BlobOwner{Kind: string(entity.Kind), Entity: entity.Name})
	if err != nil {
		t.Fatal(err)
	}
	record := seedOwnerRow(t, ctx, s, entity, "operator", map[string]any{"Картинка": blob.ID.String()})
	source := `
Процедура Выполнить()
    Сообщить(ВыполняетсяФоновоеЗадание());
    Сообщить(MD5Данных(ПрочитатьБайтыКартинки(Реф, "Картинка")));
КонецПроцедуры

Функция ReadImage(Запрос) Экспорт
    Реф = Справочники.Фотографии.НайтиПоИдентификатору(Запрос.ПараметрыURL.Получить("id"));
    Возврат ОтветТекст(200, MD5Данных(ПрочитатьБайтыКартинки(Реф, "Картинка")));
КонецФункции
`
	prog, err := parser.New(lexer.New(source, "image-read.os")).ParseProgram()
	if err != nil {
		t.Fatal(err)
	}
	s.reg.Load(runtime.LoadOptions{
		Entities:        []*metadata.Entity{entity},
		Programs:        map[string]*ast.Program{"ReadImage": prog},
		ServicePrograms: map[string]*ast.Program{"ReadImage": prog},
	})
	s.reg.LoadProcessors([]*processor.Processor{{Name: "ReadImage"}})
	sched := scheduler.New(s.store, s.reg, s.interp)
	if err := sched.LoadJobs([]*metadata.ScheduledJob{{Name: "ReadImage", Processor: "ReadImage"}}); err != nil {
		t.Fatal(err)
	}
	sched.SetVarsBuilder(func(ctx context.Context, mc *runtime.MovementsCollector, _ *[]string) (map[string]any, *interpreter.TxState) {
		vars, tx := s.buildDSLVarsTx(ctx, mc)
		vars["Реф"] = &interpreter.Ref{UUID: record.String(), Type: entity.Name}
		return vars, tx
	})
	if result, err := sched.ExecuteJobOnce(ctx, "ReadImage", nil); err != nil || !strings.Contains(result, "ba78c8657b1d02893c2e8356f3f26330") {
		t.Fatalf("queued read = %q, %v", result, err)
	}
	userCtx := auth.ContextWithUser(ctx, rowOwnerUser("operator", entity.Name, "read"))
	if _, err := sched.ExecuteJobOnce(userCtx, "ReadImage", nil); err == nil || !strings.Contains(err.Error(), "только серверному заданию") {
		t.Fatalf("job with user context: %v", err)
	}

	svc := &httpservice.Service{Name: "ReadImage", RootURL: "image-read", Auth: "none",
		Templates: []httpservice.URLTemplate{{Template: "/{id}", Methods: map[string]string{"GET": "ReadImage"}}}}
	svc.Normalize()
	s.reg.LoadHTTPServices([]*httpservice.Service{svc})
	if err := s.store.SaveNetworkEnabled(ctx, true); err != nil {
		t.Fatal(err)
	}
	w := httptest.NewRecorder()
	s.serviceDispatch(w, httptest.NewRequest(http.MethodGet, "/hs/image-read/"+record.String(), nil))
	if w.Code != http.StatusInternalServerError || !strings.Contains(w.Body.String(), "только серверному заданию") {
		t.Fatalf("anonymous HTTP read: status=%d, body=%q", w.Code, w.Body.String())
	}
}
