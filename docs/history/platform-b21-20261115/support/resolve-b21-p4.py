import pathlib,re,json
W=pathlib.Path(r'C:/Users/a.sevrjukova/AppData/Local/Temp/platform-help-admin-20261111/reports-p4-p7-b21-runtime')
pat=re.compile(r'^<<<<<<< ours\n(.*?)^=======\n(.*?)^>>>>>>> theirs\n',re.M|re.S)
for n in ['CHANGELOG.md','docs/features.md','internal/cli/schema.go','internal/configcheck/full.go','internal/report/report.go','internal/ui/handlers_reports.go','internal/ui/report_compose_render.go','internal/ui/server.go']:
 p=W/n;s=p.read_text(encoding='utf-8');count=0
 def resolve(m):
  global count
  count+=1;a,b=m.group(1),m.group(2)
  if n in ['CHANGELOG.md','docs/features.md','internal/configcheck/full.go','internal/ui/server.go']:return a+'\n'+b
  if n=='internal/cli/schema.go':return a.replace('"properties": map[string]any{','"properties": map[string]any{"drilldowns": reportDrilldownSchema(),',1)
  if n=='internal/report/report.go':return a+'\tDrilldowns []Drilldown `yaml:"drilldowns,omitempty"` // trusted root metadata, shared by variants\n'
  if n=='internal/ui/handlers_reports.go':return a.replace('skipCol != "" && strings.EqualFold(col, skipCol)','reportColumnSkipped(col, skipCols)').replace('if str, ok := v.(string); ok {','if str, ok := v.(string); ok && len(skipCols)>0 && strings.EqualFold(col,skipCols[0]) {',1)
  if n=='internal/ui/report_compose_render.go':
   if 'func (h ' in a:return a.replace('func (h *htmlComposeSink) measureOnlyCells(vals map[string]any, styles map[string]report.CellStyle) {','func (h *htmlComposeSink) measureOnlyCells(vals map[string]any, styles map[string]report.CellStyle, detail ...bool) {')
   return a.replace('h.measureOnlyCells(d.Values, d.Styles)','h.measureOnlyCells(d.Values, d.Styles, true)')+'\tif h.drilldowns.HasRows() { h.b.WriteString(`<td>` + string(h.drilldowns.Row(d.Values)) + `</td>`) }\n'
  raise AssertionError(n)
 s=pat.sub(resolve,s);assert count>0 and '<<<<<<<' not in s;p.write_text(s,encoding='utf-8')
p=W/'internal/ui/report_compose_render.go';s=p.read_text(encoding='utf-8');old='html.EscapeString(v) + `</td>`)';assert s.count(old)==1;s=s.replace(old,'string(h.drilldowns.Cell(d.Field, vals, v)) + `</td>`)');p.write_text(s,encoding='utf-8')
p=W/'internal/ui/handlers_reports.go';s=p.read_text(encoding='utf-8');old='if reportColumnSkipped(col, skipCols) {\n\t\t\t\tcontinue';assert s.count(old)==1;s=s.replace(old,'if reportColumnSkipped(col, skipCols) && !(len(skipCols)>0 && strings.EqualFold(col,skipCols[0])) {\n\t\t\t\tcontinue');p.write_text(s,encoding='utf-8')
print('P4 conflicts resolved; B20 detail-field labels retained; P4 detail_cells renderer adapted')
