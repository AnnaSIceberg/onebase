# П1 user filters — приоритетный B23 принят

Прочитан exact repro20261115-1442: ALPHA/BETA, обычный report фильтрует, result_proc получает обе строки; HTML+snapshotXLSX без предупреждения. Source B21 подтверждает: runTabularReport вызывается до compose.ApplyFilters. Это data-correctness gap; роль/ACL обход этим не объявляется.

По прямому слову analysis делаю отдельный B23 от exactB21 0d77e915, только этот фильтр-fix. B22(ACL403/ordinaryXLSXdimension) остаётся отдельной после-вечерней линией, B21 не меняю. Нужны masked/typed source rows до result_proc, сохранённые/переданные настройки, П2 variant/query filters, HTML/XLSX/PDF frozen snapshot и REST; unsupported пути должны явно отказывать, а не молча игнорировать отбор. ACL/RLS/context/положительный source budget не ослаблять.

Ориентир — кандидат и гейты до16:30 при отсутствии новых blockers; это не обещание PASS или включения. В вечерний пакет только после приёмки cc2 до17:30; иначе после-выпускная линия по её слову. Root ресурсное окно запрошено, текущий B22 native cleanup завершён, новые Go/Chrome до очереди не запускаю. Воспроизводитель не менял, БД не копировал.

Generated-with: Codex
