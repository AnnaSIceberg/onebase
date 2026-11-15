# П7 source-review 2766 — REQUEST CHANGES

Точный HEAD `2766ef28f587abbfe1334afb53e120b6304cfc2b`, base `4379007a81d1bfc8751ebcf8dbb1143ba6460ed8`, Draft2629. Code-only независимое review по поручению автора; Go/CLI/Chrome/DB/runtime-порт не запускались. Прочитан весь product diff и новые tests. Все19 changed Git blobs сверены с source manifest92e97dc5; дополнительные source anchors закреплены по Git blob, а не по изменяемому WIP.

## F1 — P1 blocker: native configurator стирает trusted context

`internal/launcher/configurator_report_module.go:40–48`: локальный `saveParam` не содержит `Context`.

`...:101–118`: `updateReportFile` декодирует существующий YAML в `saveReport`, затем сохраняет по имени только Default/Options/PeriodEnd. Неизвестный `context` теряется уже при decode в этот тип.

`...:141–145`: `setYAMLMapField(doc, "params", params)` заменяет весь массив с полученными параметрами. Сохранение существующего отчёта через обычную native форму (даже правка title/query без изменения имён/типов) удаляет `params[].context`. `cfgParam`/`configurator_load.go:415` также не содержат этого признака.

После перечитывания `Param.Context=nil`; `ResolveContextParams` на отчёте без context сразу возвращает входные значения (context.go:116–123). Параметр Actor/Unit снова попадает в editable input и перестаёт получать серверный override. Доверенный конфигуратор имеет право менять конфигурацию, но обычный save не должен молча отменять security-семантику не редактировавшегося поля.

Это source-proven детерминированная цепочка, не утверждение, что выполнен runtime GET→POST. Текущие20 local/source PG18 проверок launcher-save не покрывают.

**Минимальная доработка у owner:** сохранить существующий context по идентичности/имени вместе с прочими не редактируемыми native формой свойствами либо закрыть такой edit безопасным отказом. Не брать trusted descriptor из произвольного client POST. Нужен public native configurator GET → обычный form POST → report.LoadFile regression: user:id/login и user_card не теряют binding; повторная генерация с client spoof остаётся server-bound. Перестановка параметров не должна переносить context на чужой параметр; rename/type-change должны иметь явную семантику/отказ, а не случайный trust downgrade. Исправление делает platform2; peer WT не изменял.

## Проверенное без найденного blocker

- `ResolveContextParams` берёт auth.User из server context, копирует карту, удаляет case-variant клиентские aliases и присваивает server source. Missing actor/card/ambiguous/error отказывают; default не заменяет потерянный контекст. External reports запрещены.
- ScalarUserCardDescriptor — узкая metadata-валидация через существующий userCardDescriptor; обычное скалярное поле, единственный account reference:_users, без table parts/privileged writer. Reader использует одну SQL statement для uniqueness/value и actorUUID, ordinary catalog browse не расширяется. Нулевые bool/number не становятся отсутствием только из-за значения0/false.
- UI bind стоит в начале runReport; context исключён из defaults/request inputs/UI controls/export URL и повторного text-parser. API v2 bind до compile/execute, OpenAPI input props исключены. Export common rows path bind до compile, background worker проходит тот же путь с request actor и новым чтением карточки. Query RBAC/RLS/masking не отключались.
- Отдельной REST v1 execution-ветки отчётов в просмотренном source не найдено; актуальный публичный REST entrypoint здесь `/api/v2/report/{name}`.

## N1 — документы

`docs/report-trusted-context.md:57–58` остался статус «Go/SQLite/PostgreSQL gates ещё не выполнены». Для2766 Go/SQLite уже выполнены, теперь PG18 также подтверждён. Убрать временный статус из постоянного контракта или актуализировать в handoff; runtime availability по-прежнему не объявлять. Это не опровержение выполненных гейтов.

## Обязательные стыки следующего runtime gate

Это не дополнительные source-blockers, но текущие test fixtures покрывают user_card Department **string**, а реальный подразделенческий контекст часто reference/UUID. При runtime-порте добавить актуальный typed-reference user_card gate (UI/REST/export) с неправильным client UUID и неизменённым server UUID; context не reparsить. Сохранять0/false/date/decimal типы существующего reader.

B21 переносить только узкий source range поверх exactB20 `e6cb9a375b0bae15a430ae93b0913c02938bb45e`: P2 QueryForVariant/filters, P3 DetailFields/rowWithDetail, B19 input-limit, B20 dimension, П1 result_proc/snapshot должны остаться. Binder поставить ДО early result_proc branch. Отдельно runtime regression для result_proc trusted parameter: source main437 этой ветки П1 не содержит. CLI `report explain --sample` — прежний privileged diagnostic path; его нельзя автоматически объявлять защищённым П7 manual/report execution без отдельного bind/явной оговорки scope.

## Квитанции

Root exact SOURCE test-only APIac2664ff:3SQLite+3PG=6, UI2703f655:6SQLite+6PG=12 — **18 unique leaf PASS**, FAIL0/SKIP0, cleanup exact95a93e4b...e7dc4. Обе квитанции независимо прочитаны и SHA-сверены. Не B20 SDK и не runtime-port PASS.

`20261115-platform-П7-source-review-2766-pins.json` SHA `60fc38ce4ba53ce3d8c937846b00040653fe515d7eb250a738df049fff1cd261` хранит changed/source anchors и PG receipts. Product delta de1→2766 действительно только2 test fixtures EnsureSchema; product совпадает с de1.

**Итог:** до исправления F1 CODE REVIEW OK не даю, conditional runtime-port не начинаю. P4 Go-окно автора и frozenB20 не затрагивал. После нового exact source HEAD — review дельты + гейт launcher, не новый PLAN-цикл.

Generated-with: Codex
