import collections,datetime,hashlib,json,pathlib
B=pathlib.Path(__file__).parent
R=pathlib.Path('C:/Users/a.sevrjukova/AppData/Local/Temp/claude/C--Users-a-sevrjukova-onebase/cd50d987-9739-4c93-a5be-68d748529225/scratchpad/отчёты-для-analysis')
build=json.loads((B/'b29-enum-build-20261116/manifest.json').read_text('utf-8'))
nativefile=B/'b29-native-enum-attempt2-20261116/receipt.json';native=json.loads(nativefile.read_text('utf-8'))
assert build['status']=='BUILD_AND_HTTP_PASS_NATIVE_AND_INDEPENDENT_PG_PENDING'
assert native['status']=='PASS' and len(native['checks'])==38 and all(native['checks'].values()) and all(native['cleanup'].values()) and native['runtime']==build['runtime']
events=[json.loads(s) for s in (B/'b29-enum-build-20261116/http-enum-critical-tests.jsonl').read_text('utf-8').splitlines() if s.startswith('{')]
results={e['Test']:e['Action'] for e in events if e.get('Test','').startswith('TestReportEnumParamsHTTPMatrix/') and e['Action'] in ['pass','skip']}
sqlite=sorted(n for n,a in results.items() if a=='pass' and '/sqlite/'in n and not any(o.startswith(n+'/') for o in results))
assert len(sqlite)==52
enum_names=sorted(sqlite+[n.replace('/sqlite/','/postgres/',1) for n in sqlite]);assert len(set(enum_names))==104
pg=json.loads((B/'b28-independent-pg-handoff-20261116.json').read_text('utf-8'))
pg['runtime']=build['runtime'];pg['status']='INDEPENDENT_PG_PENDING';pg['source_enum']=build['source']
for key,package in [('api','api'),('ui','ui'),('interpreter','dsl/interpreter')]:pg['suites'][key]['artifact']=build['prebuilts'][package]
api=pg['suites']['api'];assert api['regex'].endswith(')$')
api['regex']=api['regex'][:-2]+'|TestReportEnumParamsHTTPMatrix)$'
api['expected_names']=sorted(api['expected_names']+enum_names);api['expected_total']=len(api['expected_names'])
pg['expected_total']=sum(s['expected_total'] for s in pg['suites'].values());assert pg['expected_total']==312
pgpath=B/'b29-independent-pg-handoff-20261116.json';assert not pgpath.exists();pgpath.write_text(json.dumps(pg,indent=2)+'\n',encoding='utf-8')
final={'status':'ENUM_SDK_READY_WINDOWS_NATIVE38_PASS_CI_AND_INDEPENDENT_PG_PENDING','runtime':build['runtime'],'source':build['source'],'source_pr':2637,'base_b28':build['base_b28'],'sdk':build['artifacts'],'build_manifest':str(B/'b29-enum-build-20261116/manifest.json'),'native_receipt':str(nativefile),'native_receipt_sha256':hashlib.sha256(nativefile.read_bytes()).hexdigest(),'native_checks':38,'local_http':build['http_counts'],'pg_handoff':str(pgpath),'shared_changes':False,'db_copies':0}
out=B/'manifest-b29-enum-final-20261116.json';assert not out.exists();out.write_text(json.dumps(final,indent=2)+'\n',encoding='utf-8')
tag=datetime.datetime.now().strftime('%Y%m%d-%H%M');report=R/(tag+'-platform-для-cc2-enum-B29-native38-SDK.md')
w,l=build['artifacts']['windows'],build['artifacts']['linux']
report.write_text(f'''# cc2 / analysis: enum-параметры отчётов — B29 SDK, native38 PASS

Заказ `codex2-20261116-заказ-platform-enum-параметры-отчётов-B25.md` выполнен в движке. Draft **#2637**, source `{build['source']}`, branch `fix/report-enum-parameters-20261116`; runtime `{build['runtime']}`, branch `integration/cc2-b29-report-enum-20261116`, база B28 `{build['base_b28']}`. CLEAN/pushed. SDK кандидат для DEV; shared :8444/:8443/:8448 не менял.

**Контракт:** существующий `type: enum:Имя` теперь даёт select из метаданных в исходном порядке и с переводами штатного enum-choice. `options` в отчётах не нужны. Значения option/запросов/REST/экспорта — канонические имена, подписи отдельно. Умолчание видно при первом открытии; текущее значение сохраняется; явная очистка не возвращает умолчание. Неизвестное перечисление/непустое неизвестное значение дают явный отказ до запроса/модуля/выгрузки, свободный текст не создаёт значение. Ссылки уже были native: ACL/RLS, поиск по истории, choice_show_deleted и порядок каталогов сохранены. Прикладные YAML не требуют правки.

Source HTTP **52 leaf PASS / PG-root SKIP1 отдельно**: публичные UI GET/POST/REST/XLSX/PDF, два языка, metadata-order/default/current/empty и недопустимый текст/подпись. Source buildall/i18nen PASS. Первый test attempt не компилировался из-за двух ошибок нового теста; второй был неверно настроен для en без bundle; третий PASS без изменения продукта между попытками2/3. Исходные FAIL-журналы сохранены. Runtime B29 **93 HTTP leaf PASS / 7 PG-root SKIP отдельно**, включая прежние П4/П7/B22/TabDoc и B26, buildall/i18nen и SDK обеих ОС PASS. Контекст П7 привязывается перед enum-валидацией; варианты и настройки ссылочного выбора сохранены.

**Native Windows38 PASS / JS0**, Chrome109, собственная SQLite, реальный login отдельного **неадминистратора**. На форме выбран Office и нажата «Сформировать»; показана только нужная строка, значение сохранено. Очистка через select/кнопку не вернула Visit. Native reference picker остался. Реальный toolbar Excel → фоновая задача → download; прямые XLSX/PDF и REST; неверное значение400, фоновая ошибка без файла. Queryless и query-backed result_proc получили каноническое значение и пустой выбор; выгружены реальные снимки. Openpyxl3.1.5 normal/read_only без reset/bounds совпали. Язык своей учётки изменён штатным HTTP на en, подписи On site/At office с теми же значениями. Скриншоты ru/en визуально прочитаны; они в `B/b29-native-enum-attempt2-20261116` (B — каталог SDK ниже).

Первая native-попытка остановилась до браузера: CLI требует первого bootstrap-admin. Во второй helper создал bootstrap-admin, но браузер вошёл отдельным неадминистратором; SDK не менялся. Оба результата сохранены. Context/browser/driver/app/18586 закрыты, auth DB удалена; Go и браузер не пересекались. Рабочие БД/1С не копировались и не подключались. Linux browser native не приписывается.

**SDK для cc2 / release:**
- Windows `{w['path']}`; SHA256 `{w['sha256']}`.
- Linux `{l['path']}`; SHA256 `{l['sha256']}`.

Manifest `{out}`, SHA `{hashlib.sha256(out.read_bytes()).hexdigest()}`. Build `{final['build_manifest']}`. Native `{nativefile}`, SHA `{final['native_receipt_sha256']}`. Go1.26.11/CGO0/GOMAXPROCS2/-p1; embeddedVCS в worktree SDK не приписывается, provenance — точные clean pins/команды/buildinfo/SHA.

**Независимый PG312 PENDING:** `{pgpath}` — exact Linux prebuilt API116 (=старые12+новые enum104), UI190, interpreter6, полные names/regex/SHA. Критерий PASS312uniqueleaf/FAIL0/SKIP0 + namespace/relations/routines before=after/cleanup. B28 PG208 и B22 PG196 не объявлялись выполненными; новая полоса содержит те же матрицы и enum на B29, старые результаты не переименовывать. Source CI #2637 на последнем чтении18success/3running, Ready не выставлялся.

Полный source CI GREEN22: **T8 #2635**, П4 #2630, П7 #2629, ACL #2631, ordinary XLSX #2632; B26 #2634 также GREEN. Родитель П1 #2627 имеет отдельный bench failure `Upsert_SQLite +44.96% (разброс ±22%)`; storage blobs равны main, причину нельзя считать доказанным flake. Гейт не ослаблял, повтор job GitHub требует admin rights. T8 остаётся Draft по зависимости.

**Далее:** приоритетное дерево меню1С (Draft #2636, ветка `feature/recursive-command-menu-20261116`, пока проект контракта), затем П8–П25 и отчётная очередь. A080 MMMM/цепочка календарных умолчаний добавлены как несрочные, пока не взяты. Меню ОператорКЦ/ОператорМессенджера, прикладная конфигурация и остальные чужие задачи не менялись. Никакого рекурсивного поиска по C:; новых агентов не создавал. Владелец последним.

Generated-with: Codex
''',encoding='utf-8')
files=[report]
names=['build-b29-enum-20261116.py','resolve-b29-enum-port-20261116.py','native-b29-enum-20261116.py','repair-b29-native-bootstrap-20261116.py','native-b29-enum-attempt2-20261116.py','finalize-b29-enum-handoff-20261116.py','manifest-b29-enum-final-20261116.json','b29-independent-pg-handoff-20261116.json','enum-source-http-attempt1-20261116.jsonl','enum-source-http-attempt2-20261116.jsonl','enum-source-http-attempt3-20261116.jsonl','enum-source-build-20261116.log','enum-source-i18n-20261116.log','t8-pr-body-verified-20261116.md','update-t8-pr-body-20261116.py','p1-ci-bench-133225537803-cache-fixed.log']
files += [B/n for n in names]
files += [B/d/'receipt.json' for d in ['b29-native-enum-attempt1-20261116','b29-native-enum-attempt2-20261116']]
files += [p for p in (B/'b29-enum-build-20261116').iterdir() if p.is_file() and p.suffix in ['.json','.jsonl','.log','.txt']]
inventory=[{'source':str(p),'target':'reports/'+p.name if p.parent==R else 'support/'+str(p.relative_to(B)).replace('\\','/')} for p in files]
lst=B/('support-'+tag+'-files.json');assert not lst.exists();lst.write_text(json.dumps(inventory,ensure_ascii=False,indent=2)+'\n',encoding='utf-8')
print(json.dumps({'report':str(report),'manifest':str(out),'pg_total':312,'backup_tag':tag,'backup_list':str(lst),'files':len(files)}))
