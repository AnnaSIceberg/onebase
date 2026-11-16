import datetime, hashlib, json, os, pathlib, subprocess
B = pathlib.Path(__file__).parent
O = B / 'refreshed-source-checks-20261116'
O.mkdir(exist_ok=False)
GO = 'C:/Users/a.sevrjukova/go/pkg/mod/golang.org/toolchain@v0.0.1-go1.26.11.windows-amd64/bin/go.exe'
env = dict(os.environ, GOMAXPROCS='2', CGO_ENABLED='0', GOTOOLCHAIN='local', GOFLAGS='-p=1')
tasks = [
 (2631, 'C:/Users/a.sevrjukova/platform2-report-acl-source', ['./internal/ui'], '^TestReportResultQueryAccessStatusMatrix$'),
 (2630, 'C:/Users/a.sevrjukova/platform2-report-drilldown', ['./internal/ui'], '^TestReportTypedDrilldownHTTPMatrix$'),
 (2629, 'C:/Users/a.sevrjukova/platform2-report-context', ['./internal/ui'], '^TestReportTrustedContextHTTPMatrix$'),
 (2632, 'C:/Users/a.sevrjukova/platform2-list-xlsx-source', ['./internal/excel','./internal/ui'], '^Test(ListXLSXUsedRange|OrdinaryReportXLSXUsedRangeMatrix)$'),
]
receipt = {'started_utc': datetime.datetime.now(datetime.timezone.utc).isoformat(), 'serial': True, 'runs': []}
def run(args, cwd, log):
 with log.open('wb') as f:
  p = subprocess.run(args, cwd=cwd, env=env, stdout=f, stderr=subprocess.STDOUT, creationflags=subprocess.CREATE_NO_WINDOW)
 assert p.returncode == 0, str(log)
 return hashlib.sha256(log.read_bytes()).hexdigest()
for number, directory, packages, pattern in tasks:
 head = subprocess.check_output(['git','rev-parse','HEAD'],cwd=directory,creationflags=subprocess.CREATE_NO_WINDOW).decode().strip()
 log = O / f'{number}-tests.jsonl'
 sha = run([GO,'test','-count=1','-p=1','-json',*packages,'-run',pattern],directory,log)
 events = [json.loads(s) for s in log.read_text(encoding='utf-8').splitlines() if s.startswith('{')]
 assert not [e for e in events if e['Action']=='fail']
 tests = [e for e in events if e['Action']=='pass' and e.get('Test')]
 skips = [e for e in events if e['Action']=='skip' and e.get('Test')]
 build = O / f'{number}-build.log'
 bsha = run([GO,'build','-p=1','./...'],directory,build)
 entry = {'pr':number,'head':head,'pass_events':len(tests),'skip_events':len(skips),'pg_skip_not_claimed_as_pass':True,'test_log':str(log),'test_sha256':sha,'build_all':'PASS','build_log_sha256':bsha}
 receipt['runs'].append(entry)
 (O/'receipt.json').write_text(json.dumps(receipt,ensure_ascii=False,indent=2)+'\n',encoding='utf-8')
 print(json.dumps(entry),flush=True)
