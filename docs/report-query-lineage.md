# Compiler lineage отчётного запроса (ENG16, derived WIP)

`query.CompileReportLineage(text, opts) (query.Result, *query.ReportLineage, error)`
компилирует запрос и строит неизменяемое описание поддержанного SQL-профиля.
Это основа проверки происхождения, **не разрешение чтения ПДн**. Никакой
runtime/handler/gate не переключён на этот API. Существующая обязательная
ReportOutputQueryMaskPlan продолжает запрещать защищённые сравнения.
ENG06 остаётся INTEGRATION HOLD до совместимого полного ENG16.

В этой WIP-ветке добавлены scalar derived SELECT с отдельными scope и сквозными physical occurrence. Основная source-ветка содержит strict foundation без scope/UNION. Поддержаны именованные проекции,
WHERE/GROUP/HAVING/ORDER, CASE, простые операторы, MIN/MAX/COUNT/SUM/AVG,
COALESCE/ЕСТЬNULL, SUBSTR, LENGTH, ограниченный CAST. ТЧ сохраняет владельца.
Неизвестная функция/поле/источник, неоднозначное имя, wildcard, ordinal,
correlated derived, derived reference navigation и виртуальная таблица возвращают ошибку без частичного
Result/receipt. Это намеренная граница WIP; полный A073/KPI
с VT/специальными выражениями ещё не поддержан. Поддержка парсинга SUBSTR не означает разрешения
произвольного перебора телефонных цифр.

`Outputs()` даёт detached деревья выражений выходных колонок, `Uses()` —
деревья каждого JOIN/WHERE/GROUP/HAVING/ORDER. В листьях есть canonical field,
metadata type, SourceRef(kind/name/tablepart) и отдельный Occurrence каждой
FROM/JOIN позиции. Occurrence относится к профилю, не к индексу автоматически
созданного SQL JOIN. Навигация сохраняет все hops и защищённый корень ссылки;
голое ссылочное поле, которое компилятор превращает в представление, не выдаётся
за UUID. Имена параметров выделены отдельно. Сырые значения литералов в
inspection не копируются: остаются тип и SHA256(type:value).

Выходные деревья показывают data dependencies. **Все Uses одновременно являются
control/membership dependencies всех выходов**: например, COUNT(*) зависит от
условий и JOIN, даже когда его дерево не содержит поля телефона. Проверка только
Outputs() не доказывает безопасность отчёта. Будущий owner authorization обязан
проверять весь план и источники, все чтения/RLS, вывод/порядок/лимит, отсутствие
пользовательского оракула и разрешённые фиксированные серверные операции.

`receipt.Matches(result, opts)` проверяет SHA256 скомпилированного Result и
окружения metadata/params/row filters. Типы кодируются явно, без вызова
произвольного MarshalJSON; decimal/string, bytes/string и NULL/typed nil не
схлопываются. Неизвестный SQL argument type/capability даёт STOP. Смена SQL/args/projection/metadata/RLS
инвалидирует receipt. Actor/read/роли, registry generation, snapshot, модуль,
выбранный отчёт/пресет этот helper сам не авторизует: host должен связать их
с текущим проверенным выполнением. Сериализация private receipt не переносит
полномочий, inspection-копия не изменяет receipt. Клиентский descriptor/preset
никогда не принимается как доказательство.

Targeted SQL matrix: синтетический matching self-JOIN, обе стороны RLS до
сравнения, бизнес-группа и COUNT; корень ссылки; отдельный parameter origin;
фиксированная классификация длины; ТЧ; unknown/ambiguous STOP; stale SQL/args/
metadata receipt; существующий mandatory gate по-прежнему Denied. SQLite PASS,
PostgreSQL требует общего разрешённого пакета с TEST_DATABASE_URL.

Остаток ENG16: разрешённые планы внутренних сравнений/классификаций/агрегатов
с proof всех чтений и RLS, derived/UNION scope graph, безопасный деловой номер
в отличие от сырого Представления, полный набор потребителей, отрицательные
oracle/presets/export tests, independent review и native/PG/SDK пачка. Этот
source-срез не закрывает принятый отчёт совпадений как готовый и не разрешает
его отказом заменить совместимое исполнение. Карточки/A108/field_access не меняет.

У GROUP BY разрешаются исходные поля и явные выражения: совпавший output alias
не затеняет исходное защищённое поле. GROUP BY по одному лишь output alias без
доказанного исходного поля сейчас STOP; ORDER BY использует output alias в
соответствии со своим SQL приоритетом. Обычный Query/compiler не меняется.


WIP Uses.Scope различает root и каждый derived SELECT. Все child projections
включаются как inner-projection, даже когда внешняя проекция их не выводит;
условия/группировки дочерней выборки остаются отдельными Uses того же scope.
Derived-field содержит полное дерево child expression, а не один alias/type.
Correlated scope lookup не разрешён. Раскрытое inspection имеет общий бюджет
узлов/metadata bytes/depth: компактная цепочка повторений T.X+T.X не может
создать неограниченное дерево при detached JSON inspection.

Scope/UNION delta получила отдельное bounded CODE ONLY review; это не приёмка
full privacy/runtime. SQLite nested aggregate/child RLS/root-occurrence/
GROUP shadow/unknown/expansion tests PASS.
Ни runtime gate, ни safe protected operation authorization не подключены.


UNION WIP связывает колонки по позиции, имена берёт из первой ветви и
объединяет деревья всех ветвей. Типы должны согласоваться; неизвестные/разные
типы, сложная ref-навигация и невыведенные global ORDER имена дают STOP.
UNION ALL и duplicate elimination различаются в operation/control graph.
Общий ORDER BY использует объединённый выход, а не последнюю ветвь.
SELECT DISTINCT также явно записывает membership dependencies всех outputs.
SQLite UNION ALL/DISTINCT nested aggregate+RLS и negative/global ORDER tests
PASS; PG SKIP. Никакой autorization protected operations/runtime ещё нет.

Ссылочные поля в JOIN/WHERE/HAVING доказываются как stored FK UUID, согласно
секциям обычного компилятора. Контекст предиката сохраняется внутри выражений;
origin содержит owning SourceRef, occurrence, исходное поле и reference type.
Защищённый FK не становится доступным из-за такого типа. Bare reference в
SELECT (включая CASE/COUNT), GROUP/ORDER и derived passthrough остаётся STOP:
представление не выдаётся за UUID. Ни права карточек, ни runtime gate не меняются.

Typed fingerprints используют общий provenance FormatVersion 2/domain v2.
Receipts старого формата нужно инвалидировать и пересчитать по host snapshot;
совпадение digest не доказывает доверенное происхождение пользовательского ввода.

Год/YEAR, Месяц/MONTH, День/DAY сохраняют дерево аргумента и дают числовой тип
только при одном доказанном поле/выражении metadata type date. String/number,
непривязанный параметр, неизвестная функция/arity/DISTINCT дают STOP без partial
receipt. Группировка по функции сохраняет те же control origins. SQL rewrite
остаётся в обычном compiler без изменений; это не разрешение раскрытия календарной
части защищённой даты. РАЗНОСТЬДАТ и виртуальные источники пока не поддержаны
этим профилем.

WIP periodic information slices: LastSlice/FirstSlice (русские эквиваленты),
явный AS qualifier, отсутствующий filter, boundary только omitted или параметр.
Nonperiodic/no dimensions/>64 dimensions/unknown method/filter/expression args
дают STOP. Все partition dimensions (включая скрытые и неупомянутые) и period
попадают в Uses.Scope как slice-partition/slice-order-desc|asc; boundary сохраняется
в slice-bound. Это control dependencies всех outputs, не разрешение агрегата.
RLS компилятора применяется до выбора строки LatestPerKey; typed/actor/read/RLS
admission по-прежнему обязан доказать host. Report-only mandatory guard отдельно
отклоняет protected VT до появления разрешённого operation graph. Physical tables,
ordinary Query/cards и metadata/compiled SQL compiler не меняются.

Current checkpoint не является новым PASS: prior SQLite LastSlice/RLS positive
был проверен, четыре implicit-key policy cases воспроизвели P1 до guard; owner
report-only guard6leafPASS отдельно. После carry и новых strict-boundary/budget
checks новая матрица ещё НЕ запускалась из-за disk barrier/single Go lane ENG31.
Native KPI/полный consumer query/admission/SDK remain HOLD. This WIP is not the
portable strict970 API, published independently without scopes/VT/privacy policy.
