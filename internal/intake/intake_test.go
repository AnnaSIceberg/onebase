package intake_test

import (
	"context"
	"crypto/sha256"
	"encoding/json"
	"errors"
	"fmt"
	"path/filepath"
	"sync"
	"sync/atomic"
	"testing"
	"time"

	"github.com/google/uuid"

	"github.com/ivantit66/onebase/internal/intake"
	"github.com/ivantit66/onebase/internal/metadata"
	"github.com/ivantit66/onebase/internal/storage"
)

// setup поднимает SQLite с таблицами приёмки и «бизнес-таблицей» zayavki, на
// которой проверяется транзакционность обработчика.
func setup(t *testing.T) (*intake.Engine, *storage.DB, context.Context) {
	t.Helper()
	ctx := context.Background()
	db, err := storage.ConnectSQLite(ctx, filepath.Join(t.TempDir(), "intake.db"))
	if err != nil {
		t.Fatalf("ConnectSQLite: %v", err)
	}
	t.Cleanup(func() { db.Close() })
	if err := db.EnsureIntakeSchema(ctx); err != nil {
		t.Fatalf("EnsureIntakeSchema: %v", err)
	}
	if _, err := db.Exec(ctx, `CREATE TABLE zayavki (id TEXT PRIMARY KEY, phone TEXT)`); err != nil {
		t.Fatalf("create zayavki: %v", err)
	}
	return intake.New(db), db, ctx
}

func sampleIntake() *metadata.Intake {
	in := &metadata.Intake{
		Name: "SiteLead", Transport: "http", Endpoint: "/hs/site/lead", Handler: "SiteLead",
		Idempotency: metadata.IntakeIdempotency{Key: "event_id", Scope: []string{"source", "aggregate"}},
	}
	in.Normalize()
	return in
}

func env(t *testing.T, eventID string, payload map[string]any) intake.Envelope {
	t.Helper()
	top := map[string]any{
		"event_id": eventID, "source": "site", "aggregate": "Заявка",
		"correlation_id": "corr-" + eventID, "payload": payload,
	}
	raw, err := json.Marshal(top)
	if err != nil {
		t.Fatalf("marshal envelope: %v", err)
	}
	e, err := intake.ParseEnvelope(raw)
	if err != nil {
		t.Fatalf("ParseEnvelope: %v", err)
	}
	return e
}

// creatingHandler вставляет заявку в той же транзакции приёмки и считает вызовы.
func creatingHandler(db *storage.DB, calls *int64) intake.HandlerFunc {
	return func(ctx context.Context, e intake.Envelope) (intake.HandlerResult, error) {
		atomic.AddInt64(calls, 1)
		id := uuid.NewString()
		phone, _ := e.Payload["phone"].(string)
		if _, err := db.Exec(ctx, `INSERT INTO zayavki (id, phone) VALUES (?, ?)`, id, phone); err != nil {
			return intake.HandlerResult{}, err
		}
		return intake.HandlerResult{Ref: id, BusinessResult: map[string]any{"phone": phone}}, nil
	}
}

// failingHandler создаёт заявку, но затем возвращает ошибку — проверка отката.
func failingHandler(db *storage.DB) intake.HandlerFunc {
	return func(ctx context.Context, e intake.Envelope) (intake.HandlerResult, error) {
		id := uuid.NewString()
		if _, err := db.Exec(ctx, `INSERT INTO zayavki (id, phone) VALUES (?, ?)`, id, "999"); err != nil {
			return intake.HandlerResult{}, err
		}
		return intake.HandlerResult{}, errors.New("обработчик упал")
	}
}

func count(t *testing.T, db *storage.DB, ctx context.Context, table string) int {
	t.Helper()
	var n int
	if err := db.QueryRow(ctx, `SELECT COUNT(*) FROM `+table).Scan(&n); err != nil {
		t.Fatalf("count %s: %v", table, err)
	}
	return n
}

func TestIngestCanceledDeliveryCanBeRetried(t *testing.T) {
	eng, db, ctx := setup(t)
	in := sampleIntake()
	in.DLQ.On = []string{metadata.DLQHandlerError}
	event := env(t, "E1", map[string]any{"phone": "111"})
	request, cancel := context.WithCancel(ctx)
	defer cancel()
	var calls int64
	create := creatingHandler(db, &calls)
	canceled := intake.HandlerFunc(func(tx context.Context, e intake.Envelope) (intake.HandlerResult, error) {
		_, err := create.Handle(tx, e)
		if err != nil {
			return intake.HandlerResult{}, err
		}
		cancel()
		return intake.HandlerResult{}, request.Err()
	})

	if _, err := eng.Ingest(request, in, canceled, event); !errors.Is(err, context.Canceled) {
		t.Fatalf("canceled delivery: %v", err)
	}
	for _, table := range []string{"_intake_log", "_intake_dlq", "zayavki"} {
		if n := count(t, db, ctx, table); n != 0 {
			t.Fatalf("canceled delivery left %d rows in %s", n, table)
		}
	}
	for _, expected := range []intake.Status{intake.StatusAccepted, intake.StatusDuplicate} {
		result, err := eng.Ingest(ctx, in, create, event)
		if err != nil || result.Status != expected {
			t.Fatalf("retry: status=%s, want %s, err=%v", result.Status, expected, err)
		}
	}
	if n := count(t, db, ctx, "zayavki"); n != 1 || calls != 2 {
		t.Fatalf("durable effects=%d, handler calls=%d", n, calls)
	}
}

func TestIngest_HappyPath(t *testing.T) {
	eng, db, ctx := setup(t)
	var calls int64
	res, err := eng.Ingest(ctx, sampleIntake(), creatingHandler(db, &calls), env(t, "E1", map[string]any{"phone": "111"}))
	if err != nil {
		t.Fatalf("Ingest: %v", err)
	}
	if res.Status != intake.StatusAccepted {
		t.Fatalf("статус=%q, ожидалось Принято", res.Status)
	}
	if res.ResultRef == "" {
		t.Fatalf("не заполнена ссылка результата")
	}
	if n := count(t, db, ctx, "zayavki"); n != 1 {
		t.Fatalf("заявок=%d, ожидалась 1", n)
	}
	row, _, _ := db.GetIntakeLog(ctx, "SiteLead", `["site","Заявка"]`, "E1")
	if row.Status != storage.IntakeStatusProcessed {
		t.Fatalf("строка идемпотентности не processed: %q", row.Status)
	}
}

func TestIngest_Duplicate(t *testing.T) {
	eng, db, ctx := setup(t)
	in := sampleIntake()
	var calls int64
	h := creatingHandler(db, &calls)
	payload := map[string]any{"phone": "111"}

	first, _ := eng.Ingest(ctx, in, h, env(t, "E1", payload))
	second, err := eng.Ingest(ctx, in, h, env(t, "E1", payload))
	if err != nil {
		t.Fatalf("Ingest дубль: %v", err)
	}
	if second.Status != intake.StatusDuplicate {
		t.Fatalf("статус=%q, ожидалось Дубль", second.Status)
	}
	if second.ResultRef != first.ResultRef {
		t.Fatalf("дубль вернул другую ссылку: %q != %q", second.ResultRef, first.ResultRef)
	}
	if atomic.LoadInt64(&calls) != 1 {
		t.Fatalf("обработчик вызван %d раз, ожидался 1 (без повторной обработки)", calls)
	}
	if n := count(t, db, ctx, "zayavki"); n != 1 {
		t.Fatalf("заявок=%d, ожидалась 1", n)
	}
}

func TestIngest_Mismatch(t *testing.T) {
	eng, db, ctx := setup(t)
	in := sampleIntake()
	var calls int64
	h := creatingHandler(db, &calls)

	if _, err := eng.Ingest(ctx, in, h, env(t, "E1", map[string]any{"phone": "111"})); err != nil {
		t.Fatal(err)
	}
	// Тот же event_id, другое тело → несовпадение → карантин.
	res, err := eng.Ingest(ctx, in, h, env(t, "E1", map[string]any{"phone": "222"}))
	if err != nil {
		t.Fatalf("Ingest mismatch: %v", err)
	}
	if res.Status != intake.StatusQuarantined || res.Reason != metadata.DLQSchemaMismatch {
		t.Fatalf("ожидался Карантин/schema_mismatch, получено %q/%q", res.Status, res.Reason)
	}
	if n := count(t, db, ctx, "zayavki"); n != 1 {
		t.Fatalf("заявок=%d, ожидалась 1 (вторая — карантин, без создания)", n)
	}
	open, _ := db.ListIntakeDLQ(ctx, "SiteLead", true, 0)
	if len(open) != 1 || open[0].Reason != metadata.DLQSchemaMismatch {
		t.Fatalf("ожидалась 1 запись карантина schema_mismatch, получено %d", len(open))
	}
}

// TestIngest_HandlerError_Rollback — сбой обработчика откатывает бизнес-объект
// целиком (транзакционная связка) и отправляет событие в карантин. Никаких
// полу-записей.
func TestIngest_HandlerError_Rollback(t *testing.T) {
	eng, db, ctx := setup(t)
	in := sampleIntake()

	res, err := eng.Ingest(ctx, in, failingHandler(db), env(t, "E1", map[string]any{"phone": "111"}))
	if err != nil {
		t.Fatalf("Ingest: %v", err)
	}
	if res.Status != intake.StatusQuarantined || res.Reason != metadata.DLQHandlerError {
		t.Fatalf("ожидался Карантин/handler_error, получено %q/%q", res.Status, res.Reason)
	}
	if n := count(t, db, ctx, "zayavki"); n != 0 {
		t.Fatalf("заявок=%d, ожидалось 0 (откат)", n)
	}
	row, _, _ := db.GetIntakeLog(ctx, "SiteLead", `["site","Заявка"]`, "E1")
	if row.Status != storage.IntakeStatusQuarantined {
		t.Fatalf("строка идемпотентности должна быть quarantined, получено %q", row.Status)
	}
	if open, _ := db.ListIntakeDLQ(ctx, "SiteLead", true, 0); len(open) != 1 {
		t.Fatalf("ожидалась 1 запись карантина, получено %d", len(open))
	}
}

func TestIngest_SchemaVersionMismatch(t *testing.T) {
	eng, db, ctx := setup(t)
	in := sampleIntake()
	in.SchemaVersion = "1"
	var calls int64

	res, err := eng.Ingest(ctx, in, creatingHandler(db, &calls), env(t, "E1", map[string]any{"phone": "111"}))
	if err != nil {
		t.Fatalf("Ingest: %v", err)
	}
	if res.Status != intake.StatusQuarantined || res.Reason != metadata.DLQSchemaMismatch {
		t.Fatalf("ожидался Карантин/schema_mismatch, получено %q/%q", res.Status, res.Reason)
	}
	if atomic.LoadInt64(&calls) != 0 || count(t, db, ctx, "zayavki") != 0 {
		t.Fatal("обработчик не должен запускаться при несовместимой версии схемы")
	}
}

func TestIngest_DLQWriteFailureReleasesIdempotencyKey(t *testing.T) {
	eng, db, ctx := setup(t)
	in := sampleIntake()
	event := env(t, "E1", map[string]any{"phone": "111"})
	if _, err := db.Exec(ctx, `
		CREATE TRIGGER fail_intake_dlq_insert
		BEFORE INSERT ON _intake_dlq
		BEGIN
			SELECT RAISE(ABORT, 'forced dlq failure');
		END`); err != nil {
		t.Fatalf("create trigger: %v", err)
	}

	if _, err := eng.Ingest(ctx, in, failingHandler(db), event); err == nil {
		t.Fatal("ожидалась ошибка записи DLQ")
	}
	if _, found, err := db.GetIntakeLog(ctx, "SiteLead", `["site","Заявка"]`, "E1"); err != nil || found {
		t.Fatalf("после ошибки DLQ ключ должен быть освобождён: found=%v err=%v", found, err)
	}
	if _, err := db.Exec(ctx, `DROP TRIGGER fail_intake_dlq_insert`); err != nil {
		t.Fatalf("drop trigger: %v", err)
	}

	var calls int64
	res, err := eng.Ingest(ctx, in, creatingHandler(db, &calls), event)
	if err != nil || res.Status != intake.StatusAccepted {
		t.Fatalf("retry после восстановления DLQ: status=%q err=%v", res.Status, err)
	}
	if count(t, db, ctx, "zayavki") != 1 {
		t.Fatal("retry после ошибки DLQ не создал объект")
	}
}

// TestReplay восстанавливает событие из карантина: после починки обработчик
// проходит, заявка создаётся ровно одна, запись карантина закрывается.
func TestReplay(t *testing.T) {
	eng, db, ctx := setup(t)
	in := sampleIntake()

	// Сначала загоняем в карантин сбойным обработчиком.
	if _, err := eng.Ingest(ctx, in, failingHandler(db), env(t, "E1", map[string]any{"phone": "111"})); err != nil {
		t.Fatal(err)
	}
	open, err := db.ListIntakeDLQ(ctx, "SiteLead", true, 0)
	if err != nil || len(open) != 1 {
		t.Fatalf("получить карантин: len=%d err=%v", len(open), err)
	}

	var calls int64
	res, err := eng.Replay(ctx, in, creatingHandler(db, &calls), open[0].ID)
	if err != nil {
		t.Fatalf("Replay: %v", err)
	}
	if res.Status != intake.StatusAccepted || res.ResultRef == "" {
		t.Fatalf("ожидалось Принято со ссылкой, получено %q/%q", res.Status, res.ResultRef)
	}
	if n := count(t, db, ctx, "zayavki"); n != 1 {
		t.Fatalf("заявок=%d, ожидалась 1 (сбойный проход ничего не закоммитил)", n)
	}
	if open, _ := db.ListIntakeDLQ(ctx, "SiteLead", true, 0); len(open) != 0 {
		t.Fatalf("после replay открытых записей быть не должно, есть %d", len(open))
	}
	row, _, _ := db.GetIntakeLog(ctx, "SiteLead", `["site","Заявка"]`, "E1")
	if row.Status != storage.IntakeStatusProcessed {
		t.Fatalf("после replay строка должна быть processed, получено %q", row.Status)
	}
}

func TestReplay_IsIdempotent(t *testing.T) {
	eng, db, ctx := setup(t)
	in := sampleIntake()
	if _, err := eng.Ingest(ctx, in, failingHandler(db), env(t, "E1", map[string]any{"phone": "111"})); err != nil {
		t.Fatal(err)
	}
	open, _ := db.ListIntakeDLQ(ctx, "SiteLead", true, 0)

	var calls int64
	h := creatingHandler(db, &calls)
	first, err := eng.Replay(ctx, in, h, open[0].ID)
	if err != nil || first.Status != intake.StatusAccepted {
		t.Fatalf("первый Replay: status=%q err=%v", first.Status, err)
	}
	second, err := eng.Replay(ctx, in, h, open[0].ID)
	if err != nil || second.Status != intake.StatusDuplicate {
		t.Fatalf("повторный Replay: status=%q err=%v", second.Status, err)
	}
	if second.ResultRef != first.ResultRef {
		t.Fatalf("повторный Replay вернул другую ссылку: %q != %q", second.ResultRef, first.ResultRef)
	}
	if atomic.LoadInt64(&calls) != 1 || count(t, db, ctx, "zayavki") != 1 {
		t.Fatalf("повторный Replay выполнил обработчик повторно: calls=%d", calls)
	}
}

func TestReplay_StateTransitionFailureRollsBackBusinessObject(t *testing.T) {
	eng, db, ctx := setup(t)
	in := sampleIntake()
	if _, err := eng.Ingest(ctx, in, failingHandler(db), env(t, "E1", map[string]any{"phone": "111"})); err != nil {
		t.Fatal(err)
	}
	open, _ := db.ListIntakeDLQ(ctx, "SiteLead", true, 0)

	// Имитируем сбой записи replay_state. Закрытие DLQ должно быть в той же
	// транзакции и произойти до обработчика, иначе бизнес-объект останется,
	// а открытый карантин позволит создать его повторно.
	if _, err := db.Exec(ctx, `
		CREATE TRIGGER fail_intake_replay
		BEFORE UPDATE OF replay_state ON _intake_dlq
		BEGIN
			SELECT RAISE(ABORT, 'forced replay state failure');
		END`); err != nil {
		t.Fatalf("create trigger: %v", err)
	}

	var calls int64
	if _, err := eng.Replay(ctx, in, creatingHandler(db, &calls), open[0].ID); err == nil {
		t.Fatal("ожидалась ошибка перехода replay_state")
	}
	if atomic.LoadInt64(&calls) != 0 {
		t.Fatalf("обработчик вызван до безопасного захвата DLQ: %d", calls)
	}
	if n := count(t, db, ctx, "zayavki"); n != 0 {
		t.Fatalf("после сбоя replay осталось бизнес-объектов: %d", n)
	}
	row, _, _ := db.GetIntakeLog(ctx, "SiteLead", `["site","Заявка"]`, "E1")
	if row.Status != storage.IntakeStatusQuarantined {
		t.Fatalf("после сбоя replay статус=%q, ожидался quarantined", row.Status)
	}
}

func TestReplay_RejectsSchemaMismatch(t *testing.T) {
	eng, db, ctx := setup(t)
	in := sampleIntake()
	var calls int64
	h := creatingHandler(db, &calls)

	if _, err := eng.Ingest(ctx, in, h, env(t, "E1", map[string]any{"phone": "111"})); err != nil {
		t.Fatal(err)
	}
	if _, err := eng.Ingest(ctx, in, h, env(t, "E1", map[string]any{"phone": "222"})); err != nil {
		t.Fatal(err)
	}
	open, _ := db.ListIntakeDLQ(ctx, "SiteLead", true, 0)
	if len(open) != 1 || open[0].Reason != metadata.DLQSchemaMismatch {
		t.Fatalf("ожидался schema_mismatch в карантине: %+v", open)
	}

	if _, err := eng.Replay(ctx, in, h, open[0].ID); err == nil {
		t.Fatal("schema_mismatch нельзя replay без разрешения конфликта")
	}
	if atomic.LoadInt64(&calls) != 1 || count(t, db, ctx, "zayavki") != 1 {
		t.Fatalf("replay schema_mismatch создал второй объект: calls=%d", calls)
	}
}

func TestIngest_MissingKey(t *testing.T) {
	eng, db, ctx := setup(t)
	var calls int64
	// Конверт без event_id.
	raw, _ := json.Marshal(map[string]any{"source": "site", "payload": map[string]any{"phone": "1"}})
	e, _ := intake.ParseEnvelope(raw)
	res, err := eng.Ingest(ctx, sampleIntake(), creatingHandler(db, &calls), e)
	if err != nil {
		t.Fatalf("Ingest: %v", err)
	}
	if res.Status != intake.StatusRejected {
		t.Fatalf("ожидалось Отклонено, получено %q", res.Status)
	}
	if atomic.LoadInt64(&calls) != 0 {
		t.Fatalf("обработчик не должен вызываться без ключа")
	}
}

func TestIngest_MissingScope(t *testing.T) {
	eng, db, ctx := setup(t)
	var calls int64
	raw, _ := json.Marshal(map[string]any{
		"event_id": "E1", "source": "site", "payload": map[string]any{"phone": "1"},
	})
	e, _ := intake.ParseEnvelope(raw)
	res, err := eng.Ingest(ctx, sampleIntake(), creatingHandler(db, &calls), e)
	if err != nil {
		t.Fatalf("Ingest: %v", err)
	}
	if res.Status != intake.StatusRejected || res.Reason != "missing_idempotency_scope:aggregate" {
		t.Fatalf("ожидалось отклонение отсутствующего scope, получено %q/%q", res.Status, res.Reason)
	}
	if atomic.LoadInt64(&calls) != 0 {
		t.Fatal("обработчик не должен вызываться без обязательного scope")
	}
}

func TestIngest_ScopeValuesDoNotCollide(t *testing.T) {
	eng, db, ctx := setup(t)
	in := sampleIntake()
	var calls int64
	h := creatingHandler(db, &calls)

	first := env(t, "E1", map[string]any{"phone": "111"})
	first.Top["source"] = "a|b"
	first.Top["aggregate"] = "c"
	second := env(t, "E1", map[string]any{"phone": "222"})
	second.Top["source"] = "a"
	second.Top["aggregate"] = "b|c"

	r1, err1 := eng.Ingest(ctx, in, h, first)
	r2, err2 := eng.Ingest(ctx, in, h, second)
	if err1 != nil || err2 != nil || r1.Status != intake.StatusAccepted || r2.Status != intake.StatusAccepted {
		t.Fatalf("разные scope должны приниматься независимо: r1=%q/%v r2=%q/%v", r1.Status, err1, r2.Status, err2)
	}
	if count(t, db, ctx, "zayavki") != 2 {
		t.Fatal("разные составные scope ошибочно схлопнулись")
	}
}

func TestIngest_InFlightIsNotAcknowledgedAsDuplicate(t *testing.T) {
	eng, db, ctx := setup(t)
	eng.InFlightWait = 25 * time.Millisecond
	in := sampleIntake()
	event := env(t, "E1", map[string]any{"phone": "111"})
	payload, _ := json.Marshal(event.Payload)
	hash := fmt.Sprintf("%x", sha256.Sum256(payload))
	if _, err := db.InsertIntakeLogIfNew(ctx, storage.IntakeLogRow{
		Intake: "SiteLead", Scope: `["site","Заявка"]`, Key: "E1",
		PayloadHash: hash, Status: storage.IntakeStatusReceived, ReceivedAt: 1,
	}); err != nil {
		t.Fatalf("seed received: %v", err)
	}
	var calls int64

	if res, err := eng.Ingest(ctx, in, creatingHandler(db, &calls), event); err == nil {
		t.Fatalf("in-flight событие нельзя подтверждать как %q до результата первого обработчика", res.Status)
	}
	if atomic.LoadInt64(&calls) != 0 || count(t, db, ctx, "zayavki") != 0 {
		t.Fatalf("received-дубль запустил обработчик: calls=%d", calls)
	}
}

// TestIngest_Concurrency — конкурентная приёмка одного event_id: ровно одна
// заявка, ровно одно Принято, обработчик вызван один раз. Доказывает, что
// атомарный insert-if-new гейтит обработку.
func TestIngest_Concurrency(t *testing.T) {
	eng, db, ctx := setup(t)
	in := sampleIntake()
	var calls int64
	h := creatingHandler(db, &calls)
	payload := map[string]any{"phone": "111"}

	const goroutines = 30
	var start, done sync.WaitGroup
	start.Add(1)
	var accepted int64
	errs := make(chan error, goroutines)
	for g := 0; g < goroutines; g++ {
		done.Add(1)
		go func() {
			defer done.Done()
			start.Wait()
			res, err := eng.Ingest(ctx, in, h, env(t, "E1", payload))
			if err != nil {
				errs <- err
				return
			}
			if res.Status == intake.StatusAccepted {
				atomic.AddInt64(&accepted, 1)
			}
		}()
	}
	start.Done()
	done.Wait()
	close(errs)
	for e := range errs {
		t.Fatalf("Ingest concurrency error: %v", e)
	}
	if accepted != 1 {
		t.Fatalf("Принято=%d, ожидалось ровно 1", accepted)
	}
	if c := atomic.LoadInt64(&calls); c != 1 {
		t.Fatalf("обработчик вызван %d раз, ожидался 1", c)
	}
	if n := count(t, db, ctx, "zayavki"); n != 1 {
		t.Fatalf("заявок=%d, ожидалась 1", n)
	}
}
