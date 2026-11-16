import collections
import datetime
import hashlib
import json
import os
import pathlib
import subprocess

B = pathlib.Path(__file__).parent
W = B / 'reports-after-release-b22-runtime'
D = B / 'b22-corrected-ui-tests-attempt2-20261116'
D.mkdir(exist_ok=False)
GO = 'C:/Users/a.sevrjukova/go/pkg/mod/golang.org/toolchain@v0.0.1-go1.26.11.windows-amd64/bin/go.exe'
product = '1496c60ccad535d722bb1d83d2694d7d81bfbded'
source = 'a1ac5b8af8624b05c81bf8f074555578bb3df52e'

def run(args, **kw):
    return subprocess.run(args, cwd=W, creationflags=subprocess.CREATE_NO_WINDOW, check=True, timeout=480, **kw)

def git(*args):
    return run(['git', '-c', 'gc.auto=0', *args], stdout=subprocess.PIPE, stderr=subprocess.PIPE).stdout.decode('utf-8').strip()

assert git('rev-parse', 'HEAD') == source
assert not git('status', '--porcelain')
assert git('diff', '--name-only', product, source) == 'internal/ui/report_result_access_test.go'
env = dict(os.environ, GOTOOLCHAIN='local', GOMAXPROCS='2', GOFLAGS='-p=1', GOWORK='off', CGO_ENABLED='0', GOARCH='amd64', GOOS='windows')
for key in ['DATABASE_URL', 'TEST_DATABASE_URL']:
    env.pop(key, None)
roots = ['TestReportResultQueryAccessStatusMatrix', 'TestOrdinaryReportXLSXUsedRangeMatrix']
regex = '^(' + '|'.join(roots) + ')$'
expected = json.loads((B / 'b22-ui_pg-expected.json').read_text('utf-8'))
local_expected = {name for name in expected['expected_subtests'] if name.split('/')[0] in roots and '/sqlite/' in name}
receipt = {'product_runtime': product, 'test_source': source, 'test_only_delta': ['internal/ui/report_result_access_test.go'], 'started_utc': datetime.datetime.now(datetime.timezone.utc).isoformat(), 'commands': []}

def logged(args, name, os_name):
    path = D / name
    command = [GO, *args]
    started = datetime.datetime.now(datetime.timezone.utc).isoformat()
    with path.open('xb') as log:
        result = subprocess.run(command, cwd=W, env=dict(env, GOOS=os_name), stdout=log, stderr=subprocess.STDOUT, creationflags=subprocess.CREATE_NO_WINDOW, timeout=480)
    receipt['commands'].append({'argv': command, 'goos': os_name, 'gomaxprocs': 2, 'go_parallelism': 1, 'exit': result.returncode, 'log': str(path), 'log_sha256': hashlib.sha256(path.read_bytes()).hexdigest(), 'started_utc': started, 'ended_utc': datetime.datetime.now(datetime.timezone.utc).isoformat()})
    (D / 'receipt.json').write_text(json.dumps(receipt, indent=2) + '\n', encoding='utf-8')
    print(json.dumps({'step': name, 'exit': result.returncode}), flush=True)
    assert result.returncode == 0
    return path

local_log = logged(['test', '-count=1', '-p=1', '-json', './internal/ui', '-run', regex], 'local-targeted.jsonl', 'windows')
events = [json.loads(line) for line in local_log.read_text('utf-8').splitlines() if line.startswith('{')]
assert not any(e.get('Action') == 'fail' for e in events)
passed = {e['Test'] for e in events if e.get('Action') == 'pass' and e.get('Test')}
leaves = {name for name in passed if not any(other.startswith(name + '/') for other in passed)}
assert leaves == local_expected
receipt['local_sqlite_pass'] = sorted(leaves)
receipt['local_pg_skip'] = sorted(e['Test'] for e in events if e.get('Action') == 'skip' and e.get('Test'))
artifact = D / 'b22-ui-fixture-corrected-linux-amd64.test'
logged(['test', '-c', '-p=1', '-o', str(artifact), './internal/ui'], 'linux-prebuilt-build.log', 'linux')
buildinfo = logged(['version', '-m', str(artifact)], 'linux-prebuilt-buildinfo.txt', 'linux')
assert not git('status', '--porcelain') and git('rev-parse', 'HEAD') == source
with artifact.open('rb') as stream:
    artifact_sha = hashlib.file_digest(stream, 'sha256').hexdigest()
receipt.update({'status': 'LOCAL_SQLITE12_PASS_LINUX_PREBUILT_READY_INDEPENDENT_PG_PENDING', 'artifact': {'path': str(artifact), 'sha256': artifact_sha, 'bytes': artifact.stat().st_size}, 'expected_names': expected['expected_subtests'], 'expected_total': expected['expected_total'], 'regex': expected['regex'], 'buildinfo': str(buildinfo), 'embedded_vcs_not_claimed': True, 'clean_before_after': True, 'ended_utc': datetime.datetime.now(datetime.timezone.utc).isoformat()})
assert len(leaves) == 12 and expected['expected_total'] == 190
(D / 'receipt.json').write_text(json.dumps(receipt, indent=2) + '\n', encoding='utf-8')
print(json.dumps({'status': receipt['status'], 'artifact_sha256': artifact_sha, 'expected': 190}), flush=True)
