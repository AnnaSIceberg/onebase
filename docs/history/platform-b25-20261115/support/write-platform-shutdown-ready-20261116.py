import datetime,gzip,hashlib,json,pathlib,subprocess
B=pathlib.Path(__file__).parent
R=pathlib.Path('C:/Users/a.sevrjukova/AppData/Local/Temp/claude/C--Users-a-sevrjukova-onebase/cd50d987-9739-4c93-a5be-68d748529225/scratchpad/отчёты-для-analysis')
W=pathlib.Path('C:/Users/a.sevrjukova/platform-report-group-levels-20261116')
def git(*a):return subprocess.check_output(['git','-c','gc.auto=0','-c','http.sslBackend=openssl',*a],cwd=W,creationflags=subprocess.CREATE_NO_WINDOW).decode('utf-8').strip()
refs=json.loads((B/'shutdown-platform-refs-20261116.json').read_text('utf-8'))
extra=['feature/report-tabular-result-20261114','fix/side-panel-full-height-source','fix/side-panel-full-height','wip/platform-b24-final-b25-handoff-20261115']
remote={s.split()[1].removeprefix('refs/heads/'):s.split()[0]for s in git('ls-remote','fork',*['refs/heads/'+x for x in extra]).splitlines()}
rows=refs['branches'][:]
expected_extra=['bed39b9d6083c9b8d4bf9a9dce741dbd5fc05902','2a0f4acad4df556dcd8d4676ef2967aa050fe687','513274b7e1bc230d8c0de5fcd22a34deb672345c','c15f0da4bb75e04894d7dbea0dff748d46641d86']
for branch,head in zip(extra,expected_extra):
 assert remote.get(branch)==head
 rows.append({'branch':branch,'head':head,'remote_head':remote[branch],'remote_matches':True,'status':'','worktree':'authoritative remote pin verified by ls-remote; old local aliases not changed; support ref belongs to separate support repository'})
process=json.loads((B/'shutdown-platform-processes-20261116.json').read_text('utf-8-sig'))
assert not process['ownProcesses']and not process['ownListeningPorts']and not process['globalHeavyLockHeld']
tag=datetime.datetime.now().strftime('%Y%m%d-%H%M')
table='\n'.join('| `'+x['branch']+'` | `'+x['head']+'` | совпадает |'for x in rows)
proof=B/('shutdown-platform-remote-proof-'+tag+'.json');proof.write_text(json.dumps({'checked_utc':datetime.datetime.now(datetime.timezone.utc).isoformat(),'rows':rows,'process_receipt':str(B/'shutdown-platform-processes-20261116.json')},ensure_ascii=False,indent=2)+'\n',encoding='utf-8')
report=R/(tag+'-platform-готов-к-выключению.md')
report.write_text(f'''# Platform — готов к выключению

**Контрольная точка по её команде. Все текущие изменения закоммичены и pushed. Сверены полные HEAD с `git ls-remote fork`; все13 проверенных рабочих копий CLEAN. Новые тяжёлые работы не начинал. Собственных Go/Chrome/Python/SDK/SSH процессов и портов18586/18587 нет, общий замок не держу. Можно выключать ПК.**

## Где остановился

1. **Маска даты накоплена B30, но полный source CI красный — не готовая поставка.** Runtime `d0cb9b15100ecdc6f13a8234dc4b2e3f99fa2eff`, source Draft #2611 `9a81ab6cbcb96c834c184ca24c00c1ec474ba90c`, main cb2e0acee31de5800699f46af5115b3e458a909b. HTTP131 leaf PASS/7PG-rootSKIP, Node29, source/runtime браузер PASS; native45 PASS ОТК+ОператорКЦ/JS0/cleanupalltrue на новой SQLite. Windows/Linux SDK есть; независимый PG312 PENDING. Полная передача `20261116-1532-platform-маска-даты-B30-native45-SDK.md`, manifest `B/manifest-b30-date-mask-final-20261116.json`.
2. **Следующая пачка — уровни/итоги/фон + универсальные отборы.** Source Draft #2638 `feature/report-group-levels-20261116` пока только контракт, HEAD `b36f7e93befa5ed1c5ab3c12a1dcee4c6359bcdf`, зависимость #2627; main обновлён. Незавершённые10 файлов вынесены в отдельную **WIP `579155942a5cdfe336f5148203b45fdb417dc725`**, branch `wip/platform-report-levels-checkpoint-20261116-1553`, WT `C:/Users/a.sevrjukova/platform-report-group-levels-20261116`, CLEAN/pushed. В Draft рабочий незавершённый код не отправлен. Нового SDK этой пачки нет, Go/браузер для неё не запускались. Отборы ещё не начаты, отдельного Draft/ветки отборов нет.
3. По срочному запросу ledger передан `20261116-1504-platform-ledger-контракт-для-perenos-codex.md`: server ledger-gate **не реализован ни B26, ни B29/B30**, найден SOURCE GAP PLAN/неисполняемый binding, не финальный wire. 26 anchors сравнены, критические storage/service пути идентичны source-gap. Не выдавать PLAN за готовый API.

## С чего продолжить после включения

- Сначала прочитать этот файл + память `B/platform-memory-shutdown-20261116.md`, проверить новые сообщения/журнал, не начинать прежние ветки заново.
- Разобрать CI #2611, run **43867622295**; квитанция `B/shutdown-date-source-ci-20261116.json`, полные логи/выдержки сохранены. **lint:** unused `(*Server).uiEnhancementsAllowed` в `internal/ui/ui_role_scope.go:9`. **Windows и unit-race/ui-1:** `TestManagedDateColumnBehavior`, Node VM: `window is not defined` после добавления hook в managed.js; поправить совместимость/фикстуру осмысленно, не отключать тест. **e2e:** form-density/form-group-spacing default+compact, несовпадение CSS; точные логи сохранены. PG/unit-race(other) ещё шли на момент снимка, полный GREEN не заявлять. У enum #2637 отдельный прежний общий PG timeout UI-matrix, enum SQLite+PG сами PASS. У родителя #2627 прежний bench UpsertSQLite slowdown — не доказанный flake.
- Продолжить WIP уровней в указанном WT (он уже на WIP-ветке). Сделано частично: новые цифры в rc JS, +/−/aria в группах, удалён вызов подвалов, «Итого» и request-local представления из доказанной projection metadata; заготовлены row-style carrier и стили для XLSX/PDF/direct/background. **Не закончено:** viewerScript ТабДок с цифрами; шаблон старого toolbar/CSS/тёмная тема; en-ключи; удаление мёртвых subtotal методов/комментариев; безопасная нормализация условного цвета для экспортов; gofmt; тесты и документация. `internal/ui/report_composition_presentation.go` новый файл включён в WIP. `joinStyles` оставлен двухаргументным, новая переменная CSS склеена в строку. Изменения ещё не компилировались, возможны ошибки; это WIP, не подтверждённая реализация.
- Доделать универсальные отборы: любой разрешённый result/grouping field, список полей/типизированные значения, сравнения/multiple AND, пользовательский вариант, скрытые ПДн/роли исключить и валидировать на сервере. Согласовать точки соприкосновения с platform3. Доработки дат в runtime сохранить, не менять nav/static mount дерева меню.
- **Её новое правило:** тяжёлые прогоны ПАКЕТОМ — одна проверка накопленных уровней/итогов/фона/отборов, не на каждый коммит. До неё разработка и лёгкие проверки. Go/Chrome последовательно, общий замок согласовать/использовать по актуальному runner владельца. После новых ресурсов ПК старое ограничение8ГБ автоматически не переносить в предположения; правила/замок соблюдать.
- После общего прогона накопить поверх B30 отдельную следующую runtime-линию, точные SHA/manifest/native/PG-гейты файлом. Исторические подтверждения B21/B28/B29 не переименовывать в новые.

## Кто что ждёт

- **Меню — platform2 (wV:p1)**: source c8859ae8a2060fa609e208184dfc76e3885f58db, B29-кандидат6b19acb45 из его файла1532; финальные проверки/квитанция ещё ожидаются, бинарь меню не собирал. Границы nav/static.go приняты; новые nav-recursive-section/nodes + mountNavigationTree сохранять. Передача B30/GoEND ему `20261116-1535-platform-для-platform2-B30-границы.md`.
- **Зависимости параметров Филиал→Направление, сортировка/представление и ПДн отчётных представлений — platform3 (wW:p1).** Сняты с моей очереди по последнему слову analysis. У меня остаются маска→уровни/итоги/фон→отборы. Его обязательная политика: телефоны/полные адреса никому в отчётах/экспорте/расшифровках; карточки не менять, UUID/переходы сохранять. В B30 такой новой общей политики ещё нет.
- Независимый review/PG: B30 PG312 ещё PENDING; точные prebuilt+regex+имена `B/b30-independent-pg-handoff-20261116.json`. B28 PG208/B29 PG312 не считать закрытыми без их квитанций. CI ошибки не флейковать без доказательства.
- Несрочный отчётный backlog П8–П25 после сверки (T8/П25 уже сделан), П27/П28 A080 MMMM/цепочка defaults; П29 A081 строковый LIKE-конкат. Эти три заказа пока не реализованы. Очередь владельца последняя.

## Полные HEAD — сверены ls-remote

| Ветка fork | Полный HEAD | Удалённый HEAD |
|---|---|---|
{table}

Support WIP в таблице — снимок **до записи этого checkpoint**. Этот файл, память, refs/process receipts и CI-логи сохраняются следующим коммитом в той же support-ветке; финальная точная квитанция после push — `B/support-{tag}-push.json` (она проверяет ls-remote). Не требуется менять код/ветки ради включения SHA коммита в его собственное содержимое.

## Файлы / процессы / ограничения

Квитанция18 refs `{proof}`;13WT clean — `B/shutdown-platform-refs-20261116.json`. Процессы `B/shutdown-platform-processes-20261116.json`: точный elevated CIM по моим путям, PID0/listeners0; прежние native cleanupalltrue. Чужих процессов/агентов не останавливал. Новых тяжёлых запусков после команды выключения нет, общий lock не взят/не удерживается. Выполнены только checkpoint/git/чтение готовых CI-логов. Последние короткие git/Python helper завершаются штатно после записи и push.

**Общие стенды не менялись.** Основной ЛПЦ :8448, любая выкладка ТОЛЬКО по её слову одним накопленным пакетом; :8444/:8443 тоже не трогать, B29/B30 по этим отчётам не устанавливать. Новых агентов не создавать. Рекурсивных поисков C: нет. Все коммиты Generated-with: Codex, только явные файлы, без add-A, без ручного звука.

B = `C:/Users/a.sevrjukova/AppData/Local/Temp/platform-help-admin-20261111`.
R = общая scratchpad/отчёты-для-analysis. Root checkout с чужими изменениями не трогать.

Generated-with: Codex
''',encoding='utf-8')
memory=B/'platform-memory-shutdown-20261116.md'
memory.write_text(f'''# Память platform перед выключением 16.11

Обязательная точка возобновления: `{report}`. Все ветки/полные SHA/ls-remote/status/CI/следующие шаги там. Работа остановлена по прямой команде выключения, не продолжать её автоматически до включения/возобновления.

Role platform. Не platform2. Root C:/Users/a.sevrjukova/onebase чужой dirty, свои WT. Не искать рекурсивно C:. Reports только файлами в R. Source upstream-first Draft/fork AnnaSIceberg→ivanarama. Generated-with: Codex, explicit add, всё push. Никаких shared deploy без её слова, только накопленным пакетом в ЛПЦ8448;8444/8443 также запрещены без слова. Тяжёлый прогон один на пачку. Не создавать новых агентов.

Маска B30 d0cb9b15100ecdc6f13a8234dc4b2e3f99fa2eff: SDK Win121fe5e8d8cd193a3a7e374dd9c94806080b8bdcda8648d246389ff281a40bef, Linux dee4ea598016d6c499ea118898b7099a8544dd20ba4964023ca039f95a87df87. Native45 PASS/JS0/cleanuptrue; независимыйPG312 PENDING. Source #2611 9a81ab6cbcb96c834c184ca24c00c1ec474ba90c: CI FAIL (lint unused uiEnhancementsAllowed; Node VM window undefined in ManagedDateColumn; e2e group spacing), полный GREEN не заявлять. Последний main cb2e0acee31de5800699f46af5115b3e458a909b, merge source только release docs после проверенного fc8777, нет новых Go/JS.

Активный WIP уровней579155942a5cdfe336f5148203b45fdb417dc725, WT C:/Users/a.sevrjukova/platform-report-group-levels-20261116 на wip/platform-report-levels-checkpoint-20261116-1553; 10 файлов, неполный/не собран/не проверен. Source Draft #2638 только контракт b36f7e93befa5ed1c5ab3c12a1dcee4c6359bcdf, parent #2627 bed39b9d6083c9b8d4bf9a9dce741dbd5fc05902. Никакого B31 SDK нет. Точные недоделки в checkpoint. Отборы ещё не начаты.

Меню — platform2, не трогать его nav и mountNavigationTree. Параметры и ПДн — platform3, сняты с моей очереди. Моя очередь только маска/CI → уровни+итоги+фон → отборы. П27/28/29 backlog, ownerlast. Ledger не реализован, ответ1504PLANonly. P4/P7 уже вB21 и унаследованы всемиB25+, не переносить дважды (старый отчёт1257 ошибался, исправлен1318).

Go1.26.11 path C:/Users/a.sevrjukova/go/pkg/mod/golang.org/toolchain@v0.0.1-go1.26.11.windows-amd64/bin/go.exe; gh C:/Users/a.sevrjukova/gh-cli/bin/gh.exe; Python C:/Python314/python.exe. Python subprocess CREATE_NO_WINDOW, UTF8 **явно** в read_text/write_text: одна backup-файллиста ранее испорчена cp1251 и исправлена1552b; не повторять read_text без encoding на кириллице. Node --test sandbox EPERM, запуск файла напрямую работает; Go/node/IPC пишущие git часто требуют require_escalated. PS не разворачивает glob аргумент rg — использовать rg dir -g '*.go'. gh pr edit староеProjectsclassic ломается, REST PATCH stdin JSON helper.

Support backup script B/push-platform-reboot-support.py не идемпотентный при update-ref до failed push: сначала проверить head/remote. Последний до checkpoint c15f0da4bb75e04894d7dbea0dff748d46641d86. Checkpoint backup tag{tag}, итог support-{tag}-push.json. Нативные SQLite с auth удалены; sharedDB никогда не копировались. No own Go/Chrome/Python/tunnels or heavy lock before shutdown.

Generated-with: Codex
''',encoding='utf-8')
queue=R/'20261116-platform-текущая-очередь-дата-уровни-отборы-меню.md'
queue.write_text(f'''# Platform — актуальная очередь после разделения и checkpoint

Работа остановлена для выключения ПК по её прямому слову. Главный checkpoint `{report.name}`, память `B/platform-memory-shutdown-20261116.md`.

1. Маска накоплена B30/native45 PASS, source CI FAIL: разобрать lint/e2e/Windows/ui-1, полный GREEN не заявлять. PG312 PENDING.
2. Уровни1/2/3 + без подвалов групп/общий «Итого» + представления заголовков + фон групп в HTML/Excel/PDF. Draft2638 contract-only, незавершённый WIP579155942a5cdfe336f5148203b45fdb417dc725 pushed.
3. Отбор по любому разрешённому полю результата/группировок, типизированные значения, сравнения, AND, пользовательский вариант, исключение скрытых ПДн/ролями полей. Ещё не начат.

Меню передано platform2 wV:p1; связи параметров/сортировка/представление и ПДн ссылок переданы platform3 wW:p1, **не моя очередь**. Базовая runtime B30 d0cb9b15100ecdc6f13a8234dc4b2e3f99fa2eff; ПДн-policy и ledger-gate там ещё нет. Карточки не менять. П27/П28/П29 в backlog, ownerlast.

Тяжёлые проверки — одной пачкой уровни/итоги/фон/отборы, не на каждый коммит; после команды выключения не запущено ни одной. Общие стенды без её слова не менять; только пакет в ЛПЦ8448,8444/8443 не трогать. Никаких рекурсивных поисковC:, add-A или ручного звука.

Generated-with: Codex
''',encoding='utf-8')
files=[report,memory,queue,proof,pathlib.Path(__file__),B/'shutdown-checkpoint-platform-20261116.py',B/'shutdown-platform-refs-20261116.json',B/'shutdown-platform-processes-20261116.json',B/'shutdown-date-source-ci-20261116.json',B/'group-levels-pr-body-20261116.md',B/'native-b30-date-mask-body-20261116.py.txt',R/'20261116-1535-platform-для-platform2-B30-границы.md']
for entry in refs['ci_logs']:
 if 'log'in entry:
  raw=pathlib.Path(entry['log']);compressed=raw.with_suffix(raw.suffix+'.gz');assert not compressed.exists();compressed.write_bytes(gzip.compress(raw.read_bytes(),compresslevel=1,mtime=0));files.extend([compressed,pathlib.Path(entry['failure_excerpt'])])
items=[]
for p in files:
 assert p.is_file()and p.stat().st_size<2*1024*1024,p
 items.append({'source':str(p),'target':('reports/'+p.name)if p.parent==R else 'support/'+str(p.relative_to(B)).replace('\\','/')})
lst=B/('support-'+tag+'-files.json');assert not lst.exists();lst.write_text(json.dumps(items,ensure_ascii=False,indent=2)+'\n',encoding='utf-8')
print(json.dumps({'ready_report':str(report),'memory':str(memory),'remote_refs_verified':len(rows),'tag':tag,'backup_list':str(lst),'backup_files':len(items)}),flush=True)
