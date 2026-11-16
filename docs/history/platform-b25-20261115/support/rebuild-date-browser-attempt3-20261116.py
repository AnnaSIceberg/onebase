import datetime,hashlib,json,os,pathlib,subprocess
B=pathlib.Path(__file__).parent;W=pathlib.Path('C:/Users/a.sevrjukova/platform-report-date-mask-20261116')
HEAD='fc8777bc33c1dfae23cc8bc4c9da9df378d2e18a'
def git(*a):return subprocess.check_output(['git',*a],cwd=W,creationflags=subprocess.CREATE_NO_WINDOW).decode().strip()
assert git('rev-parse','HEAD')==HEAD and not git('status','--porcelain')
D=B/'date-mask-source-checks-attempt3-20261116';D.mkdir(exist_ok=False)
exe=B/'date-mask-source-ui-attempt3-20261116.exe';assert not exe.exists()
go='C:/Users/a.sevrjukova/go/pkg/mod/golang.org/toolchain@v0.0.1-go1.26.11.windows-amd64/bin/go.exe'
env=dict(os.environ,GOMAXPROCS='2',GOFLAGS='-p=1',CGO_ENABLED='0',GOTOOLCHAIN='local')
with (D/'browser-prebuilt.log').open('xb')as f:
 p=subprocess.run([go,'test','-c','-p=1','-o',str(exe),'./internal/ui'],cwd=W,env=env,stdout=f,stderr=subprocess.STDOUT,creationflags=subprocess.CREATE_NO_WINDOW,timeout=1200)
assert p.returncode==0
with exe.open('rb')as f:sha=hashlib.file_digest(f,'sha256').hexdigest()
r={'source':HEAD,'status':'GO_END_SOURCE_GATES_PASS_BROWSER_PREBUILT_READY','binary':str(exe),'binary_sha256':sha,'source_gates':'attempt2 passed; subsequent change initializes test fixture interpreter only','ended':datetime.datetime.now(datetime.timezone.utc).isoformat()}
(D/'receipt.json').write_text(json.dumps(r,indent=2)+'\n',encoding='utf-8')
old=B/'run-date-mask-source-browser-attempt2-20261116.py';new=B/'run-date-mask-source-browser-attempt3-20261116.py';assert not new.exists();new.write_text(old.read_text().replace('attempt2','attempt3'),encoding='utf-8')
print(json.dumps(r),flush=True)
