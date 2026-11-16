import datetime
import json
import pathlib
import subprocess

B = pathlib.Path(__file__).parent
receipt_path = B / 'ci-repair-20261116.json'
assert not receipt_path.exists()
tasks = [
    ('C:/Users/a.sevrjukova/platform2-list-xlsx-source', 'fix/list-xlsx-used-range-20261115', 'db1b058739a594bcf21e29c3327cdff3440c4dfc', 2632,
     ['internal/excel/dimension_test.go', 'internal/ui/report_list_dimension_test.go'], 'test(excel): обработать закрытие XLSX и уточнить доверенный путь артефактов'),
    ('C:/Users/a.sevrjukova/platform2-report-context', 'feature/report-trusted-context-20261115', '02c62ecd322eb423b78dd0ebe66d999e384db5dd', 2629,
     ['internal/report/context.go', 'internal/ui/report_context_test.go'], 'fix(report): закрыть замечания CI доверенного контекста'),
    ('C:/Users/a.sevrjukova/platform2-report-drilldown', 'feature/report-typed-drilldown-20261115', 'cf5cf1005e75aedf3f766436b0412a053c7be19b', 2630,
     ['internal/ui/report_drilldown.go'], 'fix(report): привязать пояснение экранирования к HTML расшифровки'),
]
receipt = {'started_utc': datetime.datetime.now(datetime.timezone.utc).isoformat(), 'local_go_or_browser_started': False, 'pull_requests': []}

def run(args, cwd=None):
    result = subprocess.run(args, cwd=cwd, stdout=subprocess.PIPE, stderr=subprocess.PIPE,
                            creationflags=subprocess.CREATE_NO_WINDOW, timeout=120)
    if result.returncode:
        raise RuntimeError(f'command failed: {args[0]} {args[1]} ({result.returncode})')
    return result.stdout.decode('utf-8').strip()

for directory, branch, parent, number, files, subject in tasks:
    def git(*args):
        return run(['git', '-c', 'gc.auto=0', '-c', 'http.sslBackend=openssl', *args], directory)
    assert git('rev-parse', 'HEAD') == parent
    assert git('branch', '--show-current') == branch
    assert git('ls-remote', 'fork', 'refs/heads/' + branch).split()[0] == parent
    pr = json.loads(run(['C:/Users/a.sevrjukova/gh-cli/bin/gh.exe', 'pr', 'view', str(number), '--repo', 'ivanarama/onebase', '--json', 'headRefOid,isDraft,state,labels']))
    assert pr['headRefOid'] == parent and pr['isDraft'] and pr['state'] == 'OPEN' and not pr['labels']
    assert set(git('diff', '--name-only').splitlines()) == set(files)
    assert not git('diff', '--cached', '--name-only')
    assert not git('ls-files', '--others', '--exclude-standard')
    git('diff', '--check')
    message = B / f'ci-repair-{number}-commit.txt'
    message.write_text(subject + '\n\nGenerated-with: Codex\n', encoding='utf-8')
    git('add', '--', *files)
    assert set(git('diff', '--cached', '--name-only').splitlines()) == set(files)
    git('commit', '-F', str(message))
    head = git('rev-parse', 'HEAD')
    assert git('rev-parse', 'HEAD^') == parent
    git('push', 'fork', 'HEAD:refs/heads/' + branch)
    assert git('ls-remote', 'fork', 'refs/heads/' + branch).split()[0] == head
    assert not git('status', '--porcelain')
    receipt['pull_requests'].append({'number': number, 'branch': branch, 'parent': parent, 'head': head, 'files': files, 'remote_exact': True, 'clean': True, 'validation': 'gofmt + git diff --check; remote CI pending'})
    receipt_path.write_text(json.dumps(receipt, ensure_ascii=False, indent=2) + '\n', encoding='utf-8')
    print(json.dumps({'pr': number, 'head': head, 'pushed': True}), flush=True)
