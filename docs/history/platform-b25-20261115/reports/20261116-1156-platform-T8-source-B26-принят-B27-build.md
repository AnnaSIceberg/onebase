# platform: T8 source проверен; B26 принят, B27 SDK собирается

T8/П25 **Draft #2635**, source `79aac9d7426497fac35ba00ea42ba95232f23ff5`. Через публичный DSL вход: **49 leaf PASS**, openpyxl3.1.5 **4/4** (normal/read_only × СМ/ХД, числовые111/222, красный/зелёный, безreset/bounds), build./... PASS, i18n en PASS. Реальный API: `Новый КнигаТабличныхДокументов` / `SpreadsheetWorkbook`, `.ДобавитьЛист(Имя, ТабДок)` / `.AddSheet`, `.Количество()` / `.Count`, `.Записать(Имя,"xlsx")` / `.Write` → base64 одной книги. На AddSheet сохраняется снимок, исходный ТабДок можно переиспользовать. Имена/дубликаты/UTF-8/границы Excel и общие бюджеты проверяются, детали в `examples/tabular-workbook/README.md`. Параметры листов, freeze/повтор шапки/группы проверены на обоих листах. Старый single-document writer использует тот же общий writer.

Первый RED был в ожидании теста: OOXML без атрибута t — стандартное число, Excelize называет его CellTypeUnset. Тест исправлен с сохранением запрета строковых типов; нормальный/потоковый openpyxl независимо подтвердил числа. Затем отдельно добавлен отказ для invalid UTF-8 имени, текущий79aac проверен полностью. Первые логи сохранены, не ретегированы.

Source proof `C:\Users\a.sevrjukova\AppData\Local\Temp\platform-help-admin-20261111\t8-verify-attempt3-20261116\receipt.json`, SHA `1482b05598ec31ea5ae37b476b544ebce6ed6b03d3c626960c3607e4da62eebd`. Draft зависит от П1 #2627; freshmain adddaa объединён с существующей зависимостью, единственный docs/features conflict разрешён сохранением обеих секций. FullCI/Ready не объявлены.

**B26 принят в линию.** Runtime513274b7 отB25, Linux e977a632… на :8444 установлен release-claude по его отчёту1051; platform не устанавливал. Draft #2634 CI падал только на en «Ширина панели». В собственной CI-ветке подготовлен и i18ncheck проверен один перевод; после явного ответа пользователя «Разрешаю push в #2634» source обновлён `2a0f4acad4df556dcd8d4676ef2967aa050fe687`, remote exact проверен. Это source-only CI fix; frozen B26 SDK/runtime не ретегированы. В B26 runtime этот перевод уже присутствует. Новые CI checks идут; fullGREEN пока не заявлен.

**B27 = B26 + T8**, own runtime `7b89c38bfab4d2aa2f2c8d1843fb66f83032cfd3`, branch уже pushed. Только8 файлов T8, все6 Go blobs точно равны source79aac; CSS/панельB26 сохранены. Последовательно идёт GoM2/-p1: XLSX/DSL + HTTPcritical (height/TabDoc/dimension), buildall/i18n, Windows/Linux SDK. После GO END — собственный native HTTP/browser smoke. Кандидат не устанавливается. B22/П4/П7 пока отдельные линии, в B27 не подмешаны.

**B22:** native26 PASS, correctedprebuilt UI190 + неизменённыйinterpreter6 готовы по отчёту1047; independent PG196 ещё ждёт исполнения. Исходные Draft #2630/#2629/#2631/#2632 GitHub показывает CONFLICTING со свежим main, поэтому новые проверки пока не появились; после T8 SDK обновлю их последовательно, без изменения frozen SDK. Очередь владельца последняя.

Рекурсивных поисков по C: не было. Новых БД-копий/shared изменений/внешних отправок/расписаний нет. Work/source commits pushed, helpers и этот отчёт сохраняются следующим отдельным WIP push.

Ledger `C:\Users\a.sevrjukova\AppData\Local\Temp\platform-help-admin-20261111\platform-line-20261116-1156.json`, SHA `b5158e41b7a6fbed84107cd51715bb44ff943355d712fd8bfdafb2efe4c05089`.

Generated-with: Codex
