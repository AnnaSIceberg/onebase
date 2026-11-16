# B28: B22 + П4/П7 + T8, SDK и native75 PASS

Runtime `f045bdb11b5e563120ff0381476957e5df9e2549`, branch `integration/cc2-b28-report-actions-context-20261116` CLEAN/pushed. База B27r2 `a10677b2903e4b94cefb8d90a4104fe8a24aa639` сохраняет B26 и уже существующие П4/П7 из B21; добавлен B22 ACL403/ordinary XLSX и проверенные CI/test-only исправления. Повторной реализации П4/П7 нет. Исправление ошибочной фразы старого отчёта1257: `20261116-1318-platform-исправление-состава-B27r2-B28.md`.

Локально: **66 XLSX/DSL leaf PASS**, **85 report leaf PASS / 10 PG-root SKIP отдельно**, справочник/buildall/i18nen PASS. **Точный Windows SDK:75 native checks PASS/JS0**: ACL/XLSX26, П4/П7 19, T8 30. Свои SQLite без копирования БД; реальные CLI/login/HTTP/браузер/выгрузки/openpyxl. Все три браузера запускались последовательно после Go END. Server/browser/context/driver/ports18572/18574/18582 закрыты, own auth DB удалены. Linux browser native не заявлен.

SDK (кандидат, не установлен):
- Windows: `C:\Users\a.sevrjukova\AppData\Local\Temp\platform-help-admin-20261111\onebase-1116b28-reports-windows-amd64.exe`, SHA256 `e3a9682e547a9e41cd6aeea7df733127cfa5dee443b84b6351e01506c8c503c0`.
- Linux: `C:\Users\a.sevrjukova\AppData\Local\Temp\platform-help-admin-20261111\onebase-1116b28-reports-linux-amd64`, SHA256 `8ad3b292e9cacb722b6738da6d0a20f844946bbccd296e517f336d60e7f86268`.

Manifest `C:\Users\a.sevrjukova\AppData\Local\Temp\platform-help-admin-20261111\manifest-b28-reports-final-20261116.json`; build `C:\Users\a.sevrjukova\AppData\Local\Temp\platform-help-admin-20261111\b28-reports-build-20261116\manifest.json`; native `C:\Users\a.sevrjukova\AppData\Local\Temp\platform-help-admin-20261111\b28-native-combined-20261116.json`. Pinned Go1.26.11/CGO0/GOMAXPROCS2/-p1. VCS metadata внутри worktree-сборки не приписывается; provenance — clean pins/команды/buildinfo/SHA.

**Независимый PG208 PENDING.** `C:\Users\a.sevrjukova\AppData\Local\Temp\platform-help-admin-20261111\b28-independent-pg-handoff-20261116.json` содержит exact Linux prebuilt API12+UI190+interpreter6, SHA, regex и полные ожидаемые имена. Требуется независимый root/runner: PASS208uniqueleaf/FAIL0/SKIP0 и namespace/relations/routines before=after/cleanup. Это новый B28 gate; исторические B21 PG178 или B22 native не ретегируются. Старый B22 corrected PG196 также без полученного результата; при выполнении B28 gate он покрывает те же B22 проверки в новой линии, результаты различать по pin.

Source ACL2631 `6ac8da0f`, П42630 `892365a0`, П72629 `449bafdb`, XLSX2632 `54a8303a`: целевые HTTP-матрицы и buildall всех четырёх обновлённых веток PASS; PG локально SKIP. Все pushed, Draft. T8 #2635 source `b7c57cb62` имеет дерево, равное проверенному9b8db2dc; новый CI43846509362 ещё идёт (19success/2running на последнем чтении). Старый Windows failure сохранён; GitHub запретил повтор job без admin rights, проверки не обходились. #2634 B26 полный CI GREEN после явно разрешённого push2a0f4acad.

**Приоритеты обновлены её сообщениями:** после текущего B28/T8 шага — enum-параметры отчётов (прямая жалоба «Заявки на дату», заказ `codex2-20261116-заказ-platform-enum-параметры-отчётов-B25.md`), затем дерево меню1С, затем П8–П25 и отчётная очередь. Два A080 (MMMM и цепочка календарных умолчаний) добавлены в несрочную отчётную очередь, пока не взяты. Владелец — последним.

Меню: ранний Draft **#2636**, branch `feature/recursive-command-menu-20261116`, commit `3c88d2368843da899665a94d8113ef29eee9d715`, только проект контракта `docs/recursive-command-menu.md`. Один список `nodes`, папка/typed target, явные ID, предел8папок, роли как дополнительные ограничения, старый формат сохраняется. **Ещё не поддержано SDK, YAML не применять к B26/B28.** Реализация после enum. Операторское меню/конфигурации не менялись; source-command → report/processor сопоставляет cc2.

Shared :8443/:8444/:8448, конфигурации и1С не тронуты. Рекурсивного поиска по C: нет. Новых агентов не создавал; CPU ограничен одной Go-линией, браузер отдельно.

Generated-with: Codex
