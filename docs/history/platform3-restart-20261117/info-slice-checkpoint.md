# ENG16 info-slice control graph: WIP checkpoint90c23

Exact90c23e981b138c2c7c5d5848c3c3db398920a547, parentcf907755e5054c7645998cc23191052f66099c2d. Existing branch wip/platform3-report-lineage-derived-20261116 pushed, WT clean. Включён exact code/test/doc carry owner guard71b3; CHANGELOG объединён с прежним FK/calendar. Нового WT/архива/бинаря/Go не было. Это checkpoint, НЕ formal-ready/new ACK/SDK, peer на этот объём пока не назначать как GREEN.

Lineage поддерживает только periodic InfoReg LastSlice/FirstSlice с явным AS, omitted filter и omitted/parameter boundary. Unknown methods/filter/expression args/nonperiodic/zero or >64 dimensions STOP. Source fields сохраняют virtual-source occurrence, все partition dimensions включая скрытые и period добавлены в Uses.Scope; bound параметра отдельно. Все эти uses — control dependencies всех outputs. Никакой mask/read permission/admission этим не выдаётся, query.go/SQL generator не менялись.

До guard carry ранее проверен SQLite LastSlice RLS-before-latest positive: более поздняя исключённая запись не подменила выбранное старое значение, SUM15 вместо1004; PG SKIP. Тогда4 trace cases доходили до assertion и выявили missing mandatory VT guard (FAIL), 9negativePASS. Это intermediate evidence, не after90c23. После71b3 carry + strict empty-boundary/name/dimension-budget checks новая matrix ещё НЕ запускалась: disk barrier/single Go lane. В новой test форме также added empty boundary, empty dimension и budget65 negatives; fresh PASS отсутствует. Все runtime/native/actual-query/snapshot вопросы HOLD.

Portable e1feede5be3c761f7ae89cc6e30390d942cc8df0 с bounded static assembly ACK НЕ содержит ни этого VT graph, ни71b3/полной privacy, ни FK/calendar/scope. Его API/limits независимы и уже переданы platform2. Accepted cf остаётся accepted только доcf, не90c23.

Следующий exact tiny plan после выделенного perenos окна: на90c23 GOMAXPROCS2/CGO_ENABLED0/-p1, только TestReportOutputVirtual в internal/access и TestReportLineageInfoSlice в internal/query, принудительно PG env empty; никаких whole/новых WT/archive/SDK/native. Сначала freshfree>=2GiB, без старта до отдельного allocation разрешения; budget около2min, два packages последовательно. После PASS — bounded delta review, не повтор970/cf/e1. Это запрос будущего окна, не старт и не reservation.

Generated-with: Codex