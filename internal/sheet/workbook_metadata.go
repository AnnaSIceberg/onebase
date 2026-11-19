package sheet

import (
	"archive/zip"
	"bytes"
	"encoding/xml"
	"fmt"
	"io"
	"regexp"
	"slices"
	"strconv"
	"strings"

	"github.com/xuri/excelize/v2"
)

const workbookPart = "xl/workbook.xml"
const xlsxOfficeRelationships = "http://schemas.openxmlformats.org/officeDocument/2006/relationships"

var xlsxTagAttribute = regexp.MustCompile(`([[:space:]]+)([^[:space:]="'<>/]+)[[:space:]]*=[[:space:]]*("[^"]*"|'[^']*')`)

type workbookXMLSpan struct {
	start, openEnd, closeStart, end int
	tag                             string
	attr                            []xml.Attr
}

type workbookEdit struct {
	start, end  int
	replacement []byte
}

// Preserve the source workbook envelope and its namespace declarations. The
// library's workbook writer only models part of the extension and calc schema.
func preserveWorkbookMetadata(original []byte, f *excelize.File, names []string, replacement *Workbook) ([]byte, error) {
	root, parts, defined, err := readWorkbookSpans(original)
	if err != nil {
		return nil, fmt.Errorf("повреждён XML книги XLSX: %w", err)
	}
	edits := make([]workbookEdit, 0, len(defined)+4)

	sheets, ok := parts["sheets"]
	if !ok {
		return nil, fmt.Errorf("в исходной книге отсутствует список листов")
	}
	if len(f.WorkBook.Sheets.Sheet) > len(names) {
		var add bytes.Buffer
		tag := qualifyWorkbookTag(sheets.tag, "sheet")
		for _, sh := range f.WorkBook.Sheets.Sheet[len(names):] {
			fmt.Fprintf(&add, "<%s name=\"%s\" sheetId=\"%d\" xmlns:r=\"%s\" r:id=\"%s\"/>",
				tag, escapeWorkbookXML(sh.Name), sh.SheetID, xlsxOfficeRelationships, escapeWorkbookXML(sh.ID))
		}
		edits = append(edits, workbookEdit{sheets.closeStart, sheets.closeStart, add.Bytes()})
	}

	// Changed sheets may have a new print title or lose a previous print area.
	// Preserve the original XML of all unrelated defined names.
	replaced := make(map[string]bool, len(replacement.sheets))
	for _, sh := range replacement.sheets {
		replaced[sh.name] = true
	}
	nameGroup, hasNames := parts["definedNames"]
	for _, entry := range defined {
		sourceIndex, indexErr := strconv.Atoi(workbookAttr(entry.attr, "localSheetId"))
		if indexErr == nil && sourceIndex >= 0 && sourceIndex < len(names) &&
			replaced[names[sourceIndex]] && worksheetBuiltinName(workbookAttr(entry.attr, "name")) {
			edits = append(edits, workbookEdit{entry.start, entry.end, nil})
		}
	}

	calc, hasCalc := parts["calcPr"]
	if hasCalc {
		edits = append(edits, workbookEdit{calc.start, calc.openEnd, forceWorkbookCalculation(original[calc.start:calc.openEnd])})
	} else {
		at := workbookSuccessor(parts, root.closeStart)
		tag := qualifyWorkbookTag(root.tag, "calcPr")
		edits = append(edits, workbookEdit{at, at, fmt.Appendf(nil, "<%s fullCalcOnLoad=\"1\"/>", tag)})
	}

	var newNames bytes.Buffer
	parentTag := qualifyWorkbookTag(root.tag, "definedNames")
	if hasNames {
		parentTag = nameGroup.tag
	}
	nameTag := qualifyWorkbookTag(parentTag, "definedName")
	for _, entry := range f.GetDefinedName() {
		if !replaced[entry.Scope] || !worksheetBuiltinName(entry.Name) {
			continue
		}
		index, indexErr := f.GetSheetIndex(entry.Scope)
		if indexErr != nil || index < 0 {
			return nil, fmt.Errorf("не удалось определить лист для имени %q", entry.Name)
		}
		fmt.Fprintf(&newNames, "<%s name=\"%s\" localSheetId=\"%d\">%s</%s>",
			nameTag, escapeWorkbookXML(entry.Name), index, escapeWorkbookXML(entry.RefersTo), nameTag)
	}
	if newNames.Len() > 0 {
		switch {
		case !hasNames:
			at := workbookSuccessor(parts, root.closeStart)
			if hasCalc {
				at = calc.start
			}
			group := fmt.Appendf(nil, "<%s>%s</%s>", parentTag, newNames.Bytes(), parentTag)
			edits = append(edits, workbookEdit{at, at, group})
		case nameGroup.openEnd == nameGroup.closeStart && nameGroup.openEnd == nameGroup.end:
			// Expand an existing self-closing <definedNames/>.
			xml := original[nameGroup.start:nameGroup.openEnd]
			start := bytes.TrimSuffix(xml, []byte("/>"))
			group := fmt.Appendf(nil, "%s>%s</%s>", start, newNames.Bytes(), parentTag)
			edits = append(edits, workbookEdit{nameGroup.start, nameGroup.end, group})
		default:
			edits = append(edits, workbookEdit{nameGroup.closeStart, nameGroup.closeStart, newNames.Bytes()})
		}
	}
	slices.SortStableFunc(edits, func(a, b workbookEdit) int {
		if a.start != b.start {
			return b.start - a.start
		}
		return b.end - a.end
	})
	var result bytes.Buffer
	pos := 0
	for i := len(edits) - 1; i >= 0; i-- {
		edit := edits[i]
		if edit.start < pos {
			return nil, fmt.Errorf("пересекающиеся изменения XML книги XLSX")
		}
		result.Write(original[pos:edit.start])
		result.Write(edit.replacement)
		pos = edit.end
	}
	result.Write(original[pos:])
	return result.Bytes(), nil
}

func readWorkbookSpans(raw []byte) (root workbookXMLSpan, parts map[string]workbookXMLSpan, names []workbookXMLSpan, err error) {
	parts = make(map[string]workbookXMLSpan)
	decoder := xml.NewDecoder(bytes.NewReader(raw))
	type element struct {
		local string
		main  bool
		span  workbookXMLSpan
	}
	var stack []element
	var mainSpace string
	for {
		at := int(decoder.InputOffset())
		token, tokenErr := decoder.Token()
		if tokenErr == io.EOF {
			break
		}
		if tokenErr != nil {
			return root, nil, nil, tokenErr
		}
		end := int(decoder.InputOffset())
		switch t := token.(type) {
		case xml.StartElement:
			if len(stack) == 0 {
				if t.Name.Local != "workbook" || root.end != 0 {
					return root, nil, nil, fmt.Errorf("ожидался элемент workbook")
				}
				mainSpace = t.Name.Space
			}
			tag := raw[at+1 : end]
			stop := bytes.IndexAny(tag, " \t\r\n/>")
			stack = append(stack, element{t.Name.Local, t.Name.Space == mainSpace, workbookXMLSpan{
				start: at, openEnd: end, tag: string(tag[:stop]), attr: t.Attr,
			}})
		case xml.EndElement:
			item := stack[len(stack)-1]
			stack = stack[:len(stack)-1]
			item.span.closeStart, item.span.end = at, end
			if !item.main {
				continue
			}
			switch len(stack) {
			case 0:
				root = item.span
			case 1:
				parts[item.local] = item.span
			case 2:
				if item.local == "definedName" && stack[1].main && stack[1].local == "definedNames" {
					names = append(names, item.span)
				}
			}
		}
	}
	if root.end == 0 {
		return root, nil, nil, fmt.Errorf("не найден элемент workbook")
	}
	return root, parts, names, nil
}

func workbookSuccessor(parts map[string]workbookXMLSpan, end int) int {
	for _, name := range []string{"oleSize", "customWorkbookViews", "pivotCaches", "smartTagPr", "smartTagTypes", "webPublishing", "fileRecoveryPr", "webPublishObjects", "extLst"} {
		if part, ok := parts[name]; ok && part.start < end {
			end = part.start
		}
	}
	return end
}

func workbookAttr(attrs []xml.Attr, name string) string {
	for _, attr := range attrs {
		if attr.Name.Space == "" && attr.Name.Local == name {
			return attr.Value
		}
	}
	return ""
}

func worksheetBuiltinName(name string) bool {
	return name == "_xlnm.Print_Titles" || name == "_xlnm.Print_Area" || name == "_xlnm._FilterDatabase"
}

func qualifyWorkbookTag(existing, local string) string {
	if prefix, _, ok := strings.Cut(existing, ":"); ok {
		return prefix + ":" + local
	}
	return local
}

func escapeWorkbookXML(value string) string {
	var b strings.Builder
	_ = xml.EscapeText(&b, []byte(value))
	return b.String()
}

func forceWorkbookCalculation(tag []byte) []byte {
	for _, attr := range xlsxTagAttribute.FindAllSubmatchIndex(tag, -1) {
		if string(tag[attr[4]:attr[5]]) == "fullCalcOnLoad" {
			return slices.Replace(slices.Clone(tag), attr[6]+1, attr[7]-1, '1')
		}
	}
	at := len(tag) - 1
	if tag[at-1] == '/' {
		at--
	}
	return slices.Insert(slices.Clone(tag), at, []byte(" fullCalcOnLoad=\"1\"")...)
}

func restoreWorkbookPart(generated, metadata []byte) ([]byte, error) {
	z, err := zip.NewReader(bytes.NewReader(generated), int64(len(generated)))
	if err != nil {
		return nil, err
	}
	var output bytes.Buffer
	writer := zip.NewWriter(&output)
	found := false
	for _, part := range z.File {
		if part.Name != workbookPart {
			if err := writer.Copy(part); err != nil {
				return nil, err
			}
			continue
		}
		header := part.FileHeader
		dst, err := writer.CreateHeader(&header)
		if err != nil {
			return nil, err
		}
		if _, err = dst.Write(metadata); err != nil {
			return nil, err
		}
		found = true
	}
	if !found {
		return nil, fmt.Errorf("не удалось сохранить XML книги XLSX")
	}
	if err := writer.Close(); err != nil {
		return nil, err
	}
	return output.Bytes(), nil
}
