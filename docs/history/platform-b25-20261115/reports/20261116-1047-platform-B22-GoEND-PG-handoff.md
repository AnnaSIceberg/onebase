# B22 — GO END; исправленный prebuilt для независимого PG

После native Windows PASS26 (отчёт 20261116-1032-platform-B22-native-и-CI.md) завершён единственный ограниченный Go-прогон (-p=1, GOMAXPROCS=2). **SQLite12 PASS, PG2 SKIP отдельно**, новый Linux UI prebuilt готов. SDK, продукт и shared базы не пересобирались/не менялись.

Product/runtime `1496c60ccad535d722bb1d83d2694d7d81bfbded`. Test source `a1ac5b8af8624b05c81bf8f074555578bb3df52e` — ровно один test-only файл `internal/ui/report_result_access_test.go` относительно frozen product: вместо допустимого PG голого SELECT используется незакрытая скобка. Перед/после clean/head проверены. Старый prebuilt B22 и старые FAIL не перезаписывались.

Новый immutable UI artifact: `C:\Users\a.sevrjukova\AppData\Local\Temp\platform-help-admin-20261111\b22-corrected-ui-tests-attempt2-20261116\b22-ui-fixture-corrected-linux-amd64.test`, SHA `5e68b26cc6cea7c205d7f2563b23654f827730cad9f2d75d38a76b727f38a569`, размер 84488932 B. Команды, logs, Go1.26.11 buildinfo и привязка clean source → binary: `C:\Users\a.sevrjukova\AppData\Local\Temp\platform-help-admin-20261111\b22-corrected-ui-tests-attempt2-20261116\receipt.json`, SHA `d3da5f12a116289149f781f96538e34af11d92682246299004b75986a89aa4bb`. Встроенный VCS stamp для .test не заявляю.

**Нужна независимая PG-полоса:** новый UI190 (95 SQLite + 95 PG) и прежний неизменённый interpreter6 (3+3), всего196 exact leaf names, FAIL0/SKIP0 с before/after namespace cleanup. Полные пути/SHA/regex/имена — `C:\Users\a.sevrjukova\AppData\Local\Temp\platform-help-admin-20261111\b22-corrected-pg-handoff-20261116-1047.json`, SHA `068c187cc47cb93a0a0458609ba6020ae5be2f1a7320ef38b3a2555131fba5e8`. Старый UI `b6c5a077…` не использовать: он содержит прежний fixture. Ничего не ретегировать.

Первый запуск Go внутри песочницы получил Access denied на существующий cache и не дошёл до тестов; логи сохранены. Повтор вне песочницы использовал существующий кэш, без создания нового холодного cache: SQLite12 PASS, cross-build PASS. Новых Chrome/SDK/стендов при Go не было. Полный CI/Ready/PG PASS пока не заявлены; новые checks Draft #2631/#2632/#2629/#2630 ещё не появились.

Дальше T8/П25 в source/Draft, Б26 только после Draft release-claude; очередь владельца последняя. Go END — новый compiler не запускаю одновременно с чужим. Широких/рекурсивных поисков по C: не было.

Generated-with: Codex
