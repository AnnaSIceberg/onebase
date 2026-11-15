import collections
import datetime
import hashlib
import json
import os
import pathlib
import subprocess
import sys

B = pathlib.Path(__file__).parent
WT = B / 'report-tabular-result-runtime'
GO = pathlib.Path('C:/Users/a.sevrjukova/go/pkg/mod/golang.org/toolchain@v0.0.1-go1.26.11.windows-amd64/bin/go.exe')
HEAD = 'cd9910c81fd7812874b9ca2cafa1870857abee5e'
SOURCE = '815e1505f504659fedea159fd186eeff3ab36d33'
BASE = '64886e644da31991abd93d48390aaff7b883e9bc'
PG_ROOTS = ['TestTabularPayloadHTTPMatrix', 'TestTabularLayoutReportHTTPMatrix', 'TestTabularReportResultHTTPMatrix', 'TestTabularManagedFieldHTTPMatrix', 'TestTabularDrilldownHTTPMatrix', 'TestTabularSizeHTTPMatrix', 'TestReportFlatColumnsHTTP', 'TestTabularAutoScaleHTTPMatrix', 'TestTabularQueryInputLimitHTTPMatrix']
PG_REGEX = '^(' + '|'.join(PG_ROOTS) + ')$'
env = dict(os.environ, GOTOOLCHAIN='go1.26.11', GOMAXPROCS='2', CGO_ENABLED='0', GOOS='windows', GOARCH='amd64')
for key in ['TEST_DATABASE_URL', 'DATABASE_URL']:
    env.pop(key, None)

def git(*args):
    return subprocess.check_output(['git', *args], cwd=WT, text=True, encoding='utf-8', creationflags=subprocess.CREATE_NO_WINDOW).strip()

def check():
    if git('status', '--porcelain') or git('rev-parse', 'HEAD') != HEAD:
        raise SystemExit('STOP runtime worktree changed')

def digest(path):
    with path.open('rb') as f:
        return hashlib.file_digest(f, 'sha256').hexdigest()

def run(args, log_name, goos='windows'):
    check()
    log = B / log_name
    if log.exists():
        raise SystemExit('immutable log exists: ' + str(log))
    start = datetime.datetime.now(datetime.timezone.utc).isoformat()
    with log.open('xb') as f:
        p = subprocess.run([str(GO), *args], cwd=WT, env=dict(env, GOOS=goos), stdout=f, stderr=subprocess.STDOUT, creationflags=subprocess.CREATE_NO_WINDOW)
    check()
    receipt = {'source': SOURCE, 'runtime': HEAD, 'cwd': str(WT), 'argv': [str(GO), *args], 'env': {'GOTOOLCHAIN':'go1.26.11','GOMAXPROCS':'2','CGO_ENABLED':'0','GOOS':goos,'GOARCH':'amd64'}, 'start_utc':start, 'end_utc':datetime.datetime.now(datetime.timezone.utc).isoformat(), 'exit':p.returncode, 'log':str(log),'log_sha256':digest(log),'worktree_clean_before_after':True}
    log.with_suffix(log.suffix + '.command.json').write_text(json.dumps(receipt, ensure_ascii=False, indent=2)+'\n',encoding='utf-8')
    print(json.dumps({'step':log_name,'exit':p.returncode,'sha256':receipt['log_sha256']}),flush=True)
    if p.returncode:
        raise SystemExit('FAILED ' + str(log))
    return log

def parse(log):
    rows = [json.loads(x) for x in log.read_text(encoding='utf-8').splitlines() if x.startswith('{')]
    tests = {(r['Package'],r['Test']):r['Action'] for r in rows if r.get('Test') and r['Action'] in ('pass','fail','skip')}
    leaves = {k:v for k,v in tests.items() if not any(q[0]==k[0] and q[1].startswith(k[1]+'/') for q in tests)}
    return {'log':str(log),'sha256':digest(log),'all_named_counts':dict(collections.Counter(tests.values())),'leaf_counts':dict(collections.Counter(leaves.values())),'leaves':[{'package':p,'name':t,'result':v} for (p,t),v in sorted(leaves.items())]}

def build_artifact(key, package, goos, filename):
    path = B / filename
    if path.exists():
        raise SystemExit('immutable artifact exists ' + str(path))
    args = ['test','-p=1','-c','-o',str(path),package] if key=='ui_pg' else ['build','-p=1','-ldflags=-X github.com/ivantit66/onebase/internal/version.Build=dev-kpi-tabular-b19','-o',str(path),package]
    run(args,'b19-'+key+'-build.log',goos)
    info = run(['version','-m',str(path)],'b19-'+key+'-buildinfo.txt',goos)
    result = {'path':str(path),'sha256':digest(path),'bytes':path.stat().st_size,'runtime':HEAD,'source':SOURCE,'buildinfo':str(info),'buildinfo_sha256':digest(info)}
    (B/('b19-'+key+'-artifact.json')).write_text(json.dumps(result,indent=2)+'\n',encoding='utf-8')
    print(json.dumps({'artifact':key,**result}),flush=True)
    return result

check()
if sys.argv[1] == 'targeted-prebuilt':
    run(['build','-p=1','./...'],'b19-build-all.log')
    run(['run','./tools/i18ncheck','-lang','en'],'b19-i18n-en.log')
    results = []
    ui_regex = PG_REGEX
    ui = parse(run(['test','-p=1','-json','./internal/ui','-run',ui_regex],'b19-ui-targeted.jsonl'))
    results.append(ui)
    (B/'b19-local-targeted-summary.json').write_text(json.dumps({'runtime':HEAD,'source':SOURCE,'runs':results},ensure_ascii=False,indent=2)+'\n',encoding='utf-8')
    sqlite = [r['name'] for r in ui['leaves'] if r['result']=='pass' and '/sqlite' in r['name'] and r['name'].split('/')[0] in PG_ROOTS]
    assert len(sqlite)==59 and len(set(sqlite))==59, ('unexpected portable leaf mapping',sqlite)
    expected = sorted(sqlite + [name.replace('/sqlite','/postgres',1) for name in sqlite])
    (B/'b19-ui-pg-expected.json').write_text(json.dumps({'runtime':HEAD,'source':SOURCE,'regex':PG_REGEX,'roots':PG_ROOTS,'expected_subtests':expected,'expected_total':118,'sqlite':59,'postgres':59,'local_pg_status':'SKIP, NOT PASS'},ensure_ascii=False,indent=2)+'\n',encoding='utf-8')
    artifact = build_artifact('ui_pg','./internal/ui','linux','reports-kpi-p1-b19-ui-linux-amd64.test')
    (B/'b19-pg-handoff.json').write_text(json.dumps({'status':'IMMUTABLE_PREBUILT_ONLY_PG_PENDING','runtime':HEAD,'source':SOURCE,'base_b18':BASE,'artifact':artifact,'expected':str(B/'b19-ui-pg-expected.json'),'expected_sha256':digest(B/'b19-ui-pg-expected.json'),'regex':PG_REGEX,'expected_total':118,'sdk_status':'NOT_BUILT_YET','go_window':'STILL_ACTIVE_UNTIL_EXPLICIT_END'},indent=2)+'\n',encoding='utf-8')
elif sys.argv[1] == 'sdk':
    build_artifact('linux','./cmd/onebase','linux','onebase-1115b19-tabular-input-limit-linux-amd64')
    build_artifact('windows','./cmd/onebase','windows','onebase-1115b19-tabular-input-limit-windows-amd64.exe')
else:
    raise SystemExit('unknown stage')
