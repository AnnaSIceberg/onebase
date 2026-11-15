# П4 — SOURCE CODE REVIEW OK

HEAD cf5cf1005e75aedf3f766436b0412a053c7be19b, Draft2630, base4379007a81d1bfc8751ebcf8dbb1143ba6460ed8. Полностью прочитаны17 changed files, связанные штатные blob/attachment/entity rights и report query/masking. Manifest65029d1d независимо SHA-сверен; все17 Git blobs совпали. После tested a550 изменён только docs/report-typed-drilldowns.md. git diff --check PASS. Go/HTTP/Chrome/DB не запускал, peer WT clean и не менялся.

## Проверенный контракт

Модель Drilldown фиксирует entity/image/attachment, metadata entity только catalog/document. Dynamic selector выбирает ключ доверенного targets, не произвольный entity/URL; нет owner override для binary. Fixed и dynamic взаимоисключены, максимум64 действий/64 targets, уникальный field case-insensitive. Ошибки metadata проверяются configcheck и runtime перед ссылками/переходом. Schema/lint/CLI отражают новые поля.

Renderer парсит ненулевой UUID, неизвестный selector/invalid ID даёт escaped текст. Имена/подписи/query escaped, server route фиксирован. Служебные id_field/type_field исключены из UUID-label substitution, но НЕ из query permission/RLS/masking: maskPlan.Apply идёт раньше. Отключения security для сохранения ID не добавлено.

GET reportDrilldown заново требует report.run и текущие metadata, валидирует индекс/UUID/selector. Entity: native read + rowAllowsID; image: сохранённый owner плюс существующий blobAllowed, ownerless reject; attachment: существующие rowAllowsOwnerID(read) либо тот же узкий uploaderPreviewAllowed. Прочитан код этих helpers и native download/image routes: критерии не расширены. Затем303 на фиксированный защищённый маршрут, где проверки повторяются. GET не вызывает DSL/write/чужой callback. Права ссылки не являются snapshot capability: изменённый ID может вести только к цели, уже доступной текущему actor по текущим правилам; эта граница явно описана.

Плоский HTML получает cell/row actions; composed только detail-строки, агрегаты групп/итогов остаются без ссылок, row-action cell для них пуст. Без drilldowns nil-safe методы оставляют старый путь. Экспорт не исполняет действие и не внедряет web-actions. Native configurator меняет editable YAML root keys, не drilldowns; клиентская Composition/UserSettings не становится whitelist.

## Доказательства и пределы

Прочитан HTTP matrix: flat/composed по5actions на2строках, фактический follow303 затем штатный GET200; forged selector, foreign-row RLS, invalid index/UUID, live target permission revocation и report.run revocation. Metadata negative cases и legacy collapse/detail_link/label tests сохранены.

Local26 leaf PASS/1PG root SKIP — факты source handoff, не новая runtime приёмка. Root independent PG14 ещё ожидался на момент этого review; его receipt в reviewed handoff пока нет, PG PASS не объявляю. Native image test проверяет protected GET/status, не декодирование реального фото. Навигация register row с compound key не реализована и не заявлена. Source main437 не имеет B20 DetailFields: runtime совместимость ещё не доказана.

## Решение и обязательные runtime стыки

Новых source blockers не найдено: CODE REVIEW OK. Runtime integration/SDK не выполнялись. Перед включением в общий кандидат нужен root PG14 и runtime gate: сохранить П3 DetailFields/rowWithDetail, применить Cell к detail_cells, flat12 без служебной пустой колонки; сохранить P2 variant compile и guards, B19 input-limit и B20 XLSXdimension. Общий кандидат должен отдельно сохранить П7 binder до П1 result_proc и typed UUID user_card, B7 guards/Audio/Answer/descriptors. Не подменять source26/PG14 результатами готового SDK. Shared :8444/B20 frozen не изменены.

Generated-with: Codex
