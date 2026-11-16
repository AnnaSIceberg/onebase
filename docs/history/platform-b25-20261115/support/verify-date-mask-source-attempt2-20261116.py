import datetime,hashlib,json,os,pathlib,subprocess
B=pathlib.Path(__file__).parent;W=pathlib.Path('C:/Users/a.sevrjukova/platform-report-date-mask-20261116')
D=B/'date-mask-source-checks-attempt2-20261116';D.mkdir(exist_ok=False)
GO='C:/Users/a.sevrjukova/go/pkg/mod/golang.org/toolchain@v0.0.1-go1.26.11.windows-amd64/bin/go.exe'
HEAD='5fdabda1548dc5110aed662f69dc590614c66f41'
env=dict(os.environ,GOMAXPROCS='2',CGO_ENABLED='0',GOTOOLCHAIN='local',GOFLAGS='-p=1')
for k in ['DATABASE_URL','TEST_DATABASE_URL']:env.pop(k,None)
def git(*args):return subprocess.check_output(['git',*args],cwd=W,creationflags=subprocess.CREATE_NO_WINDOW).decode().strip()
def check():assert git('rev-parse','HEAD')==HEAD and not git('status','--porcelain')
receipt={'source':HEAD,'draft':2611,'commands':[],'browser_started':False,'status':'RUNNING'}
def save():(D/'receipt.json').write_text(json.dumps(receipt,indent=2)+'\n',encoding='utf-8')
def run(args,name):
 check();p=D/name
 with p.open('xb')as f:r=subprocess.run([GO,*args],cwd=W,env=env,stdout=f,stderr=subprocess.STDOUT,creationflags=subprocess.CREATE_NO_WINDOW,timeout=1200)
 check();receipt['commands'].append({'argv':[GO,*args],'exit':r.returncode,'log':str(p),'sha256':hashlib.sha256(p.read_bytes()).hexdigest()});save();print(json.dumps({'step':name,'exit':r.returncode}),flush=True)
 assert r.returncode==0,str(p)
 return p
run(['test','-count=1','-p=1','-json','./internal/ui','-run','^(TestDateInputBehaviorThroughDOMEvents|TestReportControlsBehaviorThroughDOMEvents|TestCompactReportsRoleScopeThroughHTTP)$'],'ui-controls-tests.jsonl')
run(['build','-p=1','./...'],'build-all.log');run(['run','-p=1','./tools/i18ncheck'],'i18n-en.log')
binary=B/'date-mask-source-ui-attempt2-20261116.exe';assert not binary.exists()
run(['test','-c','-p=1','-o',str(binary),'./internal/ui'],'browser-prebuilt.log')
receipt.update(status='GO_END_SOURCE_GATES_PASS_BROWSER_PREBUILT_READY',binary=str(binary),binary_sha256=hashlib.sha256(binary.read_bytes()).hexdigest(),ended=datetime.datetime.now(datetime.timezone.utc).isoformat());save()
print(json.dumps({'status':receipt['status'],'source':HEAD,'binary':str(binary),'sha256':receipt['binary_sha256']}),flush=True)
