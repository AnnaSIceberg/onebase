import datetime
import hashlib
import json
import pathlib
import subprocess

B = pathlib.Path(__file__).parent
W = B / 'tabular-link-color-runtime'
branch = 'wip/platform-b24-final-b25-handoff-20261115'
head = '6e7df0a16f1f875a1b818d0b4bffec342152153b'
parent = '4d49b94f3a9859cc52493c3d1f08a771c5b7d620'
out = B / 'support-20261116-1032-push.json'
assert not out.exists()
files = json.loads((B / 'support-20261116-1032-files.json').read_text('utf-8'))

def git(*args):
    return subprocess.check_output(['git', '-c', 'gc.auto=0', '-c', 'http.sslBackend=openssl', *args], cwd=W, creationflags=subprocess.CREATE_NO_WINDOW, timeout=120)

assert git('rev-parse', branch).decode().strip() == head
assert git('rev-parse', head + '^').decode().strip() == parent
assert not git('status', '--porcelain').strip()
assert git('ls-remote', 'fork', 'refs/heads/' + branch).decode().split()[0] == parent
prefix = 'docs/history/platform-b25-20261115/'
assert set(git('diff', '--name-only', '-z', parent, head).decode().rstrip('\0').split('\0')) == {prefix + f['target'] for f in files}
inventory = []
for item in files:
    raw = pathlib.Path(item['source']).read_bytes()
    assert git('show', head + ':' + prefix + item['target']) == raw
    inventory.append({'path': item['target'], 'bytes': len(raw), 'sha256': hashlib.sha256(raw).hexdigest()})
git('push', 'fork', branch)
assert git('ls-remote', 'fork', 'refs/heads/' + branch).decode().split()[0] == head
receipt = {'branch': branch, 'commit': head, 'parent': parent, 'files': inventory, 'created_utc': datetime.datetime.now(datetime.timezone.utc).isoformat(), 'scope': 'source helpers/reports/metadata only; no SDK, DB or secrets', 'recovery': 'original commit retained; sandbox blocked Git credential-helper IPC, only the verified push was resumed'}
out.write_text(json.dumps(receipt, ensure_ascii=False, indent=2) + '\n', encoding='utf-8')
print(json.dumps({'commit': head, 'file_count': len(files), 'receipt_sha256': hashlib.sha256(out.read_bytes()).hexdigest()}))
