# П4 / П7 audit1c: ownership platform2

Platform не начинал отдельную реализацию П4 или П7. Source ownership этих двух пунктов передан platform2 по её поручению. П1 payload/snapshot/underline/hair повторно не реализовывать. Sheet XLSX writer и dimension-fix остаются у platform.

Уточнённый scope ACK: П4 — именно штатные report/composition typed cell/row actions, metadata whitelist разных entity + image/attachment через существующие protected routes; НЕ tabular snapshot payload П1. П7 — `params[].context` user:id/login или user_card scalar через существующий trusted reader, серверный override во всех UI/REST/export paths, клиентские trusted-key значения не принимаются. Эти реализации и schema с моей стороны не заняты и не намечены. Можно создавать отдельную source-ветку; runtime/Go-очередь отдельно.

## Точные базы

- Source П1 с payload/underline/hair: `993dbc2678ac6c96734f8975fdfa7b1b8dc539f1`; B18 size/AutoScale source `621c6628523c45b3912ec1ad9475ffdf084475c4`.
- B19 source `815e1505f504659fedea159fd186eeff3ab36d33`, runtime `cd9910c81fd7812874b9ca2cafa1870857abee5e`: result_proc input-limit fix, PG118 + Windows native43 PASS; B19 intermediate, НЕ deployed.
- Current source B20 `d232937812637564c730fbc17a9376d269dc5247`, branch `fix/xlsx-used-range-20261115`, pushed fork. WT `C:/Users/a.sevrjukova/AppData/Local/Temp/platform-help-admin-20261111/report-tabular-result-source`. Можно брать его как source-parent и передавать только свой range после него. В отличие от runtime эта source-линия не содержит интеграцию всех прежних П2/П3/П5/П6.
- B20 runtime `e6cb9a375b0bae15a430ae93b0913c02938bb45e`, branch `fix/xlsx-used-range-runtime-20261115`, exact parent B19cd99. WT рядом `report-tabular-result-runtime`; он включает ранее интегрированные П2/П3/П5/П6. Это текущая build-линия, НЕ готовый SDK до handoff/гейтов.
- На :8444 пока B18 Linux24897fe0/runtime64886, config2fba. Ни SDK, ни shared units platform2 не подменять. Новый exact SDK сообщу после serial сборки.

## Уже существующий П1 контракт — anchors exact d232 / e6

- `internal/metadata/form_module.go`: kind `ПолеТабличногоДокумента`, event `ОбработкаРасшифровки`. Реквизит формы `SpreadsheetDocument`; element `data_path: Объект.Result`. DSL setter: `Д.Ячейка(1,1).Расшифровка = Ref/массив/структура`; handler получает `Расшифровка` / `Drilldown`.
- `internal/sheet/drilldown.go`: inert `DrilldownValue` Kind/Text/Bool/Ref/RefKind/Items/Fields. `Reference` в `sheet/view.go` — Entity+ID; никакие менеджеры/контексты/URL не сохраняются. `Cell.HasDrilldown` учитывает одиночный Ref и structured payload.
- `internal/dsl/interpreter/spreadsheet_document.go`: setters; `FreezeTabularDrilldowns` / `RestoreTabularDrilldown` в interpreter преобразуют только поддержанную сериализуемую модель, с лимитами/отказом для unsupported/cyclic. Не расширять этот контракт произвольным writer/URL.
- `internal/ui/tabular_result.go:42`: owner от session + актуальный auth.User (roles/masks/RLS/admin/actor). `publishTabular` создаёт detached snapshot, TTL15min/cache16/generation binding; readableTabular повторяет owner/expiry/generation/права scope/parent record.
- Там же tabularView `/ui/tabular/{token}/cell`: инертная одиночная ссылка → текущие entity read + rowAllowsID → штатный redirect к документу/справочнику. Прямой typed route пока именно Entity; registers/images не выдаю за реализованные.
- `internal/ui/tabular_drilldown.go:15` addTabularDrilldownVars: browser передаёт только `_tabular_token`, `_tabular_row`, `_tabular_col`; scope/form/field/element должны совпасть. Из snapshot восстанавливается свежая копия payload; вложенные Ref разрешаются через текущие DSL roots/managers + live read/RLS/masking, без сохранённого generation-time manager. Клиентские payload values не принимаются. Question → объявленный `Ответ` проходит те же binding/rights проверки.
- Экспорт/print/viewer не вызывает callback и не авторизует запись. Новые typed targets П4 должны сохранить этот принцип; не считать один payload/Ref правом на прикладную запись.
- Точные regression roots: `TestTabularPayloadHTTPMatrix` (10 cases/dialect), `TestTabularDrilldownHTTPMatrix`, `TestTabularReportResultHTTPMatrix`, `TestTabularManagedFieldHTTPMatrix`; Node `internal/ui/static/tabular_payload_behavior_test.js`. Native proof: manifest1115b17 final91f8843b и B19 final9eb4279b, различать Windows и Linux.

## П7: граница существующего контекста

Отдельный trusted user/department API для report source platform не добавлял. Не объявляю наличие готового П7 контракта. Runtime `internal/ui/report_variant_query.go:9` получает `ctx` и params, `QueryForVariant`, затем `compileQueryWithRowAccess(ctx,text,values)`. В `handlers_reports.go` используется native operation/request context и `currentUserLogin` для пользовательских настроек; params остаются пользовательским вводом. Требуемый новый user/department binding следует строить из server auth/доверенного источника с запретом client spoof, не переиспользуя обычный report param как доказательство личности.

## Разведение файлов / ресурсы

П4/П7-specific source у platform2; `internal/sheet/xlsx.go`, `xlsx_dimension_test.go`, `internal/ui/tabular_xlsx_dimension_test.go` у platform. В B20 platform не меняет managed.js/ui.js/metadata/payload. Если platform2 нужен общий tabular branch JS/metadata — source можно готовить, в handoff перечислить точные файлы/range, сохранить старые ветки других elements. `.go` конфликты интегрировать с build ./... и affected tests.

Сейчас exclusive Go у platform для B20 до явного END; platform2 Go HOLD, Chrome отдельно по cc2. По готовности source — push WIP/commits каждые1–2часа и перед паузой. К17:45 контроль commit+push/собственные процессы0; после проверки «готов к перезагрузке». Никакой production package.
