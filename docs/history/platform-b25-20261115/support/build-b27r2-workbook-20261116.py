import collections
import datetime
import hashlib
import json
import os
import pathlib
import subprocess

B = pathlib.Path(__file__).parent
W = B / 'workbook-b27-runtime'
D = B / 'b27r2-workbook-build-20261116'
D.mkdir(exist_ok=False)
GO = 'C:/Users/a.sevrjukova/go/pkg/mod/golang.org/toolchain@v0.0.1-go1.26.11.windows-amd64/bin/go.exe'
HEAD = 'a10677b2903e4b94cefb8d90a4104fe8a24aa639'
BASE = '513274b7e1bc230d8c0de5fcd22a34deb672345c'
SOURCE = '9b8db2dc37a4d799bfaaeef0cda1d0512168194b'
files = ['README.md', 'QUICKSTART.md', 'docs/dsl-reference.md', 'internal/dsl/langref/methods.go', 'docs/features.md', 'examples/tabular-workbook/README.md', 'internal/sheet/workbook.go', 'internal/sheet/workbook_test.go', 'internal/sheet/xlsx.go', 'internal/dsl/interpreter/spreadsheet_document.go', 'internal/dsl/interpreter/spreadsheet_workbook.go', 'internal/dsl/interpreter/spreadsheet_workbook_test.go']
env = dict(os.environ, GOTOOLCHAIN='local', GOMAXPROCS='2', GOFLAGS='-p=1', GOWORK='off', CGO_ENABLED='0', GOARCH='amd64', GOOS='windows')
for key in ['DATABASE_URL', 'TEST_DATABASE_URL', 'ONEBASE_TEST_WORKBOOK_LOG']:
    env.pop(key, None)

def git(*args):
    return subprocess.check_output(['git', '-c', 'gc.auto=0', *args], cwd=W, creationflags=subprocess.CREATE_NO_WINDOW, timeout=90).decode('utf-8').strip()

def sha(path):
    with pathlib.Path(path).open('rb') as stream:
        return hashlib.file_digest(stream, 'sha256').hexdigest()

def check():
    assert git('rev-parse', 'HEAD') == HEAD and not git('status', '--porcelain')

check()
assert set(git('diff', '--name-only', BASE, HEAD).splitlines()) == set(files)
for file in [f for f in files if f.endswith('.go') and f != 'internal/dsl/langref/methods.go']:
    assert git('rev-parse', HEAD + ':' + file) == git('rev-parse', SOURCE + ':' + file)
receipt = {'runtime': HEAD, 'base_b26': BASE, 'source_t8': SOURCE, 'draft': 2635, 'changed_files': files, 'core_go_files_exact_source': True, 'commands': [], 'artifacts': {}, 'shared_changes': False, 'db_copies': 0, 'status': 'RUNNING'}

def save():
    (D / 'manifest.json').write_text(json.dumps(receipt, indent=2) + '\n', encoding='utf-8')

def run(args, name, target='windows'):
    check()
    path = D / name
    start = datetime.datetime.now(datetime.timezone.utc).isoformat()
    with path.open('xb') as log:
        p = subprocess.run([GO, *args], cwd=W, env=dict(env, GOOS=target), stdout=log, stderr=subprocess.STDOUT, creationflags=subprocess.CREATE_NO_WINDOW, timeout=600)
    check()
    receipt['commands'].append({'argv': [GO, *args], 'goos': target, 'exit': p.returncode, 'log': str(path), 'sha256': sha(path), 'start': start, 'end': datetime.datetime.now(datetime.timezone.utc).isoformat(), 'gomaxprocs': 2, 'parallelism': 1})
    save()
    print(json.dumps({'step': name, 'exit': p.returncode}), flush=True)
    if p.returncode:
        raise SystemExit(p.returncode)
    return path

test_log = run(['test', '-count=1', '-p=1', '-json', './internal/sheet', './internal/dsl/interpreter', '-run', 'Test.*(Workbook|XLSX|Xlsx|Outline)'], 'xlsx-tests.jsonl')
ui_log = run(['test', '-count=1', '-p=1', '-json', './internal/ui', '-run', '^(TestShellSidePanelListFillsPanelHeight|TestTabularReportResultHTTPMatrix|TestTabularXLSXDimensionHTTPMatrix)$'], 'http-critical-tests.jsonl')
for key, path in [('xlsx', test_log), ('http', ui_log)]:
    events = [json.loads(line) for line in path.read_text('utf-8').splitlines() if line.startswith('{')]
    assert not any(e.get('Action') == 'fail' for e in events)
    results = {e['Test']: e['Action'] for e in events if e.get('Test') and e.get('Action') in ['pass', 'skip']}
    leaves = {name: action for name, action in results.items() if not any(other.startswith(name + '/') for other in results)}
    receipt[key + '_counts'] = dict(collections.Counter(leaves.values()))
    receipt[key + '_leaves'] = leaves
    if key == 'http':
        assert results.get('TestShellSidePanelListFillsPanelHeight') == 'pass'
run(['build', '-p=1', './...'], 'build-all.log')
run(['run', '-p=1', './tools/i18ncheck'], 'i18n-en.log')
for target, suffix in [('windows', 'windows-amd64.exe'), ('linux', 'linux-amd64')]:
    artifact = B / ('onebase-1116b27r2-workbook-' + suffix)
    assert not artifact.exists()
    run(['build', '-p=1', '-ldflags=-X github.com/ivantit66/onebase/internal/version.Build=dev-reports-b27r2-workbook', '-o', str(artifact), './cmd/onebase'], target + '-sdk-build.log', target)
    info = run(['version', '-m', str(artifact)], target + '-sdk-buildinfo.txt', target)
    receipt['artifacts'][target] = {'path': str(artifact), 'sha256': sha(artifact), 'bytes': artifact.stat().st_size, 'buildinfo': str(info), 'buildinfo_sha256': sha(info), 'runtime': HEAD, 'source_t8': SOURCE, 'embedded_vcs_not_claimed': True}
    save()
receipt['status'] = 'BUILD_AND_TARGETED_TESTS_PASS_SDK_READY_NATIVE_PENDING'
receipt['ended_utc'] = datetime.datetime.now(datetime.timezone.utc).isoformat()
save()
print(json.dumps({'status': receipt['status'], 'xlsx': receipt['xlsx_counts'], 'http': receipt['http_counts'], 'artifacts': receipt['artifacts']}), flush=True)
