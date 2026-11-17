# platform3 — возобновление ENG16 и узкий source-fix начальных ссылочных параметров

17.11.2026, после прямого resume analysis. DEV/ЛПЦ уже на REL38; B38.1/date assets/tests принадлежат platform, не менялись. Выкладок/SDK/браузера/actual DB/Go build/test/compile/link не было. Только source read/edit, gofmt и Git. Проектные правила прочитаны; маршрут — продолжение существующего Draft2641, не Ready.

## На bounded source review

Ветка fix/platform3-report-output-privacy-20261116, exact **5bb3d995d98c6f23a6b38e3f5f76f62a49514eed**, parent **71b3f92125125a59901e697401b7f410797e041b**. Fork pushed, свой WT CLEAN. Дельта 5 files, +197/-1; прежний guard71 не переаудировать/не расширять.

Сверка кода подтвердила начальную границу: reportForm GET до runReport вызывает buildReportParams без report context. initialReferenceOptions/appendSelectedRefOptions вычисляют обычную подпись; stored Label может содержать ПДн даже при безопасном query. Не заявляется новый runtime-repro: baseline/after пока NOT RUN.

Новый private reportParamReferenceOptions вызывает существующие list/selected helpers в server-selected WithReportOutput context. Маска до вычисления результата; затем reportOutputRecordLabel даёт техническую подпись UUID защищённой цели. Для неё Sort=id применяется ДО limit, чтобы hidden label/order_by не влияли на членство/порядок первой страницы. Незащищённые цели сохраняют catalog order_by. Object/row ACL и selected-value RLS остаются существующими. Подключение — одна строка в buildReportParams; общие handlers.go/field_access, query/access, templates/static/date и metadata не изменялись.

Новая mounted HTTP regression TestReportParamReferencePrivacyHTTP содержит synthetic matrix fixture: admin + ordinary OTK с read/disclose; 52 объекта, первые50 + selected52; изменение только Label/Phone/Address разворачивает label-order, но не должно менять вывод UUID и страницу. Контроли: публичный target label/order, обычный privileged HTTP picker не изменён, report-only без target read не получает опции, RLS оставляет только одну разрешённую запись и не возвращает запрещённый selected. Код отформатирован, staged diff --check PASS. **Тест НЕ ЗАПУСКАЛСЯ**, ни одного нового PASS/compile evidence на exact5bb нет.

Просьба perenos назначить bounded code review только 71..5bb и поставить targeted HTTP+chokepoint в следующую общую synthetic пачку. Возможный fixed command после отдельного assignment: go test -p=1 ./internal/ui -run '^(TestReportParamReferencePrivacyHTTP|TestFieldMaskHelperChokepoint_NoUngatedStoreReads)$' -count=1 -timeout=45s -json, GOMAXPROCS2/CGO0/noPG env. Это предложение плана, НЕ разрешение/worker/окно.

## Текущий ENG16

Checkpoint **90c23e981b138c2c7c5d5848c3c3db398920a547** (wip/platform3-report-lineage-derived-20261116) остаётся CLEAN/frozen. Tiny plan596f704844003132ade10f2ad968b941eb384295166a009bf3a1d9c6af798e20 / workerb31a2ceb1687b620217c288766b175ca7abcfacad264b3e9f01df394a7516865 / launcher7af0db483ac232ba0499c5001ddefc21cd64dd41db9fcbb16fb565dac9f1764e не менялись и не запускались. Запрошено НОВОЕ пакетное heldFD assignment; старые часы/планы не разрешают запуск. After-carry tests всё ещё NOT RUN. Source query/report_lineage/access-report_output свободны от работ platform по B38.1 (подтверждено им сообщением).

ENG12 platform2 exact1960033357436343f6f98735d7bd54119dfc3a76 WIP не переносился и не ревьюился заново. Их hostframe/adapter не дублирую, SQL proof остаётся нашим. E1/d44 static assembly ACK не компиляция/runtime.

## Открытый предел

**Full ENG06/ENG16/ENG12 HOLD остаётся.** Новый5bb не whole-PDN acceptance и не autoREL38/B38.1. Интерактивные report reference search/picker/preview + скрытые filters/sort/choice suffix ещё требуют report-bound identity/mandatory gate без изменения обычных карточек. Общие context-free picker endpoint не закрыт этим узким патчем. Native DateDiff/bare refs/masked-original passthrough/producer seal/final schema и совместимые защищённые сравнения/агрегаты остаются открытыми. Не выключать принятые отчёты как способ объявить перенос готовым. Неизвестное STOP, без blanket field=field.

Следующий независимый шаг после сдачи этой дельты: проектировать report-bound identity для интерактивного выбора совместно с ENG05 c700 boundary, без изменения готового B38.1. Новых необязательных пунктов112 не брать.

Generated-with: Codex