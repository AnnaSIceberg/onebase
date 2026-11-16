import collections,datetime,hashlib,json,os,pathlib,subprocess
B=pathlib.Path(__file__).parent;W=B/'report-b29-enum-runtime';D=B/'b29-enum-build-20261116';D.mkdir(exist_ok=False)
GO='C:/Users/a.sevrjukova/go/pkg/mod/golang.org/toolchain@v0.0.1-go1.26.11.windows-amd64/bin/go.exe'
HEAD='902e4d97e50e90f047ec6d27832e2f9d175d095d';BASE='f045bdb11b5e563120ff0381476957e5df9e2549';SOURCE='2e185f0615f98111cd1353d71afcc72fdabbd5fd'
env=dict(os.environ,GOTOOLCHAIN='local',GOMAXPROCS='2',GOFLAGS='-p=1',GOWORK='off',CGO_ENABLED='0',GOARCH='amd64',GOOS='windows')
for k in ['DATABASE_URL','TEST_DATABASE_URL']:env.pop(k,None)
def git(*args):return subprocess.check_output(['git',*args],cwd=W,creationflags=subprocess.CREATE_NO_WINDOW).decode().strip()
def sha(p):
 with pathlib.Path(p).open('rb') as f:return hashlib.file_digest(f,'sha256').hexdigest()
def check():assert git('rev-parse','HEAD')==HEAD and not git('status','--porcelain')
check()
files=['CHANGELOG.md','docs/features.md','docs/report-enum-parameters.md','internal/api/v2.go','internal/api/report_enum_params_test.go','internal/report/enum_params.go','internal/ui/handlers_reports.go','internal/ui/templates.go']
assert set(git('diff','--name-only',BASE,HEAD).splitlines())==set(files)
for file in ['internal/api/report_enum_params_test.go','internal/report/enum_params.go','docs/report-enum-parameters.md']:
 assert git('rev-parse',HEAD+':'+file)==git('rev-parse',SOURCE+':'+file)
receipt={'runtime':HEAD,'base_b28':BASE,'source':SOURCE,'source_pr':2637,'changed_files':files,'commands':[],'artifacts':{},'prebuilts':{},'shared_changes':False,'db_copies':0,'status':'RUNNING','trusted_context_bound_before_enum_validation':True}
def save():(D/'manifest.json').write_text(json.dumps(receipt,indent=2)+'\n',encoding='utf-8')
def run(args,name,target='windows'):
 check();p=D/name;start=datetime.datetime.now(datetime.timezone.utc).isoformat()
 with p.open('xb') as f:r=subprocess.run([GO,*args],cwd=W,env=dict(env,GOOS=target),stdout=f,stderr=subprocess.STDOUT,creationflags=subprocess.CREATE_NO_WINDOW,timeout=1200)
 check();receipt['commands'].append({'argv':[GO,*args],'goos':target,'exit':r.returncode,'log':str(p),'sha256':sha(p),'start':start,'end':datetime.datetime.now(datetime.timezone.utc).isoformat()});save();print(json.dumps({'step':name,'exit':r.returncode}),flush=True)
 assert r.returncode==0,str(p)
 return p
pattern='^(TestReportEnumParamsHTTPMatrix|TestAPIReportTrustedContextMatrix|TestReportTrustedContextHTTPMatrix|TestReportTypedDrilldownHTTPMatrix|TestReportResultQueryAccessStatusMatrix|TestOrdinaryReportXLSXUsedRangeMatrix|TestTabularReportResultHTTPMatrix|TestShellSidePanelListFillsPanelHeight)$'
p=run(['test','-count=1','-p=1','-json','./internal/api','./internal/ui','-run',pattern],'http-enum-critical-tests.jsonl')
events=[json.loads(s) for s in p.read_text('utf-8').splitlines() if s.startswith('{')]
assert not any(e['Action']=='fail' for e in events)
results={(e['Package'],e['Test']):e['Action'] for e in events if e.get('Test') and e['Action'] in ['pass','skip']}
leaves={pkg+'/'+n:a for (pkg,n),a in results.items() if not any(pkg==otherpkg and other.startswith(n+'/') for otherpkg,other in results)}
receipt['http_counts']=dict(collections.Counter(leaves.values()));receipt['http_leaves']=leaves;save()
run(['build','-p=1','./...'],'build-all.log');run(['run','-p=1','./tools/i18ncheck'],'i18n-en.log')
for target,suffix in [('windows','windows-amd64.exe'),('linux','linux-amd64')]:
 artifact=B/('onebase-1116b29-report-enum-'+suffix);assert not artifact.exists()
 run(['build','-p=1','-ldflags=-X github.com/ivantit66/onebase/internal/version.Build=dev-reports-b29-enum','-o',str(artifact),'./cmd/onebase'],target+'-sdk-build.log',target)
 info=run(['version','-m',str(artifact)],target+'-sdk-buildinfo.txt',target)
 receipt['artifacts'][target]={'path':str(artifact),'sha256':sha(artifact),'bytes':artifact.stat().st_size,'buildinfo':str(info),'embedded_vcs_not_claimed':True};save()
for package in ['api','ui','dsl/interpreter']:
 artifact=D/(package.replace('/','-')+'-linux-amd64.test')
 run(['test','-c','-p=1','-o',str(artifact),'./internal/'+package],package.replace('/','-')+'-prebuilt.log','linux')
 receipt['prebuilts'][package]={'path':str(artifact),'sha256':sha(artifact),'runtime':HEAD};save()
receipt.update(status='BUILD_AND_HTTP_PASS_NATIVE_AND_INDEPENDENT_PG_PENDING',ended=datetime.datetime.now(datetime.timezone.utc).isoformat());save()
print(json.dumps({'status':receipt['status'],'http':receipt['http_counts'],'artifacts':receipt['artifacts']}),flush=True)
