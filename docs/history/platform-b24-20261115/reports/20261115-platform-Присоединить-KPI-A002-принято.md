# Присоединить: KPI T5/T6 и штрафы A002 — взято следующим после B23

Оба repro прочитаны: codex2-20261115-KPI-T5-B20-Присоединить-repro-platform.md и cc2b-20261115-A002-П1-Присоединить-B20-RED.md. Владелец engine fix — platform; обходы в конфигурации не нужны. Подтверждён общий source-корень: Put переводит CurrentRow вниз, Append использует уже следующую строку; CurrentCol после Append попадает в следующий Put.

Делаю отдельную source-ветку/Draft и регрессии block cursor: Присоединить справа от последнего выведенного блока на его начальной строке; повторные Append наращивают ширину; следующий Вывести с колонки1 под полной максимальной высотой блока. Проверю разную высоту, sparse/merged cells и размеры колонок/строк, Put-only, HTML/XLSX/PDF. B23 immutable4fbee/B22 и существующие receipts не меняются.

На15:30 собственных Go/Chrome/app нет. Go/native-окна после очереди cc2; source работа идёт сейчас. Включение в вечерний состав заранее не обещаю: только exact гейты/review и решение cc2 до freeze. SDK/shared не изменены. Ответ дополню точным source/проверками после реализации.

Generated-with: Codex

## Source checkpoint

Source-only fix `1fad84697729162ea54bbd7f3a91a763cbcde1fa`, parent P1 source d232. Worktree Temp/platform-help-admin-20261111/tabular-block-output-source. Шесть файлов: neutral block cursor, narrow DSL shifted widths/group header, three regression files, CHANGELOG. Source source branch и существующий upstream Draft2627 fast-forward pushed; Draft сохранён. Gofmt/diff-check PASS, Go/test/native НЕ запускались, GREEN ещё не заявлен. B23/shared не изменены.

Следом отдельно в очереди T7: цвет текста HTML-ссылки расшифровки; source html.go owner platform. В Присоединить не смешивается.

## Уточнение source/runtime mapping

Runtime code-only6b5ffd49 atop B234fbee, sourcefe71da19 atop d232. Source prerequisite: в исходной d232 не было runtime-переноса ширин colWidths/Put/maket/ColumnWidthsPx; скопирован существующий native контракт только в source, runtime baseline сохранён. Полный pin B/b24-source-runtime-pins-wip.json SHA d08edbd49e86c1c9cc064485682f8eb01ec2fecddce7570e87d43219e0c4d968:6/8 целых файлов byte-equal; CHANGELOG и declarative BuildSheet loop/helper различаются указанными hunks. Раннее утверждение5equal было остановлено сравнением и отозвано до тестов; не product/runtime failure. Всё clean/pushed, тесты ещё НЕ запускались; собственный Go/Chrome HOLD до root canonical END.
