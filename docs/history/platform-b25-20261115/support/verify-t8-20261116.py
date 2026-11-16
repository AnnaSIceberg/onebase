import collections
import datetime
import hashlib
import json
import os
import pathlib
import re
import base64
import subprocess

B = pathlib.Path(__file__).parent
W = pathlib.Path('C:/Users/a.sevrjukova/platform-workbook-xlsx-20261116')
D = B / 't8-verify-attempt1-20261116'
D.mkdir(exist_ok=False)
GO = 'C:/Users/a.sevrjukova/go/pkg/mod/golang.org/toolchain@v0.0.1-go1.26.11.windows-amd64/bin/go.exe'
H = '84f146757cb91f353067cb03ab5e34ccc1d4c6ac'
env = dict(os.environ, GOTOOLCHAIN='local', GOMAXPROCS='2', GOFLAGS='-p=1', GOWORK='off', CGO_ENABLED='0', GOOS='windows', GOARCH='amd64', ONEBASE_TEST_WORKBOOK_LOG='1')
for name in ['DATABASE_URL', 'TEST_DATABASE_URL']:
    env.pop(name, None)

def git(*args):
    return subprocess.check_output(['git', '-c', 'gc.auto=0', *args], cwd=W, creationflags=subprocess.CREATE_NO_WINDOW, timeout=90).decode('utf-8').strip()

def check():
    assert git('rev-parse', 'HEAD') == H and not git('status', '--porcelain')

receipt = {'source': H, 'pr': 2635, 'started_utc': datetime.datetime.now(datetime.timezone.utc).isoformat(), 'commands': [], 'database_access': False, 'sdk_rebuilt': False}

def save():
    (D / 'receipt.json').write_text(json.dumps(receipt, ensure_ascii=False, indent=2) + '\n', encoding='utf-8')

def run(args, name):
    check()
    path = D / name
    start = datetime.datetime.now(datetime.timezone.utc).isoformat()
    with path.open('xb') as log:
        p = subprocess.run([GO, *args], cwd=W, env=env, stdout=log, stderr=subprocess.STDOUT, creationflags=subprocess.CREATE_NO_WINDOW, timeout=600)
    check()
    receipt['commands'].append({'argv': [GO, *args], 'exit': p.returncode, 'log': str(path), 'sha256': hashlib.sha256(path.read_bytes()).hexdigest(), 'gomaxprocs': 2, 'parallelism': 1, 'start': start, 'end': datetime.datetime.now(datetime.timezone.utc).isoformat()})
    save()
    print(json.dumps({'step': name, 'exit': p.returncode}), flush=True)
    if p.returncode:
        raise SystemExit(p.returncode)
    return path

tests = run(['test', '-count=1', '-p=1', '-json', './internal/sheet', './internal/dsl/interpreter', '-run', 'Test.*(Workbook|XLSX|Xlsx|Outline)'], 'xlsx-and-workbook-tests.jsonl')
events = [json.loads(line) for line in tests.read_text('utf-8').splitlines() if line.startswith('{')]
results = {(e['Package'], e['Test']): e['Action'] for e in events if e.get('Test') and e.get('Action') in ['pass', 'fail', 'skip']}
leaves = {key: value for key, value in results.items() if not any(other[0] == key[0] and other[1].startswith(key[1] + '/') for other in results)}
assert leaves and set(leaves.values()) == {'pass'}
receipt['leaf_counts'] = dict(collections.Counter(leaves.values()))
receipt['leaf_names'] = [list(key) for key in sorted(leaves)]
encoded = [re.search(r'WORKBOOK_XLSX_BASE64:([A-Za-z0-9+/=]+)', e.get('Output', '')) for e in events]
encoded = [match.group(1) for match in encoded if match]
assert len(encoded) == 1
artifact = D / 'two-named-snapshots.xlsx'
artifact.write_bytes(base64.b64decode(encoded[0], validate=True))
import openpyxl
checks = []
for read_only in [False, True]:
    book = openpyxl.load_workbook(artifact, read_only=read_only, data_only=True)
    assert book.sheetnames == ['СМ', 'ХД']
    for name, number, color in [('СМ', 111, 'FF0000'), ('ХД', 222, '00FF00')]:
        cell = book[name]['A1']
        assert cell.value == number and type(cell.value) in (int, float)
        assert cell.fill.fgColor.rgb[-6:].upper() == color
        checks.append({'read_only': read_only, 'sheet': name, 'value': cell.value, 'type': type(cell.value).__name__, 'fill': cell.fill.fgColor.rgb, 'dimension': book[name].calculate_dimension()})
    book.close()
receipt['independent_reader'] = {'version': openpyxl.__version__, 'checks': checks, 'reset_dimensions': False, 'forced_bounds': False, 'artifact': str(artifact), 'sha256': hashlib.sha256(artifact.read_bytes()).hexdigest()}
save()
run(['build', '-p=1', './...'], 'build-all.log')
run(['run', '-p=1', './tools/i18ncheck'], 'i18n-en.log')
receipt['status'] = 'TARGETED_TESTS_AND_BUILD_ALL_PASS_OPENPYXL_PASS'
receipt['ended_utc'] = datetime.datetime.now(datetime.timezone.utc).isoformat()
save()
print(json.dumps({'status': receipt['status'], 'counts': receipt['leaf_counts'], 'reader_checks': len(checks)}), flush=True)
