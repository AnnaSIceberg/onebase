package ui

import (
	"context"
	"crypto/sha256"
	"encoding/json"
	"net/http"
	"net/http/httptest"
	"net/url"
	"strings"
	"sync"
	"testing"
	"time"

	"github.com/go-chi/chi/v5"
	"github.com/google/uuid"

	"github.com/ivantit66/onebase/internal/metadata"
	"github.com/ivantit66/onebase/internal/processor"
)

func closeIntentBody(intentID, reason, name string) url.Values {
	return url.Values{
		"_close_intent_id": {intentID},
		"_close_reason":    {reason},
		"_close_mode":      {"discard"},
		"_kind":            {"object"},
		"Наименование":     {name},
	}
}

func executeFormCloseIntent(t *testing.T, s *Server, ent *metadata.Entity, body url.Values) *httptest.ResponseRecorder {
	t.Helper()
	req := httptest.NewRequest(http.MethodPost, "/ui/catalog/"+ent.Name+"/form-close-intent", strings.NewReader(body.Encode()))
	req.Header.Set("Content-Type", "application/x-www-form-urlencoded; charset=utf-8")
	rec := httptest.NewRecorder()
	router := chi.NewRouter()
	s.Mount(router)
	router.ServeHTTP(rec, req)
	return rec
}

func decodeCloseIntentResponse(t *testing.T, rec *httptest.ResponseRecorder) formEventResponse {
	t.Helper()
	var response formEventResponse
	if err := json.Unmarshal(rec.Body.Bytes(), &response); err != nil {
		t.Fatalf("decode close response: %v; body=%s", err, rec.Body.String())
	}
	return response
}

func executeProcessorCloseIntent(t *testing.T, s *Server, proc *processor.Processor, body url.Values) *httptest.ResponseRecorder {
	t.Helper()
	req := httptest.NewRequest(http.MethodPost, "/ui/processor/"+proc.Name+"/form-close-intent", strings.NewReader(body.Encode()))
	req.Header.Set("Content-Type", "application/x-www-form-urlencoded; charset=utf-8")
	rec := httptest.NewRecorder()
	router := chi.NewRouter()
	s.Mount(router)
	router.ServeHTTP(rec, req)
	return rec
}

func TestManagedFormCloseIntentOutputCancelAndReason(t *testing.T) {
	srv, ent := setupManagedEventsServer(t, `
Процедура ПроверитьЗакрытие(Отказ)
	Сообщить(ПричинаЗакрытия + ":" + CloseReason);
	Если Объект.Наименование = "" Тогда
		Отказ = Истина;
		Возврат;
	КонецЕсли;
КонецПроцедуры
`, map[metadata.FormEventType]string{metadata.FormEventBeforeClose: "ПроверитьЗакрытие"}, nil)

	deniedID := uuid.NewString()
	denied := decodeCloseIntentResponse(t, executeFormCloseIntent(t, srv, ent, closeIntentBody(deniedID, "escape", "")))
	if !denied.OK || denied.Close == nil || denied.Close.IntentID != deniedID || denied.Close.Allowed {
		t.Fatalf("cancel output did not keep form open: %+v", denied)
	}
	if strings.Join(denied.Messages, "|") != "escape:escape" {
		t.Fatalf("close reason was not injected: %v", denied.Messages)
	}

	allowedID := uuid.NewString()
	allowed := decodeCloseIntentResponse(t, executeFormCloseIntent(t, srv, ent, closeIntentBody(allowedID, "cross", "готово")))
	if !allowed.OK || allowed.Close == nil || allowed.Close.IntentID != allowedID || !allowed.Close.Allowed {
		t.Fatalf("allowed close was rejected: %+v", allowed)
	}
}

func TestManagedFormCloseIntentExceptionAndMissingHandlerFailClosedCorrectly(t *testing.T) {
	t.Run("exception", func(t *testing.T) {
		srv, ent := setupManagedEventsServer(t, `
Процедура ПроверитьЗакрытие()
	ВызватьИсключение("заполните обязательные поля");
КонецПроцедуры
`, map[metadata.FormEventType]string{metadata.FormEventBeforeClose: "ПроверитьЗакрытие"}, nil)
		resp := decodeCloseIntentResponse(t, executeFormCloseIntent(t, srv, ent, closeIntentBody(uuid.NewString(), "cross", "x")))
		if resp.OK || resp.Close == nil || resp.Close.Allowed || !strings.Contains(resp.Error, "заполните обязательные поля") {
			t.Fatalf("exception did not fail closed: %+v", resp)
		}
	})

	t.Run("no handler permits close", func(t *testing.T) {
		srv, ent := setupManagedEventsServer(t, "", nil, nil)
		resp := decodeCloseIntentResponse(t, executeFormCloseIntent(t, srv, ent, closeIntentBody(uuid.NewString(), "close", "x")))
		if !resp.OK || resp.Close == nil || !resp.Close.Allowed {
			t.Fatalf("form without BeforeClose must close: %+v", resp)
		}
	})

	t.Run("declared handler without module fails closed", func(t *testing.T) {
		srv, ent := setupManagedEventsServer(t, "", map[metadata.FormEventType]string{
			metadata.FormEventBeforeClose: "НетТакойПроцедуры",
		}, nil)
		resp := decodeCloseIntentResponse(t, executeFormCloseIntent(t, srv, ent, closeIntentBody(uuid.NewString(), "cross", "x")))
		if resp.OK || resp.Close == nil || resp.Close.Allowed || !strings.Contains(resp.Error, "НетТакойПроцедуры") {
			t.Fatalf("declared handler without module did not fail closed: %+v", resp)
		}
	})
}

func TestManagedFormCloseIntentCannotBeForgedThroughFormEvent(t *testing.T) {
	srv, ent := setupManagedEventsServer(t, `
Процедура ПроверитьЗакрытие()
	Сообщить("не должно выполняться");
КонецПроцедуры
`, map[metadata.FormEventType]string{metadata.FormEventBeforeClose: "ПроверитьЗакрытие"}, nil)
	body := url.Values{"_event": {string(metadata.FormEventBeforeClose)}, "_kind": {"object"}}
	resp := decodeFormEventResponse(t, executeFormEvent(t, srv, ent, body).Body.Bytes())
	if resp.OK || resp.Error == "" || len(resp.Messages) != 0 {
		t.Fatalf("lifecycle event was remotely forged through /form-event: %+v", resp)
	}
}

func TestManagedFormCloseIntentReplayAndConflict(t *testing.T) {
	srv, ent := setupManagedEventsServer(t, `
Процедура ПроверитьЗакрытие()
	Объект.Записать();
КонецПроцедуры
`, map[metadata.FormEventType]string{metadata.FormEventBeforeClose: "ПроверитьЗакрытие"}, nil)
	id := uuid.New()
	if err := srv.store.Upsert(context.Background(), ent.Name, id, map[string]any{"Наименование": "A"}, ent); err != nil {
		t.Fatal(err)
	}
	intentID := uuid.NewString()
	body := closeIntentBody(intentID, "programmatic", "A")
	body.Set("_id", id.String())
	body.Set("_version", "1")
	first := executeFormCloseIntent(t, srv, ent, body)
	second := executeFormCloseIntent(t, srv, ent, body)
	if first.Code != http.StatusOK || second.Code != http.StatusOK || first.Body.String() != second.Body.String() {
		t.Fatalf("exact replay changed terminal response: first=%d/%s second=%d/%s", first.Code, first.Body, second.Code, second.Body)
	}
	row, err := srv.store.GetByID(context.Background(), ent.Name, id, ent)
	if err != nil {
		t.Fatal(err)
	}
	if got := row["_version"]; got != int64(2) {
		t.Fatalf("replayed handler executed more than once: version=%v row=%v", got, row)
	}

	changed := closeIntentBody(intentID, "programmatic", "B")
	changed.Set("_id", id.String())
	changed.Set("_version", "2")
	conflict := executeFormCloseIntent(t, srv, ent, changed)
	resp := decodeCloseIntentResponse(t, conflict)
	if conflict.Code != http.StatusConflict || resp.OK || resp.Close == nil || resp.Close.Allowed {
		t.Fatalf("same intent with another payload was accepted: status=%d response=%+v", conflict.Code, resp)
	}
}

func TestManagedFormCloseIntentRejectsInvalidProtocolAndSignature(t *testing.T) {
	srv, ent := setupManagedEventsServer(t, `
Процедура ПроверитьЗакрытие(Первый, Второй)
КонецПроцедуры
`, map[metadata.FormEventType]string{metadata.FormEventBeforeClose: "ПроверитьЗакрытие"}, nil)

	badReason := closeIntentBody(uuid.NewString(), "forged", "x")
	badReasonResp := executeFormCloseIntent(t, srv, ent, badReason)
	if badReasonResp.Code != http.StatusBadRequest || decodeCloseIntentResponse(t, badReasonResp).Close.Allowed {
		t.Fatalf("unknown reason was accepted: %d %s", badReasonResp.Code, badReasonResp.Body)
	}

	badSig := executeFormCloseIntent(t, srv, ent, closeIntentBody(uuid.NewString(), "cross", "x"))
	resp := decodeCloseIntentResponse(t, badSig)
	if resp.OK || resp.Close == nil || resp.Close.Allowed || !strings.Contains(resp.Error, "0 или 1") {
		t.Fatalf("invalid BeforeClose signature was accepted: %+v", resp)
	}
}

func TestProcessorFormCloseIntentUsesOutputCancelAndReason(t *testing.T) {
	program := mustParse(t, `
Процедура ПроверитьЗакрытие(Cancel)
	Сообщить(CloseReason);
	Если Объект.Имя = "" Тогда
		Cancel = Истина;
	КонецЕсли;
КонецПроцедуры
`)
	form := processorExecutionForm(&metadata.FormElement{
		Kind: metadata.FormElementField, Name: "ПолеИмя", DataPath: "Объект.Имя",
	})
	form.Handlers = map[metadata.FormEventType]string{metadata.FormEventBeforeClose: "ПроверитьЗакрытие"}
	form.ProgramAST = program
	proc := &processor.Processor{
		Name: "ЗакрываемаяОбработка", Params: []processor.Param{{Name: "Имя", Type: "string"}},
		Forms: []*metadata.FormModule{form},
	}
	srv, _ := newProcessorFormEventExecutionServer(t, proc, nil)
	body := url.Values{"Имя": {""}}
	body.Set(processorServiceFieldName(proc.Params, "_close_intent_id"), uuid.NewString())
	body.Set(processorServiceFieldName(proc.Params, "_close_reason"), "popup_cancel")
	body.Set(processorServiceFieldName(proc.Params, "_close_mode"), "discard")

	response := decodeCloseIntentResponse(t, executeProcessorCloseIntent(t, srv, proc, body))
	if !response.OK || response.Close == nil || response.Close.Allowed {
		t.Fatalf("processor output Cancel did not keep the form open: %+v", response)
	}
	if strings.Join(response.Messages, "|") != "popup_cancel" {
		t.Fatalf("processor close reason was not injected: %v", response.Messages)
	}
}

func TestFormCloseReplayLedgerWaitsForConcurrentExactRequest(t *testing.T) {
	ledger := newFormCloseReplayLedger()
	hash := sha256.Sum256([]byte("same form state"))
	reservation, replay, err := ledger.reserve(context.Background(), "user", "user\x00route|intent", hash)
	if err != nil || reservation == nil || replay != nil {
		t.Fatalf("initial reserve: reservation=%v replay=%v err=%v", reservation, replay, err)
	}

	type outcome struct {
		replay *formCloseReplayResult
		err    error
	}
	ready := make(chan struct{})
	result := make(chan outcome, 1)
	var once sync.Once
	go func() {
		once.Do(func() { close(ready) })
		_, got, gotErr := ledger.reserve(context.Background(), "user", "user\x00route|intent", hash)
		result <- outcome{replay: got, err: gotErr}
	}()
	<-ready
	select {
	case got := <-result:
		t.Fatalf("concurrent exact request returned before completion: %+v", got)
	case <-time.After(20 * time.Millisecond):
	}

	want := formCloseReplayResult{status: http.StatusOK, header: http.Header{"X-Test": {"yes"}}, body: []byte("terminal")}
	reservation.complete(want)
	select {
	case got := <-result:
		if got.err != nil || got.replay == nil || got.replay.status != want.status || string(got.replay.body) != string(want.body) {
			t.Fatalf("concurrent replay mismatch: %+v", got)
		}
	case <-time.After(time.Second):
		t.Fatal("concurrent exact request did not wake after completion")
	}
}
