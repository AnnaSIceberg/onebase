import pathlib,json,hashlib,subprocess,collections,datetime
B=pathlib.Path(r'C:/Users/a.sevrjukova/AppData/Local/Temp/platform-help-admin-20261111');W=B/'reports-after-release-b22-runtime';R=pathlib.Path(r'C:/Users/a.sevrjukova/AppData/Local/Temp/claude/C--Users-a-sevrjukova-onebase/cd50d987-9739-4c93-a5be-68d748529225/scratchpad/отчёты-для-analysis');H='1496c60ccad535d722bb1d83d2694d7d81bfbded';BASE='0d77e91557b12bada4668fd7bfa5f1b13a37de08'
def sha(p):
 with pathlib.Path(p).open('rb')as f:return hashlib.file_digest(f,'sha256').hexdigest()
def load(p):return json.loads(pathlib.Path(p).read_text('utf-8-sig'))
def git(*a):return subprocess.check_output(['git',*a],cwd=W,creationflags=subprocess.CREATE_NO_WINDOW).decode('utf-8').strip()
assert git('rev-parse','HEAD')==H and not git('status','--porcelain')
end=B/'b22-go-end-processes.json';assert load(end)['go_process_count']==0
art={k:load(B/('b22-'+k+'-artifact.json'))for k in ['ui_pg','interpreter_pg','linux','windows']}
for a in art.values():
 assert a['runtime']==a['source']==H and sha(a['path'])==a['sha256'];info=pathlib.Path(a['buildinfo']).read_text('utf-8');assert 'go1.26.11'in info and 'CGO_ENABLED=0'in info;a['embedded_vcs_revision_present']='vcs.revision'in info;a['revision_binding']='clean combined runtime/source tree + command receipts + SHA; do not invent .test embeddedVCS'
s=load(B/'b22-local-targeted-summary.json');count=collections.Counter()
for r in s['runs']:count.update(r['counts'])
assert count=={'pass':480,'skip':17}
expected={}
for k,num in [('ui_pg',190),('interpreter_pg',6)]:
 p=B/('b22-'+k+'-expected.json');e=load(p);assert len(set(e['expected_subtests']))==e['expected_total']==num;expected[k]={'path':str(p),'sha256':sha(p),**e}
changed=git('diff','--name-only',BASE,H).splitlines();assert len(changed)==9
protected=['internal/sheet','internal/printform','internal/query','internal/access','internal/report','internal/launcher','internal/ui/tabular_result.go','internal/ui/report_context_test.go','internal/ui/report_drilldown.go','internal/ui/handlers_events.go','internal/ui/handlers_entity.go','internal/api']
for p in protected:assert not git('diff','--name-only',BASE,H,'--',p),p
files=[]
for name in changed:
 raw=subprocess.check_output(['git','show',H+':'+name],cwd=W,creationflags=subprocess.CREATE_NO_WINDOW);files.append({'path':name,'sha256_git_blob':hashlib.sha256(raw).hexdigest(),'bytes':len(raw)})
frozen=[('manifest-1115b21-report-context-drilldown-DEV-final.json','6f4f74ea7ffd5ef4511a9dc1050b5e8e816f3e8c059ed64c2cb7333f354ca70e'),('onebase-1115b21-report-context-drilldown-linux-amd64','881d78ad9d6d37916350b6bca099a8e7b9208f14a35b8f742f7e3776c92f300a'),('onebase-1115b21-report-context-drilldown-windows-amd64.exe','e9bbbce60ce5b67e1867c0087eda77897b8e6ec540638bd8164814127bf6eb06')]
for n,h in frozen:assert sha(B/n)==h
red=B/'b22-red-b21-overlay/receipt.json';rr=load(red);assert rr['exit']==1 and len(rr['failed_leaves'])==14
m={'status':'B22_AFTER_RELEASE_DEV_LOCAL_BUILD_GREEN_PG_NATIVE_PENDING','runtime':H,'source':H,'source_binding':'combined B22 integration tree; upstream source heads separate','base_b21':BASE,'branch':'fix/reports-after-release-b22-20261115','pushed':True,'upstream_inputs':art['linux']['upstream_inputs'],'drafts':{'acl':2631,'ordinary_xlsx':2632,'status':'Draft, no CI/Ready claim'},'artifacts':art,'fixes':['typed native source ACL denial stays in queryProxy userError/DSLError cause chain; result_proc403 via errors.Is, genuine module/query/SQL errors500','ordinary list/report buffered and stream XLSX dimension from headers and actual value/style cells; TableDocument writer unchanged'],'tests':{'local_leaf_pass':480,'local_pg_root_skip':17,'skip_not_pg_pass':True,'summary':str(B/'b22-local-targeted-summary.json'),'summary_sha256':sha(B/'b22-local-targeted-summary.json'),'build_all':{'exit':0,'log':str(B/'b22-build-all.log'),'sha256':sha(B/'b22-build-all.log')},'i18n_en':{'exit':0,'log':str(B/'b22-i18n-en.log'),'sha256':sha(B/'b22-i18n-en.log')},'red_old_b21':{'receipt':str(red),'sha256':sha(red),'expected_fail_leaf':14,'frozen_sdk_executed':False,'scope':'Go overlay exactB21 product + new test-source43c00427; old assert parentT stopped ordinary background after direct fail','test_adapter':'final1496 changes assertion helper to current subtest T, same assertions, no product delta'}},'expected':expected,'independent_pg_status':'PENDING exact UI190+interpreter6; no source/baseline receipts substituted','native_status':'PENDING separate root backendcleanup/Chrome grant; prepared helper SOURCE ONLY','native_helper':{'path':str(B/'b22-native-afterrelease.py'),'sha256':sha(B/'b22-native-afterrelease.py'),'not_imported_or_run':True},'protected_paths_byte_equal_b21':protected,'changed_files':files,'frozen_b21_verified':[{'path':str(B/n),'sha256':h}for n,h in frozen],'go_end':{'explicit':True,'path':str(end),'sha256':sha(end),'go_process_count':0},'scope':{'current_evening_b21_untouched':True,'shared_changes':False,'db_copies':0,'new_Chrome_started':False,'rollout':'shadow LPC/:8444 only after evening release by cc2 when ready; production next package only by userword','checkpoint':'17:45 fresh push/process check remains required, not completed early'}}
out=B/'manifest-1115b22-report-acl-list-xlsx-DEV.json';assert not out.exists();out.write_text(json.dumps(m,ensure_ascii=False,indent=2)+'\n',encoding='utf-8');out.with_suffix('.json.sha256').write_text(sha(out)+'  '+out.name+'\n',encoding='utf-8')
f=R/'20261115-platform-B22-after-release-build-handoff.md';assert not f.exists();f.write_text(f'''# B22 — exactB21 + ACL result_proc403 / ordinary XLSXdimension

Combinedruntime/source tree {H}, exactB21parent{BASE}; sourceACL a00d3238/Draft2631 иordinarydb1b0587/Draft2632. Отдельная fork ветка clean/pushed. Всего9files vsB21:2reviewed fixes+tests/changelog; последнийtest-onlyadapter текущего testing.T не меняет assertions/product. B21SDK/manifest6f4f SHA checked unchanged, вечерняя линия не затронута.

Build./.../i18nen PASS. Targeted Query/TxGuard/factory/unload/exception/rethrow/cause иExcel+HTTP/наследованныеB21:480localleafPASS/17localPGrootSKIP отдельно. Independent exact UI190 (95SQLite+95PG) +interpreter6 (3+3) пока PENDING. СтарыйB21 overlay RED14fails сохранён, это не исполнение frozen SDK. Field/permission/query budgets иsheet/П1/П4/П7/APIB7 guards не менялись.

SDKLinux {art['linux']['sha256']} ({art['linux']['path']}); Windows {art['windows']['sha256']} ({art['windows']['path']}). Prebuilts UI{art['ui_pg']['sha256']} /interpreter{art['interpreter_pg']['sha256']}; полныйexpectednames/regex/SHAs вmanifest.

Manifest {out} SHA {sha(out)}. GOEND fresh0. Native helper подготовленSOURCEONLY, доrootbackendcleanup/отдельногоSTART не запускался. Нужны реальные403/500controls иfreshordinarydirect/backgroundXLSX defaultopenpyxl3.1.5 безreset/forcedbounds противnormal/28rows/grand253. Новыйкандидатпослевечернеговыпуска: shadowЛПЦ/:8444 попорядкуcc2, рабочаявследующемпакететолькоеёсловом. Shared/DBcopies0. К1745отдельныйfreshcheckpoint/push.

Generated-with: Codex
''',encoding='utf-8');print(json.dumps({'manifest':str(out),'sha256':sha(out),'report_sha256':sha(f),'runtime':H,'localpass':480,'localskip':17}))
