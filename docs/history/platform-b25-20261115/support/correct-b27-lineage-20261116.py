import datetime, hashlib, json, pathlib, subprocess
B = pathlib.Path(__file__).parent
W = B/'report-b28-runtime'
R = pathlib.Path('C:/Users/a.sevrjukova/AppData/Local/Temp/claude/C--Users-a-sevrjukova-onebase/cd50d987-9739-4c93-a5be-68d748529225/scratchpad/отчёты-для-analysis')
def git(*args):
 return subprocess.check_output(['git',*args],cwd=W,creationflags=subprocess.CREATE_NO_WINDOW).decode().strip()
b21='0d77e91557b12bada4668fd7bfa5f1b13a37de08'
b27='a10677b2903e4b94cefb8d90a4104fe8a24aa639'
subprocess.run(['git','merge-base','--is-ancestor',b21,b27],cwd=W,creationflags=subprocess.CREATE_NO_WINDOW,check=True)
p4='892365a03c8c35e3c48dea25ed897a3c9a072d95'
p7='449bafdb8a8c86f177e8dbbe768a62af885b2ab0'
head=git('rev-parse','HEAD')
assert head=='f045bdb11b5e563120ff0381476957e5df9e2549' and not git('status','--porcelain')
matches=[]
for source,files in [(p4,['internal/report/drilldown.go','internal/ui/report_drilldown.go','internal/configcheck/report_drilldown.go']), (p7,['internal/report/context.go','internal/access/scalar_user_card.go','internal/configcheck/report_context.go'])]:
 for file in files:
  a=git('rev-parse',head+':'+file); b=git('rev-parse',source+':'+file)
  assert a==b,file
  matches.append({'file':file,'source':source,'blob':a})
receipt={'b21':b21,'b27r2':b27,'b21_is_b27r2_ancestor':True,'b28':head,'source_matches':matches,'b28_not_built_yet':True,'shared_changes':False,'incorrect_report':'20261116-1257-platform-B27r2-T8-native30-SDK-handoff.md','correction':'P4/P7 present in B27r2 via B21; only B22 was absent','timestamp':datetime.datetime.now().isoformat()}
out=B/'b27r2-lineage-correction-20261116.json'
assert not out.exists()
out.write_text(json.dumps(receipt,ensure_ascii=False,indent=2)+'\n',encoding='utf-8')
report=R/'20261116-1318-platform-исправление-состава-B27r2-B28.md'
assert not report.exists()
report.write_text(f'''# Исправление состава B27r2; B28 подготовлен

В моём отчёте `20261116-1257-platform-B27r2-T8-native30-SDK-handoff.md` ошибочно сказано, что П4/П7 не входят в B27r2. **П4/П7 уже перенесены в B21 `{b21}` и сохранены в B25/B26/B27r2.** Проверена ancestry B21 → B27r2 `{b27}`. SDK, их SHA и native30 не изменились. Старый отчёт сохранён как история, это явное исправление его фразы о составе.

П4/П7 исторически прошли exact B21 независимый root178uniqueleaf PASS/FAIL0/SKIP0 и Windows native19PASS/JS0: `20261115-platform-B21-П4-П7-PG-native-PASS.md`. Эти результаты относятся к B21, не ретегируются в новые SDK. В B27r2 отсутствовал только B22. Повторный полный перенос П4 отменён до фиксации; на его месте оставлены существующие расширенные runtime-матрицы, П3/П1/choice-поведение.

Новая собственная линия B28 `{head}`, branch `integration/cc2-b28-report-actions-context-20261116`, CLEAN/pushed. База — frozen B27r2; добавлен точный B22 (ACL403 + ordinary XLSX used range + исправленные тестовые фикстуры), а у существующих П4/П7 только проверенные CI-исправления. Ещё SDK B28 не собирался. Шесть ключевых Go-файлов П4/П7 совпадают с актуальными source Draft #2630/#2629; квитанция `{out}`. B28 ждёт последовательное Go-окно после source-проверок.

Source #2627 обновлён main и справочником: `bed39b9d6083c9b8d4bf9a9dce741dbd5fc05902`, CLI/langref PASS. ACL #2631 обновлён этим родителем: `6ac8da0fa9bab99c57078cfc04d6235070097cd7`, целевые тесты и buildall PASS; локальный PostgreSQL SKIP отдельно. П4 source `892365a0` также targeted/buildall PASS. П7/#2632 проверяются последовательно. Все ветки pushed; Draft сохраняются.

T8 #2635 теперь `{git('rev-parse','b7c57cb62')}`: merge актуального родителя; дерево **точно совпадает** с проверенным source9b8db2dc (оба tree `6bba9b293f5e7b534f5e7c18f52b069bbd651514`). Новый CI run43846509362. Предыдущий Windows job133209160123 упал на инициализации двух SlickGrid. Прямой повтор job GitHub отклонил HTTP403: нужны admin rights; это ограничение GitHub, не отказ автоматического согласования Codex. Никакие проверки не обходились. Описание Draft обновлено реальными доказательствами.

Независимый B22 corrected PG196 остаётся PENDING (handoff1047). Shared/production/конфигурации/1С не менялись, DBcopies0. Рекурсивных поисков по диску нет. Очередь П8–П25 далее, владелец последним.

Generated-with: Codex
''',encoding='utf-8')
print(json.dumps({'report':str(report),'receipt':str(out),'sha256':hashlib.sha256(out.read_bytes()).hexdigest()}))
