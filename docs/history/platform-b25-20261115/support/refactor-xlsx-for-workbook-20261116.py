import hashlib
import json
import pathlib

path = pathlib.Path('C:/Users/a.sevrjukova/platform-workbook-xlsx-20261116/internal/sheet/xlsx.go')
text = path.read_text('utf-8')
before = hashlib.sha256(path.read_bytes()).hexdigest()
start = text.index('func (d *Document) XLSX() ([]byte, error) {')
end = text.index('// xlsxView keeps grouping', start)
original = text[start:end]
body_start = original.index('\tmaxRow, maxCol := d.ContentBounds()')
body_end = original.index('\tbuf, err := f.WriteToBuffer()')
initialization = original[:body_start]
body = original[body_start:body_end].replace('return nil, err', 'return nil, 0, err')
tail = original[body_end:]
replacement = initialization + '''\tcollapsed, outlineLevel, err := d.writeXLSXSheet(f, sheetName)
\tif err != nil {
\t\treturn nil, err
\t}
''' + tail + '''// writeXLSXSheet is shared by single-document and named-workbook exports.
// It never creates, copies or serializes another workbook.
func (d *Document) writeXLSXSheet(f *excelize.File, sheetName string) (map[int]bool, int, error) {
''' + body + '\treturn collapsed, outlineLevel, nil\n}\n\n'
text = text[:start] + replacement + text[end:]
old = 'func xlsxOutlineXML(data []byte, collapsed map[int]bool, level int) ([]byte, error) {'
assert text.count(old) == 1
text = text.replace(old, '''type xlsxSheetOutline struct {
\tcollapsed map[int]bool
\tlevel int
}

func xlsxOutlineXML(data []byte, collapsed map[int]bool, level int) ([]byte, error) {
\treturn xlsxWorkbookOutlineXML(data, map[string]xlsxSheetOutline{
\t\t"xl/worksheets/sheet1.xml": {collapsed: collapsed, level: level},
\t})
}

// This only completes outline metadata in the one file generated above; it
// does not combine independently generated XLSX/ZIP documents.
func xlsxWorkbookOutlineXML(data []byte, outlines map[string]xlsxSheetOutline) ([]byte, error) {''')
old = '\t\tif entry.Name != "xl/worksheets/sheet1.xml" {'
assert text.count(old) == 1
text = text.replace(old, '\t\toutline, apply := outlines[entry.Name]\n\t\tif !apply {')
assert text.count('if !collapsed[row] {') == 1
text = text.replace('if !collapsed[row] {', 'if !outline.collapsed[row] {')
old = 'fmt.Sprintf(`<sheetFormatPr outlineLevelRow="%d"`, level)'
assert text.count(old) == 1
text = text.replace(old, 'fmt.Sprintf(`<sheetFormatPr outlineLevelRow="%d"`, outline.level)')
path.write_text(text, encoding='utf-8')
out = pathlib.Path(__file__).with_suffix('.json')
out.write_text(json.dumps({'path': str(path), 'before': before, 'after': hashlib.sha256(path.read_bytes()).hexdigest()}, indent=2) + '\n', encoding='utf-8')
print('shared XLSX sheet writer extracted; legacy entry point retained')
