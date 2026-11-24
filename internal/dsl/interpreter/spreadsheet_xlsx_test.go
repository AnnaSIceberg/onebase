package interpreter_test

import (
	"archive/zip"
	"bytes"
	"encoding/base64"
	"io"
	"path/filepath"
	"testing"

	"github.com/ivantit66/onebase/internal/dsl/interpreter"
	"github.com/ivantit66/onebase/internal/printform"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
	"github.com/xuri/excelize/v2"
)

const tabDocMailSrc = `
  ТабДок = Новый ТабличныйДокумент;
  Шапка = ТабДок.Ячейка(1, 1);
  Шапка.Текст = "Заявки бригаде";
  Шапка.Жирный = Истина;
  Шапка.РазмерШрифта = 11;
  ТабДок.Объединить(1, 1, 1, 3);
  ТабДок.Ячейка(2, 1).Текст = "Пл8000641";
  Сумма = ТабДок.Ячейка(2, 2);
  Сумма.Значение = 1250.5;
  Сумма.Текст = "1 250,50";
  ТабДок.ШиринаКолонки(1, 145);
  Возврат ТабДок.Записать("заявки.xlsx", "xlsx");
`

// Base64 export does not depend on optional email-profile extensions.
func TestSpreadsheetXLSXBase64Export(t *testing.T) {
	value := runHTTPSrc(t, "Функция Тест()"+tabDocMailSrc+"КонецФункции", withSpreadsheet(map[string]any{}))
	data, err := base64.StdEncoding.DecodeString(value.(string))
	require.NoError(t, err)
	require.True(t, bytes.HasPrefix(data, []byte("PK")))
}

// Вложение — настоящая книга Excel с оформлением документа: шапка жирная,
// объединена, число — числом, ширина колонки сохранена.
func TestSpreadsheetXLSXAttachmentIsWorkbook(t *testing.T) {
	value := runHTTPSrc(t, "Функция Тест()"+tabDocMailSrc+"КонецФункции", withSpreadsheet(map[string]any{}))
	data, err := base64.StdEncoding.DecodeString(value.(string))
	require.NoError(t, err)
	f, err := excelize.OpenReader(bytes.NewReader(data))
	require.NoError(t, err, "вложение не открывается как xlsx")
	defer f.Close() //nolint:errcheck
	v, _ := f.GetCellValue("Лист1", "A1")
	assert.Equal(t, "Заявки бригаде", v)
	view, err := f.GetSheetView("Лист1", 0)
	require.NoError(t, err)
	require.NotNil(t, view.ZoomScale)
	require.Equal(t, 100.0, *view.ZoomScale)
	id, _ := f.GetCellStyle("Лист1", "A1")
	st, err := f.GetStyle(id)
	require.NoError(t, err)
	require.NotNil(t, st.Font)
	assert.True(t, st.Font.Bold)
	assert.Equal(t, 11.0, st.Font.Size)
	merges, _ := f.GetMergeCells("Лист1")
	require.Len(t, merges, 1)
	assert.Equal(t, "C1", merges[0].GetEndAxis())
	raw, _ := f.GetCellValue("Лист1", "B2", excelize.Options{RawCellValue: true})
	assert.Equal(t, "1250.5", raw, "сумма — числом, не текстом")
	w, _ := f.GetColWidth("Лист1", "A")
	assert.InDelta(t, 20, w, 1)
}

// withSpreadsheet добавляет фабрику Новый ТабличныйДокумент — как её
// подмешивает исполнение модулей (ui/handlers_dsl.go).
func withSpreadsheet(vars map[string]any) map[string]any {
	for k, v := range interpreter.NewSpreadsheetFunctions() {
		vars[k] = v
	}
	return vars
}

// Макет в духе «Партнеры» из А_ОтправкаНаПочту: шапка и строка Calibri 11,
// шапка жирная, рамки ячеек, ширины колонок.
const partnersLayout = `name: Партнеры
columns:
  - width: 90px
  - width: 250px
  - width: 100px
areas:
  Шапка:
    rows:
      - cells:
          - {text: "Номер", bold: true, fontFamily: Calibri, fontSize: 11, border: all, align: center}
          - {text: "Адрес", bold: true, fontFamily: Calibri, fontSize: 11, border: all, align: center}
          - {text: "Сумма", bold: true, fontFamily: Calibri, fontSize: 11, border: all, align: center}
  Строка:
    rows:
      - cells:
          - {parameter: Номер, fontFamily: Calibri, fontSize: 11, border: all}
          - {parameter: Адрес, fontFamily: Calibri, fontSize: 11, border: all}
          - {parameter: Сумма, fontFamily: Calibri, fontSize: 11, border: all, align: right}
`

// ТабДок.Вывести(Макет.ПолучитьОбласть(…)) → Записать(…, "xlsx") → вложение:
// оформление и ширины колонок макета доходят до книги, как XLS97 в 1С.
func TestSpreadsheetXLSXFromLayoutKeepsTemplate(t *testing.T) {
	lt, err := printform.ParseLayoutBytes([]byte(partnersLayout))
	require.NoError(t, err)
	sender := &profileSender{}
	vars := withSpreadsheet(interpreter.NewEmailFunctions(sender, nil))
	interpreter.InjectMaket(vars, lt)
	runHTTPSrc(t, `Процедура Тест()
  ТабДок = Новый ТабличныйДокумент;
  ТабДок.Вывести(Макет.ПолучитьОбласть("Шапка"));
  Строки = Новый Массив;
  Строки.Добавить(Новый Структура("Номер, Адрес, Сумма", "Пл8000641", "Москва, ул. Ленина, 1", 1250.5));
  Строки.Добавить(Новый Структура("Номер, Адрес, Сумма", "Пл8000642", "Химки, ул. Мира, 7", 980));
  Для Каждого Стр Из Строки Цикл
    Обл = Макет.ПолучитьОбласть("Строка");
    Обл.Параметры.Номер = Стр.Номер;
    Обл.Параметры.Адрес = Стр.Адрес;
    Обл.Параметры.Сумма = Стр.Сумма;
    ТабДок.Вывести(Обл);
  КонецЦикла;
  Письмо = Новый ПисьмоEmail;
  Письмо.Кому = "brigada@iceberg.test";
  Письмо.Тема = "Заявки на завтра";
  Письмо.ДобавитьВложение("заявки.xlsx", ТабДок.Записать("заявки.xlsx", "xlsx"), , Истина);
  Письмо.Отправить();
КонецПроцедуры`, vars)
	require.Len(t, sender.msg.Files, 1)
	f, err := excelize.OpenReader(bytes.NewReader(sender.msg.Files[0].Data))
	require.NoError(t, err)
	defer f.Close() //nolint:errcheck
	const sh = "Лист1"

	head, _ := f.GetCellStyle(sh, "B1")
	hs, err := f.GetStyle(head)
	require.NoError(t, err)
	assert.Equal(t, "Calibri", hs.Font.Family)
	assert.Equal(t, 11.0, hs.Font.Size)
	assert.True(t, hs.Font.Bold, "шапка жирная")
	assert.Len(t, hs.Border, 4, "рамки шапки со всех сторон")

	v, _ := f.GetCellValue(sh, "B3")
	assert.Equal(t, "Химки, ул. Мира, 7", v)
	raw, _ := f.GetCellValue(sh, "C2", excelize.Options{RawCellValue: true})
	assert.Equal(t, "1250.5", raw, "сумма — числом")
	row, _ := f.GetCellStyle(sh, "C3")
	rs, err := f.GetStyle(row)
	require.NoError(t, err)
	assert.False(t, rs.Font.Bold)
	assert.Equal(t, "right", rs.Alignment.Horizontal)
	assert.Len(t, rs.Border, 4, "рамки строки")

	for col, want := range map[string]float64{"A": (90 - 5) / 7.0, "B": (250 - 5) / 7.0, "C": (100 - 5) / 7.0} {
		w, _ := f.GetColWidth(sh, col)
		assert.InDelta(t, want, w, 0.5, "ширина колонки %s из макета", col)
	}
}

// Layout and runtime cell styles must survive both paths of the shared workbook writer.
func TestSpreadsheetXLSXStyleAndNestedOutlineOnReplace(t *testing.T) {
	layout, err := printform.ParseLayoutBytes([]byte(`name: Отчёт
areas:
  Строка:
    rows:
      - cells:
          - {text: "Строка", fontFamily: Calibri, fontSize: 11, border: thin, borderColor: "#B3AC86", wrap: false}
          - {text: "Обычный текст"}
`))
	require.NoError(t, err)
	vars := withSpreadsheet(map[string]any{})
	interpreter.InjectMaket(vars, layout)
	value := runHTTPSrc(t, `Функция Тест()
    ТабДок = Новый ТабличныйДокумент;
    ТабДок.ФиксацияСверху = 3;
    ТабДок.ФиксацияСлева = 2;
    ТабДок.МасштабПросмотра = 90;
    Обл = Макет.Область("Строка");
    ТабДок.Вывести(Обл);
    ТабДок.Вывести(Обл);
    ТабДок.Вывести(Обл);
    ТабДок.Вывести(Обл, 1, "Итог", Ложь);
    ТабДок.Вывести(Обл, 2, "Раздел", Истина);
    ТабДок.Вывести(Обл);
    ТабДок.ЗакончитьАвтогруппировкуСтрок(2);
    ТабДок.Вывести(Обл);
    ТабДок.ЗакончитьАвтогруппировкуСтрок();
    ТабДок.Вывести(Обл);
    Яч = ТабДок.Ячейка(8, 3);
    Яч.Текст = "Без рамки";
    Яч.ИмяШрифта = "Calibri";
    Яч.Граница = "none";
    Яч.Переносить = Ложь;
    Книга = Новый КнигаТабличныхДокументов;
    Книга.ДобавитьЛист("Данные", ТабДок);
    Возврат Книга;
КонецФункции`, vars)
	book := value.(*interpreter.SpreadsheetWorkbook)

	fresh, err := book.XLSX()
	require.NoError(t, err)
	dir := t.TempDir()
	filename := filepath.Join(dir, "архив.xlsx")
	old := excelize.NewFile()
	require.NoError(t, old.SetSheetName("Sheet1", "Архив"))
	_, err = old.NewSheet("Данные")
	require.NoError(t, err)
	require.NoError(t, old.SetCellStr("Архив", "A1", "2010"))
	require.NoError(t, old.SaveAs(filename))
	require.NoError(t, old.Close())
	replaced, err := book.UpdateXLSXFile(t.Context(), dir, filename)
	require.NoError(t, err)

	for _, tc := range []struct {
		name, worksheet string
		data            []byte
	}{
		{"new", "xl/worksheets/sheet1.xml", fresh},
		{"replace", "xl/worksheets/sheet2.xml", replaced},
	} {
		t.Run(tc.name, func(t *testing.T) {
			const sh = "Данные"
			f, err := excelize.OpenReader(bytes.NewReader(tc.data))
			require.NoError(t, err)
			t.Cleanup(func() { require.NoError(t, f.Close()) })
			panes, err := f.GetPanes(sh)
			require.NoError(t, err)
			require.Equal(t, "C4", panes.TopLeftCell)
			view, err := f.GetSheetView(sh, 0)
			require.NoError(t, err)
			require.NotNil(t, view.ZoomScale)
			require.Equal(t, 90.0, *view.ZoomScale)
			for i, want := range []uint8{0, 0, 0, 0, 1, 2, 1, 0} {
				level, err := f.GetRowOutlineLevel(sh, i+1)
				require.NoError(t, err)
				require.Equal(t, want, level, "row %d", i+1)
				visible, err := f.GetRowVisible(sh, i+1)
				require.NoError(t, err)
				require.Equal(t, want == 0, visible, "row %d", i+1)
			}
			style := func(ref string) *excelize.Style {
				id, err := f.GetCellStyle(sh, ref)
				require.NoError(t, err)
				st, err := f.GetStyle(id)
				require.NoError(t, err)
				return st
			}
			st := style("A1")
			require.Equal(t, "Calibri", st.Font.Family)
			require.Equal(t, 11.0, st.Font.Size)
			require.False(t, st.Alignment.WrapText)
			require.Len(t, st.Border, 4)
			for _, edge := range st.Border {
				require.Equal(t, 1, edge.Style)
				require.Equal(t, "B3AC86", edge.Color)
			}
			require.True(t, style("B1").Alignment.WrapText)
			require.Equal(t, "Calibri", style("C8").Font.Family)
			require.Empty(t, style("C8").Border)
			require.False(t, style("C8").Alignment.WrapText)

			z, err := zip.NewReader(bytes.NewReader(tc.data), int64(len(tc.data)))
			require.NoError(t, err)
			part, err := z.Open(tc.worksheet)
			require.NoError(t, err)
			data, err := io.ReadAll(part)
			require.NoError(t, err)
			require.NoError(t, part.Close())
			require.Regexp(t, `<row\b[^>]*(?:\br="4"[^>]*\bcollapsed="1"|\bcollapsed="1"[^>]*\br="4")`, string(data))
			require.Contains(t, string(data), `outlineLevelRow="2"`)
			if tc.name == "replace" {
				archived, err := f.GetCellValue("Архив", "A1")
				require.NoError(t, err)
				require.Equal(t, "2010", archived)
			}
		})
	}
}
