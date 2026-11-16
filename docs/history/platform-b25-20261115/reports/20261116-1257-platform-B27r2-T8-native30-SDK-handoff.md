# T8/П25 — API и SDK B27r2; native30 PASS

Draft **#2635**, source `9b8db2dc37a4d799bfaaeef0cda1d0512168194b`. Кандидат runtime `a10677b2903e4b94cefb8d90a4104fe8a24aa639` отB26 `513274b7e1bc230d8c0de5fcd22a34deb672345c`; рабочие базы/стенды не переключались. B22 и П4/П7 пока отдельные линии, в этот кандидат не включены.

Штатный API для perenos:
```bsl
Книга = Новый КнигаТабличныхДокументов;
Книга.ДобавитьЛист("СМ", ТабДокСМ);
Книга.ДобавитьЛист("ХД", ТабДокХД);
СодержимоеBase64 = Книга.Записать("служебная-записка.xlsx", "xlsx");
```
English: `SpreadsheetWorkbook`, `AddSheet`, `Count`, `Write`. На добавлении сохраняется снимок; исходный ТабДок можно очистить/переиспользовать. Одна книга, листы в порядке добавления. API возвращает base64, файловые пути не открывает; транспорт/расписания остаются в прикладном слое. Имена/UTF-8/дубликаты/Excel bounds и общие budgets — в `examples/tabular-workbook/README.md`. Методы зарегистрированы в штатном langref/autocomplete.

Source core79aac: **49 leaf PASS**, independent openpyxl4/4, buildall/i18nen PASS. Source9b добавил справку/autocomplete и исправил унаследованный дрейф `ПолучитьМакет`; CLI/langref gates PASS. RuntimeB27r2: **50 XLSX/DSL PASS**, **14 HTTP PASS/2 PG-root SKIP отдельно**, buildall/i18nen и оба SDK PASS, pinned Go1.26.11/CGO0/GOMAXPROCS2/-p1. HTTP включает сохранение высоты side_panelB26 и прежние TabDoc/dimension guards.

**Native Windows30 PASS, JS0**, Chrome109, собственная SQLite без копирования БД. Публичный `check`; реальный login → форма отчёта → POST303 → viewer200 → `result_proc` с обычным runtime factory map → XLSX base64. Два листа СМ/ХД, числа111/222, красный/зелёный, source переиспользован и затем изменён до999, снимки сохранены. Openpyxl3.1.5 normal/read_only безreset/bounds: A1:B2 на обоих листах; объединения, разные ширины/высоты и landscape/portrait/fitWidth сохранены. Пустая книга даёт явный module error500, частичного результата нет. Отдельно native `langref --format json` вернул все3 метода книги. Browser/context/driver/app/18582 закрыты, own-auth DB удалена. Go/Chrome не работали одновременно.

Первый native B27 FAIL был только ожиданием статуса redirect в событии Chrome: реальный server log уже содержал303→200. Исправлен наблюдатель, SDK не менялся; B27 получил29 PASS. После добавления metadata/help построен отдельный B27r2, прошёл30 PASS. Старые SDK/FAIL/receipts сохранены, не ретегировались. Linux native этим Windows-прогоном не заявлен.

**SDK для изолированной проверки конфигурации (не установлен):**
- Linux: `C:\Users\a.sevrjukova\AppData\Local\Temp\platform-help-admin-20261111\onebase-1116b27r2-workbook-linux-amd64`; SHA `6c27eddfff6d92566e680b33f3704ac327d78673bdad36671319805ee045b3d9`.
- Windows: `C:\Users\a.sevrjukova\AppData\Local\Temp\platform-help-admin-20261111\onebase-1116b27r2-workbook-windows-amd64.exe`; SHA `7bd6521a53b3c9a4342c89a72076cc031115c06f0b3f235209ec872436f587e1`.

Manifest `C:\Users\a.sevrjukova\AppData\Local\Temp\platform-help-admin-20261111\manifest-b27r2-workbook-final-20261116-1257.json`, SHA `3319441bdde0bb5ab8d62896c70eb1dec1e0e955621435988ed4870a26e5017e`. Build `C:\Users\a.sevrjukova\AppData\Local\Temp\platform-help-admin-20261111\b27r2-workbook-build-20261116\manifest.json`, SHA `d88170fb1c6ac0e412129cb7b2fd914e9ee10e495d89808e9294bd2586a88d91`; native `C:\Users\a.sevrjukova\AppData\Local\Temp\platform-help-admin-20261111\b27r2-native-workbook-attempt1-20261116\receipt.json`, SHA `aef74fb5084fd501877184a6746c226c4b80642d12c1a0837c125d9e3e0542d0`. EmbeddedVCS не приписываю .git-worktree сборке: provenance — clean runtime/source pins, команды, Go buildinfo и SHA. Core Go blobs T8 совпадают с upstream source; runtime сохраняет дополнительные descriptors (187 функций/172 метода), а source имеет176/169. Оба справочника сгенерированы отдельно и проверены.

**Открыто:** full source CI #2635 — в новом прогоне один Windows failure разбирается, Ready/production не заявлены. Родитель #2627 остаётся Draft; его CI/docs drift исправляется отдельно и пойдёт в ACL source #2631. #2630/#2629/#2632 уже обновлены main (892365a0/449bafdb/54a8303a), CI идут. B22 correctedprebuilt UI190+interpreter6 всё ещё ждёт независимую PG-полосу (отчёт1047).

B26 #2634: после одобренного пользователем push source2a0f4acad **полный CI GREEN**, включаяPG/Windows/e2e. Runtime513274b7 на :8444 установлен release-claude, platform не менял shared. Очередь владельца последняя; рекурсивных поисков по C: не было.

Generated-with: Codex
