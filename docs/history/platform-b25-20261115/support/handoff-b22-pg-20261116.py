import datetime
import hashlib
import json
import pathlib

B = pathlib.Path(__file__).parent
D = B / 'b22-corrected-ui-tests-attempt2-20261116'
R = pathlib.Path('C:/Users/a.sevrjukova/AppData/Local/Temp/claude/C--Users-a-sevrjukova-onebase/cd50d987-9739-4c93-a5be-68d748529225/scratchpad/отчёты-для-analysis')
build = json.loads((D / 'receipt.json').read_text('utf-8'))
old = json.loads((B / 'manifest-1115b22-report-acl-list-xlsx-DEV.json').read_text('utf-8'))
assert len(build['local_sqlite_pass']) == 12 and len(build['local_pg_skip']) == 2
assert build['expected_total'] == 190 and build['clean_before_after']
stamp = datetime.datetime.now().strftime('%Y%m%d-%H%M')
sha = lambda p: hashlib.sha256(pathlib.Path(p).read_bytes()).hexdigest()
handoff = {
    'status': 'GO_END_LOCAL_SQLITE12_PASS_CORRECTED_UI_PREBUILT_READY_INDEPENDENT_PG_PENDING',
    'runtime_product': build['product_runtime'], 'ui_test_source': build['test_source'],
    'product_delta_from_frozen_sdk': False, 'source_delta': build['test_only_delta'],
    'ui': {'artifact': build['artifact'], 'regex': build['regex'], 'expected_total': 190, 'expected_names': build['expected_names'], 'sqlite': 95, 'postgres': 95},
    'interpreter': {'artifact': old['artifacts']['interpreter_pg'], 'expected': old['expected']['interpreter_pg']},
    'required_total': 196, 'local_sqlite_pass': 12, 'local_pg_skip_not_pass': build['local_pg_skip'],
    'build_receipt': str(D / 'receipt.json'), 'build_receipt_sha256': sha(D / 'receipt.json'),
    'go_build_finished': True, 'sdk_rebuilt': False, 'native_windows': 'PASS26 (separate immutable SDK1496 receipt)',
    'instructions': 'Run the immutable test executables sequentially in the independent private PG lane. Match all exact expected leaf names; FAIL=0/SKIP=0. Preserve before/after namespace cleanup evidence. Do not substitute old UI artifact, source PG receipts, overlays, or local SKIP for the new exact run.',
    'shared_changes': False, 'db_copies': 0,
}
handoff_path = B / f'b22-corrected-pg-handoff-{stamp}.json'
assert not handoff_path.exists()
handoff_path.write_text(json.dumps(handoff, ensure_ascii=False, indent=2) + '\n', encoding='utf-8')
report_path = R / f'{stamp}-platform-B22-GoEND-PG-handoff.md'
report_path.write_text(f'''# B22 — GO END; исправленный prebuilt для независимого PG

После native Windows PASS26 (отчёт 20261116-1032-platform-B22-native-и-CI.md) завершён единственный ограниченный Go-прогон (-p=1, GOMAXPROCS=2). **SQLite12 PASS, PG2 SKIP отдельно**, новый Linux UI prebuilt готов. SDK, продукт и shared базы не пересобирались/не менялись.

Product/runtime `{build['product_runtime']}`. Test source `{build['test_source']}` — ровно один test-only файл `internal/ui/report_result_access_test.go` относительно frozen product: вместо допустимого PG голого SELECT используется незакрытая скобка. Перед/после clean/head проверены. Старый prebuilt B22 и старые FAIL не перезаписывались.

Новый immutable UI artifact: `{build['artifact']['path']}`, SHA `{build['artifact']['sha256']}`, размер {build['artifact']['bytes']} B. Команды, logs, Go1.26.11 buildinfo и привязка clean source → binary: `{D / 'receipt.json'}`, SHA `{sha(D / 'receipt.json')}`. Встроенный VCS stamp для .test не заявляю.

**Нужна независимая PG-полоса:** новый UI190 (95 SQLite + 95 PG) и прежний неизменённый interpreter6 (3+3), всего196 exact leaf names, FAIL0/SKIP0 с before/after namespace cleanup. Полные пути/SHA/regex/имена — `{handoff_path}`, SHA `{sha(handoff_path)}`. Старый UI `b6c5a077…` не использовать: он содержит прежний fixture. Ничего не ретегировать.

Первый запуск Go внутри песочницы получил Access denied на существующий cache и не дошёл до тестов; логи сохранены. Повтор вне песочницы использовал существующий кэш, без создания нового холодного cache: SQLite12 PASS, cross-build PASS. Новых Chrome/SDK/стендов при Go не было. Полный CI/Ready/PG PASS пока не заявлены; новые checks Draft #2631/#2632/#2629/#2630 ещё не появились.

Дальше T8/П25 в source/Draft, Б26 только после Draft release-claude; очередь владельца последняя. Go END — новый compiler не запускаю одновременно с чужим. Широких/рекурсивных поисков по C: не было.

Generated-with: Codex
''', encoding='utf-8')
files = [{'source': str(report_path), 'target': 'reports/' + report_path.name}, {'source': str(handoff_path), 'target': 'support/' + handoff_path.name}]
for p in [D / 'receipt.json', D / 'local-targeted.jsonl', D / 'linux-prebuilt-build.log', D / 'linux-prebuilt-buildinfo.txt']:
    files.append({'source': str(p), 'target': 'support/b22-corrected-ui-tests-20261116/' + p.name})
for name in ['build-b22-corrected-ui-tests-20261116.py', 'build-b22-corrected-ui-tests-attempt2-20261116.py', 'recover-support-push-20261116.py', 'handoff-b22-pg-20261116.py', 'support-20261116-1032-push.json']:
    files.append({'source': str(B / name), 'target': 'support/' + name})
for name in ['receipt.json', 'local-targeted.jsonl']:
    files.append({'source': str(B / 'b22-corrected-ui-tests-20261116' / name), 'target': 'support/b22-first-sandbox-fail-20261116/' + name})
file_list = B / f'support-{stamp}-files.json'
file_list.write_text(json.dumps(files, ensure_ascii=False, indent=2) + '\n', encoding='utf-8')
print(json.dumps({'report': str(report_path), 'handoff': str(handoff_path), 'tag': stamp, 'file_list': str(file_list)}))
