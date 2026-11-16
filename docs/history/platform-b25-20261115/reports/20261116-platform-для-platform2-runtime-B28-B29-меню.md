# Platform2: точная база движка B28/B29; дерево меню передано вам

Её разделение 16.11 принято: **дерево меню делает platform2 (wV:p1)**. Platform снимает его со своей активной очереди и не продолжает код/PR меню. Прикладные деревья и соответствие команд 1С переносит cc2 после вашего контракта/бинаря.

## Точные сохранённые runtime

- **B29, актуальная накопленная база:** `902e4d97e50e90f047ec6d27832e2f9d175d095d`, fork `integration/cc2-b29-report-enum-20261116`, WT `C:/Users/a.sevrjukova/AppData/Local/Temp/platform-help-admin-20261111/report-b29-enum-runtime`. CLEAN/pushed на передаче. Это B28 + enum-параметры отчётов, source Draft #2637 `2e185f0615f98111cd1353d71afcc72fdabbd5fd`.
- **B28:** `f045bdb11b5e563120ff0381476957e5df9e2549`, fork `integration/cc2-b28-report-actions-context-20261116`, WT `B/report-b28-runtime`. Это B27r2 + B22 ACL403/ordinary XLSX и CI/test-only исправления. П4/П7 уже унаследованы из B21, не переносить повторно.
- B27r2 `a10677b2903e4b94cefb8d90a4104fe8a24aa639` содержит T8 многолистную книгу, B26 `513274b7e1bc230d8c0de5fcd22a34deb672345c`, B25 и П4/П7 из B21 `0d77e91557b12bada4668fd7bfa5f1b13a37de08`. В старом отчёте1257 была ошибочная фраза об отсутствии П4/П7; она исправлена отдельным отчётом1318. База B29 включает всё это, плюс B22 и enum.

B = `C:/Users/a.sevrjukova/AppData/Local/Temp/platform-help-admin-20261111`.

## Готовые SDK и доказательства

B29 Windows `B/onebase-1116b29-report-enum-windows-amd64.exe`, SHA256 `0cdf623d05427656b219020edc91c48984855831c56f89b662ad57d45a91010d`; Linux `B/onebase-1116b29-report-enum-linux-amd64`, SHA256 `b9f2a748fd5e29f39215d35112450df4ebc3a635de7dbe07ded521469d16e6a0`.

Manifest `B/manifest-b29-enum-final-20261116.json`; build `B/b29-enum-build-20261116/manifest.json`; native `B/b29-native-enum-attempt2-20261116/receipt.json`. Native38PASS/JS0 под неадминистратором, cleanupalltrue, ownSQLite/noDBcopy. Локально runtime93HTTP leafPASS/7PG-rootSKIP; независимый **PG312 PENDING**, exact prebuilt/regex/names в `B/b29-independent-pg-handoff-20261116.json`.

B28 native75PASS (ACL/XLSX26 + П4/П7 19 + T8 30), отчёт `20261116-1333-platform-B28-native75-SDK-и-очередь.md`. Source CI GREEN22 у B26 #2634/T8 #2635/П4 #2630/П7 #2629/ACL #2631/ordinary XLSX #2632. У enum #2637 source enum SQLite+PG матрица прошла в CI, но общий job PG завершился20минутным таймаутом `TestWritePost_FormHTTPMatrix/.../sqlite`; общий GREEN не заявляется. У родителя П1 #2627 отдельный bench failure. Гейты не ослабляются, Draft/Ready не менял.

## Что осталось от начатого меню

Draft **#2636** `https://github.com/ivanarama/onebase/pull/2636`, ветка `feature/recursive-command-menu-20261116`, source `3c88d2368843da899665a94d8113ef29eee9d715`, fresh main base `adddaa8d95f04b4b7b3ff57991edbbd9b8740fa2`. Только **проект** контракта `docs/recursive-command-menu.md`; рабочего API/SDK меню ещё нет. WT `C:/Users/a.sevrjukova/platform-recursive-menu-20261116` очищен от собственных незавершённых правок после remote backup. Можно принять эту линию по её поручению, не создавая конкурирующий delivery PR.

Незавершённая проба модели: `B/menu-recursive-first-model-WIP-20261116.patch`, SHA256 `c39e124b505bdbfb4800e5a04a6245239abeba0229f29d089a477ef9dd4fcae4`, `B/menu-first-model-WIP-20261116.json`. Архив pushed в `wip/platform-b24-final-b25-handoff-20261115` (8bcfaabb...). В SDK её нет, Go-проверок нет; это не принятая реализация и не обязательная архитектура. Fixed-depth обходы есть в `internal/navigation` delta/layout/Diff, role projection, renderer, settings editor/preview. Простого поля Nodes недостаточно. Исходный точный заказ cc2 — `20261116-1320-cc2-дерево-меню-1С-план-заказ-platform.md`.

## Правила и текущая параллельная работа

Основной общий стенд — **ЛПЦ :8448**, любая выкладка только по её слову, одним накопленным пакетом. :8444 и :8443 также без её слова не менять; **B29 на :8444 не ставить**. Движок копится B26→B29+. Свою доработку проверять на собственной копии; операторы КЦ/мессенджера в меню без изменений. Используйте отдельный WT; все свои коммиты/ветки пушить, только явный add, Generated-with: Codex.

Platform сейчас делает маску даты в source Draft #2611, HEAD `d570ea4a40f3681bde8dec518ffee8565c4eddd3`, WT `C:/Users/a.sevrjukova/platform-report-date-mask-20261116`; **этого ещё нет в runtime B29**. Будущий B30+ накопит её поверх B29 после native-проверок. На записи этого файла локальное Go-окно platform занято source проверкой, журналы `B/date-mask-source-checks-attempt1-20261116`; браузер запускается только после Go END. Одновременно несколько Go/Chrome линий на ПК8ГБ не запускать; Go -p1/GOMAXPROCS2, чужие процессы не останавливать.

У platform дальше уровни/итоги/фон групп, отборы, связь параметров и ПДн; дерево меню дальше не изменяет. Новых агентов не создавал. Никаких рекурсивных поисков по C:.

Generated-with: Codex
