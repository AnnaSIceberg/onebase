import datetime
import json
import pathlib
import subprocess

B = pathlib.Path(__file__).parent
W = pathlib.Path('C:/Users/a.sevrjukova/platform-workbook-xlsx-20261116')
branch = 'feature/tabular-workbook-xlsx-20261116'
parent = 'c63fe8e37'
files = ['docs/features.md', 'examples/tabular-workbook/README.md', 'internal/sheet/workbook.go', 'internal/sheet/workbook_test.go', 'internal/sheet/xlsx.go', 'internal/dsl/interpreter/spreadsheet_document.go', 'internal/dsl/interpreter/spreadsheet_workbook.go', 'internal/dsl/interpreter/spreadsheet_workbook_test.go']
out = B / 't8-draft-20261116.json'
assert not out.exists()
GH = 'C:/Users/a.sevrjukova/gh-cli/bin/gh.exe'

def run(args):
    return subprocess.check_output(args, cwd=W, creationflags=subprocess.CREATE_NO_WINDOW, timeout=150).decode('utf-8').strip()

def git(*args):
    return run(['git', '-c', 'gc.auto=0', '-c', 'http.sslBackend=openssl', *args])

head_before = git('rev-parse', 'HEAD')
assert head_before.startswith(parent)
assert git('branch', '--show-current') == branch
assert not git('ls-remote', 'fork', 'refs/heads/' + branch)
assert json.loads(run([GH, 'pr', 'list', '--repo', 'ivanarama/onebase', '--state', 'all', '--head', branch, '--json', 'number'])) == []
git('diff', '--check')
git('add', '--', *files)
assert set(git('diff', '--cached', '--name-only').splitlines()) == set(files)
assert not git('ls-files', '--others', '--exclude-standard')
git('diff', '--cached', '--check')
message = B / 't8-feature-20261116-commit.txt'
message.write_text('feat(sheet): собрать именованные ТабДок в одну XLSX-книгу\n\nT8/П25: server-side снимки листов, общий XLSX writer, DSL API и общие лимиты. Draft до проверки DSL/геометрии и объединённой сборки.\n\nGenerated-with: Codex\n', encoding='utf-8')
git('commit', '-F', str(message))
head = git('rev-parse', 'HEAD')
assert git('rev-parse', 'HEAD^') == head_before
receipt = {'branch': branch, 'head': head, 'parent': head_before, 'base_main': 'adddaa8d95f04b4b7b3ff57991edbbd9b8740fa2', 'dependency': 'Draft #2627 + test-only 0594980b', 'files': files, 'status': 'COMMITTED_PUSH_PENDING', 'created_utc': datetime.datetime.now(datetime.timezone.utc).isoformat()}
out.write_text(json.dumps(receipt, indent=2) + '\n', encoding='utf-8')
git('push', 'fork', 'HEAD:refs/heads/' + branch)
assert git('ls-remote', 'fork', 'refs/heads/' + branch).split()[0] == head
assert not git('status', '--porcelain')
receipt['status'] = 'PUSHED_DRAFT_CREATE_PENDING'
out.write_text(json.dumps(receipt, indent=2) + '\n', encoding='utf-8')
body = B / 't8-draft-20261116-body.md'
body.write_text('''Служебная записка T8/П25 по нескольким направлениям должна быть одной книгой с отдельным листом на направление. Добавлен серверный `КнигаТабличныхДокументов` / `SpreadsheetWorkbook`: `ДобавитьЛист(Имя, ТабДок)` сохраняет снимок, `Записать(Имя, "xlsx")` возвращает base64 одной книги.

**Stacked Draft, зависит от #2627.** Свежий upstream main `adddaa8d` объединён с существующим источником П1 и его test-only исправлением `0594980b`. Родительский XLSX writer ещё не в main; до его принятия в diff видна зависимость. Перед Ready нужно отделить принятую базу и подтвердить оставшийся самостоятельный результат T8. П1 заново не реализуется.

Все листы пишет один Excelize file через общий писатель одиночного ТабДок: типы значений, стили, объединения, размеры, печать/fit width, freeze, повтор шапки и группы. Никаких COM, копирования отдельных XLSX-книг, файловых путей или внешних отправок. Имена явные, с отказом при ошибке и совпадении регистра; лимиты проверяются на всю книгу, включая разреженные прямоугольники и работу групп.

Добавлены проверки через опубликованный DSL-вход: СМ/ХД, числа111/222, красный/зелёный, снимок при переиспользовании ТабДок; параметры листов и отрицательные входы. Публичный пример/контракт — `examples/tabular-workbook/README.md`.

Проверка пока в процессе: gofmt и diff-check выполнены, Go/tests/SDK для нового T8 ещё не запускались. Draft оставлен до проверки контракта, сборки и CI. Стенды, конфигурации КЦ и внешние выходы не переключались.
''', encoding='utf-8')
url = run([GH, 'pr', 'create', '--repo', 'ivanarama/onebase', '--base', 'main', '--head', 'AnnaSIceberg:' + branch, '--draft', '--title', 'feat(sheet): одна XLSX-книга из именованных ТабДок (T8/П25)', '--body-file', str(body)])
pr = json.loads(run([GH, 'pr', 'view', url, '--repo', 'ivanarama/onebase', '--json', 'number,url,isDraft,headRefOid']))
assert pr['isDraft'] and pr['headRefOid'] == head
receipt.update({'status': 'DRAFT_PUBLISHED_TESTS_PENDING', 'pull_request': pr})
out.write_text(json.dumps(receipt, ensure_ascii=False, indent=2) + '\n', encoding='utf-8')
print(json.dumps({'number': pr['number'], 'url': pr['url'], 'head': head, 'draft': True}))
