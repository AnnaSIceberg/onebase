# After-release ACL/XLSX — независимое source RO review

Прочитаны узкие source ranges a00d323898c6ca40ab093eca8319dae812faf90e (Draft2631, parentd2329378,6files) и db1b058739a594bcf21e29c3327cdff3440c4dfc (Draft2632, parentmain4379007a,4files), весь новый test code и связанные error/writer реализации. Manifest698e39d0 SHA independentlyverified, все перечисленные source/runtime Git blobs совпали; три owner WT clean. Own runtime f33e2470/19abe9cc наe6 просмотрен отдельно, не принят за B21. Go/build/test/HTTP/Chrome/DB/SDK0, peer source не менял.

## ACL — CODE-ONLY OK

querySourceAccessDenied.Error сохраняет прежний текст, Unwrap несёт native interpreter.ErrRowAccessDenied. queryProxy теперь сохраняет исходный err в userError.Err на compile/SQL/guard failure. Существующие userError.asDSLError и DSLError.Unwrap сохраняют цепь. execTry/rethrowFn без аргументов повторно возбуждает исходный userError; явно user-raised строка создаёт новый userError без native cause. Поэтому сравнение errors.Is типизированное, а не совпадение пользовательского текста.

Оба result_proc caller paths используют один reportResultErrorStatus:403 только для native cause, иначе500. Сам отказ/права/состав данных не ослабляются. Runtime WIP наe6 меняет только эти два места внутри handlers_reports: QueryForVariant и B19 input-limit ветки сохранены. П7 там ещё нет: futureport на acceptedB21 обязан сохранить binder доqueryless result_proc.

TestReportResultQueryAccessStatusMatrix содержит10expectedcases/dialect:4denied direct/query+proc/queryless/rethrow,3realreadpositive включая native viewer,3genuineerrors(fake same-text/compile/SQLmissingtable) остаются500. Это дизайн не результаты. До Go/PG нельзя объявлять rethrow/status поведение runtime-доказанным; queryProxy изменение общее, нужны затронутые interpreter query/exception regressions вместе с HTTP матрицей. Информацию/реальный source dataset не читали.

## Ordinary XLSX — CODE-ONLY OK

Изменён internal/excel, sheet/xlsx.go не тронут. ExportList и WriteList задают dimension перед serialization; для StreamWriter правильно доNewStreamWriter. setListUsedRange включает ширину header/максимальную ragged row, пустые значения со стилями и положение последней непустой по len(row) строки; nil/empty row только с row-height used extent не расширяет. A1 для пустого случая, inclusive1-based coords в правильном col,row порядке, API errors возвращаются. Старые value/style/format/freeze/column-width/stream RowOpts пути не переписаны.

12unitcases ожидаемы (6shapes×2writers); ZIPdimension проверяется напрямую, ragged C2 content отдельно. Public ordinary-report direct/background matrix ожидает28XMLrows/A1:B28 и grand253; это собственные22synthetic source rows+groups/subtotals/grand/header, а не изменение оригинального repro. Optional export dir test-only, no actual exports generated reviewerом. Default installed openpyxl read_only безreset/forcedbounds против normal на НОВЫХ exports остаётся обязательным будущим gate; excelize.GetRows и pandas не заменяют его. Excel/LibreOffice совместимость не заявлена.

## Решение / очередь

Новых source blockers в просмотренных дельтах не найдено: CODE-ONLY REVIEW OK обоим. **Tests NOT RUN**, ни local PASS, ни PG PASS/SDK readiness не объявляются. Исполнение source Go только по отдельному root cc2 окну. После тестов повторный review фактического testedHEAD/logs/PG и новый candidate на актуальной post-release базе с сохранением B21 П4/П7/UUID/flat12 и B20/B19 guards. Старый e6 WIP не становится release binary автоматически.

Ownership source остаётся platform2. Только ПОСЛЕ вечернего выпуска; в immutable B21/manifest6f4f не добавлять. Root текущий Go/Chrome HOLD соблюдён, shared :8444/общие службы неизменны. Никаких новых GitHub labels/Ready/CI claims этим review не сделано.

Generated-with: Codex
