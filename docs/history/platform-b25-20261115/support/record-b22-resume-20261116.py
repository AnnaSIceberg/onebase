import datetime
import hashlib
import json
import pathlib

B = pathlib.Path(__file__).parent
R = pathlib.Path('C:/Users/a.sevrjukova/AppData/Local/Temp/claude/C--Users-a-sevrjukova-onebase/cd50d987-9739-4c93-a5be-68d748529225/scratchpad/отчёты-для-analysis')
native_path = B / 'b22-native-acl-xlsx-attempt3-20261116/receipt.json'
native = json.loads(native_path.read_text('utf-8'))
assert native['status'] == 'PASS' and all(native['checks'].values()) and all(native['cleanup'].values())
assert len(native['checks']) == 26
assert native['exports']['direct']['sha256'] == native['exports']['background']['sha256']
ci = json.loads((B / 'ci-repair-20261116.json').read_text('utf-8'))
acl = json.loads((B / 'b22-acl-testfix-20261116.json').read_text('utf-8'))
stamp = datetime.datetime.now().strftime('%Y%m%d-%H%M')
report_path = R / f'{stamp}-platform-B22-native-и-CI.md'
assert not report_path.exists()
sha = lambda path: hashlib.sha256(path.read_bytes()).hexdigest()
supplement = {
    'status': 'B22_WINDOWS_NATIVE_PASS_PG_CORRECTED_FIXTURE_PREBUILT_PENDING',
    'created_utc': datetime.datetime.now(datetime.timezone.utc).isoformat(),
    'runtime_product': native['runtime'], 'sdk_windows_sha256': native['sdk_sha256'],
    'immutable_build_manifest': str(B / 'manifest-1115b22-report-acl-list-xlsx-DEV.json'),
    'immutable_build_manifest_sha256': sha(B / 'manifest-1115b22-report-acl-list-xlsx-DEV.json'),
    'native': {'receipt': str(native_path), 'sha256': sha(native_path), 'checks': 26, 'status': 'PASS', 'cleanup': native['cleanup'], 'helper_sha256': sha(B / 'b22-native-afterrelease-attempt3-20261116.py'), 'exports': native['exports']},
    'source_heads': ci['pull_requests'] + [acl],
    'ci': 'New HEAD checks have not appeared yet. No full-CI/Ready claim.',
    'postgres': 'Independent exact-name PG gate remains pending; old UI prebuilt has the known SELECT fixture, must not be retagged.',
    'scope': {'shared_changes': False, 'db_copies': 0, 'local_go_started': False, 'recursive_disk_search': False, 'b26_not_duplicated': True},
}
manifest_path = B / f'b22-native-resume-{stamp}.json'
manifest_path.write_text(json.dumps(supplement, ensure_ascii=False, indent=2) + '\n', encoding='utf-8')
rows = '\n'.join(f"| #{p['number']} | `{p['head']}` |" for p in ci['pull_requests'])
report = f'''# platform — B22 native PASS; исправления исходных Draft PR

Возобновление по её поручению 16.11. Общие/платформенные вводные, CLAUDE.md целиком, CONTRIBUTING.md, checkpoint 17:45 и хвост журнала прочитаны. Рекурсивных поисков по диску не запускал; только точные пути и Git metadata. После восстановления exec CPU 1,6%, свободная RAM 2147602432 байт. Новых Go-сборок в этом отчёте нет.

B22 exact product `{native['runtime']}`, готовый Windows SDK `{native['sdk_sha256']}`: **26 именованных проверок PASS, JS0**, SHA бинаря проверен отдельно; Chrome 109.0.5414.46, собственная SQLite без копирования БД. Настоящие ACL: direct/query/queryless/rethrow → 403 без выдачи данных; текстовая имитация ACL и ошибка запроса → 500; разрешённые direct/viewer → 200. Прямой публичный XLSX-маршрут и настоящий клик Excel → фоновая задача → download дают одну и ту же книгу: A1:B28, 28 строк, итог 253, 56 заполненных ячеек, openpyxl 3.1.5 normal=read_only по координатам/значениям/типам, без reset_dimensions и forced bounds. XLSX SHA `{native['exports']['direct']['sha256']}`.

Нативный сбой 15.11 был в наблюдателе: helper искал несуществующую прямую ссылку `/B22Ordinary/excel`, тогда как toolbar использует lower-case `/export/excel`. Исправлен только helper: direct проверяется публичным HTTP, background — реальным кликом по toolbar. SDK/продукт не менялись. Сохранён отдельный sandbox IPC FAIL attempt2 (WinError5), затем ограниченный loopback smoke вне песочницы PASS attempt3. Browser/context/driver/app/port18574 закрыты, own-auth DB удалена, tunnel не использовался.

Причины старого красного CI сверены по GitHub check annotations. В существующие Draft PR отправлены узкие исправления, HEAD и remote проверены, WT clean:

| Draft | Новый HEAD |
|---|---|
{rows}
| #2631 | `{acl['head']}` |

#2632: обработаны Close ошибок XLSX readers; opt-in путь тестовых артефактов отмечен как заданный доверенным test runner (G703), assertions сохранены. #2629: эквивалентный De Morgan + обработка Close. #2630: существующий комментарий об экранировании перенесён на правильный template.HTML cast. #2631: сохранённое B22 test-only исправление a1ac перенесено в source: `ВЫБРАТЬ (` действительно ошибочно и в PostgreSQL, в отличие от голого SELECT. Продуктовые изменения в этих четырёх коммитах отсутствуют (П7 условие эквивалентно). gofmt/diff-check; новые remote CI пока не появились — GREEN/Ready не заявлены.

**Открыто по B22:** independent exact-name PG gate UI190+interpreter6. Старый UI prebuilt содержит прежний fixture: его нельзя объявить зелёным или ретегировать. Следующий шаг — отдельный test-only prebuilt от сохранённого a1ac, затем independent gate; production/shared установки не выполнялись. П4/П7 имеют отдельные сохранённые B21 PG178/Windows19 PASS; не выдаю их за новый CI этих HEAD.

**Далее:** T8/П25 штатная многолистная книга; затем оставшаяся проверка П4/П7 и П8–П25 по утреннему заказу perenos. B26: height source/runtime WT release-claude обнаружены только по Git metadata, нового Draft по точной ветке пока нет; правку не дублировал. Очередь владельца последняя.

Manifest supplement: `{manifest_path}`, SHA `{sha(manifest_path)}`. Native receipt `{native_path}`, SHA `{sha(native_path)}`. Исходный B22 manifest и исторические FAIL сохранены. Отчёт/новые helper sources/квитанции подлежат отдельному WIP push, квитанция push будет отдельным дополнением.

Generated-with: Codex
'''
report_path.write_text(report, encoding='utf-8')
files = [{'source': str(report_path), 'target': 'reports/' + report_path.name},
         {'source': str(manifest_path), 'target': 'support/' + manifest_path.name},
         {'source': str(native_path), 'target': 'support/b22-native-attempt3-20261116-receipt.json'}]
for name in ['ci-repair-20261116.py', 'ci-repair-20261116.json', 'push-b22-acl-testfix-20261116.py', 'b22-acl-testfix-20261116.json', 'repair-b22-native-helper-20261116.py', 'b22-native-afterrelease-attempt2-20261116.py', 'b22-native-afterrelease-attempt3-20261116.py', 'resource-sample-20261116.py', 'record-b22-resume-20261116.py']:
    files.append({'source': str(B / name), 'target': 'support/' + name})
file_list = B / f'support-{stamp}-files.json'
file_list.write_text(json.dumps(files, ensure_ascii=False, indent=2) + '\n', encoding='utf-8')
print(json.dumps({'report': str(report_path), 'supplement': str(manifest_path), 'support_file_list': str(file_list), 'support_tag': stamp}, ensure_ascii=False))
