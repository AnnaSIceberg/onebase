package storage

import (
	"context"
	"strings"
	"sync/atomic"
)

// Счётчик записей запроса. Отвечает на вопрос «писал ли этот запрос в базу»:
// событие формы перечитывает строки табличных частей после обработчика,
// только если обработчик что-то записал (мог переписать их через общий
// модуль, #579), — иначе перечитывать незачем, а у заявки колл-центра это 17
// запросов на событие.
//
// Счётчик живёт в контексте, а не в процессе: на живом сервере базу
// постоянно пишут фоновые исполнители (очередь заданий, heartbeat), и общий
// счётчик сдвигался бы на каждом событии. Записи наследуют контекст
// обработчика — и прямые, и в транзакции (BeginTx выводит свой контекст из
// переданного). Учёт с запасом: всё, что не чистый SELECT и не служебная
// команда транзакции, считается записью.

type writeCounterKey struct{}

// WithWriteCounter добавляет в контекст счётчик записей и возвращает его.
func WithWriteCounter(ctx context.Context) (context.Context, *atomic.Uint64) {
	c := &atomic.Uint64{}
	return context.WithValue(ctx, writeCounterKey{}, c), c
}

func countWrite(ctx context.Context) {
	if ctx == nil {
		return
	}
	if c, _ := ctx.Value(writeCounterKey{}).(*atomic.Uint64); c != nil {
		c.Add(1)
	}
}

// noteExec учитывает оператор Exec: всё, кроме служебных команд транзакции и
// настроек, считается записью.
func noteExec(ctx context.Context, sqlText string) {
	switch firstSQLWord(sqlText) {
	case "SAVEPOINT", "RELEASE", "BEGIN", "COMMIT", "ROLLBACK", "SET", "PRAGMA", "ANALYZE", "SHOW":
		return
	}
	countWrite(ctx)
}

// noteQuery учитывает Query/QueryRow: запись — всё, что не чистый SELECT
// (INSERT … RETURNING, CTE с изменяющими предложениями), по тому же правилу,
// что и режим только чтения (queryAllowed).
func noteQuery(ctx context.Context, sqlText string) {
	if firstSQLWord(sqlText) == "SELECT" && !strings.Contains(sqlText, ";") {
		return
	}
	countWrite(ctx)
}

func firstSQLWord(sqlText string) string {
	fields := strings.Fields(sqlText)
	if len(fields) == 0 {
		return ""
	}
	return strings.ToUpper(strings.TrimLeft(fields[0], "("))
}
