package interpreter

import (
	"context"
	"encoding/base64"
	"strings"

	"github.com/shopspring/decimal"

	"github.com/ivantit66/onebase/internal/sheet"
)

// SpreadsheetWorkbook is a server-side collection of named document snapshots.
// Write returns base64 data, like SpreadsheetDocument.Write, and never opens a path.
type SpreadsheetWorkbook struct {
	book *sheet.Workbook
}

func newSpreadsheetWorkbook(args []any) any {
	if len(args) != 0 {
		RaiseUserError("SpreadsheetWorkbook constructor takes no arguments")
	}
	return &SpreadsheetWorkbook{book: sheet.NewWorkbook()}
}

func (b *SpreadsheetWorkbook) XLSX() ([]byte, error) { return b.book.XLSX() }

func (b *SpreadsheetWorkbook) UpdateXLSXFile(ctx context.Context, dir, filename string) ([]byte, error) {
	return b.book.UpdateXLSXFile(ctx, dir, filename)
}

func (b *SpreadsheetWorkbook) KnownMethods() (string, []string) {
	return "SpreadsheetWorkbook", []string{"ДобавитьЛист", "AddSheet", "ДобавитьЛистДописывания", "AddPeriodSummary", "Количество", "Count", "Записать", "Write"}
}

func (b *SpreadsheetWorkbook) CallMethod(name string, args []any) any {
	switch strings.ToLower(name) {
	case "добавитьлист", "addsheet":
		if len(args) != 2 {
			RaiseUserError("SpreadsheetWorkbook.AddSheet requires a name and a tabular document")
		}
		name, ok := args[0].(string)
		if !ok {
			RaiseUserError("SpreadsheetWorkbook sheet name must be a string")
		}
		doc, ok := args[1].(*SpreadsheetDocument)
		if !ok || doc == nil {
			RaiseUserError("SpreadsheetWorkbook.AddSheet requires a tabular document")
		}
		if err := b.book.AddSheet(name, doc.Doc); err != nil {
			RaiseUserError(err.Error())
		}
		return nil
	case "добавитьлистдописывания", "addperiodsummary":
		return b.addPeriodSummary(args)
	case "количество", "count":
		if len(args) != 0 {
			RaiseUserError("SpreadsheetWorkbook.Count takes no arguments")
		}
		return float64(b.book.SheetCount())
	case "записать", "write":
		if len(args) != 1 && len(args) != 2 {
			RaiseUserError("SpreadsheetWorkbook.Write requires a filename and optional xlsx format")
		}
		if _, ok := args[0].(string); !ok {
			RaiseUserError("SpreadsheetWorkbook filename must be a string")
		}
		if len(args) == 2 && !strings.EqualFold(strArg(args, 1), "xlsx") {
			RaiseUserError("SpreadsheetWorkbook supports only the xlsx format")
		}
		data, err := b.XLSX()
		if err != nil {
			RaiseUserError(err.Error())
		}
		return base64.StdEncoding.EncodeToString(data)
	}
	RaiseUserError("Unknown SpreadsheetWorkbook method: " + name)
	return nil
}

func (b *SpreadsheetWorkbook) addPeriodSummary(args []any) any {
	if len(args) != 6 && len(args) != 7 {
		RaiseUserError("ДобавитьЛистДописывания: ожидаются имя, документ, месяц, итог, прежние месяцы и строки расчёта годового итога")
	}
	name, nameOK := args[0].(string)
	doc, docOK := args[1].(*SpreadsheetDocument)
	period, periodOK := args[2].(string)
	total, totalOK := args[3].(string)
	previous, previousOK := args[4].(*Array)
	rows, rowsOK := args[5].(*Array)
	if !nameOK || !docOK || doc == nil || !periodOK || !totalOK || !previousOK || !rowsOK {
		RaiseUserError("ДобавитьЛистДописывания: неверные аргументы")
	}
	if len(previous.items) > 12 || len(rows.items) > sheet.MaxWorkbookCells/12 {
		RaiseUserError("ДобавитьЛистДописывания: слишком много строк или периодов")
	}
	u := sheet.PeriodSummary{Period: period, Total: total}
	if len(args) == 7 {
		marker, ok := args[6].(string)
		if !ok {
			RaiseUserError("ДобавитьЛистДописывания: метка группы должна быть строкой")
		}
		u.GroupMarker = marker
	}
	for _, item := range previous.items {
		label, ok := item.(string)
		if !ok {
			RaiseUserError("ДобавитьЛистДописывания: заголовок месяца должен быть строкой")
		}
		u.Previous = append(u.Previous, label)
	}
	for _, item := range rows.items {
		row, ok := item.(*Array)
		if !ok || (len(row.items) != 1 && len(row.items) != 3) {
			RaiseUserError("ДобавитьЛистДописывания: строка расчёта содержит номер строки либо номер строки и две строки-основания")
		}
		var indices [3]int
		for i, cell := range row.items {
			index, ok := toDecimal(cell)
			if !ok || !decimalWithinSafeBounds(index) || index.LessThan(decimal.NewFromInt(3)) ||
				index.GreaterThan(decimal.NewFromInt(1_048_576)) || !index.IsInteger() {
				RaiseUserError("ДобавитьЛистДописывания: номер строки должен быть целым числом от 3 до 1048576")
			}
			indices[i] = int(index.IntPart())
		}
		u.Rows = append(u.Rows, sheet.PeriodSummaryRow{Row: indices[0], Numerator: indices[1], Denominator: indices[2]})
	}
	if err := b.book.AddPeriodSummary(name, doc.Doc, u); err != nil {
		RaiseUserError(err.Error())
	}
	return nil
}
