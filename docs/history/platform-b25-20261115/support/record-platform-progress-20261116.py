import datetime
import hashlib
import json
import pathlib

B = pathlib.Path(__file__).parent
R = pathlib.Path('C:/Users/a.sevrjukova/AppData/Local/Temp/claude/C--Users-a-sevrjukova-onebase/cd50d987-9739-4c93-a5be-68d748529225/scratchpad/отчёты-для-analysis')
proof_path = B / 't8-verify-attempt3-20261116/receipt.json'
proof = json.loads(proof_path.read_text('utf-8'))
assert proof['status'] == 'TARGETED_TESTS_AND_BUILD_ALL_PASS_OPENPYXL_PASS'
assert proof['source'] == '79aac9d7426497fac35ba00ea42ba95232f23ff5' and proof['leaf_counts'] == {'pass': 49}
stamp = datetime.datetime.now().strftime('%Y%m%d-%H%M')
sha = lambda p: hashlib.sha256(pathlib.Path(p).read_bytes()).hexdigest()
ledger = {
    'current_dev_baseline': {'name': 'B26', 'runtime': '513274b7e1bc230d8c0de5fcd22a34deb672345c', 'linux_sdk_sha256': 'e977a632cfefc6c1b34575b4ac7797aeb1c057090738f470243fb03b48b67e6d', 'deployment_provenance': 'release-claude report 20261116-1051; no platform live-install or independent SDK file hash claimed'},
    'b26_source_ci': {'pr': 2634, 'head': '2a0f4acad4df556dcd8d4676ef2967aa050fe687', 'parent': 'ed5636a5dd89387f91359b412c06240fecc1b5c2', 'own_branch': 'fix/side-panel-full-height-ci-20261116', 'source_branch': 'fix/side-panel-full-height-source', 'change': 'one en translation: Panel width', 'i18n_exit': 0, 'source_push_user_approved': True, 'runtime_rebuilt': False},
    't8_source': {'pr': 2635, 'head': proof['source'], 'proof': str(proof_path), 'proof_sha256': sha(proof_path), 'leaf_pass': 49, 'openpyxl_checks': 4, 'build_all': 'PASS', 'i18n_en': 'PASS'},
    'b27_candidate': {'runtime': '7b89c38bfab4d2aa2f2c8d1843fb66f83032cfd3', 'branch': 'integration/cc2-b27-workbook-20261116', 'base_b26': '513274b7e1bc230d8c0de5fcd22a34deb672345c', 'status': 'SDK_BUILD_ACTIVE_NATIVE_PENDING', 'contains': 'B26 + T8 only; B22/P4/P7 follow separately', 'installed': False},
    'b22': {'status': 'Windows native26 PASS; corrected UI prebuilt ready; independent PG196 pending', 'handoff': str(B / 'b22-corrected-pg-handoff-20261116-1047.json')},
    'source_pr_conflicts': [2630, 2629, 2631, 2632],
    'shared_changes_by_platform': False, 'recursive_disk_search': False,
}
ledger_path = B / f'platform-line-{stamp}.json'
ledger_path.write_text(json.dumps(ledger, ensure_ascii=False, indent=2) + '\n', encoding='utf-8')
report_path = R / f'{stamp}-platform-T8-source-B26-принят-B27-build.md'
report_path.write_text(f'''# platform: T8 source проверен; B26 принят, B27 SDK собирается

T8/П25 **Draft #2635**, source `{proof['source']}`. Через публичный DSL вход: **49 leaf PASS**, openpyxl3.1.5 **4/4** (normal/read_only × СМ/ХД, числовые111/222, красный/зелёный, безreset/bounds), build./... PASS, i18n en PASS. Реальный API: `Новый КнигаТабличныхДокументов` / `SpreadsheetWorkbook`, `.ДобавитьЛист(Имя, ТабДок)` / `.AddSheet`, `.Количество()` / `.Count`, `.Записать(Имя,"xlsx")` / `.Write` → base64 одной книги. На AddSheet сохраняется снимок, исходный ТабДок можно переиспользовать. Имена/дубликаты/UTF-8/границы Excel и общие бюджеты проверяются, детали в `examples/tabular-workbook/README.md`. Параметры листов, freeze/повтор шапки/группы проверены на обоих листах. Старый single-document writer использует тот же общий writer.

Первый RED был в ожидании теста: OOXML без атрибута t — стандартное число, Excelize называет его CellTypeUnset. Тест исправлен с сохранением запрета строковых типов; нормальный/потоковый openpyxl независимо подтвердил числа. Затем отдельно добавлен отказ для invalid UTF-8 имени, текущий79aac проверен полностью. Первые логи сохранены, не ретегированы.

Source proof `{proof_path}`, SHA `{sha(proof_path)}`. Draft зависит от П1 #2627; freshmain adddaa объединён с существующей зависимостью, единственный docs/features conflict разрешён сохранением обеих секций. FullCI/Ready не объявлены.

**B26 принят в линию.** Runtime513274b7 отB25, Linux e977a632… на :8444 установлен release-claude по его отчёту1051; platform не устанавливал. Draft #2634 CI падал только на en «Ширина панели». В собственной CI-ветке подготовлен и i18ncheck проверен один перевод; после явного ответа пользователя «Разрешаю push в #2634» source обновлён `2a0f4acad4df556dcd8d4676ef2967aa050fe687`, remote exact проверен. Это source-only CI fix; frozen B26 SDK/runtime не ретегированы. В B26 runtime этот перевод уже присутствует. Новые CI checks идут; fullGREEN пока не заявлен.

**B27 = B26 + T8**, own runtime `{ledger['b27_candidate']['runtime']}`, branch уже pushed. Только8 файлов T8, все6 Go blobs точно равны source79aac; CSS/панельB26 сохранены. Последовательно идёт GoM2/-p1: XLSX/DSL + HTTPcritical (height/TabDoc/dimension), buildall/i18n, Windows/Linux SDK. После GO END — собственный native HTTP/browser smoke. Кандидат не устанавливается. B22/П4/П7 пока отдельные линии, в B27 не подмешаны.

**B22:** native26 PASS, correctedprebuilt UI190 + неизменённыйinterpreter6 готовы по отчёту1047; independent PG196 ещё ждёт исполнения. Исходные Draft #2630/#2629/#2631/#2632 GitHub показывает CONFLICTING со свежим main, поэтому новые проверки пока не появились; после T8 SDK обновлю их последовательно, без изменения frozen SDK. Очередь владельца последняя.

Рекурсивных поисков по C: не было. Новых БД-копий/shared изменений/внешних отправок/расписаний нет. Work/source commits pushed, helpers и этот отчёт сохраняются следующим отдельным WIP push.

Ledger `{ledger_path}`, SHA `{sha(ledger_path)}`.

Generated-with: Codex
''', encoding='utf-8')
files = [{'source': str(report_path), 'target': 'reports/' + report_path.name}, {'source': str(ledger_path), 'target': 'support/' + ledger_path.name}]
for name in ['refactor-xlsx-for-workbook-20261116.py', 'refactor-xlsx-for-workbook-20261116.json', 'publish-t8-draft-20261116.py', 't8-draft-20261116.json', 'verify-t8-20261116.py', 'verify-t8-attempt2-20261116.py', 'verify-t8-attempt3-20261116.py', 'build-b27-workbook-20261116.py', 'record-platform-progress-20261116.py', 'support-20261116-1047-push.json']:
    files.append({'source': str(B / name), 'target': 'support/' + name})
for attempt in [1, 2, 3]:
    directory = B / f't8-verify-attempt{attempt}-20261116'
    for name in ['receipt.json', 'xlsx-and-workbook-tests.jsonl', 'build-all.log', 'i18n-en.log']:
        path = directory / name
        if path.exists():
            files.append({'source': str(path), 'target': f'support/t8-verify-attempt{attempt}-20261116/' + name})
file_list = B / f'support-{stamp}-files.json'
file_list.write_text(json.dumps(files, ensure_ascii=False, indent=2) + '\n', encoding='utf-8')
print(json.dumps({'report': str(report_path), 'tag': stamp, 'file_list': str(file_list)}))
