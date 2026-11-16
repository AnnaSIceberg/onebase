# Platform3: остановка перед выключением ПК, 16.11.2026 15:55

Пользователь остановил работу для добавления CPU/RAM. Новых тяжёлых работ не начинать до возобновления. Роль — platform3, не platform2. Задачи: (1) Филиал → Направление в параметрах отчётов; (2) политика ПДн в отчётах, прежде всего подписи ссылок. Ничего не выкладывать: пакет в ЛПЦ :8448 только её словом; :8443/:8444 также не менять.

## Сохранённый код

- Source ветка `feature/platform3-report-choice-20261116`, полный HEAD `e52801ca03ef6f4c36a5dc153a64fa30b18d50dc`, fork AnnaSIceberg/onebase. Сверен через git ls-remote 16.11 в 15:54. Upstream Draft https://github.com/ivanarama/onebase/pull/2639, не Ready. Основа origin/main `cb2e0acee` (полный SHA получить git show); runtime B29 не использован как head upstream PR.
- Source WT `C:/Users/a.sevrjukova/AppData/Local/Temp/platform3-report-choice-source-20261116` (после checkpoint переключён на отдельную WIP ветку для этого снимка памяти).
- Отдельный runtime WT `C:/Users/a.sevrjukova/AppData/Local/Temp/platform3-report-b29-20261116`, ветка `integration/platform3-report-choice-pdn-b29-20261116`, HEAD `902e4d97e50e90f047ec6d27832e2f9d175d095d`: чистый B29, source ещё НЕ перенесён. Бинарь не собирался.
- WIP ветка `wip/platform3-checkpoint-20261116` содержит этот снимок и копию плана. Её точный итоговый HEAD и сверка ls-remote — в отчёте готовности к выключению.

## Что реализовано, но НЕ принято

Параметры `report.Param` получили:

```yaml
params:
  - name: Филиал
    type: reference:Филиалы
  - name: Направление
    type: reference:Направления
    choice_filter:
      - field: Филиал
        param: Филиал
    choice_label_suffix: Филиал
```

Новый `internal/ui/report_parameter_choice.go`: серверные условия равенства с ведущими ссылочными параметрами; пустой ведущий параметр не ограничивает выбор. Представление дочерней ссылки — обычная подпись + подпись ссылочного реквизита в скобках. Сортировка остаётся `order_by` справочника. Начальная выдача и GET-подбор используют эти ограничения. Контекст `form_kind=report`, form=params передаётся в существующий механизм data-ref-choice-context (ui.js НЕ изменён). Источники/операторы из браузера не принимаются. Проверка несовместимого UUID подключена к runReport и reportExportRowsWithContext.

Добавлены `internal/report/parameter_choice.go` (проверка типов/существования источников/циклов) и вызов из CheckLintReports. Обновлены schema report, lint schema и сохранение параметров в конфигураторе. Контракт `docs/report-parameter-choice.md`.

HTTP-матрица `internal/ui/report_parameter_choice_test.go`: SQLite/PG, пустой филиал, разные филиалы, порядок, подпись, поиск, пагинация, несовместимый selected_id, malformed source, права report run, форма, несовместимая пара при построении/Excel/PDF. Тесты добавлены, но НЕ ЗАПУЩЕНЫ. Только gofmt + git diff --check. Первую синтаксическую ошибку composite literal в тесте исправил до commit. Компиляция не проверена.

Изменены существующие файлы: internal/report/report.go, internal/ui/handlers_reports.go, internal/ui/ref_choice_filter.go, internal/ui/handlers_entity.go, internal/ui/templates.go (один data-атрибут), internal/cli/schema.go, internal/configcheck/lint.go, internal/configcheck/report_param_default.go, internal/launcher/configurator_report_module.go. Новые специализированные файлы перечислены выше. Разделение с platform пока только планом: готовый патч требуется согласовать при интеграции с уровнями/отборами, которые platform делает параллельно. Не утверждать, что непересечение уже подтверждено.

## С чего продолжить

1. Проверить source CI #2639 (на остановке IN_PROGRESS, classify SUCCESS; общего GREEN нет). Исправить компиляцию/замечания после просмотра результатов; тяжёлых локальных прогонов до конца checkpoint не запускать.
2. Переключить свой source WT с WIP обратно на feature/platform3-report-choice-20261116. Проверить публичные пути: REST internal/api/v2.go пока НЕ валидирует зависимые параметры; POST preview подбора пока НЕ адаптирован (обычный GET адаптирован). Проверить, требуется ли preview для справочника направлений. Проверить, что CheckLintReports даёт нужный уровень ошибки, а не только advisory.
3. Проверить наследование роли/row access и поведения старых параметров: старый путь source использовал refOptionsChoice, B29 использует refOptionsFilter.withMarked(p.ChoiceShowDeleted). При переносе на B29 сохранить B29 choice_show_deleted, enum, trusted context. Нельзя механически заменить B29 handlers_reports.go source-файлом.
4. Для новой возможности добавить docs/features.md согласно CLAUDE; не пересекаться с незавершёнными блоками platform. Уточнить контракт у codex2 чтением его актуального П14 заказа. Отдельного заказа П14 среди codex2-20261116-* в общей папке пока не было; использованы журнал её решений и реальная карточка звонка.
5. Справочник конфигурации `C:/Users/a.sevrjukova/cc2-wt-kpi-export-codex2/catalogs/Айсберг_Направления.yaml`: order_by [ПорядокВОтчетах, Наименование]. Карточка `forms/а_звонок/объекта.form.yaml`, элемент ПолеНаправление (~726): Филиал eq Объект.ФилиалАйсберг; parent_id not_in_hierarchy c976331e-6119-41c6-9e9e-a4e26fefd169; is_folder=false; ОбщийСклад=false. Новая минимальная грамматика report choice_filter пока поддерживает только eq по параметру: проверить, нужен ли полный повтор этих статических исключений в отчётах; не обещать его.
6. ПДн: только исследование, НИКАКОГО кода/PR/исправления ещё нет. Нужен отдельный source Draft от свежего main и поддерживаемая политика вывода, независимая от прав карточек. Не маскировать сырые строки наивным regex и не менять данные справочников. Уточнить механизм для произвольного result_proc/управляемых KPI, прямых значений и подписей ссылок; не заявлять полное покрытие без происхождения данных/финального output-gate.
7. После обоих результатов один общий пакет: общий замок, Go -p1/GOMAXPROCS2, SQLite/PG HTTP, браузер отдельно. SDK собирает/накапливает platform после передачи source/runtime SHA, либо мы только по согласованной сборке. Выкладки нет.

## ПДн: точные входы и требование

Общая папка R = `C:/Users/a.sevrjukova/AppData/Local/Temp/claude/C--Users-a-sevrjukova-onebase/cd50d987-9739-4c93-a5be-68d748529225/scratchpad/отчёты-для-analysis`.

- R/codex2-20261116-заказ-platform-ПДн-представления-ссылок-B29.md — прочитан. Синтетика codex2 f024f8ec; продукт81bf88cc, DSL afa3b77f, WIP после уточнения ee647026. Оригиналы в C:/Users/a.sevrjukova/cc2-wt-kpi-export-codex2/tools/otk/task15/http-B29/ (в текущем WT на остановке каталога task15 не было; найти git-ветку, не считать файлы доказательств потерянными).
- Воспроизведение: одна только замена Представления заявки на строку с 79990009987 / ДОМ_ПДН_987 / КВ_ПДН_654 приводит к этим данным в HTML/XLSX/PDF, даже если телефон/адрес не выбраны запросом. Источник — resolveUUIDsInReport и maskedRecordLabel, который учитывает обычные ролевые права, поэтому admin не защищён.
- Требование: телефоны и полные адреса скрыты ВСЕМ в отчётах, включая admin/disclose/read:full; адрес без дома/квартиры, телефон не выводить/полная маска. Покрыть HTML, внутренние расшифровки, tooltip, печать, XLSX/XML/relationships/скрытые данные, PDF с нормализацией переносов, управляемые отчётные KPI. UUID/переход сохранить. Обычную карточку и field_access/disclose не менять (её уточнение 16.11 ~16:45).

## Процессы/замок/инструменты

Свои серверы, Go-тесты/сборки, Chrome, Python, SSH/туннели за этот запуск не запускались; Go только gofmt. Проверка Get-CimInstance с правами 15:54 не нашла go/chrome/python/ssh/onebase с platform3-report в command line. Чужие процессы не останавливать.

Общий lock `C:/Users/a.sevrjukova/AppData/Local/Temp/claude/c--Users-a-sevrjukova-onebase/cd50d987-9739-4c93-a5be-68d748529225/scratchpad/perenos-codex/тяжёлый-прогон.lock` не захватывал. На чтении 15:53 там был чужой владелец platform2 menu Build pid=21356; не удалял и не менял.

gh: C:/Users/a.sevrjukova/gh-cli/bin/gh.exe. Git мутации в .git и сеть git ls-remote требуют require_escalated; обычная песочница отвечает Permission denied / schannel SEC_E_NO_CREDENTIALS. Это не отказ политики; разрешённые повторы с require_escalated выполняются. PowerShell login:false, UTF8 OutputEncoding/Get-Content -Encoding UTF8. Не использовать wildcard в позиционных путях rg на Windows: rg -g по каталогу. Фоновые окна скрытые. Звук вручную не вызывать.

Generated-with: Codex
