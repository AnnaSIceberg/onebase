# platform — B22 native PASS; исправления исходных Draft PR

Возобновление по её поручению 16.11. Общие/платформенные вводные, CLAUDE.md целиком, CONTRIBUTING.md, checkpoint 17:45 и хвост журнала прочитаны. Рекурсивных поисков по диску не запускал; только точные пути и Git metadata. После восстановления exec CPU 1,6%, свободная RAM 2147602432 байт. Новых Go-сборок в этом отчёте нет.

B22 exact product `1496c60ccad535d722bb1d83d2694d7d81bfbded`, готовый Windows SDK `8ee6d46c2665bee1eba4388e5d4d6a433d9603058286a05683874717c51fbd2f`: **26 именованных проверок PASS, JS0**, SHA бинаря проверен отдельно; Chrome 109.0.5414.46, собственная SQLite без копирования БД. Настоящие ACL: direct/query/queryless/rethrow → 403 без выдачи данных; текстовая имитация ACL и ошибка запроса → 500; разрешённые direct/viewer → 200. Прямой публичный XLSX-маршрут и настоящий клик Excel → фоновая задача → download дают одну и ту же книгу: A1:B28, 28 строк, итог 253, 56 заполненных ячеек, openpyxl 3.1.5 normal=read_only по координатам/значениям/типам, без reset_dimensions и forced bounds. XLSX SHA `d5719131735859b010adb77af13200953b5acd39135b5e57d3bda157d1eca7ca`.

Нативный сбой 15.11 был в наблюдателе: helper искал несуществующую прямую ссылку `/B22Ordinary/excel`, тогда как toolbar использует lower-case `/export/excel`. Исправлен только helper: direct проверяется публичным HTTP, background — реальным кликом по toolbar. SDK/продукт не менялись. Сохранён отдельный sandbox IPC FAIL attempt2 (WinError5), затем ограниченный loopback smoke вне песочницы PASS attempt3. Browser/context/driver/app/port18574 закрыты, own-auth DB удалена, tunnel не использовался.

Причины старого красного CI сверены по GitHub check annotations. В существующие Draft PR отправлены узкие исправления, HEAD и remote проверены, WT clean:

| Draft | Новый HEAD |
|---|---|
| #2632 | `6f9a28d81b056bfe5fda42d1d161f182e9737395` |
| #2629 | `54409bcda023220f32d1e723db7656ee0ec6216e` |
| #2630 | `f022b846801ed2d953c21a20b68f94bf7e0ece61` |
| #2631 | `6f4e46893706169822c33d980428fc5eb5bc8a17` |

#2632: обработаны Close ошибок XLSX readers; opt-in путь тестовых артефактов отмечен как заданный доверенным test runner (G703), assertions сохранены. #2629: эквивалентный De Morgan + обработка Close. #2630: существующий комментарий об экранировании перенесён на правильный template.HTML cast. #2631: сохранённое B22 test-only исправление a1ac перенесено в source: `ВЫБРАТЬ (` действительно ошибочно и в PostgreSQL, в отличие от голого SELECT. Продуктовые изменения в этих четырёх коммитах отсутствуют (П7 условие эквивалентно). gofmt/diff-check; новые remote CI пока не появились — GREEN/Ready не заявлены.

**Открыто по B22:** independent exact-name PG gate UI190+interpreter6. Старый UI prebuilt содержит прежний fixture: его нельзя объявить зелёным или ретегировать. Следующий шаг — отдельный test-only prebuilt от сохранённого a1ac, затем independent gate; production/shared установки не выполнялись. П4/П7 имеют отдельные сохранённые B21 PG178/Windows19 PASS; не выдаю их за новый CI этих HEAD.

**Далее:** T8/П25 штатная многолистная книга; затем оставшаяся проверка П4/П7 и П8–П25 по утреннему заказу perenos. B26: height source/runtime WT release-claude обнаружены только по Git metadata, нового Draft по точной ветке пока нет; правку не дублировал. Очередь владельца последняя.

Manifest supplement: `C:\Users\a.sevrjukova\AppData\Local\Temp\platform-help-admin-20261111\b22-native-resume-20261116-1032.json`, SHA `57ed4de17e2afc40636d8fb780093aef92633d4c24942eb67e9fde64ce65e5b3`. Native receipt `C:\Users\a.sevrjukova\AppData\Local\Temp\platform-help-admin-20261111\b22-native-acl-xlsx-attempt3-20261116\receipt.json`, SHA `ef34a3dceceb871e2116ce76f357b9e4d3c915ff36f72deac9838379ba9a8493`. Исходный B22 manifest и исторические FAIL сохранены. Отчёт/новые helper sources/квитанции подлежат отдельному WIP push, квитанция push будет отдельным дополнением.

Generated-with: Codex
