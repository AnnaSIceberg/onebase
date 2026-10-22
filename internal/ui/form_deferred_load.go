package ui

import (
	"net/http"
	"reflect"
	"strings"

	"github.com/ivantit66/onebase/internal/metadata"
)

// ПослеОткрытия — отложенная загрузка данных формы (история клиента из 1С и
// т.п.): обработчик группы запускается клиентом ПОСЛЕ ответа ПриОткрытии
// отдельным запросом мимо общей очереди событий, пока на группе крутится
// индикатор. Форма открывается сразу, оператор работает, не дожидаясь ответа.
//
// Раз ответ приходит, когда пользователь и другие события уже могли изменить
// форму, он несёт только то, что изменил сам обработчик: реквизиты, значение
// которых отличается от присланного, и таблицы значений, чьи строки
// изменились. Состояния элементов, условное оформление и табличные части
// документа он не трогает — иначе откатил бы чужие правки.
const formEventAfterOpen = string(metadata.FormEventAfterOpen)

// deferredFormEvent — запрос отложенной загрузки (клиент шлёт _deferred=1 и
// событие ПослеОткрытия).
func deferredFormEvent(r *http.Request) bool {
	return r.FormValue("_deferred") == "1" && r.FormValue("_event") == formEventAfterOpen
}

// filterDeferredResponse сравнивает два типизированных снимка. Сырое POST-тело
// не подходит: браузер не отправляет false-флажки, а числа и даты присылает
// строками. nil baseline означает обычное синхронное событие.
func filterDeferredResponse(baseline *formEventState, resp *formEventResponse) {
	if baseline == nil {
		return
	}
	resp.Deferred = true
	for k, v := range resp.Values {
		if before, ok := baseline.Values[k]; ok && reflect.DeepEqual(before, v) {
			delete(resp.Values, k)
		}
	}
	for name, rows := range resp.FormTables {
		if deferredRowsEqual(baseline.FormTables[name], rows) {
			delete(resp.FormTables, name)
		}
	}
	resp.TableParts = nil
	resp.ElementStates = nil
	resp.ConditionalCSS = ""
	resp.Dirty = nil
}

func deferredRowsEqual(before, rows []map[string]any) bool {
	if len(before) != len(rows) {
		return false
	}
	for i, row := range rows {
		for k, v := range row {
			if strings.HasPrefix(k, "_") {
				continue // служебное оформление не является изменением данных
			}
			if !reflect.DeepEqual(before[i][k], v) {
				return false
			}
		}
	}
	return true
}
