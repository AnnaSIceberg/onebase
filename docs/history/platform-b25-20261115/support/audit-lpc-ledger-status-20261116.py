import datetime,hashlib,json,pathlib,subprocess
B=pathlib.Path(__file__).parent;D=B/'lpc-historical-ledger-PLAN-20261114'
R=pathlib.Path('C:/Users/a.sevrjukova/AppData/Local/Temp/claude/C--Users-a-sevrjukova-onebase/cd50d987-9739-4c93-a5be-68d748529225/scratchpad/отчёты-для-analysis')
W=B/'report-b29-enum-runtime'
def sha(p):return hashlib.sha256(pathlib.Path(p).read_bytes()).hexdigest()
manifest=json.loads((D/'04-PLAN-MANIFEST.json').read_text('utf-8'))
for a in manifest['artifacts']:assert sha(a['path'])==a['sha256'],a['path']
proposal=json.loads((D/'02-ledger-binding.PROPOSAL.json').read_text('utf-8'))
assert not proposal['executable'] and not proposal['existing_native_contract'] and not proposal['admission_protocol_status']['runtime_witness']
sources=json.loads((D/'03-SOURCES.json').read_text('utf-8'))
pins={'source_plan':'f49a9e0bfd2eb5184ce9e13e583a5a163e967534','B26':'513274b7e1bc230d8c0de5fcd22a34deb672345c','B29':'902e4d97e50e90f047ec6d27832e2f9d175d095d'}
def git(*args):return subprocess.check_output(['git',*args],cwd=W,creationflags=subprocess.CREATE_NO_WINDOW)
rows=[]
for entry in sources['engine_sources']:
 path=entry['path'];assert path.startswith('internal/') and '..' not in path
 raw=git('show',pins['source_plan']+':'+path);assert hashlib.sha256(raw).hexdigest()==entry['sha256_git_blob_bytes']
 row={'path':path,'source_plan_blob':entry['git_blob'],'source_plan_sha256':entry['sha256_git_blob_bytes']}
 for label in ['B26','B29']:
  content=git('show',pins[label]+':'+path)
  row[label+'_blob']=git('rev-parse',pins[label]+':'+path).decode().strip();row[label+'_sha256']=hashlib.sha256(content).hexdigest();row[label+'_equals_plan_source']=content==raw
 rows.append(row)
assert len(rows)==26
record={'status':'SOURCE_GAP_PLAN_ONLY_NO_IMPLEMENTED_GATE_CERTIFIED','pins':pins,'plan_artifacts_verified':manifest['artifacts'],'source_comparison':rows,'B26_unchanged_count':sum(r['B26_equals_plan_source']for r in rows),'B29_unchanged_count':sum(r['B29_equals_plan_source']for r in rows),'no_go_browser_http_ssh_db_for_this_audit':True}
receipt=B/'lpc-ledger-status-readonly-20261116.json';assert not receipt.exists();receipt.write_text(json.dumps(record,indent=2)+'\n',encoding='utf-8')
paths=[R/'20261114-cc2-ЛПЦ-PLAN-v2-ПРИНЯТ-и-native-gaps.md',R/'20261114-2036-review-ЛПЦ-PLAN-v2-движения.md',pathlib.Path('C:/Users/a.sevrjukova/AppData/Local/Temp/cc2-lpc-movement-plan-v2-root-audit.json')]
reviews=[{'path':str(p),'sha256':sha(p)}for p in paths]
tag=datetime.datetime.now().strftime('%Y%m%d-%H%M');out=R/(tag+'-platform-ledger-контракт-для-perenos-codex.md')
table='\n'.join('| `'+r['path']+'` | '+('тот же blob'if r['B26_equals_plan_source']else'изменён, см. квитанцию')+' | '+('тот же blob'if r['B29_equals_plan_source']else'изменён, см. квитанцию')+' |'for r in rows if r['path']in ['internal/entityservice/service.go','internal/storage/register.go','internal/storage/crud.go','internal/storage/movement_cleanup.go','internal/exchange/package.go','internal/ui/dsl_documents.go'])
out.write_text(f'''# ЛПЦ PLAN v2: summary-only и ledger — статус для perenos-codex / cc2

**В части platform это по-прежнему SOURCE GAP PLAN, не реализованный server ledger-gate. Ни B26, ни B29 я не выпускаю как имеющий такую защиту. Готовую реализацию и review/runtime-квитанции ledger предъявить не могу: этот этап не был реализован.** В B29+ ledger также не добавлялся; текущие B28/B29 меняют отчёты/XLSX/enum, а не исторический импорт.

## Что действительно существует

1. Предложение summary-only профиля: `{D/'01-source-summary-profile-PLAN.md'}`; SHA256 `a4d196b0f51e0907a0699308f4e8ecdbc6322f8607a37736510a1fca2449f606`. В первой строке статуса прямо PLAN/SOURCE-ONLY, «не реализация CC_V2». Это требования к coverage/counts/identity/completeness, не финальная wire-схема: нет утверждённых mode/полей запроса-ответа/численных budgets/реализованного CC_V2 SHA. Использовать как реализованный wire нельзя. Окончательный summary-only wire должен назвать и подтвердить cc2; новым ответом cc2 этот план не подменяю.
2. Неисполняемый ledger binding: `{D/'02-ledger-binding.PROPOSAL.json'}`; SHA256 `1a7d41ddeeb23e9e704e39de9b1f5aa6cda20aafcbec7327d479d279ecb7ebef`. `executable=false`, `existing_native_contract=false`, `runtime_witness=false`, реальные bindings не заполнены. DDL/admission/authority ещё проектные.
3. Источник анализа26 engine blobs: `{D/'03-SOURCES.json'}`; SHA256 `1d7b4a9ff1e7f728352075b6a0234b5f56f0f8380d18f181e27acbd2656986c9`, pin `{pins['source_plan']}`.
4. Полный source-gap план: `{R/'20261114-2035-platform-ЛПЦ-historical-ledger-SOURCE-GAP-PLAN.md'}`; SHA256 `836333f0ce4c55934c5eb2a56438904f7cbea1957ed52fbbf7ba63a6e84d98cd`.
5. Manifest этих4 artifacts: `{D/'04-PLAN-MANIFEST.json'}`; SHA256 `{sha(D/'04-PLAN-MANIFEST.json')}`. Сейчас повторно проверены bytes/SHA всех4, без HTTP/БД/экспорта.

## Что принято review

- PLAN v2 perenos: `perenos-codex/отчёты/20261114-ЛПЦ-PLAN-исходные-движения-v2.md`, SHA256 `bd1d1915803fbc9287dddfa9d698cdfa06ca42463e3e7217bf7848e106611397`.
- Принятие cc2: `{reviews[0]['path']}`, SHA256 `{reviews[0]['sha256']}`. Там явно: согласован PLAN; следующий результат cc2 — SOURCE summary-only wire; ledger DDL/installation/admission/manifest authority не готовая реализация, platform имел только PLAN/source scope.
- Независимое review: `{reviews[1]['path']}`, SHA256 `{reviews[1]['sha256']}`. Защита регистратора закрыта **как проектное требование**, прямо «реализации нет»; native apply остаётся блокирован этим пробелом.
- Root audit PLAN v2: `{reviews[2]['path']}`, SHA256 `{reviews[2]['sha256']}`. Это квитанция проверки плана/metadata/mapping, не исполнения ledger и не release gate.

## Привязка к B26 / B29

- B26 runtime `{pins['B26']}` (`fix/side-panel-full-height`), Linux SDK SHA256 `e977a632cfefc6c1b34575b4ac7797aeb1c057090738f470243fb03b48b67e6d`. Его handoff `20261116-1051-выпуск-claude-полоса-B26-на-8444.md` описывает B25+высоту полосы, не ledger. Историческая установка на8444 не меняет нового правила выкладки.
- B29 runtime `{pins['B29']}`, branch `integration/cc2-b29-report-enum-20261116`; его SDK/native38/PG312 handoff — `20261116-1403-platform-для-cc2-enum-B29-native38-SDK.md`. Ledger туда не переносился.
- Новая RO-квитанция `{receipt}`, SHA256 `{sha(receipt)}`: сопоставлены **Git bytes** всех26 source anchors f49 с exact B26 и B29. Совпадают B26:{record['B26_unchanged_count']}/26, B29:{record['B29_unchanged_count']}/26; изменённые файлы перечислены отдельно в JSON. Вывод не строится на поиске имён ledger/wire в индексе.

| Критический source путь | B26 относительно source-gap f49 | B29 |
|---|---|---|
{table}

## Что ещё нужно и когда

Последний принятый этап14.11 — PLAN. Дальше: cc2 выдаёт точный readonly summary-only wire/source commit и его review; по подтверждённому profile выбирается native scope/модель. Затем отдельная реализация server gate до persistence hooks/первой записи, сериализация ordinary/import/release, storage backstops и публичная SQLite+PG матрица всех UI/REST/DSL/exchange/repair путей, включая admin. Нужны точные executable manifest/undo/CAS и отдельная runtime-квитанция. Этих реализаций/проверок сейчас у platform нет; срок не назначен, в текущей реализации отчётного UI он не скрыт. Perenos не должен считать PLAN уже доступным API или готовым B26 gate.

Новых тяжёлых действий по этому запросу нет: только точные локальные документы/Git blobs и SHA. БД/1С/HTTP/SSH/выкладки не выполнялись. Для общих стендов действует её правило: основной ЛПЦ :8448, только её слово и один накопленный пакет; B29 на8444 не ставить.

Generated-with: Codex
''',encoding='utf-8')
files=[out,receipt,pathlib.Path(__file__)]
lst=B/('support-'+tag+'-files.json');assert not lst.exists();lst.write_text(json.dumps([{'source':str(p),'target':'reports/'+p.name if p.parent==R else 'support/'+p.name}for p in files],ensure_ascii=False,indent=2)+'\n',encoding='utf-8')
print(json.dumps({'report':str(out),'receipt':str(receipt),'unchanged_B26':record['B26_unchanged_count'],'unchanged_B29':record['B29_unchanged_count'],'changed_B26':[r['path']for r in rows if not r['B26_equals_plan_source']],'tag':tag,'backup_list':str(lst)}))
