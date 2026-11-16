import collections, datetime, hashlib, json, os, pathlib, subprocess
B = pathlib.Path(__file__).parent
W = B/'report-b28-runtime'
D = B/'b28-reports-build-20261116'
D.mkdir(exist_ok=False)
GO = 'C:/Users/a.sevrjukova/go/pkg/mod/golang.org/toolchain@v0.0.1-go1.26.11.windows-amd64/bin/go.exe'
HEAD = 'f045bdb11b5e563120ff0381476957e5df9e2549'
BASE = 'a10677b2903e4b94cefb8d90a4104fe8a24aa639'
env = dict(os.environ,GOTOOLCHAIN='local',GOMAXPROCS='2',GOFLAGS='-p=1',GOWORK='off',CGO_ENABLED='0',GOARCH='amd64',GOOS='windows')
for key in ['DATABASE_URL','TEST_DATABASE_URL','ONEBASE_TEST_WORKBOOK_LOG']:
 env.pop(key,None)
def git(*args):
 return subprocess.check_output(['git',*args],cwd=W,creationflags=subprocess.CREATE_NO_WINDOW).decode().strip()
def sha(path):
 with pathlib.Path(path).open('rb') as f: return hashlib.file_digest(f,'sha256').hexdigest()
def check():
 assert git('rev-parse','HEAD')==HEAD and not git('status','--porcelain')
check()
changed = ['CHANGELOG.md','internal/dsl/interpreter/query_proxy.go','internal/excel/dimension_test.go','internal/excel/excel.go','internal/report/context.go','internal/sheet/block_output_test.go','internal/sheet/drilldown_color_test.go','internal/ui/handlers_reports.go','internal/ui/report_context_test.go','internal/ui/report_drilldown.go','internal/ui/report_list_dimension_test.go','internal/ui/report_result_access.go','internal/ui/report_result_access_test.go','internal/ui/row_access.go','internal/ui/tabular_block_output_test.go','internal/ui/tabular_color_test.go']
assert set(git('diff','--name-only',BASE,HEAD).splitlines())==set(changed)
receipt={'runtime':HEAD,'base_b27r2':BASE,'b21_inherited_p4_p7':'0d77e91557b12bada4668fd7bfa5f1b13a37de08','changed_files':changed,'shared_changes':False,'db_copies':0,'commands':[],'artifacts':{},'prebuilts':{},'status':'RUNNING'}
def save():
 (D/'manifest.json').write_text(json.dumps(receipt,indent=2)+'\n',encoding='utf-8')
def run(args,name,target='windows'):
 check(); log=D/name; start=datetime.datetime.now(datetime.timezone.utc).isoformat()
 with log.open('xb') as f:
  p=subprocess.run([GO,*args],cwd=W,env=dict(env,GOOS=target),stdout=f,stderr=subprocess.STDOUT,creationflags=subprocess.CREATE_NO_WINDOW,timeout=1200)
 check(); receipt['commands'].append({'argv':[GO,*args],'goos':target,'exit':p.returncode,'log':str(log),'sha256':sha(log),'start':start,'end':datetime.datetime.now(datetime.timezone.utc).isoformat(),'gomaxprocs':2,'parallelism':1});save()
 print(json.dumps({'step':name,'exit':p.returncode}),flush=True)
 assert p.returncode==0,str(log)
 return log
def summarize(key,path):
 events=[json.loads(s) for s in path.read_text('utf-8').splitlines() if s.startswith('{')]
 assert not any(e['Action']=='fail' for e in events)
 results={e['Test']:e['Action'] for e in events if e.get('Test') and e['Action'] in ['pass','skip']}
 leaves={n:a for n,a in results.items() if not any(o.startswith(n+'/') for o in results)}
 assert leaves
 receipt[key+'_leaves']=leaves; receipt[key+'_counts']=dict(collections.Counter(leaves.values())); save()
 return leaves
summarize('xlsx',run(['test','-count=1','-p=1','-json','./internal/sheet','./internal/dsl/interpreter','./internal/excel','-run','Test.*(Workbook|XLSX|Xlsx|Outline|Query.*Access)'],'xlsx-dsl-tests.jsonl'))
pattern='^(TestAPIReportTrustedContextMatrix|TestReportTrustedContextHTTPMatrix|TestReportTypedDrilldownHTTPMatrix|TestReportResultQueryAccessStatusMatrix|TestOrdinaryReportXLSXUsedRangeMatrix|TestShellSidePanelListFillsPanelHeight|TestTabularReportResultHTTPMatrix|TestTabularXLSXDimensionHTTPMatrix|TestTabularDrilldownHTTPMatrix|TestTabularLayoutReportHTTPMatrix|TestTabularManagedFieldHTTPMatrix|TestConfiguratorSaveReport_ContextRoundTrip|TestReportTypedDrilldownMetadata|TestReportTrustedContext|TestReportContextYAML)$'
summarize('reports',run(['test','-count=1','-p=1','-json','./internal/api','./internal/ui','./internal/launcher','./internal/report','-run',pattern],'report-critical-tests.jsonl'))
run(['test','-count=1','-p=1','./internal/cli','./internal/dsl/langref','-run','Test(PublicDocsLanguageCountsUpToDate|DSLReference|LangrefMarkdown|Completeness|Descriptors|Examples|Alias)'],'langref-tests.log')
run(['build','-p=1','./...'],'build-all.log')
run(['run','-p=1','./tools/i18ncheck'],'i18n-en.log')
for target,suffix in [('windows','windows-amd64.exe'),('linux','linux-amd64')]:
 artifact=B/('onebase-1116b28-reports-'+suffix);assert not artifact.exists()
 run(['build','-p=1','-ldflags=-X github.com/ivantit66/onebase/internal/version.Build=dev-reports-b28','-o',str(artifact),'./cmd/onebase'],target+'-sdk-build.log',target)
 info=run(['version','-m',str(artifact)],target+'-sdk-buildinfo.txt',target)
 receipt['artifacts'][target]={'path':str(artifact),'sha256':sha(artifact),'bytes':artifact.stat().st_size,'buildinfo':str(info),'runtime':HEAD,'embedded_vcs_not_claimed':True};save()
for package in ['api','ui','dsl/interpreter']:
 artifact=D/(package.replace('/','-')+'-linux-amd64.test')
 run(['test','-c','-p=1','-o',str(artifact),'./internal/'+package],package.replace('/','-')+'-prebuilt.log','linux')
 receipt['prebuilts'][package]={'path':str(artifact),'sha256':sha(artifact),'runtime':HEAD};save()
receipt['status']='BUILD_AND_TARGETED_TESTS_PASS_NATIVE_AND_INDEPENDENT_PG_PENDING'
receipt['ended_utc']=datetime.datetime.now(datetime.timezone.utc).isoformat();save()
print(json.dumps({'status':receipt['status'],'xlsx':receipt['xlsx_counts'],'reports':receipt['reports_counts'],'artifacts':receipt['artifacts']}),flush=True)
