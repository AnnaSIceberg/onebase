import pathlib,subprocess,hashlib,json,os,datetime
B=pathlib.Path(__file__).parent
S=B/'report-tabular-result-source';W=B/'report-tabular-result-runtime'
SOURCE='d232937812637564c730fbc17a9376d269dc5247';BASE='cd9910c81fd7812874b9ca2cafa1870857abee5e'
GO='C:/Users/a.sevrjukova/go/pkg/mod/golang.org/toolchain@v0.0.1-go1.26.11.windows-amd64/bin/go.exe'
def git(w,*a):return subprocess.check_output(['git',*a],cwd=w,creationflags=subprocess.CREATE_NO_WINDOW)
assert git(S,'rev-parse','HEAD').decode().strip()==SOURCE and not git(S,'status','--porcelain').strip()
assert git(W,'rev-parse','HEAD').decode().strip()==BASE and not git(W,'status','--porcelain').strip()
D=B/'b20-red-old-xlsx';D.mkdir(exist_ok=False)
old=D/'xlsx.go';old.write_bytes(git(S,'show','815e1505f504659fedea159fd186eeff3ab36d33:internal/sheet/xlsx.go'))
overlay=D/'overlay.json';overlay.write_text(json.dumps({'Replace':{str(S/'internal/sheet/xlsx.go'):str(old)}}),encoding='utf-8')
env=dict(os.environ,GOTOOLCHAIN='go1.26.11',GOMAXPROCS='2',CGO_ENABLED='0',GOOS='windows',GOARCH='amd64')
for k in ['TEST_DATABASE_URL','DATABASE_URL']:env.pop(k,None)
args=[GO,'test','-p=1','-json','-overlay='+str(overlay),'./internal/sheet','-run','^TestDocumentXLSXDimension$']
log=D/'red.jsonl'
with log.open('xb')as f:p=subprocess.run(args,cwd=S,env=env,stdout=f,stderr=subprocess.STDOUT,creationflags=subprocess.CREATE_NO_WINDOW)
rows=[json.loads(x)for x in log.read_text('utf-8').splitlines()if x.startswith('{')]
fails=[r['Test']for r in rows if r.get('Test')and r['Action']=='fail'and r['Test'].count('/')==1]
assert p.returncode==1 and len(fails)==7,(p.returncode,fails)
(D/'receipt.json').write_text(json.dumps({'source_tests':SOURCE,'old_writer_source':'815e1505f504659fedea159fd186eeff3ab36d33','writer_sha256':hashlib.sha256(old.read_bytes()).hexdigest(),'expected_red_leaf_count':7,'failed_leaves':fails,'exit':p.returncode,'argv':args,'not_frozen_sdk_execution':True},ensure_ascii=False,indent=2)+'\n',encoding='utf-8')
print('Expected old writer RED: 7 dimension failures',flush=True)
subprocess.run(['git','switch','-c','fix/xlsx-used-range-runtime-20261115',BASE],cwd=W,creationflags=subprocess.CREATE_NO_WINDOW,check=True)
subprocess.run(['git','-c','gc.auto=0','cherry-pick',SOURCE],cwd=W,creationflags=subprocess.CREATE_NO_WINDOW,check=True)
head=git(W,'rev-parse','HEAD').decode().strip()
s=(B/'build-kpi-b19.py').read_text('utf-8')
s=s.replace(BASE,head).replace('815e1505f504659fedea159fd186eeff3ab36d33',SOURCE)
s=s.replace("BASE = '64886e644da31991abd93d48390aaff7b883e9bc'","BASE = '"+BASE+"'")
s=s.replace("'TestTabularQueryInputLimitHTTPMatrix']","'TestTabularQueryInputLimitHTTPMatrix', 'TestTabularXLSXDimensionHTTPMatrix']")
s=s.replace('b19','b20').replace('base_b18','base_b19')
s=s.replace('==59','==66').replace("'expected_total':118,'sqlite':59,'postgres':59","'expected_total':132,'sqlite':66,'postgres':66").replace("'expected_total':118,","'expected_total':132,")
s=s.replace('onebase-1115b20-tabular-input-limit','onebase-1115b20-xlsx-dimension')
s=s.replace("['run','./tools/i18ncheck','-lang','en']","['run','./tools/i18ncheck']")
s=s.replace('    results = []',"    results = []\n    results.append(parse(run(['test','-p=1','-json','./internal/sheet','./internal/printform'],'b20-sheet-printform-full.jsonl')))\n    results.append(parse(run(['test','-p=1','-json','./internal/dsl/interpreter','-run','XLSX|LayoutFormatReaches|MaketFormatReaches'],'b20-interpreter-xlsx.jsonl')))")
builder=B/'build-kpi-b20.py';assert not builder.exists();builder.write_text(s,encoding='utf-8')
print(json.dumps({'runtime':head,'source':SOURCE,'base_b19':BASE,'builder':str(builder)}),flush=True)
