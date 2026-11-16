# Owner contract v1: ENG04/ENG12 inputs, producer schema, native passthrough

Для platform2, cc2 и analysis. Ответ на 20261116-platform2-ENG04-ENG12-typed-extras-фильтры-следующий-контракт.md и 20261116-platform2-ENG12-A004-A005-actual-input-остаток.md. Это нормативная граница следующей реализации, НЕ готовый SDK/API/SHA нового слоя. Текущие ReportOutputFilterAllowed/ReportOutputQueryMaskPlan сохраняются; generic allow/exemption/переменная effectiveFilters не вводятся этим файлом. ENG06 integration HOLD до совместимого ENG16 сохраняется.

## 1. Final filter schema — только успешное host evidence

Exported rowsource.Result, Columns/Origins, JSON и объявление RowSourceColumn{Name,Type} не являются authority. Final schema создаёт один host producer только ПОСЛЕ успешного compile/read/guard/typed codec/validated program/immutable originals/output checks. При ошибке/cancel/unknown — нет partial result и нет final filter schema.

Нужен private sealed receipt/opaque producer handle. Его внешним литералом/JSON нельзя сконструировать. Связь включает canonical report+variant, selected query/template hash, module hash+pure-profile version, metadata generation/fingerprint, actor/run/read/ordinary-field/RLS/policy identity, exact columns/types/ref roots/data+control origins и параметры/input bindings. Изменение публичных Rows/Columns/schema или потеря private evidence делает handle непригодным. Inspector/JSON — копии данных, не capability.

Фильтр выбирает server column ID из текущей producer schema; клиентские Source/Type/UUID/Origins игнорируются. Mandatory + object read + RLS + ordinary field policy проверяются вновь в единственном owner chokepoint, включая preset/API/export/background. Hidden/unknown field, неизвестный ref root, stale snapshot — STOP. UUID flag не снимает запрет защищённого root field. Extra type/ref не угадываются по строке. Декларация type:string/literal/validated-program сама не разрешает фильтрацию.

Все control/membership dependencies также обязательны: COUNT(*)/extra зависят от исходного отбора/JOIN/ветвления, даже если в их projection нет Phone. Согласованное ENG16 declassification evidence потребуется отдельно; его нельзя заменить флагом private/internal или одним fixed literal origin.

## 2. Точные origin kinds и обязательный binding

Ниже закрытые host kinds. Любой неизвестный kind -> STOP, клиент/DSL kind не задаёт:

- `source-field`: canonical SourceRef(kind,name,tablepart), occurrence/scope, stable field identity+canonical name, metadata type, ref target/root/path, current read/RLS/policy proof.
- `fixed-literal`: лексический literal доверенного неизменного SQL/template или проверенного module AST; binding template/module hash + node/ordinal identity + type + typed value digest. Нужен host proof статичности текста/AST, не только совпадение значения.
- `user-input`: любое принятое request/preset значение, включая params, result filters, sort/search/selected variant; значение явно пусто/false/0 также user input.
- `server-default`: реально вычисленное host из canonical metadata при отсутствии принятого request value; binding default-expression hash, slot, dependencies, clock/timezone/context snapshot. User-controllable slot и user/control dependencies сохраняются; такой default НЕ становится fixed literal для protected comparison.
- `server-context`: actor/locale/timezone/branch/context value из конкретного server context; binding context slot + actor/context fingerprint. Это не fixed literal; raw protected context value в DSL не открывается.
- `validated-program`: exact module/profile/operation proof плюс КОНСЕРВАТИВНОЕ объединение всех input/data/control origins. Это преобразование, а не очистка/повышение доверия.
- `masked`: host policy заменило/скрыло исходное поле. Это не SQL NULL, не пустая дата/UUID и не fixed literal; raw value в DSL не попадает.

Binding `user-input` содержит: canonical report и variant; trusted query/template identity; metadata/policy generation; server slot kind (`report-param`, `result-filter`, `sort`, `search`, `variant`); canonical param ID либо server filter-column ID+validated operator+operand index; declared parsed type и доказанный ref root (если есть); presence (`value`, `explicit-empty`, `missing`) и источник request/preset в private execution frame; private typed value digest. Identity/slot/field/operator разрешает HOST, не returned descriptor. Raw value не идёт в публичный trace/логи.

Для report-param имя связывается с текущим metadata Param и нормализуется существующим parser. Для result-filter — с текущим server schema field/operator до DSL. `__has`/клиентский `origin=default` не является доказательством доверия: если значение принято из запроса/сохранённого пресета, оно user-input, даже если равно default/пустому шаблону. Missing и explicit-empty сохраняют разные semantics, false/0 не считаются missing.

Matches SQL/args/metadata/RLS подтверждает целостность, но НЕ происхождение значения. Одинаковые typed bytes могут иметь разные kinds/bindings. SHA/имя alias/тип не превращают user input в literal. SQL literal, полученный конкатенацией user value в Query.Text, тоже user-tainted: module hash сам по себе не доказывает статичность этого query text. Dynamic/unknown text не допускается в protected-operation recipe. Выбор из заранее доверенных variants связывается с отдельным approved template, а не с клиентским текстом SQL.

## 3. Readonly effective filters в DSL

Пока host frame/seal/taint нет — такую переменную не вводить. Последующая переменная, если нужна, содержит только уже validated typed user values и read-only data view, без host DB/context/callbacks/metadata authority. Неизменяемость включает вложенные коллекции и методы, String() не форматирует host internals. Source/field/operator IDs и evidence остаются private host side; возвращённый DSL descriptor не может их изменить.

Каждое прочтённое effective value имеет `user-input` binding. Pure transformations/branching/Query parameter assignment распространяют data+control origins; минимум безопасного первого профиля — whole-execution union, как для текущих extras. Сложение/trim/alias/возврат той же строки не очищают origin. Даже literal, выбранный user-controlled веткой, несёт user control origin. Если runtime не может это доказать — не давать этому значению допуск protected operation/вход final schema по типу fixed literal.

User-input сам по себе не означает запрет обычного безопасного параметра/отбора. Он НИКОГДА не служит proof fixed empty-template/normalizer/сравнения защищённых ключей. ReportOutput mandatory/ordinary/read/RLS gates сохраняются. Hidden Phone/Address нельзя сделать полем поиска/distinct/filter/preset через extra/alias; protected input всё ещё STOP до отдельного ENG16.

Effective filters не могут задавать SQL identifiers/source namespace/entry/code/schema/allow callbacks. Обычные значения передаются параметрами. Для дополнительных Query сохраняется точный canonical input binding, а не bare value с подписью literal. Ни module whitelist, ни CodeLookup не дают автоматического allow пользователю.

## 4. Native originals и passthrough (A004/A005 уже начаты)

Current701 all-inputs direct Schema остаётся строгой. Native literal1 Заявок не равен synthetic declared Count; DateDiff/РАЗНОСТЬДАТ не доказан наличием alias/type:number; bare ref presentation не равен UUID; Адрес pii:true не разрешён даже admin. Общую Schema/ReportAllowed ради всего native result НЕ ослаблять.

Допустим следующий отдельный host pipeline: compiler/host доказывает КАЖДУЮ исходную колонку и её policy до выполнения; host сохраняет immutable ORIGINAL CHANNEL, уже пригодный для безопасного вывода, а DSL получает лишь минимальный явно разрешённый typed INPUT SUBSET. Число/порядок/дубликаты строк сохраняются по private ordinal/row identity, не dedup по одному UUID. После DSL host пересобирает результат из originals и проверенных extras; DSL не возвращает и не заменяет originals целиком.

Protected original не проходит в DSL ни как raw, ни под видом opaque raw object. Для output host применяет mandatory mask/hide и сохраняет tracked `masked`; предпочтительно не выбирать raw hide-field в Go вовсе, если SQL plan доказанно заменяет его NULL. Скрытый nil не нормализовать в правдоподобную пустую дату/ссылку/0 и не открывать через lookup/label. Неизвестная output lineage не становится безопасной только потому, что passthrough immutable: UNKNOWN STOP, а не blanket all columns outside DSL.

Фиксированный SQL literal1 может иметь отдельный host `fixed-literal` origin с доверенным template/node и number type; пользовательское значение 1 — другой origin. DateDiff допускается только после доказанного compiler/host expression type/операций, lineage всех аргументов и control/RLS; текущий unknown остаётся STOP. На уровне фильтра применяется тот же owner policy, не исключение field=field/number. Bare ref presentation требует отдельного owner label proof; до него не показывать исходный stored label. Доказанный UUID/root сохраняется, безопасная подпись применяется до group/render/export; карточки/field_access не меняются.

Нужен отдельный API/receipt для host originals + minimal DSL inputs, а не callback allow-all в старом Run. Literal/date/ref support проверяются точными native A004/A005 cases; synthetic fixture не закрывает эти различия. Этот контракт не разрешает передать весь native set в interpreter и не объявляет A004/A005 готовыми.

## 5. Единственный порядок и gate интеграции

Canonical report/variant + текущие actor/run/read/RLS/field/metadata/policy -> host input bindings -> compile/provenance -> безопасное read/mask/input subset -> validated readonly source -> immutable originals+typed extras -> successful sealed final schema -> validated result filters -> user limit/compose/group/render/export/API/background. Safety budgets не являются business limit: превышение даёт отказ, не обрезанную выдачу. Если известный отбор можно безопасно push до producer — только тем же validated binding и повторным доказательством; позднее ApplyFilters после лимита не возвращать.

Для snapshot после сериализации нельзя доверять Columns/Origins/HasTrace из JSON. Нужен server-side handle с current authorization/revalidation либо повторное выполнение; persisted proof требует отдельного server-only signing/validation contract. Нового snapshot разрешения здесь нет. UI/API/export/background/presets обязаны использовать один producer/schema/policy boundary. При несовпадении actor/query/module/metadata/RLS/input frame — STOP без частичного результата.

До этих доказательств действуют current source STOP и ENG06 HOLD. Технический ответ не равен source SHA нового frame API, не вводит новую DSL variable/allow и не требует менять карточки/рабочие формы. Продолжаем все замечания16.11 и весь KPI; новые прочие пункты112 не берём.

Generated-with: Codex
