import hashlib,json,os,pathlib,subprocess
B=pathlib.Path(__file__).parent
r=json.loads((B/'date-mask-source-checks-attempt1-20261116/receipt.json').read_text('utf-8'))
assert r['status']=='GO_END_SOURCE_GATES_PASS_BROWSER_PREBUILT_READY'
p=pathlib.Path(r['binary'])
with p.open('rb')as f:assert hashlib.file_digest(f,'sha256').hexdigest()==r['binary_sha256']
log=B/'date-mask-source-browser-attempt1-20261116.log';assert not log.exists()
with log.open('wb')as f:
 result=subprocess.run([str(p),'-test.run','^TestDateMaskBrowserReportEditing$','-test.v','-test.timeout','160s'],cwd='C:/Users/a.sevrjukova/platform-report-date-mask-20261116/internal/ui',env=dict(os.environ,GOMAXPROCS='2'),stdout=f,stderr=subprocess.STDOUT,creationflags=subprocess.CREATE_NO_WINDOW,timeout=180)
out={'source':r['source'],'binary':str(p),'binary_sha256':r['binary_sha256'],'exit':result.returncode,'log':str(log),'log_sha256':hashlib.sha256(log.read_bytes()).hexdigest(),'go_started':False,'browser_fixture':'public HTTP report + trusted CDP keyboard + clipboard restored by fixture'}
dest=B/'date-mask-source-browser-attempt1-20261116.json';assert not dest.exists();dest.write_text(json.dumps(out,indent=2)+'\n',encoding='utf-8');print(json.dumps(out))
raise SystemExit(result.returncode)
