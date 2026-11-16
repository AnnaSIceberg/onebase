import datetime
import hashlib
import json
import pathlib

B = pathlib.Path(__file__).parent
R = pathlib.Path('C:/Users/a.sevrjukova/AppData/Local/Temp/claude/C--Users-a-sevrjukova-onebase/cd50d987-9739-4c93-a5be-68d748529225/scratchpad/отчёты-для-analysis')
build_path = B / 'b27r2-workbook-build-20261116/manifest.json'
native_path = B / 'b27r2-native-workbook-attempt1-20261116/receipt.json'
build = json.loads(build_path.read_text('utf-8'))
native = json.loads(native_path.read_text('utf-8'))
assert build['status'] == 'BUILD_AND_TARGETED_TESTS_PASS_SDK_READY_NATIVE_PENDING'
assert native['status'] == 'PASS' and len(native['checks']) == 30 and all(native['checks'].values()) and all(native['cleanup'].values())
assert build['runtime'] == native['runtime'] == 'a10677b2903e4b94cefb8d90a4104fe8a24aa639'
assert build['artifacts']['windows']['sha256'] == native['sdk_windows_sha256']
sha = lambda p: hashlib.sha256(pathlib.Path(p).read_bytes()).hexdigest()
stamp = datetime.datetime.now().strftime('%Y%m%d-%H%M')
supplement = {'status': 'T8_B27R2_WINDOWS_NATIVE30_PASS_SOURCE_CI_PENDING', 'runtime': build['runtime'], 'source': build['source_t8'], 'base_b26': build['base_b26'], 'source_pr': 2635, 'sdk': build['artifacts'], 'build_manifest': str(build_path), 'build_manifest_sha256': sha(build_path), 'native_receipt': str(native_path), 'native_receipt_sha256': sha(native_path), 'native_checks': 30, 'cleanup': native['cleanup'], 'go_started_by_native': False, 'full_ci': 'PENDING: latest source run has Windows failure under investigation; not Ready', 'shared_changes': False, 'db_copies': 0, 'linux_native': 'not claimed'}
manifest = B / f'manifest-b27r2-workbook-final-{stamp}.json'
manifest.write_text(json.dumps(supplement, ensure_ascii=False, indent=2) + '\n', encoding='utf-8')
report = R / f'{stamp}-platform-B27r2-T8-native30-SDK-handoff.md'
report.write_text(f'''# T8/П25 — API и SDK B27r2; native30 PASS

Draft **#2635**, source `{build['source_t8']}`. Кандидат runtime `{build['runtime']}` отB26 `{build['base_b26']}`; рабочие базы/стенды не переключались. B22 и П4/П7 пока отдельные линии, в этот кандидат не включены.

Штатный API для perenos:
```bsl
Книга = Новый КнигаТабличныхДокументов;
Книга.ДобавитьЛист("СМ", ТабДокСМ);
Книга.ДобавитьЛист("ХД", ТабДокХД);
СодержимоеBase64 = Книга.Записать("служебная-записка.xlsx", "xlsx");
```
English: `SpreadsheetWorkbook`, `AddSheet`, `Count`, `Write`. На добавлении сохраняется снимок; исходный ТабДок можно очистить/переиспользовать. Одна книга, листы в порядке добавления. API возвращает base64, файловые пути не открывает; транспорт/расписания остаются в прикладном слое. Имена/UTF-8/дубликаты/Excel bounds и общие budgets — в `examples/tabular-workbook/README.md`. Методы зарегистрированы в штатном langref/autocomplete.

Source core79aac: **49 leaf PASS**, independent openpyxl4/4, buildall/i18nen PASS. Source9b добавил справку/autocomplete и исправил унаследованный дрейф `ПолучитьМакет`; CLI/langref gates PASS. RuntimeB27r2: **50 XLSX/DSL PASS**, **14 HTTP PASS/2 PG-root SKIP отдельно**, buildall/i18nen и оба SDK PASS, pinned Go1.26.11/CGO0/GOMAXPROCS2/-p1. HTTP включает сохранение высоты side_panelB26 и прежние TabDoc/dimension guards.

**Native Windows30 PASS, JS0**, Chrome109, собственная SQLite без копирования БД. Публичный `check`; реальный login → форма отчёта → POST303 → viewer200 → `result_proc` с обычным runtime factory map → XLSX base64. Два листа СМ/ХД, числа111/222, красный/зелёный, source переиспользован и затем изменён до999, снимки сохранены. Openpyxl3.1.5 normal/read_only безreset/bounds: A1:B2 на обоих листах; объединения, разные ширины/высоты и landscape/portrait/fitWidth сохранены. Пустая книга даёт явный module error500, частичного результата нет. Отдельно native `langref --format json` вернул все3 метода книги. Browser/context/driver/app/18582 закрыты, own-auth DB удалена. Go/Chrome не работали одновременно.

Первый native B27 FAIL был только ожиданием статуса redirect в событии Chrome: реальный server log уже содержал303→200. Исправлен наблюдатель, SDK не менялся; B27 получил29 PASS. После добавления metadata/help построен отдельный B27r2, прошёл30 PASS. Старые SDK/FAIL/receipts сохранены, не ретегировались. Linux native этим Windows-прогоном не заявлен.

**SDK для изолированной проверки конфигурации (не установлен):**
- Linux: `{build['artifacts']['linux']['path']}`; SHA `{build['artifacts']['linux']['sha256']}`.
- Windows: `{build['artifacts']['windows']['path']}`; SHA `{build['artifacts']['windows']['sha256']}`.

Manifest `{manifest}`, SHA `{sha(manifest)}`. Build `{build_path}`, SHA `{sha(build_path)}`; native `{native_path}`, SHA `{sha(native_path)}`. EmbeddedVCS не приписываю .git-worktree сборке: provenance — clean runtime/source pins, команды, Go buildinfo и SHA. Core Go blobs T8 совпадают с upstream source; runtime сохраняет дополнительные descriptors (187 функций/172 метода), а source имеет176/169. Оба справочника сгенерированы отдельно и проверены.

**Открыто:** full source CI #2635 — в новом прогоне один Windows failure разбирается, Ready/production не заявлены. Родитель #2627 остаётся Draft; его CI/docs drift исправляется отдельно и пойдёт в ACL source #2631. #2630/#2629/#2632 уже обновлены main (892365a0/449bafdb/54a8303a), CI идут. B22 correctedprebuilt UI190+interpreter6 всё ещё ждёт независимую PG-полосу (отчёт1047).

B26 #2634: после одобренного пользователем push source2a0f4acad **полный CI GREEN**, включаяPG/Windows/e2e. Runtime513274b7 на :8444 установлен release-claude, platform не менял shared. Очередь владельца последняя; рекурсивных поисков по C: не было.

Generated-with: Codex
''', encoding='utf-8')
files = [{'source': str(report), 'target': 'reports/' + report.name}, {'source': str(manifest), 'target': 'support/' + manifest.name}]
for directory, names in [('b27-workbook-build-20261116', ['manifest.json']), ('b27r2-workbook-build-20261116', ['manifest.json', 'http-critical-tests.jsonl', 'xlsx-tests.jsonl', 'build-all.log', 'i18n-en.log', 'windows-sdk-buildinfo.txt', 'linux-sdk-buildinfo.txt']), ('b27-native-workbook-attempt1-20261116', ['receipt.json']), ('b27-native-workbook-attempt2-20261116', ['receipt.json']), ('b27r2-native-workbook-attempt1-20261116', ['receipt.json'])]:
    for name in names:
        files.append({'source': str(B / directory / name), 'target': 'support/' + directory + '/' + name})
for name in ['native-b27-workbook-20261116.py', 'native-b27-workbook-attempt2-20261116.py', 'native-b27r2-workbook-20261116.py', 'build-b27r2-workbook-20261116.py', 'resolve-b27-langref-port-20261116.py', 't8-langref-final-20261116.jsonl', 'b27-langref-final-20261116.jsonl', 'support-20261116-1156-push.json', 'finalize-t8-handoff-20261116.py']:
    files.append({'source': str(B / name), 'target': 'support/' + name})
file_list = B / f'support-{stamp}-files.json'
file_list.write_text(json.dumps(files, ensure_ascii=False, indent=2) + '\n', encoding='utf-8')
print(json.dumps({'report': str(report), 'tag': stamp, 'file_list': str(file_list)}))
