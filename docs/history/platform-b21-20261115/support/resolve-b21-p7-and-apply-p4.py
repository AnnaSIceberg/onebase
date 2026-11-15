import pathlib,re,subprocess,json
B=pathlib.Path(r'C:/Users/a.sevrjukova/AppData/Local/Temp/platform-help-admin-20261111');W=B/'reports-p4-p7-b21-runtime'
pat=re.compile(r'^<<<<<<< ours\n(.*?)^=======\n(.*?)^>>>>>>> theirs\n',re.M|re.S)
for n in ['CHANGELOG.md','docs/features.md','internal/api/v2.go','internal/cli/schema.go','internal/configcheck/lint.go','internal/launcher/configurator_report_module.go','internal/ui/handlers_reports.go']:
 p=W/n;s=p.read_text(encoding='utf-8');count=0
 def resolve(m):
  global count
  count+=1;a,b=m.group(1),m.group(2)
  if n in ['CHANGELOG.md','docs/features.md']:return a+'\n'+b
  if n=='internal/api/v2.go':return b[:b.index('\t\tcompiled, err :=')]+a
  if n=='internal/cli/schema.go':return a+'\t\t\t"context": reportContextSchema(),\n'
  if n=='internal/configcheck/lint.go':return a.replace('"labels": freeMap()', '"labels": freeMap(), "context": with(obj("user"), map[string]*yamlLintSchema{"user_card": obj("catalog", "account_field", "field", "table_part", "column")})')
  if n=='internal/launcher/configurator_report_module.go':
   return a+ ('\t\tContext *report.ParamContext `yaml:"context,omitempty"`\n' if 'Name ' in a else b)
  if n=='internal/ui/handlers_reports.go':return b[:b.index('\tcompiled, err :=')]+a
  raise AssertionError(n)
 s=pat.sub(resolve,s);assert count>0 and '<<<<<<<' not in s;p.write_text(s,encoding='utf-8')
# P7 binder must precede inherited P1 queryless result_proc branch.
s=(W/'internal/ui/handlers_reports.go').read_text(encoding='utf-8');print(json.dumps({'bind_at':s.index('ResolveContextParams(opCtx'), 'queryless_at':s.find('runTabular'), 'remaining_conflicts':False}))
p=subprocess.run(['git','add','-A'],cwd=W,creationflags=subprocess.CREATE_NO_WINDOW);assert p.returncode==0
p=subprocess.run(['git','-c','gc.auto=0','commit','-m','feat(report): перенести доверенный контекст П7 на B20','-m','Source: platform2 Draft #2629, 4379007a..02c62ecd322eb423b78dd0ebe66d999e384db5dd. Preserve B20 choice_show_deleted and P2 variants; bind before result_proc. Runtime gates pending.\n\nGenerated-with: Codex'],cwd=W,capture_output=True,creationflags=subprocess.CREATE_NO_WINDOW);assert p.returncode==0,p.stderr;print(p.stdout.decode(errors='replace'))
p=subprocess.run(['git','apply','--3way',str(B/'b21-p4-source.patch')],cwd=W,capture_output=True,creationflags=subprocess.CREATE_NO_WINDOW);(B/'b21-p4-apply.log').write_bytes(p.stdout+p.stderr);print(json.dumps({'p4_apply_exit':p.returncode,'output':(p.stdout+p.stderr).decode(errors='replace')}))
