# KPI T3: существующий маршрут обработки и отдельный API gap

Прочитан заказ `perenos-codex/отчёты/codex2-20261115-KPI-T3-переход-к-форме-подготовки-platform.md`. Source RO exactB20 runtime `e6cb9a375b0bae15a430ae93b0913c02938bb45e`; HTTP/Chrome/Go/конфигурацию не запускал и не менял. Не блокер сегодняшнего выпуска по поручению perenos.

## Что действительно отсутствует

Глобальный `ОткрытьФорму(Ссылка)` в `internal/ui/form_navigation.go:30–62` принимает только сохранённый typed Ref сущности. Строка `Обработка.А_ОтчетKPIБригадиру` этим builtin не поддержана. `ОткрытьФормуРезультата` тоже не является универсальным processor-opener: требует исходную сохранённую карточку, typed Ref и declared parameter/guard. Не предлагаю ради T3 создавать фиктивный документ или обходить guard.

Штатного handle `ПолучитьФорму(...).Открыть()` с выбором произвольной именованной формы/набором processor-параметров этим RO не найдено; такой общий API остаётся отдельным узким заказом после выпуска.

## Уже существующий официальный путь к default форме обработки

В B20 есть server DSL notification-команда открытия зарегистрированной обработки. Для интерактивного обработчика, который открывает подготовку самому пользователю со штатными defaults:

```bsl
Цель = Новый Структура("вид,сущность", "processor", "А_ОтчетKPIБригадиру");
ОтправитьУведомление(ИмяПользователя(), "ui.открытьФорму", Цель);
```

Это fixed metadata name в серверном коде, без произвольного URL/DOM/JS вставок. `ИмяПользователя()` — server login интерактивного actor (langref/functions.go:1859); не брать адресата/цель из недоверенного client input.

Source цепочка:

- `internal/ui/static/ui.js:6822–6825` formURL: kind processor → `/ui/processor/` + encodeURIComponent(entity), id для обработки не нужен. Вкладочная оболочка открывает отдельную вкладку; вне неё native fallback использует текущее окно.
- `internal/ui/handlers_processors.go:420–432` getProcessor проверяет регистрацию и processors.run на целевом GET; processorForm дополнительно проверяет external processor permission и вызывает pickProcessorManagedFormCtx для штатного role-selected варианта. Первоначальные значения берутся из proc.Params defaults.
- `internal/ui/handlers_events.go:37–57`: команды самому actor в form-event scoped к исходной оболочке, CommandID/dedup и response uiCommands fallback рядом с SSE; это уже имеющаяся инфраструктура, не новый raw navigation bypass.

**Границы:** в существующем JS formURL параметры добавляются только для kind report; processor queryparams/имя формы этим payload не передаются. Permission проверяется при целевом GET, не обещаю новый typed pre-publish guard у глобального ОткрытьФорму. Для заявленного T3 «отдельная зарегистрированная подготовка со своими defaults, перенос параметров родителя не нужен» этот существующий маршрут потенциально достаточен, но конкретный T3 actor/menu/role-form в runtime мной не воспроизведён — source-supported, НЕ PASS.

Если требуется именно именованная форма/типизированные processor params/предварительный серверный run-check с синхронным refusal, это остаётся новым native API, не объявляется реализованным notification-примером. Права или модуль KPI ради этого не расширял; после выпуска — отдельная согласованная регрессия/реализация недостающей части.

Generated-with: Codex
