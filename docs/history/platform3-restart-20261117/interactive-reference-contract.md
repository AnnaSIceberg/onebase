# ENG05/ENG06: следующий interactive reference boundary — owner contract, не runtime

17.11.2026. Продолжение после сдачи exact5bb3d995d98c6f23a6b38e3f5f76f62a49514eed на bounded source review. Это проект следующего собственного среза, НЕ ACK реализации/новый YAML/API runtime и НЕ предложение включить full privacy в текущий выпуск.

## Сверенный integration base

В engine396ae11defdfeffdeabd60cd8c36b95780006108 уже есть report_parameter_choice.go и reportInitialChoiceOptions. В отличие от исходника c700, runtime buildReportParams принимает report metadata, исключает Context params, сохраняет VisibleWhen, enum options и ChoiceShowDeleted. ChoiceContext/resolveReportChoiceRequest также учитывают Context sources. Новую mandatory policy подключать точечно к этим фактическим helpers, а не заменять buildReportParams целиком из source71/5bb или c700. Наличие файла в engine не доказывает native acceptance всех его потребителей. B38.1 date assets/tests/CHANGELOG принадлежит platform, не трогать.

## Подтверждённая открытая граница

ChoiceContext сейчас пуст для reference-param без choice_filter/suffix; resolveReportChoiceRequest такой параметр отвергает. Generic GET /ui/_ref-options/{entity} и POST preview/page остаются обычными endpoints карточек. Контекст отчётного результата не наследуется отдельным HTTP запросом автоматически. При существующем report-choice всё равно недостаточно безопасно вычислять label ПОСЛЕ поиска по сырому Label/Phone: total, membership, ordering и selected_allowed сами являются выходами.

5bb закрывает только начальные bounded/selected options в своём fullprivacy source. Ни generic picker/search/preview, ни branch suffix/declared field filter, ни actual396 этим SHA не закрыты. Full ENG06/16/12 HOLD сохраняется.

## Контракт следующего server boundary

1. Отчётный UI всегда передаёт серверно восстановимую identity report+parameter, даже без choice_filter/suffix. Предпочтителен отдельный report-scoped маршрут: имя отчёта и имя параметра принадлежат route; server registry проверяет report.run, reference type и target object.read до чтения. Browser header, target UUID, произвольный form_kind, client schema или готовая строка label не доказывают identity. Не добавлять глобальный report-mode обычному endpoint карточек. Удаление client context не должно переключать тот же report route на ordinary policy.
2. User/default/Context values разрешаются по фактическому runtime-контракту. Context source определяется host, не browser. Пустой branch снимает только своё Eq-условие, не ACL/standalone filters. Existing archive/folder/activity/deleted и order_by сохраняются, пока конкретное поле допустимо report-output policy. Не расширять старый заказ полным фильтром карточки звонка.
3. До List/Count/selected_allowed восстановить все client-влияемые Search/Filters/ChoicePredicates/Sort и доказать исходные поля. Для report mandatory: ReportOutputFilterAllowed + ordinary FieldDecisions + object read/RLS. UUID не отменяет запрет protected FK. Непроверенные tablepart/navigation/unknown paths STOP. Никакого blanket field=field. Служебный RLS применяется сервером, не объявляется пользовательским фильтром и не снимается.
4. Search должен затрагивать только доказанные доступные поля; default all-string search не подходит. При отсутствии безопасного набора — единообразный STOP до запроса, не скрытый raw search с последующей маской и не Count по исходным данным. Метаданные общего registry/карточек не мутировать. Ограничения search и Count одинаковы; offset/limit применяются после допустимого отбора.
5. Сортировка до limit. Разрешённый бизнес order_by публичной цели сохранить; защищённую произвольную stored label не использовать для порядка. Техническое представление UUID допускает order=id. Для mixed order_by запретный/unknown ключ STOP либо отдельный явный безопасный order contract; нельзя молча сохранить скрытую вторичную сортировку.
6. Поля маскируются до `_label`, branch suffix, preview, row style и JSON. Existing reportOutputRecordLabel — masking helper, не exemption. Suffix требует доказанного безопасного FK и безопасной целевой подписи с object/RLS; protected/unknown suffix STOP до чтения, без утечки через наличие/отсутствие суффикса. Ответ report route ограничен необходимыми id/label и безопасными transport fields. UUID/переход в обычную карточку сохраняются, права карточки прежние.
7. Preview proc / computed text не обходят output policy. Неизвестное происхождение DSL/готовой строки STOP; полноценный ENG12 producer trace/approved computation нужен отдельно. Безопасная initial option не разрешает произвольный preview. selected_allowed использует ровно те же допустимые predicates/activity/deleted/RLS, что и список, и не раскрывает скрытую принадлежность.
8. API/export/background/presets повторяют server validation на каждом запросе; сохранённая client field identity/label не permission и не lineage. Mandatory policy не зависит от admin/disclose. Обычные карточки, A108 и context-free endpoints продолжают обычные права.

## Минимальный публичный regression набор следующего среза

- admin + ordinary OTK: меняем только protected Phone/Address/storedLabel при неизменном UUID/public fields; list labels/order/count/search/selected/preview не приобретают зависимость от них;
- reference-param БЕЗ choice_filter и с ним; forged report/parameter/target/Context sources/duplicate identity отвергаются до чтения;
- safe branch Eq, empty-parent own-filter clear, suffix, catalog order, enum/context/choice_show_deleted controls сохранены;
- protected filter/sort/search/suffix и unknown tablepart/nav дают STOP; безопасный City/Street/business field остаётся разрешён при доказанных metadata/policy;
- selected за первой страницей, недоступный selected и RLS excluded rows не обходят read;
- ordinary privileged card/picker unchanged; report→card UUID transition остаётся, report output itself безопасен.

Новых runtime endpoints/metadata/JS этот файл не добавляет. Новых Go/suites/HTTP/browser/native запусков0. Конкретная реализация и targeted fixed commands требуют отдельного exact и очередного perenos batch assignment. Старый tiny90 план неизменён и по-прежнему NOTRUN.

Generated-with: Codex