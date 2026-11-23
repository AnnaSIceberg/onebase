package ui

import (
	"bytes"
	"net/http"
	"net/http/httptest"
	"net/url"
	"os"
	"path/filepath"
	"strings"
	"sync"
	"testing"
	"time"

	"github.com/ivantit66/onebase/internal/auth"
	"github.com/ivantit66/onebase/internal/dbtest"
	"github.com/ivantit66/onebase/internal/metadata"
	"github.com/ivantit66/onebase/internal/processor"
	"github.com/ivantit66/onebase/internal/storage"
	"github.com/stretchr/testify/require"
	"github.com/xuri/excelize/v2"
)

func TestManagedWorkbookDownloadAndUpdateHTTP(t *testing.T) {
	dbtest.ForEachDialect(t, func(t *testing.T, db *storage.DB) {
		dir := t.TempDir()
		t.Setenv("ONEBASE_REPORT_XLSX_DIR", dir)
		s, user, _ := tabularFixture(t, db)
		const filename = "Отчёт 400.xlsx"
		form := processorExecutionForm(
			&metadata.FormElement{Kind: metadata.FormElementField, Name: "Branch", DataPath: "Объект.Branch"},
			&metadata.FormElement{Kind: metadata.FormElementFormField, Name: "Destination", WorkbookSavePicker: filename},
			&metadata.FormElement{Kind: metadata.FormElementButton, Name: "Build", WorkbookSave: true, Handlers: map[metadata.FormEventType]string{metadata.FormEventOnClick: "Build"}},
			&metadata.FormElement{Kind: metadata.FormElementButton, Name: "Replace", Handlers: map[metadata.FormEventType]string{metadata.FormEventOnClick: "Replace"}},
			&metadata.FormElement{Kind: metadata.FormElementButton, Name: "ReplaceAgain", Handlers: map[metadata.FormEventType]string{metadata.FormEventOnClick: "ReplaceAgain"}},
			&metadata.FormElement{Kind: metadata.FormElementButton, Name: "AddOther", Handlers: map[metadata.FormEventType]string{metadata.FormEventOnClick: "AddOther"}},
			&metadata.FormElement{Kind: metadata.FormElementButton, Name: "Confirm", WorkbookUpdateOnAnswer: true, Handlers: map[metadata.FormEventType]string{metadata.FormEventOnClick: "Ask", metadata.FormEventOnAnswer: "Confirm"}},
			&metadata.FormElement{Kind: metadata.FormElementButton, Name: "Unmarked", Handlers: map[metadata.FormEventType]string{metadata.FormEventOnAnswer: "Confirm"}},
		)
		form.ProgramAST = mustParse(t, `Функция MakeBook(Число, ИмяЛиста = "Итого")
  Лист = Новый ТабличныйДокумент;
  Лист.Ячейка(1,1).Значение = Число;
  Книга = Новый КнигаТабличныхДокументов;
  Книга.ДобавитьЛист(ИмяЛиста, Лист);
  Возврат Книга;
КонецФункции
Процедура Build()
  СообщитьПрогрессФормирования(1, 2);
  СкачатьКнигуExcel("Отчёт 400.xlsx", MakeBook(41));
КонецПроцедуры
Процедура Replace()
  ЗаменитьЛистыExcel("Отчёт 400.xlsx", MakeBook(42));
  Сообщить("Книга обновлена");
КонецПроцедуры
Процедура ReplaceAgain()
  ЗаменитьЛистыExcel("Отчёт 400.xlsx", MakeBook(43));
КонецПроцедуры
Процедура AddOther()
  ЗаменитьЛистыExcel("Отчёт 400.xlsx", MakeBook(44, "Период"));
КонецПроцедуры
Процедура Ask()
  ПоказатьВопрос("Продолжить?", "ДаНет");
КонецПроцедуры
Процедура Confirm()
  Если ВопросОтвет = "Да" Тогда
    ЗаменитьЛистыExcel("Отчёт 400.xlsx", MakeBook(45));
  КонецЕсли;
КонецПроцедуры`)
		proc := &processor.Processor{Name: "WorkbookTest", Forms: []*metadata.FormModule{form}, Params: []processor.Param{{Name: "Branch", Type: "string"}}}
		s.reg.LoadProcessors([]*processor.Processor{proc})
		user.Roles[0].Permissions.Processors = map[string][]string{proc.Name: {"run"}}
		base := "/ui/processor/" + proc.Name
		page := tabularHTTP(s, user, "own", "GET", base, nil)
		require.Equal(t, http.StatusOK, page.Code)
		for _, marker := range []string{"data-ob-workbook-picker=", "data-ob-workbook-save", "data-ob-workbook-progress", "/static/tabular.js"} {
			require.Contains(t, page.Body.String(), marker)
		}
		require.NotContains(t, page.Body.String(), "data-ob-tabular-field=")
		s.cfg.UIExceptRoles = []string{"ОператорКЦ", "ОператорМессенджера"}
		for _, role := range s.cfg.UIExceptRoles {
			mixed := *user
			mixed.Roles = []*auth.Role{user.Roles[0], {Name: role}}
			legacy := tabularHTTP(s, &mixed, "own", "GET", base, nil)
			require.Equal(t, http.StatusOK, legacy.Code, role)
			require.Contains(t, legacy.Body.String(), `data-ob-fire-click="Build"`, role)
			require.NotContains(t, legacy.Body.String(), "data-ob-workbook-picker=", role)
			require.NotContains(t, legacy.Body.String(), "data-ob-workbook-save", role)
			require.NotContains(t, legacy.Body.String(), "рендеринг не реализован", role)
		}
		clickRequest := func(session, button string) *httptest.ResponseRecorder {
			body := url.Values{"_form": {form.Name}, "_element": {button}, "_event": {"Нажатие"}}
			return tabularHTTP(s, user, session, "POST", base+"/form-event", body)
		}
		click := func(session, button string) formEventResponse {
			w := clickRequest(session, button)
			require.Equal(t, http.StatusOK, w.Code)
			return decodeFormEventResponse(t, w.Body.Bytes())
		}
		assertCell := func(data []byte, sheet, want string) {
			f, err := excelize.OpenReader(bytes.NewReader(data))
			require.NoError(t, err)
			value, err := f.GetCellValue(sheet, "A1")
			require.NoError(t, err)
			require.Equal(t, want, value)
			require.NoError(t, f.Close())
		}
		download := func(session string, resp formEventResponse) *httptest.ResponseRecorder {
			return tabularHTTP(s, user, session, "GET", resp.WorkbookDownloadURL, nil)
		}
		assertDownload := func(w *httptest.ResponseRecorder, sheet, want string) []byte {
			require.Equal(t, http.StatusOK, w.Code)
			require.Contains(t, w.Header().Get("Content-Disposition"), "attachment")
			assertCell(w.Body.Bytes(), sheet, want)
			return w.Body.Bytes()
		}
		read := func(session string, resp formEventResponse, want string) []byte {
			require.True(t, resp.OK, resp.Error)
			require.Contains(t, resp.WorkbookDownloadURL, "/ui/tabular/")
			require.Equal(t, http.StatusNotFound, download("foreign", resp).Code)
			return assertDownload(download(session, resp), "Итого", want)
		}
		missing := click("own", "Replace")
		require.False(t, missing.OK)
		require.Empty(t, missing.WorkbookDownloadURL)
		require.Zero(t, s.tabularResults.reservedBooks)
		request := url.Values{"_form": {form.Name}, "_element": {"Build"}, "_event": {"Нажатие"}}
		stream := tabularHTTPAccept(s, user, "own", "POST", base+"/form-event", request, tabularProgressType)
		require.Equal(t, http.StatusOK, stream.Code)
		require.True(t, stream.Flushed)
		require.Equal(t, tabularProgressType, stream.Header().Get("Content-Type"))
		records := strings.Split(strings.TrimSpace(stream.Body.String()), "\n")
		require.Len(t, records, 2)
		require.JSONEq(t, `{"progress":50}`, records[0])
		first := decodeFormEventResponse(t, []byte(records[1]))
		newFile := read("own", first, "41")
		path := filepath.Join(dir, filename)
		require.NoFileExists(t, path)
		require.NoError(t, os.WriteFile(path, newFile, 0o600))

		updated := click("second", "Replace")
		require.Contains(t, updated.Messages, "Книга обновлена")
		latest := click("third", "ReplaceAgain")
		read("own", first, "41")
		read("second", updated, "42")
		result := read("third", latest, "43")
		disk, err := os.ReadFile(path)
		require.NoError(t, err)
		require.Equal(t, result, disk)

		runTogether := func(actions ...func()) {
			var group sync.WaitGroup
			start := make(chan struct{})
			for _, action := range actions {
				group.Go(func() {
					<-start
					action()
				})
			}
			close(start)
			group.Wait()
		}
		var previous, replaceTotal, addPeriod *httptest.ResponseRecorder
		runTogether(
			func() { previous = download("own", first) },
			func() { replaceTotal = clickRequest("total", "Replace") },
			func() { addPeriod = clickRequest("period", "AddOther") },
		)
		require.Equal(t, newFile, assertDownload(previous, "Итого", "41"))
		require.Equal(t, http.StatusOK, replaceTotal.Code)
		require.Equal(t, http.StatusOK, addPeriod.Code)
		total := decodeFormEventResponse(t, replaceTotal.Body.Bytes())
		period := decodeFormEventResponse(t, addPeriod.Body.Bytes())
		require.True(t, total.OK, total.Error)
		require.True(t, period.OK, period.Error)
		require.NotEmpty(t, total.WorkbookDownloadURL)
		require.NotEmpty(t, period.WorkbookDownloadURL)
		require.NotEqual(t, total.WorkbookDownloadURL, period.WorkbookDownloadURL)
		var totalDownload, periodDownload *httptest.ResponseRecorder
		runTogether(
			func() { totalDownload = download("total", total) },
			func() { periodDownload = download("period", period) },
		)
		totalFile := assertDownload(totalDownload, "Итого", "42")
		periodFile := assertDownload(periodDownload, "Период", "44")
		require.Equal(t, http.StatusNotFound, download("total", period).Code)
		require.Equal(t, http.StatusNotFound, download("period", total).Code)
		disk, err = os.ReadFile(path)
		require.NoError(t, err)
		assertCell(disk, "Итого", "42")
		assertCell(disk, "Период", "44")
		require.True(t, bytes.Equal(disk, totalFile) || bytes.Equal(disk, periodFile))
		result = disk

		for i := 5; i < maxWorkbookDownloads; i++ {
			next := click("third", "Build")
			require.True(t, next.OK, next.Error)
			require.NotEmpty(t, next.WorkbookDownloadURL)
		}
		full := click("third", "Replace")
		require.False(t, full.OK)
		require.Empty(t, full.WorkbookDownloadURL)
		require.Contains(t, full.Error, "предел выдачи")
		disk, err = os.ReadFile(path)
		require.NoError(t, err)
		require.Equal(t, result, disk)
		read("own", first, "41")
		read("second", updated, "42")

		store := &s.tabularResults
		store.mu.Lock()
		for _, item := range store.workbooks {
			item.expires = time.Now().Add(-time.Second)
		}
		store.mu.Unlock()
		recovered := click("own", "Replace")
		read("own", recovered, "42")
		require.Equal(t, http.StatusNotFound, tabularHTTP(s, user, "second", "GET", updated.WorkbookDownloadURL, nil).Code)
		question := click("own", "Confirm")
		require.True(t, question.OK)
		require.NotNil(t, question.Question)
		require.Empty(t, question.WorkbookDownloadURL)
		require.NotEmpty(t, question.Question.WorkbookAnswerToken)
		answer := func(session, button, value, token, branch string) formEventResponse {
			body := url.Values{"_form": {form.Name}, "_element": {button}, "_event": {"Ответ"}, "_question_answer": {value},
				workbookAnswerTokenField: {token}, "Branch": {branch}}
			rec := tabularHTTP(s, user, session, http.MethodPost, base+"/form-event", body)
			require.Equal(t, http.StatusOK, rec.Code)
			return decodeFormEventResponse(t, rec.Body.Bytes())
		}
		token := question.Question.WorkbookAnswerToken
		denied := func(resp formEventResponse) {
			t.Helper()
			require.False(t, resp.OK)
			require.Empty(t, resp.WorkbookDownloadURL)
		}
		denied(answer("own", "Confirm", "Да", "", ""))
		denied(answer("second", "Confirm", "Да", token, ""))
		denied(answer("own", "Confirm", "Да", token, "Изменённый филиал"))
		denied(answer("own", "Unmarked", "Да", token, ""))
		cancelled := answer("own", "Confirm", "Нет", token, "")
		require.True(t, cancelled.OK, cancelled.Error)
		require.Empty(t, cancelled.WorkbookDownloadURL)
		denied(answer("own", "Confirm", "Да", token, ""))
		beforeApproval, err := os.ReadFile(path)
		require.NoError(t, err)
		assertCell(beforeApproval, "Итого", "42")
		newQuestion := click("own", "Confirm")
		require.NotNil(t, newQuestion.Question)
		token = newQuestion.Question.WorkbookAnswerToken
		read("own", answer("own", "Confirm", "Да", token, ""), "45")
		denied(answer("own", "Confirm", "Да", token, ""))
		user.Roles[0].Permissions.Processors = nil
		require.Equal(t, http.StatusNotFound, tabularHTTP(s, user, "own", "GET", recovered.WorkbookDownloadURL, nil).Code)
	})
}
