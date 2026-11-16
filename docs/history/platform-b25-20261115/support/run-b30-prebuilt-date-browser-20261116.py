import hashlib,json,os,pathlib,subprocess
B=pathlib.Path(__file__).parent
m=json.loads((B/'b30-date-mask-build-20261116/manifest.json').read_text('utf-8'))
assert m['status']=='BUILD_AND_HTTP_PASS_NATIVE_AND_INDEPENDENT_PG_PENDING'
artifact=m['prebuilts']['ui-windows'];p=pathlib.Path(artifact['path'])
with p.open('rb')as f:assert hashlib.file_digest(f,'sha256').hexdigest()==artifact['sha256']
log=B/'b30-prebuilt-date-browser-attempt1-20261116.log';assert not log.exists()
with log.open('xb')as f:
 r=subprocess.run([str(p),'-test.run','^TestDateMaskBrowserReportEditing$','-test.v','-test.timeout','160s'],cwd=B/'report-b30-date-mask-runtime/internal/ui',env=dict(os.environ,GOMAXPROCS='2'),stdout=f,stderr=subprocess.STDOUT,creationflags=subprocess.CREATE_NO_WINDOW,timeout=180)
out={'runtime':m['runtime'],'source':m['source'],'binary':artifact,'exit':r.returncode,'log':str(log),'sha256':hashlib.sha256(log.read_bytes()).hexdigest(),'go_started':False,'shared_changes':False}
dest=B/'b30-prebuilt-date-browser-attempt1-20261116.json';assert not dest.exists();dest.write_text(json.dumps(out,indent=2)+'\n',encoding='utf-8');print(json.dumps(out))
raise SystemExit(r.returncode)
