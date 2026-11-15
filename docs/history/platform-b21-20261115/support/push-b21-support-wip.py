import pathlib,json,hashlib,subprocess,os,datetime
B=pathlib.Path(r'C:/Users/a.sevrjukova/AppData/Local/Temp/platform-help-admin-20261111');W=B/'reports-p4-p7-b21-runtime';R=pathlib.Path(r'C:/Users/a.sevrjukova/AppData/Local/Temp/claude/C--Users-a-sevrjukova-onebase/cd50d987-9739-4c93-a5be-68d748529225/scratchpad/отчёты-для-analysis');H='0d77e91557b12bada4668fd7bfa5f1b13a37de08'
def git(*a,data=None,env=None):return subprocess.check_output(['git','-c','gc.auto=0',*a],cwd=W,input=data,env=env,creationflags=subprocess.CREATE_NO_WINDOW)
assert git('rev-parse','HEAD').decode().strip()==H and not git('status','--porcelain').strip()
state=R/'20261115-platform-B21-current-state.md';assert not state.exists();state.write_text('''# Platform current state — B21 handed to cc2

Combined runtime/source integration0d77e91557b12bada4668fd7bfa5f1b13a37de08, baseB20e6, sourceinputs P4cf5/P702c. Main runtime fork branch feature/reports-p4-p7-b21-runtime-20261115 clean/pushed. Final manifest6f4f74ea7ffd5ef4511a9dc1050b5e8e816f3e8c059ed64c2cb7333f354ca70e; original build2de91957 untouched. Linux881d78ad/Windows e9bbbce6, full SHA/path inmanifest. Local383PASS/14localPGrootSKIP; independentAPI12+UI166=178PASS/0SKIP/exactcleanup; actualWindowsnative19PASS/JS0. FourPNGs Desktop/ТабДок-B21-П4-П7-native-20261115-platform. Native positive own synthetic admin; nonadmin/RLS/revocation/binary guards HTTP matrices, not real OTK dataset.

Explicit own Go END + browser/app/port18572/driver/context/authDB/tunnel cleanup accepted. cc2 NOW holds canonical Chrome/ownPG for merged config47701d26; platform Go/Chrome HOLD until root END. Shared :8444 B18 unchanged by platform, release prepare NOT execute before evening userword. No DBcopy/migrate to shared. Root acceptance and3base release execution separate.

After-release ownership ACK platform2: nested result_proc ACL deniedsource500vs403 and ordinary YAMLreport XLSXdimensionA1; neither implemented byplatform, neither added B21. Exact ownsource parent d2329378 / runtimeB20e6 supplied. П4/П7source author platform2, engineintegration platform. Partners prebind consultation RO only report5d3d4e81: no standalone nativeCLI no-seed, exportedEnsurePredefinedColumnsAPI + ordinaryMigrate alwaysseed; sameTX prebindproposal only, no actualDDL/database read here.

Source Drafts2629/2630 remainDraft; no Ready/fullCI/production readiness claimed. All helper failures and corrected receipts retained; no retag oldSDK/PG. User requires regularWIPpush +17:30releasefreeze/17:45freshrebootcheckpoint; that future checkpoint is not declared completed now.

Generated-with: Codex
''',encoding='utf-8')
branch='wip/platform-b21-handoff-20261115';assert subprocess.run(['git','show-ref','--verify','--quiet','refs/heads/'+branch],cwd=W,creationflags=subprocess.CREATE_NO_WINDOW).returncode==1
index=B/'b21-support-wip.index';assert not index.exists();env=dict(os.environ,GIT_INDEX_FILE=str(index));git('read-tree',H,env=env)
files=[]
for n in ['prepare-b21-runtime.py','resolve-b21-p7-and-apply-p4.py','resolve-b21-p4.py','fixtures-b21-runtime.py','build-b21.py','b21-native-context-drilldown.py','b21-native-context-drilldown-attempt2.py','b21-native-context-drilldown-attempt3.py','finalize-b21-native.py','manifest-1115b21-report-context-drilldown-DEV.json','manifest-1115b21-report-context-drilldown-DEV-final.json','b21-local-targeted-summary.json','b21-api_pg-expected.json','b21-ui_pg-expected.json','b21-api_pg-handoff.json','b21-ui_pg-handoff.json','b21-go-end-processes.json','b21-native-cleanup-end.json']:
 files.append((B/n,'support/'+n))
for n in ['b21-api_pg-artifact.json','b21-ui_pg-artifact.json','b21-linux-artifact.json','b21-windows-artifact.json','b21-build-all.log.command.json','b21-i18n-en.log.command.json','b21-api_pg-buildinfo.txt','b21-ui_pg-buildinfo.txt','b21-linux-buildinfo.txt','b21-windows-buildinfo.txt']:
 files.append((B/n,'support/'+n))
for folder,name in [('b21-native-context-drilldown-attempt1','receipt-recovered.json'),('b21-native-context-drilldown-attempt2','receipt.json'),('b21-native-context-drilldown-attempt2','redirect-server-observation.json'),('b21-native-context-drilldown-attempt3','receipt.json')]:files.append((B/folder/name,'receipts/'+folder+'/'+name))
for n in ['20261115-platform-Partners-prebind-B20-source-RO.md','20261115-platform-Partners-prebind-B20-source-pins.json','20261115-platform-B21-П4-П7-build-handoff.md','20261115-platform-B21-П4-П7-PG-native-PASS.md','20261115-platform-B21-current-state.md']:files.append((R/n,'reports/'+n))
inventory=[]
for p,target in files:
 assert p.is_file();raw=p.read_bytes();oid=git('hash-object','-w','--stdin',data=raw).decode().strip();git('update-index','--add','--cacheinfo','100644,'+oid+',docs/history/platform-b21-20261115/'+target,env=env);inventory.append({'path':target,'bytes':len(raw),'sha256':hashlib.sha256(raw).hexdigest()})
tree=git('write-tree',env=env).decode().strip();commit=git('commit-tree',tree,'-p',H,data='docs(wip): сохранить полный handoff B21 и квитанции\n\nGenerated-with: Codex\n'.encode()).decode().strip();git('update-ref','refs/heads/'+branch,commit);git('-c','http.sslBackend=openssl','push','fork',branch)
out=B/'b21-support-wip-push.json';assert not out.exists();receipt={'branch':branch,'commit':commit,'parent':H,'files':inventory,'scope':'source helpers, metadata, reports, sanitized receipts only; no SDK binaries/DB/password/cookie/customer source fixture/image/export payload','runtime_worktree_clean':not git('status','--porcelain').strip(),'created_utc':datetime.datetime.now(datetime.timezone.utc).isoformat()};out.write_text(json.dumps(receipt,ensure_ascii=False,indent=2)+'\n',encoding='utf-8');print(json.dumps({'branch':branch,'commit':commit,'files':len(files),'receipt_sha256':hashlib.sha256(out.read_bytes()).hexdigest(),'runtime_clean':receipt['runtime_worktree_clean']}))
