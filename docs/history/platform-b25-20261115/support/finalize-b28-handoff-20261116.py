import datetime,hashlib,json,pathlib
B=pathlib.Path(__file__).parent
R=pathlib.Path('C:/Users/a.sevrjukova/AppData/Local/Temp/claude/C--Users-a-sevrjukova-onebase/cd50d987-9739-4c93-a5be-68d748529225/scratchpad/отчёты-для-analysis')
build=json.loads((B/'b28-reports-build-20261116/manifest.json').read_text('utf-8'))
native=json.loads((B/'b28-native-combined-20261116.json').read_text('utf-8'))
assert native['status']=='PASS' and native['checks']==75 and native['runtime']==build['runtime']
old=json.loads((B/'b22-corrected-pg-handoff-20261116-1047.json').read_text('utf-8'))
b21=json.loads((B/'manifest-1115b21-report-context-drilldown-DEV.json').read_text('utf-8'))
gate={'runtime':build['runtime'],'status':'INDEPENDENT_PG_PENDING','required':'unique leaf PASS exact expected names; FAIL0 SKIP0; namespace/relations/routines before=after and cleanup; historical proofs not retagged','suites':{}}
for key,package,definition,nameskey in [('api','api',b21['expected']['api_pg'],'expected_subtests'),('ui','ui',old['ui'],'expected_names'),('interpreter','dsl/interpreter',old['interpreter']['expected'],'expected_subtests')]:
 names=definition[nameskey]
 gate['suites'][key]={'artifact':build['prebuilts'][package],'regex':definition['regex'],'expected_names':names,'expected_total':len(names)}
gate['expected_total']=sum(x['expected_total'] for x in gate['suites'].values());assert gate['expected_total']==208
pg=B/'b28-independent-pg-handoff-20261116.json';assert not pg.exists();pg.write_text(json.dumps(gate,indent=2)+'\n',encoding='utf-8')
final={'runtime':build['runtime'],'base_b27r2':build['base_b27r2'],'status':'SDK_READY_WINDOWS_NATIVE75_PASS_INDEPENDENT_PG_PENDING','build_manifest':str(B/'b28-reports-build-20261116/manifest.json'),'native_receipt':str(B/'b28-native-combined-20261116.json'),'artifacts':build['artifacts'],'local_xlsx':build['xlsx_counts'],'local_reports':build['reports_counts'],'native_checks':75,'pg_handoff':str(pg),'shared_changes':False,'db_copies':0}
out=B/'manifest-b28-reports-final-20261116.json';assert not out.exists();out.write_text(json.dumps(final,indent=2)+'\n',encoding='utf-8')
tag=datetime.datetime.now().strftime('%Y%m%d-%H%M')
report=R/(tag+'-platform-B28-native75-SDK-и-очередь.md')
w,l=build['artifacts']['windows'],build['artifacts']['linux']
report.write_text(f'''# B28: B22 + П4/П7 + T8, SDK и native75 PASS

Runtime `{build['runtime']}`, branch `integration/cc2-b28-report-actions-context-20261116` CLEAN/pushed. База B27r2 `{build['base_b27r2']}` сохраняет B26 и уже существующие П4/П7 из B21; добавлен B22 ACL403/ordinary XLSX и проверенные CI/test-only исправления. Повторной реализации П4/П7 нет. Исправление ошибочной фразы старого отчёта1257: `20261116-1318-platform-исправление-состава-B27r2-B28.md`.

Локально: **66 XLSX/DSL leaf PASS**, **85 report leaf PASS / 10 PG-root SKIP отдельно**, справочник/buildall/i18nen PASS. **Точный Windows SDK:75 native checks PASS/JS0**: ACL/XLSX26, П4/П7 19, T8 30. Свои SQLite без копирования БД; реальные CLI/login/HTTP/браузер/выгрузки/openpyxl. Все три браузера запускались последовательно после Go END. Server/browser/context/driver/ports18572/18574/18582 закрыты, own auth DB удалены. Linux browser native не заявлен.

SDK (кандидат, не установлен):
- Windows: `{w['path']}`, SHA256 `{w['sha256']}`.
- Linux: `{l['path']}`, SHA256 `{l['sha256']}`.

Manifest `{out}`; build `{final['build_manifest']}`; native `{final['native_receipt']}`. Pinned Go1.26.11/CGO0/GOMAXPROCS2/-p1. VCS metadata внутри worktree-сборки не приписывается; provenance — clean pins/команды/buildinfo/SHA.

**Независимый PG208 PENDING.** `{pg}` содержит exact Linux prebuilt API12+UI190+interpreter6, SHA, regex и полные ожидаемые имена. Требуется независимый root/runner: PASS208uniqueleaf/FAIL0/SKIP0 и namespace/relations/routines before=after/cleanup. Это новый B28 gate; исторические B21 PG178 или B22 native не ретегируются. Старый B22 corrected PG196 также без полученного результата; при выполнении B28 gate он покрывает те же B22 проверки в новой линии, результаты различать по pin.

Source ACL2631 `6ac8da0f`, П42630 `892365a0`, П72629 `449bafdb`, XLSX2632 `54a8303a`: целевые HTTP-матрицы и buildall всех четырёх обновлённых веток PASS; PG локально SKIP. Все pushed, Draft. T8 #2635 source `b7c57cb62` имеет дерево, равное проверенному9b8db2dc; новый CI43846509362 ещё идёт (19success/2running на последнем чтении). Старый Windows failure сохранён; GitHub запретил повтор job без admin rights, проверки не обходились. #2634 B26 полный CI GREEN после явно разрешённого push2a0f4acad.

**Приоритеты обновлены её сообщениями:** после текущего B28/T8 шага — enum-параметры отчётов (прямая жалоба «Заявки на дату», заказ `codex2-20261116-заказ-platform-enum-параметры-отчётов-B25.md`), затем дерево меню1С, затем П8–П25 и отчётная очередь. Два A080 (MMMM и цепочка календарных умолчаний) добавлены в несрочную отчётную очередь, пока не взяты. Владелец — последним.

Меню: ранний Draft **#2636**, branch `feature/recursive-command-menu-20261116`, commit `3c88d2368843da899665a94d8113ef29eee9d715`, только проект контракта `docs/recursive-command-menu.md`. Один список `nodes`, папка/typed target, явные ID, предел8папок, роли как дополнительные ограничения, старый формат сохраняется. **Ещё не поддержано SDK, YAML не применять к B26/B28.** Реализация после enum. Операторское меню/конфигурации не менялись; source-command → report/processor сопоставляет cc2.

Shared :8443/:8444/:8448, конфигурации и1С не тронуты. Рекурсивного поиска по C: нет. Новых агентов не создавал; CPU ограничен одной Go-линией, браузер отдельно.

Generated-with: Codex
''',encoding='utf-8')
files=[report,R/'20261116-1318-platform-исправление-состава-B27r2-B28.md']
names=['correct-b27-lineage-20261116.py','b27r2-lineage-correction-20261116.json','verify-refreshed-source-20261116.py','resolve-acl-parent-20261116.py','build-b28-reports-20261116.py','prepare-b28-native-20261116.py','run-b28-native-20261116.py','finalize-b28-handoff-20261116.py','b28-native-plan-20261116.json','manifest-b28-native-input-20261116.json','b28-native-combined-20261116.json','b28-independent-pg-handoff-20261116.json','manifest-b28-reports-final-20261116.json','menu-draft-body-20261116.md','t8-pr-body-verified-20261116.md','update-t8-pr-body-20261116.py']
files += [B/n for n in names]
for d in ['refreshed-source-checks-20261116','b28-reports-build-20261116']:
 files += [p for p in (B/d).iterdir() if p.is_file() and p.suffix in ['.json','.jsonl','.log','.txt']]
for t in json.loads((B/'b28-native-plan-20261116.json').read_text('utf-8')):
 files += [pathlib.Path(t['helper']),pathlib.Path(t['receipt'])]
inventory=[{'source':str(p),'target':('reports/' if p.parent==R else 'support/'+str(p.relative_to(B)).replace('\\','/'))+p.name if p.parent==R else 'support/'+str(p.relative_to(B)).replace('\\','/')} for p in files]
lst=B/('support-'+tag+'-files.json');assert not lst.exists();lst.write_text(json.dumps(inventory,ensure_ascii=False,indent=2)+'\n',encoding='utf-8')
print(json.dumps({'report':str(report),'manifest':str(out),'pg_total':gate['expected_total'],'backup_tag':tag,'backup_list':str(lst),'files':len(files)},ensure_ascii=False))
