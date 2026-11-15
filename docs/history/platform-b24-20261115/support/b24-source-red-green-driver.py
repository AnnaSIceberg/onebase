from pathlib import Path
import subprocess,os,json,hashlib,datetime
B=Path(__file__).parent;S=B/'tabular-block-output-source';W=B/'tabular-block-output-runtime';GO=Path(r'C:/Users/a.sevrjukova/go/pkg/mod/golang.org/toolchain@v0.0.1-go1.26.11.windows-amd64/bin/go.exe');env=dict(os.environ,GOTOOLCHAIN='go1.26.11',GOMAXPROCS='2',GOFLAGS='-p=1',CGO_ENABLED='0',GOWORK='off',GOOS='windows',GOARCH='amd64');env.pop('TEST_DATABASE_URL',None);env.pop('DATABASE_URL',None)
SOURCE='6a7d082bdfab850a1d6b79b182bb4b7e9f64a0fd';RUNTIME='f1a2855d592701bb1f498115c43af68ccef38ee7';regex='^(TestTabularBlockOutput|TestSpreadsheetBlockOutputDSL|TestTabularHorizontalBlockHTTPMatrix)$'
def git(cwd,*a):return subprocess.check_output(['git',*a],cwd=cwd,creationflags=subprocess.CREATE_NO_WINDOW).decode().strip()
def check():assert git(S,'rev-parse','HEAD')==SOURCE and not git(S,'status','--porcelain')and git(W,'rev-parse','HEAD')==RUNTIME and not git(W,'status','--porcelain')
def run(cwd,argv,log):
 check();p=B/log;assert not p.exists();start=datetime.datetime.now(datetime.timezone.utc).isoformat()
 with p.open('xb')as f:r=subprocess.run([str(GO),*argv],cwd=cwd,env=env,stdout=f,stderr=subprocess.STDOUT,creationflags=subprocess.CREATE_NO_WINDOW)
 check();rec={'cwd':str(cwd),'argv':[str(GO),*argv],'exit':r.returncode,'start':start,'end':datetime.datetime.now(datetime.timezone.utc).isoformat(),'sha256':hashlib.sha256(p.read_bytes()).hexdigest(),'toolchain':'go1.26.11 GOMAX2 p1 CGO0','source':SOURCE,'runtime':RUNTIME,'clean_before_after':True};p.with_suffix(p.suffix+'.command.json').write_text(json.dumps(rec,indent=2)+'\n',encoding='utf-8');print(json.dumps({'log':log,'exit':r.returncode,'sha256':rec['sha256']}),flush=True);return r.returncode,p
exit,p=run(S,['test','-count=1','-p=1','-json','./internal/sheet','./internal/dsl/interpreter','./internal/ui','-run',regex],'b24-source-third-targeted.jsonl')
if exit:raise SystemExit(exit)
exit,p=run(W,['test','-count=1','-p=1','-json','-overlay',str(B/'b24-red-old-sheet/overlay.json'),'./internal/sheet','./internal/dsl/interpreter','./internal/ui','-run',regex],'b24-red-old-sheet-targeted.jsonl')
events=[json.loads(l)for l in p.read_text('utf-8').splitlines()if l.startswith('{')];failed=[x['Test']for x in events if x.get('Test')and x['Action']=='fail'];assert exit!=0 and failed and not any('build failed'in x.get('Output','')for x in events);(B/'b24-red-old-sheet/receipt.json').write_text(json.dumps({'status':'EXPECTED_RED','old_sheet_source':'4fbee90500a26f9bc59336e0ae0453027fd941ea','test_runtime':RUNTIME,'source':SOURCE,'exit':exit,'failed_events':failed,'log':str(p),'sha256':hashlib.sha256(p.read_bytes()).hexdigest(),'not_frozen_SDK_execution':True},indent=2)+'\n',encoding='utf-8')
raise SystemExit(subprocess.run([r'C:/Python314/python.exe',str(B/'build-b24.py'),'gates-prebuilt'],creationflags=subprocess.CREATE_NO_WINDOW).returncode)
