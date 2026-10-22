package storage

import (
	"strings"
	"sync/atomic"
)

// writeSeq — счётчик операторов, способных изменить данные, по всему процессу.
// Он отвечает на вопрос «писал ли кто-нибудь в базу за этот отрезок»: событие
// формы перечитывает строки табличных частей после обработчика, только если
// счётчик сдвинулся (обработчик мог переписать их через общий модуль, #579), —
// иначе перечитывать незачем, а у заявки колл-центра это 17 запросов на
// событие. Учёт намеренно с запасом: запись соседнего пользователя тоже
// сдвигает счётчик, и лишнее перечитывание безопасно, пропущенное — нет.
var writeSeq atomic.Uint64

// WriteSeq — текущее значение счётчика записей.
func WriteSeq() uint64 { return writeSeq.Load() }

// noteExec учитывает оператор Exec: всё, кроме служебных команд транзакции и
// настроек, считается записью.
func noteExec(sqlText string) {
	switch firstSQLWord(sqlText) {
	case "SAVEPOINT", "RELEASE", "BEGIN", "COMMIT", "ROLLBACK", "SET", "PRAGMA", "ANALYZE", "SHOW":
		return
	}
	writeSeq.Add(1)
}

// noteQuery учитывает Query/QueryRow: запись — всё, что не чистый SELECT
// (INSERT … RETURNING, CTE с изменяющими предложениями), по тому же правилу,
// что и режим только чтения (queryAllowed).
func noteQuery(sqlText string) {
	if firstSQLWord(sqlText) == "SELECT" && !strings.Contains(sqlText, ";") {
		return
	}
	writeSeq.Add(1)
}

func firstSQLWord(sqlText string) string {
	fields := strings.Fields(sqlText)
	if len(fields) == 0 {
		return ""
	}
	return strings.ToUpper(strings.TrimLeft(fields[0], "("))
}
