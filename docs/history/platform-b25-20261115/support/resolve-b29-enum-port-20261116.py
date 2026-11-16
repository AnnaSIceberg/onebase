import pathlib,re,subprocess
W=pathlib.Path(__file__).parent/'report-b29-enum-runtime'
def git(*args):return subprocess.check_output(['git',*args],cwd=W,creationflags=subprocess.CREATE_NO_WINDOW).decode('utf-8')
source='2e185f0615f98111cd1353d71afcc72fdabbd5fd'
for name,heading,end in [('CHANGELOG.md','# История изменений\n\n','\nЗдесь видно'),('docs/features.md','# Возможности OneBase\n\n','\n## Страница приложения')]:
 ours=git('show',':2:'+name);theirs=git('show',source+':'+name)
 assert ours.startswith(heading) and theirs.startswith(heading)
 block=theirs[len(heading):].split(end,1)[0].rstrip()+'\n\n'
 (W/name).write_text(heading+block+ours[len(heading):],encoding='utf-8')
pattern=r'^<<<<<<< HEAD\n(.*?)^=======\n(.*?)^>>>>>>> 2e185f061 \(fix\(report\): render and validate enum parameter choices\)\n'
for name,expected in [('internal/api/v2.go',1),('internal/ui/handlers_reports.go',2)]:
 p=W/name;s=p.read_text('utf-8');s,n=re.subn(pattern,lambda m:m[1],s,flags=re.M|re.S);assert n==expected and '<<<<<<<'not in s
 if name.endswith('v2.go'):
  before='\t\tsourceSettings, err := reportpkg.ParseSourceSettings(r.URL.Query().Get("__settings"))'
  assert s.count(before)==1
  block='\t\tif err := reportpkg.ValidateEnumParams(rep.Params, params, h.reg); err != nil {\n\t\t\twriteError(w, http.StatusBadRequest, err.Error(), "", 0)\n\t\t\treturn\n\t\t}\n'
  s=s.replace(before,block+before)
 else:
  anchor='\tIsSel      bool\n';assert s.count(anchor)==1;s=s.replace(anchor,anchor+'\tIsEnum bool\n\tEnumOptions []EnumOption\n')
  block='\tif err := reportpkg.ValidateEnumParams(rep.Params, paramValues, s.reg); err != nil {\n\t\thttp.Error(w, s.errText(r, err), http.StatusBadRequest)\n\t\treturn\n\t}\n'
  assert s.count(block)==1;s=s.replace(block,'',1)
  anchor='\n\t// Выбранный вариант компоновки (параметр __variant); пусто → основной.'
  assert s.count(anchor)==1
  s=s.replace(anchor,'\n'+block.replace('\t\thttp.Error','\t\topStatus = "error"\n\t\thttp.Error')+anchor)
  start=s.index('func (s *Server) reportExportRowsWithContext(');a,b=s[:start],s[start:]
  anchor='\tvariant := r.FormValue("__variant")';assert b.count(anchor)==1
  block='\tif err := reportpkg.ValidateEnumParams(rep.Params, paramValues, s.reg); err != nil {\n\t\treturn nil, nil, newReportExportError(http.StatusBadRequest, "invalid report parameter", err)\n\t}\n'
  b=b.replace(anchor,block+anchor);s=a+b
 p.write_text(s,encoding='utf-8')
print('B29 conflicts resolved: keep B28 variants/context/choice flags; validate enums after trusted context')
