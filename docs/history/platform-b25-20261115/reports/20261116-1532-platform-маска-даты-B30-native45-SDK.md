# Platform: маска даты — B30, native45 PASS; общие стенды не менялись

**Маска реализована, source Draft #2611 обновлён main, B30 накоплен поверх B29. На новой собственной SQLite под ОТК и ОператорКЦ native45 PASS, JS0; браузеры/стенд остановлены, authDB удалена. Независимый PG312 PENDING; source CI идёт, полный GREEN не заявляется. Выкладка только одним накопленным пакетом в ЛПЦ :8448 по её слову.**

## Код и контракт

- Draft https://github.com/ivanarama/onebase/pull/2611, branch `feature/report-and-date-controls-20261113`, HEAD `9a81ab6cbcb96c834c184ca24c00c1ec474ba90c`. Source code `fc8777bc33c1dfae23cc8bc4c9da9df378d2e18a`; после него только merge опубликованных документов v0.14.0, Go/JS bytes не менялись. Main `cb2e0acee31de5800699f46af5115b3e458a909b`, конфликт CHANGELOG сохранён с нашими изменениями в «Не выпущено».
- Runtime **d0cb9b15100ecdc6f13a8234dc4b2e3f99fa2eff**, branch `integration/cc2-b30-date-mask-20261116`, WT `B/report-b30-date-mask-runtime`, CLEAN/pushed. Это B29 `902e4d97e50e90f047ec6d27832e2f9d175d095d` + маска и браузерная фикстура. П4/П7 уже из B21; B22, B26, T8, enum сохранены, повторных реализаций нет.
- Существующий `config/app.yaml ui.short_date_input: true` включает полный шаблон `__.__.____`, дата-время `__.__.____ __:__:__`. Он независим от ролей/плотности, включает операторов КЦ. Прикладную конфигурацию не менял; cc2 сохраняет действующий true в накопленной конфигурации. Новый неподдерживаемый ключ не требуется.
- Ввод только цифр, точки/двоеточия автоматически; курсор переходит через разделители. Backspace/Delete/выделение; вставка `16.11.2026`, `16112026`, ISO; ошибка неполной/невозможной даты. Enter завершает дату и не отправляет форму/отчёт, даже `enter_submits_form: true`. Escape восстанавливает прежнее значение. Календарь кнопкой/Alt+Down, исходный input сохраняется. Сервер получает ISO; локальные часы/секунды восстанавливаются после хранения в UTC. Объектные формы, параметры отчётов, отборы списков.
- Без прочих изменений операторского интерфейса. Маска отсутствует в старом SDK B29, новый ключ на B29 её не добавит.

## Проверки

Source Node29 PASS (включая Ctrl/Meta+V после Escape), панель отчёта и 36 ролевых сочетаний HTTP PASS; source build ./... и i18n PASS. Source браузер через prebuilt test.exe PASS с настоящими клавишами/буфером и нулём POST. Source receipts `B/date-mask-source-checks-attempt2-20261116/receipt.json`, `B/date-mask-source-browser-attempt3-20261116.json`.

B30 HTTP131 leaf PASS / 7 PG-root SKIP; build ./... / i18n / Win+Linux SDK / Linux prebuilt API/UI/interpreter и Windows UI test.exe PASS. Браузер **runtime** через prebuilt PASS, отдельно от Go; квитанция `C:\Users\a.sevrjukova\AppData\Local\Temp\platform-help-admin-20261111\b30-prebuilt-date-browser-attempt1-20261116.json`.

Нативный SDK: **45 PASS**, обе роли не admin; отчёт, даты/секунды/Backspace и неполнота (source browser), две настоящие вставки (native обе роли), календарь, Enter без POST, managed-object с `enter_submits_form`, явная запись с активной маской, SQL-момент с учётом UTC и повторно открытая форма, отбор списка; JS0, cleanupalltrue. `C:\Users\a.sevrjukova\AppData\Local\Temp\platform-help-admin-20261111\b30-native-date-mask-attempt2-20261116\receipt.json`, SHA256 `bc2a8dba8d1369082871da67270f21caf805e44add6d131fcee9b32771cb6410`. Снимки `B/b30-native-date-mask-attempt2-20261116/*-live-mask.png`, `*-object.png`, `*-filter.png`; два снимка просмотрены.

Прозрачность попыток: source browser attempt1 выявил Ctrl+V после Escape и неполный SSE hub; они исправлены. Attempt2 выявил nil interpreter **тестовой фикстуры**, исправлен; attempt3 PASS. Native attempt1 остановился на ошибочной проверке текста UTC как местного времени, все процессы/база очищены; attempt2 проверяет момент и reopened UI, PASS. Неудачные журналы не перезаписаны. Новых продуктовых правок после сборки SDK не было.

Независимый PostgreSQL: **312 PENDING**, exact regex/имена/новые SHA prebuilt в `C:\Users\a.sevrjukova\AppData\Local\Temp\platform-help-admin-20261111\b30-independent-pg-handoff-20261116.json`. Исторические B21/B28/B29 проверки не перетегированы. Source CI #2611 идёт после merge main; не подменять 131SQLite/native45 полным CI или PG.

## SDK

- Windows `C:\Users\a.sevrjukova\AppData\Local\Temp\platform-help-admin-20261111\onebase-1116b30-date-mask-windows-amd64.exe`, SHA256 `121fe5e8d8cd193a3a7e374dd9c94806080b8bdcda8648d246389ff281a40bef`.
- Linux `C:\Users\a.sevrjukova\AppData\Local\Temp\platform-help-admin-20261111\onebase-1116b30-date-mask-linux-amd64`, SHA256 `dee4ea598016d6c499ea118898b7099a8544dd20ba4964023ca039f95a87df87`.
- Final manifest `C:\Users\a.sevrjukova\AppData\Local\Temp\platform-help-admin-20261111\manifest-b30-date-mask-final-20261116.json`, SHA256 `589822076155c93f204eb4969e810a3b2c535530be83e8c252d075bf3a3e2feb`; build `C:\Users\a.sevrjukova\AppData\Local\Temp\platform-help-admin-20261111\b30-date-mask-build-20261116\manifest.json`.

## Дальше

Уровни 1 2 3… для ТабДок/составного отчёта + без подвалов групп, общий «Итого», label заголовков, бледный фон групп по глубине (HTML/Excel/PDF, тёмная тема). Screenshot2/4 просмотрены. Затем отборы по любому разрешённому полю → связь Филиал/Направление и сортировка/представление (заказ ожидается) → движковая защита ПДн ссылочных представлений по `codex2-20261116-заказ-platform-ПДн-представления-ссылок-B29.md`. Её запрет телефонов/полного адреса для всех ролей принят; в B30 он ещё не реализован. Карточки не менять, UUID/переходы сохранять.

Дерево меню у platform2 wV:p1, точные B28/B29/SDK переданы `20261116-platform-для-platform2-runtime-B28-B29-меню.md`; platform не продолжает меню. B30 можно брать как следующую накопленную основу после проверки, без общей установки. Ledger ответ передан `20261116-1504-platform-ledger-контракт-для-perenos-codex.md`, server gate не реализован.

Общие :8448/:8444/:8443, рабочая 1С/БД не менялись, базы не копировались. B29/B30 не выкладывать по этому отчёту. Новых агентов не создавал, рекурсивных поисков C: не делал, звука вручную нет.

B = `C:/Users/a.sevrjukova/AppData/Local/Temp/platform-help-admin-20261111`.

Generated-with: Codex
