import pathlib, subprocess, json, hashlib, os, datetime
B=pathlib.Path(__file__).parent
S=B/'report-tabular-result-source'
W=B/'report-tabular-result-runtime'
G='C:/Users/a.sevrjukova/go/pkg/mod/golang.org/toolchain@v0.0.1-go1.26.11.windows-amd64/bin/go.exe'
F='815e1505f504659fedea159fd186eeff3ab36d33'
BASE='64886e644da31991abd93d48390aaff7b883e9bc'
def git(w,*a):
 return subprocess.check_output(['git',*a],cwd=w,creationflags=subprocess.CREATE_NO_WINDOW)
assert not git(S,'status','--porcelain').strip()
assert git(S,'rev-parse','HEAD').decode().strip()==F
assert not git(W,'status','--porcelain').strip()
assert git(W,'rev-parse','HEAD').decode().strip()==BASE
D=B/'b19-red-frozen621';D.mkdir(exist_ok=False)
old=D/'handlers_reports.go'
old.write_bytes(git(S,'show','621c6628523c45b3912ec1ad9475ffdf084475c4:internal/ui/handlers_reports.go'))
overlay=D/'overlay.json'
overlay.write_text(json.dumps({'Replace':{str(S/'internal/ui/handlers_reports.go'):str(old)}}),encoding='utf-8')
env=dict(os.environ,GOTOOLCHAIN='go1.26.11',GOMAXPROCS='2',CGO_ENABLED='0',GOOS='windows',GOARCH='amd64')
for k in ['TEST_DATABASE_URL','DATABASE_URL']:env.pop(k,None)
args=[G,'test','-p=1','-json','-overlay='+str(overlay),'./internal/ui','-run','^TestTabularQueryInputLimitHTTPMatrix/sqlite/(GET|POST)/(limit-hit|module-must-not-run|no-stale-viewer)$']
start=datetime.datetime.now(datetime.timezone.utc).isoformat()
log=D/'red.jsonl'
with log.open('xb') as f:
 p=subprocess.run(args,cwd=S,env=env,stdout=f,stderr=subprocess.STDOUT,creationflags=subprocess.CREATE_NO_WINDOW)
rows=[json.loads(x) for x in log.read_text(encoding='utf-8').splitlines() if x.startswith('{')]
fails=[r['Test'] for r in rows if r.get('Test') and r['Action']=='fail' and r['Test'].count('/')==3]
assert p.returncode==1 and len(fails)==6, (p.returncode,fails)
(D/'receipt.json').write_text(json.dumps({'source_fixture':F,'overlaid_handler_source':'621c6628523c45b3912ec1ad9475ffdf084475c4','handler_sha256':hashlib.sha256(old.read_bytes()).hexdigest(),'argv':args,'start_utc':start,'end_utc':datetime.datetime.now(datetime.timezone.utc).isoformat(),'exit':p.returncode,'failed_leaves':fails,'expected_red':True,'not_frozen_sdk_execution':True},ensure_ascii=False,indent=2),encoding='utf-8')
print('EXPECTED RED: six cropped-input checks fail on exact621 handler',flush=True)
subprocess.run(['git','switch','-c','fix/tabular-query-input-runtime-20261115',BASE],cwd=W,check=True,creationflags=subprocess.CREATE_NO_WINDOW)
subprocess.run(['git','-c','gc.auto=0','cherry-pick',F],cwd=W,check=True,creationflags=subprocess.CREATE_NO_WINDOW)
print('RUNTIME_HEAD='+git(W,'rev-parse','HEAD').decode().strip(),flush=True)
