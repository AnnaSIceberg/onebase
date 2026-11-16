import datetime,hashlib,json,pathlib,subprocess
B=pathlib.Path(__file__).parent
R=pathlib.Path('C:/Users/a.sevrjukova/AppData/Local/Temp/claude/C--Users-a-sevrjukova-onebase/cd50d987-9739-4c93-a5be-68d748529225/scratchpad/отчёты-для-analysis')
build=json.loads((B/'b30-date-mask-build-20261116/manifest.json').read_text('utf-8'))
native=json.loads((B/'b30-native-date-mask-attempt2-20261116/receipt.json').read_text('utf-8'))
browser=json.loads((B/'b30-prebuilt-date-browser-attempt1-20261116.json').read_text('utf-8'))
assert build['runtime']=='d0cb9b15100ecdc6f13a8234dc4b2e3f99fa2eff'
assert build['status']=='BUILD_AND_HTTP_PASS_NATIVE_AND_INDEPENDENT_PG_PENDING'
assert native['status']=='PASS'and len(native['checks'])==45 and all(native['checks'].values())and all(native['cleanup'].values())and native['page_errors']==0
assert browser['exit']==0
def sha(p):
 with pathlib.Path(p).open('rb')as f:return hashlib.file_digest(f,'sha256').hexdigest()
for item in [*build['artifacts'].values(),*build['prebuilts'].values()]:assert sha(item['path'])==item['sha256']
source='9a81ab6cbcb96c834c184ca24c00c1ec474ba90c'
def git(*a):return subprocess.check_output(['git',*a],cwd=B/'report-b30-date-mask-runtime',creationflags=subprocess.CREATE_NO_WINDOW).decode().strip()
changed=git('diff','--name-only',build['source'],source).splitlines()
assert changed==['CHANGELOG.md','docs/images/releases/v0.14.0-help.png','docs/releases/release_v0.14.0.md']
old=json.loads((B/'b29-independent-pg-handoff-20261116.json').read_text('utf-8'))
old['runtime']=build['runtime'];old['source_date']=source;old['base_b29']=build['base_b29']
for package,key in [('api','api'),('ui','ui'),('interpreter','dsl/interpreter')]:old['suites'][package]['artifact']=build['prebuilts'][key]
pg=B/'b30-independent-pg-handoff-20261116.json';assert not pg.exists();pg.write_text(json.dumps(old,indent=2)+'\n',encoding='utf-8')
final={'runtime':build['runtime'],'branch':'integration/cc2-b30-date-mask-20261116','base_b29':build['base_b29'],'source':source,'tested_source_code':build['source'],'source_docs_refresh_only':changed,'draft':2611,'build_manifest':str(B/'b30-date-mask-build-20261116/manifest.json'),'build_manifest_sha256':sha(B/'b30-date-mask-build-20261116/manifest.json'),'artifacts':build['artifacts'],'native_checks':45,'native_receipt':str(B/'b30-native-date-mask-attempt2-20261116/receipt.json'),'native_receipt_sha256':sha(B/'b30-native-date-mask-attempt2-20261116/receipt.json'),'runtime_browser_receipt':str(B/'b30-prebuilt-date-browser-attempt1-20261116.json'),'runtime_browser_sha256':sha(B/'b30-prebuilt-date-browser-attempt1-20261116.json'),'http_leaf_counts':build['http_counts'],'independent_pg':str(pg),'independent_pg_status':'PENDING_312','source_ci_status':'RUNNING_NOT_FULL_GREEN','shared_changes':False,'db_copies':0,'deployment_rule':'LPC8448 only explicit user word, one accumulated package; no8444/8443 updates'}
fp=B/'manifest-b30-date-mask-final-20261116.json';assert not fp.exists();fp.write_text(json.dumps(final,indent=2)+'\n',encoding='utf-8')
tag=datetime.datetime.now().strftime('%Y%m%d-%H%M')
p=R/(tag+'-platform-маска-даты-B30-native45-SDK.md')
p.write_text(f'''# Platform: маска даты — B30, native45 PASS; общие стенды не менялись

**Маска реализована, source Draft #2611 обновлён main, B30 накоплен поверх B29. На новой собственной SQLite под ОТК и ОператорКЦ native45 PASS, JS0; браузеры/стенд остановлены, authDB удалена. Независимый PG312 PENDING; source CI идёт, полный GREEN не заявляется. Выкладка только одним накопленным пакетом в ЛПЦ :8448 по её слову.**

## Код и контракт

- Draft https://github.com/ivanarama/onebase/pull/2611, branch `feature/report-and-date-controls-20261113`, HEAD `{source}`. Source code `{build['source']}`; после него только merge опубликованных документов v0.14.0, Go/JS bytes не менялись. Main `cb2e0acee31de5800699f46af5115b3e458a909b`, конфликт CHANGELOG сохранён с нашими изменениями в «Не выпущено».
- Runtime **{build['runtime']}**, branch `{final['branch']}`, WT `B/report-b30-date-mask-runtime`, CLEAN/pushed. Это B29 `{build['base_b29']}` + маска и браузерная фикстура. П4/П7 уже из B21; B22, B26, T8, enum сохранены, повторных реализаций нет.
- Существующий `config/app.yaml ui.short_date_input: true` включает полный шаблон `__.__.____`, дата-время `__.__.____ __:__:__`. Он независим от ролей/плотности, включает операторов КЦ. Прикладную конфигурацию не менял; cc2 сохраняет действующий true в накопленной конфигурации. Новый неподдерживаемый ключ не требуется.
- Ввод только цифр, точки/двоеточия автоматически; курсор переходит через разделители. Backspace/Delete/выделение; вставка `16.11.2026`, `16112026`, ISO; ошибка неполной/невозможной даты. Enter завершает дату и не отправляет форму/отчёт, даже `enter_submits_form: true`. Escape восстанавливает прежнее значение. Календарь кнопкой/Alt+Down, исходный input сохраняется. Сервер получает ISO; локальные часы/секунды восстанавливаются после хранения в UTC. Объектные формы, параметры отчётов, отборы списков.
- Без прочих изменений операторского интерфейса. Маска отсутствует в старом SDK B29, новый ключ на B29 её не добавит.

## Проверки

Source Node29 PASS (включая Ctrl/Meta+V после Escape), панель отчёта и 36 ролевых сочетаний HTTP PASS; source build ./... и i18n PASS. Source браузер через prebuilt test.exe PASS с настоящими клавишами/буфером и нулём POST. Source receipts `B/date-mask-source-checks-attempt2-20261116/receipt.json`, `B/date-mask-source-browser-attempt3-20261116.json`.

B30 HTTP131 leaf PASS / 7 PG-root SKIP; build ./... / i18n / Win+Linux SDK / Linux prebuilt API/UI/interpreter и Windows UI test.exe PASS. Браузер **runtime** через prebuilt PASS, отдельно от Go; квитанция `{final['runtime_browser_receipt']}`.

Нативный SDK: **45 PASS**, обе роли не admin; отчёт, даты/секунды/Backspace и неполнота (source browser), две настоящие вставки (native обе роли), календарь, Enter без POST, managed-object с `enter_submits_form`, явная запись с активной маской, SQL-момент с учётом UTC и повторно открытая форма, отбор списка; JS0, cleanupalltrue. `{final['native_receipt']}`, SHA256 `{final['native_receipt_sha256']}`. Снимки `B/b30-native-date-mask-attempt2-20261116/*-live-mask.png`, `*-object.png`, `*-filter.png`; два снимка просмотрены.

Прозрачность попыток: source browser attempt1 выявил Ctrl+V после Escape и неполный SSE hub; они исправлены. Attempt2 выявил nil interpreter **тестовой фикстуры**, исправлен; attempt3 PASS. Native attempt1 остановился на ошибочной проверке текста UTC как местного времени, все процессы/база очищены; attempt2 проверяет момент и reopened UI, PASS. Неудачные журналы не перезаписаны. Новых продуктовых правок после сборки SDK не было.

Независимый PostgreSQL: **312 PENDING**, exact regex/имена/новые SHA prebuilt в `{pg}`. Исторические B21/B28/B29 проверки не перетегированы. Source CI #2611 идёт после merge main; не подменять 131SQLite/native45 полным CI или PG.

## SDK

- Windows `{build['artifacts']['windows']['path']}`, SHA256 `{build['artifacts']['windows']['sha256']}`.
- Linux `{build['artifacts']['linux']['path']}`, SHA256 `{build['artifacts']['linux']['sha256']}`.
- Final manifest `{fp}`, SHA256 `{sha(fp)}`; build `{final['build_manifest']}`.

## Дальше

Уровни 1 2 3… для ТабДок/составного отчёта + без подвалов групп, общий «Итого», label заголовков, бледный фон групп по глубине (HTML/Excel/PDF, тёмная тема). Screenshot2/4 просмотрены. Затем отборы по любому разрешённому полю → связь Филиал/Направление и сортировка/представление (заказ ожидается) → движковая защита ПДн ссылочных представлений по `codex2-20261116-заказ-platform-ПДн-представления-ссылок-B29.md`. Её запрет телефонов/полного адреса для всех ролей принят; в B30 он ещё не реализован. Карточки не менять, UUID/переходы сохранять.

Дерево меню у platform2 wV:p1, точные B28/B29/SDK переданы `20261116-platform-для-platform2-runtime-B28-B29-меню.md`; platform не продолжает меню. B30 можно брать как следующую накопленную основу после проверки, без общей установки. Ledger ответ передан `20261116-1504-platform-ledger-контракт-для-perenos-codex.md`, server gate не реализован.

Общие :8448/:8444/:8443, рабочая 1С/БД не менялись, базы не копировались. B29/B30 не выкладывать по этому отчёту. Новых агентов не создавал, рекурсивных поисков C: не делал, звука вручную нет.

B = `C:/Users/a.sevrjukova/AppData/Local/Temp/platform-help-admin-20261111`.

Generated-with: Codex
''',encoding='utf-8')
q=R/'20261116-platform-текущая-очередь-дата-уровни-отборы-меню.md';s=q.read_text('utf-8');start=s.index('1. **Маска');end=s.index('2. **Панель',start)
s=s[:start]+f'1. **Маска даты — реализована и накоплена B30.** Draft #2611 source {source}, runtime {build["runtime"]}, native45 PASS/JS0/cleanupalltrue; source Node29, source/runtime браузеры PASS. Windows/Linux SDK готовы. Source CI RUNNING, PG312 PENDING. Отчёт `{p.name}`. На общих стендах не установлена.\n'+s[end:];s=s.replace('Screenshot_3 просмотрен, 2/4 ещё не просмотрены','Screenshot_2/3/4 просмотрены');q.write_text(s,encoding='utf-8')
# Keep all reproducibility metadata, but no SDK, prebuilt, database or session.
files=[p,q,R/'20261116-platform-для-platform2-runtime-B28-B29-меню.md',fp,pg,pathlib.Path(__file__)]
patterns=['*date-mask*20261116.py','*date-mask*20261116.md','*date-browser*20261116.py','*date-browser*20261116.json','*date-browser*20261116.log','*date-mask*20261116.json','resolve-b30-date-port-20261116.py','update-platform-queue-20261116.py','finish-date-main-refresh-20261116.py']
for pattern in patterns:files.extend(p for p in B.glob(pattern)if p.is_file())
for dirname in ['date-mask-source-checks-attempt1-20261116','date-mask-source-checks-attempt2-20261116','date-mask-source-checks-attempt3-20261116','b30-date-mask-build-20261116','b30-native-date-mask-attempt1-20261116','b30-native-date-mask-attempt2-20261116']:
 for f in (B/dirname).iterdir():
  if f.is_file()and(f.name=='receipt.json'or f.suffix in ['.png','.jsonl']or f.name in ['manifest.json','build-all.log','i18n-en.log']):files.append(f)
items=[]
for f in sorted(set(files)):
 assert f.stat().st_size<2*1024*1024
 items.append({'source':str(f),'target':('reports/'+f.name)if f.parent==R else 'support/'+str(f.relative_to(B)).replace('\\','/')})
fl=B/('support-'+tag+'-files.json');assert not fl.exists();fl.write_text(json.dumps(items,ensure_ascii=False,indent=2)+'\n',encoding='utf-8')
print(json.dumps({'report':str(p),'manifest':str(fp),'backup_tag':tag,'backup_list':str(fl),'backup_files':len(items)}))
