import datetime
import json
import pathlib
import subprocess

B = pathlib.Path(__file__).parent
W = pathlib.Path('C:/Users/a.sevrjukova/platform2-report-acl-source')
branch = 'fix/report-result-access-status-20261115'
parent = 'a00d323898c6ca40ab093eca8319dae812faf90e'
file = 'internal/ui/report_result_access_test.go'
out = B / 'b22-acl-testfix-20261116.json'
assert not out.exists()

def run(args):
    return subprocess.check_output(args, cwd=W, creationflags=subprocess.CREATE_NO_WINDOW, timeout=120).decode('utf-8').strip()

def git(*args):
    return run(['git', '-c', 'gc.auto=0', '-c', 'http.sslBackend=openssl', *args])

run(['C:/Users/a.sevrjukova/go/pkg/mod/golang.org/toolchain@v0.0.1-go1.26.11.windows-amd64/bin/gofmt.exe', '-w', file])
assert git('rev-parse', 'HEAD') == parent
assert git('branch', '--show-current') == branch
assert git('ls-remote', 'fork', 'refs/heads/' + branch).split()[0] == parent
pr = json.loads(run(['C:/Users/a.sevrjukova/gh-cli/bin/gh.exe', 'pr', 'view', '2631', '--repo', 'ivanarama/onebase', '--json', 'headRefOid,isDraft,state,labels']))
assert pr['headRefOid'] == parent and pr['isDraft'] and pr['state'] == 'OPEN' and not pr['labels']
assert git('diff', '--name-only') == file
assert not git('diff', '--cached', '--name-only')
assert not git('ls-files', '--others', '--exclude-standard')
git('diff', '--check')
message = B / 'b22-acl-testfix-20261116-commit.txt'
message.write_text('test(report): сделать ошибочный запрос неверным на обеих СУБД\n\nПеренос сохранённого исправления fixture B22 a1ac5b8a в исходный Draft PR. Продуктовый код не меняется.\n\nGenerated-with: Codex\n', encoding='utf-8')
git('add', '--', file)
git('commit', '-F', str(message))
head = git('rev-parse', 'HEAD')
assert git('rev-parse', 'HEAD^') == parent
git('push', 'fork', 'HEAD:refs/heads/' + branch)
assert git('ls-remote', 'fork', 'refs/heads/' + branch).split()[0] == head
assert not git('status', '--porcelain')
receipt = {'pr': 2631, 'head': head, 'parent': parent, 'branch': branch, 'file': file, 'product_delta': False, 'remote_exact': True, 'clean': True, 'created_utc': datetime.datetime.now(datetime.timezone.utc).isoformat()}
out.write_text(json.dumps(receipt, ensure_ascii=False, indent=2) + '\n', encoding='utf-8')
print(json.dumps(receipt))
