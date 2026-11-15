# B23 — отборы входа result_proc исправлены

Exact runtime/source `4fbee90500a26f9bc59336e0ae0453027fd941ea`, parent B21 `0d77e91557b12bada4668fd7bfa5f1b13a37de08`. Девять файлов, только source-filter datafix; B22 не включён. Clean/pushed fork branch `fix/report-result-source-filters-b23-20261115`; Draft https://github.com/AnnaSIceberg/onebase/pull/5 показывает минимальную runtime-дельту принятой линии. Отдельный перенос зависимого П1/П2 source в upstream этим handoff не объявляется завершённым; CI не приписывается PASS.

Пользовательский/сохранённый отбор применяется после RBAC/RLS/масок до передачи массива Результат в модуль. П2 fixed predicates остаются в SQL до report_max_rows. Некорректный JSON/поле/оператор/неподдержанный reference filter явно отклоняются, queryless+filter HTTP422. REST фильтрует полный доступный источник до пагинации, в существующем max1000; превышение HTTP422. Бюджеты не повышены. Компилятор проекций ссылок не менялся: явная .Ссылка сохраняет UUID, обычная bare reference сохраняет прежнее представление.

Build ./... и i18n en PASS. Local156 leaf PASS,16 локальных PG roots SKIP — это не PG PASS. Независимый PG cc2: UI212+API42=254 unique PASS (127SQLite+127PG), FAIL0/SKIP0, все exact names; namespace/relations/routines95a93e4b…7dc4 неизменны/cleanup exact.

WINDOWS SDK `36405653a12e8732f2cce90222537e3681f206fb34818fa33f44f76fb1861fc7`: native22 checks PASS/JS0, own nonadmin. Ordinary control ALPHA-only; native Generate без фильтра COUNT2; GET и public POST с фильтром COUNT1, ALPHA есть/BETA нет. HTML, XLSX (default installed openpyxl) и PDF (pypdf text) одного P1 snapshot совпадают. P2 AND и REST total1/rows1 PASS; unknown filter HTTP400 без snapshot redirect. Native Linux proof не заявлен. Attempt1 до Chrome остановлен из-за обязательного первого admin; FAIL сохранён, поправлен только bootstrap own fixture.

Linux SDK: `C:\Users\a.sevrjukova\AppData\Local\Temp\platform-help-admin-20261111\onebase-1115b23-report-source-filters-linux-amd64` SHA256 `92cf861981ade515e1f254fbce1c7a17e9b1ac3b9bf51807c31a2bb7ef861180`.
Windows SDK: `C:\Users\a.sevrjukova\AppData\Local\Temp\platform-help-admin-20261111\onebase-1115b23-report-source-filters-windows-amd64.exe` SHA256 `36405653a12e8732f2cce90222537e3681f206fb34818fa33f44f76fb1861fc7`.
Final supplement: `C:\Users\a.sevrjukova\AppData\Local\Temp\platform-help-admin-20261111\manifest-1115b23-report-source-filters-DEV-final.json` SHA256 `0ea02d35bc464edcd6f1a0a2b5b338a87cd13c1e04deabeec0cc60f226063d5b`. Original build3bce035c не ретегирован. Три PNG: `C:\Users\a.sevrjukova\Desktop\Отборы-ТабДок-B23-native-20261115-platform`; SHA в supplement. Квитанция native86e2db20, cleanupcef8875e; browser/context/driver/app/port18576/authDB0, tunnel unused, Go0. No DBcopy/shared mutation.

Известный ordinary XLSX dimension=A1 и nested result_proc ACL500 относятся к B21/B22 after-release и НЕ исправлены в B23. P1 result XLSX inherits B20 fix. Новый T5 Присоединить оставлен отдельной очередью. B21 и B22 артефакты неизменны. Включение в вечерний состав — только exact acceptance cc2, применение общих служб — её вечерним словом. До17:45 нужен отдельный свежий reboot-checkpoint.

Generated-with: Codex
