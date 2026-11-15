import pathlib,subprocess,os,sys,hashlib,json,datetime,collections
B=pathlib.Path(__file__).parent; W=B/'report-source-filters-b23-runtime'; GO=pathlib.Path(r'C:/Users/a.sevrjukova/go/pkg/mod/golang.org/toolchain@v0.0.1-go1.26.11.windows-amd64/bin/go.exe')
HEAD='4fbee90500a26f9bc59336e0ae0453027fd941ea'; BASE='0d77e91557b12bada4668fd7bfa5f1b13a37de08'
INPUTS={'accepted_b21':BASE,'delta':'__settings.filters source datafix only; B22 excluded'}
UI_ROOTS=['TestTabularPayloadHTTPMatrix','TestTabularLayoutReportHTTPMatrix','TestTabularReportResultHTTPMatrix','TestTabularManagedFieldHTTPMatrix','TestTabularDrilldownHTTPMatrix','TestTabularSizeHTTPMatrix','TestReportFlatColumnsHTTP','TestTabularAutoScaleHTTPMatrix','TestTabularQueryInputLimitHTTPMatrix','TestTabularXLSXDimensionHTTPMatrix','TestReportTrustedContextHTTPMatrix','TestReportTypedDrilldownHTTPMatrix','TestReportResultSourceFiltersHTTPMatrix']
API_ROOTS=['TestAPIReportTrustedContextMatrix','TestReportVariantDataMatrix','TestAPIReportSourceFiltersMatrix']
env=dict(os.environ,GOTOOLCHAIN='go1.26.11',GOMAXPROCS='2',GOFLAGS='-p=1',GOWORK='off',CGO_ENABLED='0',GOARCH='amd64',GOOS='windows')
for k in ['TEST_DATABASE_URL','DATABASE_URL']:env.pop(k,None)
def git(*a):return subprocess.check_output(['git',*a],cwd=W,creationflags=subprocess.CREATE_NO_WINDOW).decode('utf-8').strip()
def sha(p):return hashlib.file_digest(pathlib.Path(p).open('rb'),'sha256').hexdigest()
def check():assert git('rev-parse','HEAD')==HEAD and not git('status','--porcelain')
def run(args,name,osname='windows'):
 check();log=B/name;assert not log.exists();start=datetime.datetime.now(datetime.timezone.utc).isoformat()
 with log.open('xb') as f:p=subprocess.run([str(GO),*args],cwd=W,env=dict(env,GOOS=osname,ONEBASE_TEST_XLSX_DIR=str(B/'b23-targeted-new-xlsx')),stdout=f,stderr=subprocess.STDOUT,creationflags=subprocess.CREATE_NO_WINDOW)
 check();r={'runtime':HEAD,'source':HEAD,'source_binding':'combined runtime integration source tree; separate upstream_inputs','upstream_inputs':INPUTS,'argv':[str(GO),*args],'env':{'GOTOOLCHAIN':'go1.26.11','pinned_go':'1.26.11','GOMAXPROCS':'2','CGO_ENABLED':'0','GOOS':osname,'GOARCH':'amd64','GOFLAGS':'-p=1'},'exit':p.returncode,'start':start,'end':datetime.datetime.now(datetime.timezone.utc).isoformat(),'log':str(log),'sha256':sha(log),'clean_before_after':True}
 log.with_suffix(log.suffix+'.command.json').write_text(json.dumps(r,indent=2)+'\n',encoding='utf-8');print(json.dumps({'step':name,'exit':p.returncode,'sha256':r['sha256']}),flush=True)
 if p.returncode:raise SystemExit('FAILED '+name)
 return log
def parse(p):
 events=[json.loads(l) for l in p.read_text('utf-8').splitlines() if l.startswith('{')];result={(e['Package'],e['Test']):e['Action'] for e in events if e.get('Test') and e['Action'] in ['pass','skip','fail']};leaves={k:v for k,v in result.items() if not any(x[0]==k[0] and x[1].startswith(k[1]+'/') for x in result)}
 return {'log':str(p),'sha256':sha(p),'counts':dict(collections.Counter(leaves.values())),'leaves':[{'package':k[0],'name':k[1],'result':v} for k,v in sorted(leaves.items())]}
def artifact(key,pkg,filename,osname,test=False):
 p=B/filename;assert not p.exists();args=['test','-c','-p=1','-o',str(p),pkg] if test else ['build','-p=1','-ldflags=-X github.com/ivantit66/onebase/internal/version.Build=dev-reports-b23','-o',str(p),pkg]
 run(args,'b23-'+key+'-build.log',osname);info=run(['version','-m',str(p)],'b23-'+key+'-buildinfo.txt',osname)
 a={'path':str(p),'sha256':sha(p),'bytes':p.stat().st_size,'runtime':HEAD,'source':HEAD,'source_binding':'combined runtime integration tree','upstream_inputs':INPUTS,'buildinfo':str(info),'buildinfo_sha256':sha(info)}
 out=B/('b23-'+key+'-artifact.json');assert not out.exists();out.write_text(json.dumps(a,indent=2)+'\n',encoding='utf-8');print(json.dumps({'artifact':key,**a}),flush=True);return a
check()
if sys.argv[1]=='gates-prebuilt':
 run(['build','-p=1','./...'],'b23-build-all.log')
 run(['run','-p=1','./tools/i18ncheck'],'b23-i18n-en.log')
 results=[]
 results.append(parse(run(['test','-count=1','-p=1','-json','./internal/report/compose','./internal/report','-run','TestApplySourceFilters|TestApplyFilters|TestReportVariant|TestUserSettings|TestParseUserSettings'],'b23-compose-targeted.jsonl')))
 for key,pkg,roots in [('ui_pg','./internal/ui',UI_ROOTS),('api_pg','./internal/api',API_ROOTS)]:
  regex='^('+'|'.join(roots)+')$';extra='|TestRenderComposed|TestWriteDetailLink|TestApplyResolvedLabels|TestReportVariantDataScreenAndExports' if key=='ui_pg' else ''
  result=parse(run(['test','-count=1','-p=1','-json',pkg,'-run',regex+extra],'b23-'+key+'-targeted.jsonl'));results.append(result)
  names=[x['name'] for x in result['leaves'] if x['result']=='pass' and '/sqlite' in x['name'] and x['name'].split('/')[0] in roots];assert len(names)==len(set(names)) and names
  expected=sorted(names+[n.replace('/sqlite','/postgres',1) for n in names]);out=B/('b23-'+key+'-expected.json');assert not out.exists();out.write_text(json.dumps({'runtime':HEAD,'source':HEAD,'source_binding':'combined runtime integration tree','upstream_inputs':INPUTS,'regex':regex,'roots':roots,'expected_subtests':expected,'expected_total':len(expected),'sqlite':len(names),'postgres':len(names),'local_pg_status':'SKIP_NOT_PASS'},indent=2)+'\n',encoding='utf-8')
  a=artifact(key,pkg,'reports-source-filters-b23-'+key.replace('_pg','')+'-linux-amd64.test','linux',True)
  handoff=B/('b23-'+key+'-handoff.json');assert not handoff.exists();handoff.write_text(json.dumps({'status':'IMMUTABLE_PREBUILT_ONLY_PG_PENDING','runtime':HEAD,'source':HEAD,'upstream_inputs':INPUTS,'base_b21':BASE,'artifact':a,'expected':str(out),'expected_sha256':sha(out),'regex':regex,'expected_total':len(expected),'go_window':'ACTIVE_UNTIL_EXPLICIT_END'},indent=2)+'\n',encoding='utf-8');print(json.dumps({'HANDOFF':str(handoff),'sha256':sha(handoff),'expected_total':len(expected)}),flush=True)
 out=B/'b23-local-targeted-summary.json';assert not out.exists();out.write_text(json.dumps({'runtime':HEAD,'source':HEAD,'upstream_inputs':INPUTS,'runs':results},indent=2)+'\n',encoding='utf-8')
elif sys.argv[1]=='sdk':
 artifact('linux','./cmd/onebase','onebase-1115b23-report-source-filters-linux-amd64','linux')
 artifact('windows','./cmd/onebase','onebase-1115b23-report-source-filters-windows-amd64.exe','windows')
else:raise SystemExit('unknown stage')
