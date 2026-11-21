package ui

import (
	"fmt"
	"os"
	"path/filepath"
	"strings"
	"testing"
	"time"

	"github.com/stretchr/testify/require"
	"github.com/xuri/excelize/v2"
)

func TestScheduledWorkbookReplacesSheetsWithoutHTTP(t *testing.T) {
	dir := t.TempDir()
	t.Setenv("ONEBASE_REPORT_XLSX_DIR", dir)
	path := filepath.Join(dir, "Сайты 400.xlsx")
	outside := filepath.Join(t.TempDir(), filepath.Base(path))
	titles := []string{"Исходный", "Ноябрь", "Итог 1", "Итог 2", "Итог 3", "Итог 4", "Архив"}
	initial := excelize.NewFile()
	require.NoError(t, initial.SetSheetName("Sheet1", titles[0]))
	style, err := initial.NewStyle(&excelize.Style{Font: &excelize.Font{Bold: true}})
	require.NoError(t, err)
	for i, title := range titles {
		if i != 0 {
			_, err := initial.NewSheet(title)
			require.NoError(t, err)
		}
		require.NoError(t, initial.SetCellInt(title, "A1", int64(i)))
	}
	require.NoError(t, initial.SetCellStyle("Архив", "A1", "A1", style))
	require.NoError(t, initial.SaveAs(path))
	require.NoError(t, initial.Close())
	original, err := os.ReadFile(path)
	require.NoError(t, err)
	require.NoError(t, os.WriteFile(outside, original, 0o600))

	makeBody := func(destination, book string) string {
		return fmt.Sprintf(`Лист = Новый ТабличныйДокумент;
Книга = Новый КнигаТабличныхДокументов;
Для Номер = 1 По 5 Цикл
  Если Номер = 1 Тогда
    Имя = "Ноябрь";
  Иначе
    Имя = "Итог " + Строка(Номер - 1);
  КонецЕсли;
  Лист.Ячейка(1, 1).Значение = 40 + Номер;
  Книга.ДобавитьЛист(Имя, Лист);
КонецЦикла;
Если ЗаменитьЛистыExcelФоном("%s", %s) Тогда
  Сообщить("обновлено");
КонецЕсли;`, strings.ReplaceAll(filepath.ToSlash(destination), `"`, `""`), book)
	}
	for _, tc := range []struct {
		name, dest, book, status string
		catch                    bool
	}{
		{"absolute file in root", path, "Книга", "success", false},
		{"relative path", filepath.Base(path), "Книга", "error", false},
		{"outside the root", outside, "Книга", "error", false},
		{"invalid workbook", path, "Неопределено", "error", false},
		{"recover by file", outside, "Книга", "success", true},
	} {
		t.Run(tc.name, func(t *testing.T) {
			before, err := os.ReadFile(path)
			require.NoError(t, err)
			body := makeBody(tc.dest, tc.book)
			if tc.catch {
				body = "Попытка\n" + body + "\nИсключение\nСообщить(ОписаниеОшибки());\nКонецПопытки;"
			}
			s := schedTestServer(t, "Сайты400", body, false)
			s.sched.SetVarsBuilder(s.BuildScheduledJobDSLVars)

			id, err := s.sched.RunNow(t.Context(), "Сайты400")
			require.NoError(t, err)
			var status, output, runError string
			require.Eventually(t, func() bool {
				run, err := s.sched.RunByID(t.Context(), id)
				if err != nil || run == nil || run.FinishedAt == nil {
					return false
				}
				status, output, runError = run.Status, run.Output, run.Error
				return true
			}, 5*time.Second, 20*time.Millisecond)
			require.Equal(t, tc.status, status, runError)
			require.Empty(t, s.tabularResults.workbooks)
			if status == "success" && !tc.catch {
				require.Equal(t, "обновлено", output)
				return
			}
			if tc.catch {
				require.Contains(t, output, "ЗаменитьЛистыExcelФоном")
			} else {
				require.Contains(t, runError, "ЗаменитьЛистыExcelФоном")
				require.Empty(t, output)
			}
			after, err := os.ReadFile(path)
			require.NoError(t, err)
			require.Equal(t, before, after)
			outsideAfter, err := os.ReadFile(outside)
			require.NoError(t, err)
			require.Equal(t, original, outsideAfter)
		})
	}

	file, err := excelize.OpenFile(path)
	require.NoError(t, err)
	defer func() { require.NoError(t, file.Close()) }()
	require.Equal(t, titles, file.GetSheetList())
	for i, title := range titles {
		value, err := file.GetCellValue(title, "A1")
		require.NoError(t, err)
		want := i
		if i > 0 && i < 6 {
			want += 40
		}
		require.Equal(t, fmt.Sprint(want), value, title)
	}
	archiveStyle, err := file.GetCellStyle("Архив", "A1")
	require.NoError(t, err)
	require.Equal(t, style, archiveStyle)

	s := schedTestServer(t, "Сайты400", makeBody(path, "Книга"), false)
	_, runErr, setupErr := s.RunProcessor(t.Context(), s.reg, "ФоноваяОбработка", nil, nil, nil)
	require.NoError(t, setupErr)
	require.ErrorContains(t, runErr, "ЗаменитьЛистыExcelФоном")
}
