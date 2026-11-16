# Compiler lineage отчётного запроса (ENG16, derived WIP)

`query.CompileReportLineage(text, opts) (query.Result, *query.ReportLineage, error)`
компилирует запрос и строит неизменяемое описание поддержанного SQL-профиля.
Это основа проверки происхождения, **не разрешение чтения ПДн**. Никакой
runtime/handler/gate не переключён на этот API. Существующая обязательная
ReportOutputQueryMaskPlan продолжает запрещать защищённые сравнения.
ENG06 остаётся INTEGRATION HOLD до совместимого полного ENG16.

В этой WIP-ветке добавлены scalar derived SELECT с отдельными scope и сквозными physical occurrence. Стабильный Draft017 для review содержит только исходный single SELECT профиль. Поддержаны именованные проекции,
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

Это непринимаемый пока WIP: SQLite nested aggregate/child RLS/root-occurrence/
GROUP shadow/unknown/expansion tests PASS; stable017 independent repeat pending.
Ни runtime gate, ни safe protected operation authorization не подключены.


UNION WIP связывает колонки по позиции, имена берёт из первой ветви и
объединяет деревья всех ветвей. Типы должны согласоваться; неизвестные/разные
типы, сложная ref-навигация и невыведенные global ORDER имена дают STOP.
UNION ALL и duplicate elimination различаются в operation/control graph.
Общий ORDER BY использует объединённый выход, а не последнюю ветвь.
SELECT DISTINCT также явно записывает membership dependencies всех outputs.
SQLite UNION ALL/DISTINCT nested aggregate+RLS и negative/global ORDER tests
PASS; PG SKIP. Никакой autorization protected operations/runtime ещё нет.
