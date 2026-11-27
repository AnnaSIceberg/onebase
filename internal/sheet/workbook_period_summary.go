package sheet

import (
	"fmt"
	"slices"
	"strings"
	"unicode"

	"github.com/shopspring/decimal"
	"github.com/xuri/excelize/v2"
)

// PeriodSummary updates one period and its running total in an existing sheet.
// Columns 1 and 2 identify metrics, with period headers on row 1.
type PeriodSummary struct {
	Period, Total string
	GroupMarker   string
	Previous      []string
	Rows          []PeriodSummaryRow
}

// PeriodSummaryRow uses 1-based coordinates in the generated document.
// A sum has no numerator or denominator. A ratio uses two amount rows.
type PeriodSummaryRow struct {
	Row, Numerator, Denominator int
}

func (b *Workbook) AddPeriodSummary(name string, doc *Document, update PeriodSummary) error {
	keys := map[string]bool{periodKey(update.Period): true, periodKey(update.Total): true}
	if keys[""] || len(keys) != 2 || len(update.Previous) > 12 || len(update.Rows) == 0 || len(update.Rows) > MaxWorkbookCells/12 {
		return fmt.Errorf("некорректные параметры периодов листа %q", name)
	}
	for _, header := range update.Previous {
		key := periodKey(header)
		if key == "" || keys[key] {
			return fmt.Errorf("повтор периода на листе %q", name)
		}
		keys[key] = true
	}
	amounts := make(map[int]bool, len(update.Rows))
	for _, row := range update.Rows {
		_, repeated := amounts[row.Row]
		if row.Row < 3 || row.Row > maxExcelRows || repeated || (row.Numerator == 0) != (row.Denominator == 0) {
			return fmt.Errorf("некорректная строка %d итогов листа %q", row.Row, name)
		}
		amounts[row.Row] = row.Numerator == 0
	}
	for _, row := range update.Rows {
		if row.Numerator != 0 && (!amounts[row.Numerator] || !amounts[row.Denominator]) {
			return fmt.Errorf("некорректные основания процента в строке %d листа %q", row.Row, name)
		}
	}
	if err := b.AddSheet(name, doc); err != nil {
		return err
	}
	update.Previous = append([]string(nil), update.Previous...)
	update.Rows = append([]PeriodSummaryRow(nil), update.Rows...)
	b.sheets[len(b.sheets)-1].summary = &update
	return nil
}

func periodKey(s string) string {
	return strings.Map(func(r rune) rune {
		if unicode.IsSpace(r) || r == '.' {
			return -1
		}
		return unicode.ToLower(r)
	}, s)
}

func summaryRowKey(left, right string) [2]string {
	normalize := func(s string) string {
		return strings.ReplaceAll(strings.ToLower(strings.Join(strings.Fields(s), " ")), "ё", "е")
	}
	return [2]string{normalize(left), normalize(right)}
}

func readSummaryCells(f *excelize.File, name string) ([][]string, error) {
	rows, err := f.Rows(name)
	if err != nil {
		return nil, err
	}
	defer rows.Close()
	var data [][]string
	width := 0
	for rows.Next() {
		values, err := rows.Columns(excelize.Options{RawCellValue: true})
		if err != nil {
			return nil, err
		}
		width = max(width, len(values))
		if int64(len(data)+1)*int64(max(width, 1)) > MaxWorkbookCells {
			return nil, fmt.Errorf("итоговый лист XLSX слишком большой")
		}
		data = append(data, values)
	}
	return data, rows.Error()
}

func summaryText(values []string, column int) string {
	if column < 1 || column > len(values) {
		return ""
	}
	return values[column-1]
}

func summaryCellText(doc *Document, row, col int) string {
	if cell := doc.GetCell(row-1, col-1); cell != nil {
		return cell.DisplayText()
	}
	return ""
}

func summaryHeaders(headers []string, use map[string]bool) (map[string]int, error) {
	columns := make(map[string]int, len(use))
	for idx, text := range headers {
		key := periodKey(text)
		if idx < 2 || !use[key] {
			continue
		}
		if columns[key] != 0 {
			return nil, fmt.Errorf("повтор заголовка периода %q", text)
		}
		columns[key] = idx + 1
	}
	return columns, nil
}

func writeSummaryNumber(f *excelize.File, name string, col, row int, value decimal.Decimal) error {
	ref, err := excelize.CoordinatesToCellName(col, row)
	if err != nil {
		return err
	}
	if value.IsZero() {
		return f.SetCellValue(name, ref, nil)
	}
	return f.SetCellDefault(name, ref, value.String())
}

func patchPeriodSummary(f *excelize.File, sh workbookSheet) error {
	p := sh.summary
	original, err := readSummaryCells(f, sh.name)
	if err != nil {
		return err
	}
	if len(original) < 2 || len(original[0]) < 3 {
		return fmt.Errorf("в исходной вкладке %q нет шапки итогов", sh.name)
	}
	period, total := periodKey(p.Period), periodKey(p.Total)
	known := map[string]bool{period: true, total: true}
	for _, header := range p.Previous {
		known[periodKey(header)] = true
	}
	oldCols, err := summaryHeaders(original[0], known)
	if err != nil {
		return err
	}
	_, lastSourceCol := sh.doc.ContentBounds()
	sourceHeaders := make([]string, lastSourceCol+1)
	for i := range sourceHeaders {
		sourceHeaders[i] = summaryCellText(sh.doc, 1, i+1)
	}
	newCols, err := summaryHeaders(sourceHeaders, known)
	if err != nil {
		return err
	}
	sourceCol := newCols[period]
	if sourceCol == 0 {
		return fmt.Errorf("нет обновляемого месяца %q", p.Period)
	}

	periodCol, totalCol := oldCols[period], oldCols[total]
	lastOriginalCol := 0
	for _, row := range original {
		lastOriginalCol = max(lastOriginalCol, len(row))
	}
	if periodCol == 0 {
		periodCol = lastOriginalCol + 1
		if totalCol != 0 {
			periodCol = totalCol
			col, err := excelize.ColumnNumberToName(totalCol)
			if err != nil {
				return err
			}
			if err := f.InsertCols(sh.name, col, 1); err != nil {
				return err
			}
			totalCol++
		}
		ref, err := excelize.CoordinatesToCellName(periodCol, 1)
		if err != nil {
			return err
		}
		if err := f.SetCellStr(sh.name, ref, p.Period); err != nil {
			return err
		}
	}
	if totalCol == 0 {
		totalCol = max(lastOriginalCol, periodCol) + 1
		ref, err := excelize.CoordinatesToCellName(totalCol, 1)
		if err != nil {
			return err
		}
		if err := f.SetCellStr(sh.name, ref, p.Total); err != nil {
			return err
		}
	}

	lastSourceRow, _ := sh.doc.ContentBounds()
	sourceKeys := summaryMetricKeys(lastSourceRow+1, p.GroupMarker, func(i int) (string, string) {
		return summaryCellText(sh.doc, i+1, 1), summaryCellText(sh.doc, i+1, 2)
	})
	existingKeys := summaryKeysForRows(original, p.GroupMarker)
	if p.GroupMarker != "" {
		original, err = insertNewSummaryGroups(f, sh, sourceCol, sourceKeys, original, existingKeys)
		if err != nil {
			return err
		}
		existingKeys = summaryKeysForRows(original, p.GroupMarker)
	}
	oldRows := make(map[summaryMetricKey][]int)
	for i, key := range existingKeys {
		if key != (summaryMetricKey{}) {
			oldRows[key] = append(oldRows[key], i+1)
		}
	}
	newRows := make(map[summaryMetricKey]int)
	for _, row := range p.Rows {
		if row.Row > len(sourceKeys) || sourceKeys[row.Row-1] == (summaryMetricKey{}) {
			return fmt.Errorf("у строки %d итогов нет подписи", row.Row)
		}
		newRows[sourceKeys[row.Row-1]]++
	}
	positions := make(map[summaryMetricKey]int)
	targets := make(map[int]int, len(p.Rows))
	sums := make(map[int]decimal.Decimal, len(p.Rows))
	last := len(original)
	for _, row := range p.Rows {
		key := sourceKeys[row.Row-1]
		pos := positions[key]
		positions[key]++
		old := oldRows[key]
		cell := sh.doc.GetCell(row.Row-1, sourceCol-1)
		text := ""
		if cell != nil {
			text = cell.DisplayText()
		}
		if len(old) != 0 && len(old) != newRows[key] {
			// An occurrence without a stable match must never overwrite a
			// different metric carrying the same visible label.
			continue
		}
		dst := 0
		if len(old) > 0 {
			dst = old[pos]
		} else if text != "" && key.Group == "" {
			last++
			dst = last
			for col := 1; col <= 2; col++ {
				ref, err := excelize.CoordinatesToCellName(col, dst)
				if err != nil {
					return err
				}
				if err := f.SetCellStr(sh.name, ref, summaryCellText(sh.doc, row.Row, col)); err != nil {
					return err
				}
			}
		}
		if dst == 0 {
			continue
		}
		targets[row.Row] = dst
		ref, err := excelize.CoordinatesToCellName(periodCol, dst)
		if err != nil {
			return err
		}
		if text == "" {
			err = f.SetCellValue(sh.name, ref, nil)
		} else {
			err = setXLSXValue(f, sh.name, ref, cell, false)
		}
		if err != nil {
			return err
		}
		if row.Numerator != 0 {
			continue
		}
		amount, err := summaryAmount(summaryCellValue(cell))
		if err != nil {
			return fmt.Errorf("месячное значение строки %d: %w", row.Row, err)
		}
		if len(old) > 0 {
			for _, header := range p.Previous {
				raw := summaryText(original[dst-1], oldCols[periodKey(header)])
				value, err := summaryAmount(raw)
				if err != nil {
					return fmt.Errorf("старое значение строки %d, %q: %w", dst, header, err)
				}
				amount = amount.Add(value)
			}
		}
		sums[row.Row] = amount
	}
	for _, row := range p.Rows {
		dst := targets[row.Row]
		if dst == 0 {
			continue
		}
		amount, ok := sums[row.Row]
		if row.Numerator != 0 {
			n, hasN := sums[row.Numerator]
			d, hasD := sums[row.Denominator]
			ok = hasN && hasD
			if ok && !d.IsZero() {
				amount = n.Mul(decimal.NewFromInt(100)).DivRound(d, 1)
			}
		}
		if ok {
			if err := writeSummaryNumber(f, sh.name, totalCol, dst, amount); err != nil {
				return err
			}
		}
	}
	return nil
}

func summaryAmount(value string) (decimal.Decimal, error) {
	if value == "" {
		return decimal.Zero, nil
	}
	value = strings.TrimSpace(value)
	if len(value) > 64 {
		return decimal.Zero, fmt.Errorf("слишком длинное число")
	}
	d, err := decimal.NewFromString(value)
	if err == nil && (d.Exponent() < -40 || d.Exponent() > 40) {
		err = fmt.Errorf("недопустимый порядок числа")
	}
	return d, err
}

func summaryCellValue(cell *Cell) string {
	if cell == nil {
		return ""
	}
	if cell.Value != nil {
		return fmt.Sprint(cell.Value)
	}
	return cell.Text
}

type summaryMetricKey struct {
	Label, Code, Group string
}

func summaryKeysForRows(rows [][]string, marker string) []summaryMetricKey {
	return summaryMetricKeys(len(rows), marker, func(i int) (string, string) {
		return summaryText(rows[i], 1), summaryText(rows[i], 2)
	})
}

// A new site belongs before the separator of the next section. Inserting a
// complete block keeps its identity on the next update, including when another
// section already contains the same site.
func insertNewSummaryGroups(f *excelize.File, sh workbookSheet, periodCol int, source []summaryMetricKey,
	original [][]string, existing []summaryMetricKey) ([][]string, error) {
	present := make(map[string]bool)
	sectionEnds := make(map[string]int)
	for i, key := range existing {
		if key.Group == "" {
			continue
		}
		present[key.Group] = true
		section, _, child := strings.Cut(key.Group, "\x00")
		if !child && sectionEnds[""] == 0 {
			// The preceding blank row separates the first channel from all sites.
			sectionEnds[""] = i
		}
		sectionEnds[section] = i + 2
	}
	needed := make(map[string]bool)
	for _, row := range sh.summary.Rows {
		if row.Row > len(source) {
			continue
		}
		key := source[row.Row-1]
		if strings.Contains(key.Group, "\x00") && !present[key.Group] &&
			summaryCellText(sh.doc, row.Row, periodCol) != "" {
			needed[key.Group] = true
		}
	}
	insertions := make(map[int][][]string)
	for _, row := range sh.summary.Rows {
		if row.Row > len(source) {
			continue
		}
		key := source[row.Row-1]
		if !needed[key.Group] {
			continue
		}
		section, _, _ := strings.Cut(key.Group, "\x00")
		next := sectionEnds[section]
		if next == 0 {
			if section != "" {
				return nil, fmt.Errorf("на листе %q отсутствует раздел %q", sh.name, section)
			}
			next = len(original) + 1
		}
		captions := []string{summaryCellText(sh.doc, row.Row, 1), summaryCellText(sh.doc, row.Row, 2)}
		insertions[next] = append(insertions[next], captions)
	}
	positions := make([]int, 0, len(insertions))
	for row := range insertions {
		positions = append(positions, row)
	}
	slices.Sort(positions)
	for i := len(positions) - 1; i >= 0; i-- {
		row := positions[i]
		captions := insertions[row]
		if row <= len(original) {
			if err := f.InsertRows(sh.name, row, len(captions)); err != nil {
				return nil, err
			}
		}
		for offset, cells := range captions {
			ref, err := excelize.CoordinatesToCellName(1, row+offset)
			if err != nil {
				return nil, err
			}
			if err := f.SetSheetRow(sh.name, ref, &cells); err != nil {
				return nil, err
			}
		}
		original = slices.Insert(original, row-1, captions...)
	}
	return original, nil
}

// A matching marker in column B starts a named metric block. A blank
// separator between blocks starts a parent group; following blocks are
// identified within that parent. This lets the same metric appear in more
// than one block without matching by an unstable absolute row number.
func summaryMetricKeys(count int, marker string, captions func(int) (string, string)) []summaryMetricKey {
	keys := make([]summaryMetricKey, count)
	var parent, group string
	var separated bool
	marker = summaryRowKey("", marker)[1]
	for i := 2; i < count; i++ {
		a, b := captions(i)
		key := summaryRowKey(a, b)
		if key == ([2]string{}) {
			separated = true
			continue
		}
		if marker != "" && key[1] == marker {
			if separated {
				parent = key[0]
				group = parent
			} else {
				group = parent + "\x00" + key[0]
			}
		}
		keys[i] = summaryMetricKey{Label: key[0], Code: key[1], Group: group}
		separated = false
	}
	return keys
}
