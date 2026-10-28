# Передача: доработки платформы OneBase для конфигурации cc2

Состояние на 28.10.2026. Документ для разработчика, который принимает
доработку платформы под конфигурацию колл-центра (cc2). Секретов здесь нет:
адреса серверов, учётные записи, ключи и токены выдаёт администратор.

## 1. Что такое OneBase и наш форк

- **OneBase** — 1С-подобная бизнес-платформа на Go: один бинарь исполняет
  прикладные конфигурации (метаданные YAML + модули на встроенном языке `.os`)
  поверх PostgreSQL или SQLite. Устройство движка описано в `CLAUDE.md`
  (раздел «Архитектура движка»), форматы объектов — в `DEVELOPER.md`,
  управляемые формы — в `docs/forms.md`, язык — в `docs/dsl-reference.md`.
- **Upstream** — `ivanarama/onebase`, владелец и правообладатель — Иван Титов.
  Все изменения платформы идут в upstream через PR. Правило «только
  локальных» коммитов нет: каждая правка сразу оформляется Draft PR.
- **Форк** — `AnnaSIceberg/onebase`. В нём живут ветки наших PR
  (`feature/*`, `fix/*`) и эта интеграционная ветка.
- **Конфигурация cc2** — отдельный репозиторий `oneservice-cc_v2`: формы,
  модули, сценарии проверки. Платформа и конфигурация выкладываются
  на сервер колл-центра (КЦ) вместе: конфигурация пользуется ключами из ещё
  не влитых PR.

## 2. Интеграционная ветка (dev-сборка)

Ветка `integration/cc2-dev-20261028` в форке — точная копия локальной ветки
`local/cc2-2026-10-22`, из которой собираются бинари для сервера КЦ.

- База — `origin/main` на 21.10.2026 (`55322710`). Поверх — коммиты всех наших
  открытых PR (cherry-pick).
- `11af0d5f` — сборка **1027f**. Она стоит у оператора, выпуск-14 от
  28.10.2026.
- `39106287` — сборка **1028a** = 1027f + #2397 (`Оповестить`). Передана
  для стенда разработки.
- Последний коммит ветки — этот документ, код не меняет.

Номер сборки виден в `onebase --version` как `dev-<короткий sha>`.

**Чего в ветке нет.** У части PR в ветке самого PR есть более поздние
коммиты: правки по ревью владельца или синхронизация с `main`. В dev-сборку
они не переносились. В таблице ниже это отмечено как «кроме поздних
коммитов (N/M)». При пересборке брать надо актуальный HEAD ветки PR.

## 3. Состав dev-сборки — наши открытые PR

Статусы и метки на 28.10.2026:
- Draft / Ready — состояние PR на GitHub;
- `reviewed`, `changes-requested`, `needs-decision`, `ship`, `hold` — метки
  конвейера владельца (раздел 6).

Всего открыто 69 PR: 38 Ready, 31 Draft. Из них:
- 19 с `changes-requested` — владелец ждёт правок;
- 17 с `needs-decision` — ждут решения владельца;
- 7 с `ship`, но у всех семи стоит и `needs-decision`, а это стоп даже при `ship`.

| PR | Суть | Статус | Метки | Зависимость | В dev-сборке |
|---|---|---|---|---|---|
| [#1983](https://github.com/ivanarama/onebase/pull/1983) | fix(query): PostgreSQL — отбор виртуальных таблиц по колонкам регистра, пустая ссылка — NULL, UUID-строка без ::uuid | Ready | hold | — | да |
| [#2110](https://github.com/ivanarama/onebase/pull/2110) | fix(документы): пометка и удаление проведённого документа — через ОбработкаУдаленияПроведения | Ready | ship, needs-decision, reviewed | — | да, кроме поздних коммитов (8/9) |
| [#2172](https://github.com/ivanarama/onebase/pull/2172) | feat(списки): постоянный отбор формы списка, условие — только для ролей | Ready | needs-decision | — | да, кроме поздних коммитов (6/9) |
| [#2175](https://github.com/ivanarama/onebase/pull/2175) | feat(запросы): табличная часть справочника/документа как источник запроса | Ready | ship, needs-decision, reviewed | — | да, кроме поздних коммитов (5/6) |
| [#2177](https://github.com/ivanarama/onebase/pull/2177) | feat(ui): ПолеКартинки с data_path — превью вложения и открытие в полном размере | Ready | needs-decision | — | да, кроме поздних коммитов (7/8) |
| [#2188](https://github.com/ivanarama/onebase/pull/2188) | fix(query): дата регистра на SQLite сравнивается с параметром в формате регистра | Ready | needs-decision | — | да, кроме поздних коммитов (5/6) |
| [#2190](https://github.com/ivanarama/onebase/pull/2190) | feat(ui): заголовок колонки табличной части из реквизита (title_from) | Ready | ship, needs-decision, reviewed | — | да, кроме поздних коммитов (2/3) |
| [#2221](https://github.com/ivanarama/onebase/pull/2221) | fix(ui): ПередЗаписью/ПриЗаписи формы видят ссылку записанного объекта и верный ЭтоНовый() | Ready | needs-decision | — | да, кроме поздних коммитов (2/3) |
| [#2222](https://github.com/ivanarama/onebase/pull/2222) | fix(ui): Объект.Провести() в обработчике управляемой формы проводит документ | Ready | reviewed | — | да, кроме поздних коммитов (2/3) |
| [#2253](https://github.com/ivanarama/onebase/pull/2253) | fix(query): сортировка по ссылке в сгруппированном запросе — агрегатом представления | Ready | needs-decision | — | да, кроме поздних коммитов (2/6) |
| [#2257](https://github.com/ivanarama/onebase/pull/2257) | feat(формы): поля в диалоге ПоказатьВопрос — ДиалогПоля и многострочный ввод | Ready | ship, needs-decision, reviewed | — | да |
| [#2258](https://github.com/ivanarama/onebase/pull/2258) | feat(списки): список помнит выбранную пользователем сортировку | Ready | needs-decision | — | да, кроме поздних коммитов (1/4) |
| [#2264](https://github.com/ivanarama/onebase/pull/2264) | fix(query): ссылочный и булев реквизит регистра в виртуальных таблицах | Ready | ship, needs-decision, reviewed | — | да, кроме поздних коммитов (1/3) |
| [#2270](https://github.com/ivanarama/onebase/pull/2270) | feat(dsl): ПолучитьПисьма — чтение почты по POP3 и IMAP | Ready | reviewed, changes-requested | — | да, кроме поздних коммитов (2/6) |
| [#2273](https://github.com/ivanarama/onebase/pull/2273) | feat(формы): колонки таблицы значений формы — состав, подпись, ширина, readonly, ПриАктивизацииСтроки | Ready | changes-requested | — | да, кроме поздних коммитов (1/3) |
| [#2275](https://github.com/ivanarama/onebase/pull/2275) | feat(формы): таблица-выдача — rows, show_count, empty_text у ValueTable (поверх #2273) | Ready | changes-requested | поверх #2273 | да |
| [#2276](https://github.com/ivanarama/onebase/pull/2276) | feat(формы): hint — всплывающая подсказка у всех полей, текст под полем по hint_inline | Ready | ship, needs-decision, reviewed | — | да, кроме поздних коммитов (1/2) |
| [#2277](https://github.com/ivanarama/onebase/pull/2277) | feat(формы): density: compact — плотная раскладка отдельной формы | Ready | needs-decision | — | да, кроме поздних коммитов (3/4) |
| [#2279](https://github.com/ivanarama/onebase/pull/2279) | feat(формы): высота закладок — fit_window, equal_height, height с прокруткой внутри | Ready | reviewed, changes-requested | — | да, кроме поздних коммитов (4/8) |
| [#2280](https://github.com/ivanarama/onebase/pull/2280) | feat(формы): кнопка в строке команд документа — placement: header; «? Справка» туда же | Ready | reviewed, changes-requested | — | да, кроме поздних коммитов (1/4) |
| [#2281](https://github.com/ivanarama/onebase/pull/2281) | feat(формы): deadline_warn — поле-срок подсвечивается по времени | Ready | ship, needs-decision, reviewed | — | да, кроме поздних коммитов (1/2) |
| [#2282](https://github.com/ivanarama/onebase/pull/2282) | feat(формы): conceal/reveal_when — значение поля не уходит в браузер, пока форма его не раскроет | Ready | changes-requested | — | да, кроме поздних коммитов (2/11) |
| [#2283](https://github.com/ivanarama/onebase/pull/2283) | feat(формы): tab_skip_readonly и autofocus — Tab по редактируемому, курсор в первое пустое поле | Ready | needs-decision, reviewed | — | да, кроме поздних коммитов (1/6) |
| [#2284](https://github.com/ivanarama/onebase/pull/2284) | feat(формы): ПослеОткрытия — отложенная загрузка данных группы с индикатором, мимо очереди событий | Ready | needs-decision | — | да, кроме поздних коммитов (1/4) |
| [#2285](https://github.com/ivanarama/onebase/pull/2285) | feat(dsl): отправка почты с профилем SMTP, несколькими получателями, HTML и вложениями из данных (поверх #2270) | Draft | — | поверх #2270 | да |
| [#2286](https://github.com/ivanarama/onebase/pull/2286) | feat(табдок): Записать(…, "xlsx") — книга Excel с оформлением макета | Draft | — | после #2285 | да |
| [#2291](https://github.com/ivanarama/onebase/pull/2291) | feat(формы): actions.copy.visible — скрыть «Скопировать» у формы объекта | Draft | — | — | да |
| [#2292](https://github.com/ivanarama/onebase/pull/2292) | feat(формы): group_titles: bold — жирные заголовки групп и таблиц формы | Draft | — | — | да |
| [#2293](https://github.com/ivanarama/onebase/pull/2293) | feat(формы): background и border_color у ПолеВвода — фон и рамка самого ввода | Draft | — | — | да |
| [#2294](https://github.com/ivanarama/onebase/pull/2294) | feat(формы): header_background у ТабличнаяЧасть — цвет шапки таблицы | Draft | — | — | да |
| [#2295](https://github.com/ivanarama/onebase/pull/2295) | fix(формы): закрытие ждёт текущее событие формы вместо отказа | Draft | — | — | да |
| [#2296](https://github.com/ivanarama/onebase/pull/2296) | feat(формы): frame: false у ГруппаФормы — группа без рамки, только раскладка | Draft | — | — | да |
| [#2297](https://github.com/ivanarama/onebase/pull/2297) | feat(формы): header_inline — заголовок формы и строка команд одной строкой | Draft | — | — | да |
| [#2298](https://github.com/ivanarama/onebase/pull/2298) | feat(формы): grow — элемент занимает остаток строки горизонтальной группы | Draft | — | — | да |
| [#2299](https://github.com/ivanarama/onebase/pull/2299) | feat(формы): title_hidden у ТабличнаяЧасть — таблица без заголовка | Draft | — | — | да |
| [#2300](https://github.com/ivanarama/onebase/pull/2300) | feat(формы): background у СтраницыФормы — фон набора страниц и полосы закладок | Draft | — | — | да |
| [#2301](https://github.com/ivanarama/onebase/pull/2301) | feat(формы): placement: header_end и поля в строке команд документа | Draft | — | — | да, кроме поздних коммитов (2/3) |
| [#2302](https://github.com/ivanarama/onebase/pull/2302) | fix(формы): ответ события с вопросом применяется до открытия вопроса | Draft | — | — | да, кроме поздних коммитов (1/2) |
| [#2303](https://github.com/ivanarama/onebase/pull/2303) | feat(dsl): ТекущаяУниверсальнаяДатаВМиллисекундах и ЗаписьЖурналаРегистрации | Draft | — | — | да |
| [#2304](https://github.com/ivanarama/onebase/pull/2304) | feat(формы): Server-Timing у открытия и событий формы (ONEBASE_SERVER_TIMING=1) | Draft | — | — | да |
| [#2305](https://github.com/ivanarama/onebase/pull/2305) | perf(формы): событие формы пересылает варианты ТЧ только для изменившихся ТЧ формы | Draft | — | поверх #2304 | да |
| [#2306](https://github.com/ivanarama/onebase/pull/2306) | perf(формы): открытие формы — списки выбора только для полей и ТЧ на форме | Draft | — | поверх #2305 | да |
| [#2307](https://github.com/ivanarama/onebase/pull/2307) | perf(формы): событие формы не перечитывает ТЧ из базы, если за обработчик никто не писал | Draft | — | поверх #2306 | да |
| [#2308](https://github.com/ivanarama/onebase/pull/2308) | fix(формы): отбор choice_filter не очищает запертое поле и не шлёт ПриИзменении | Draft | — | — | да |
| [#2309](https://github.com/ivanarama/onebase/pull/2309) | perf(формы): закрытие формы не считает варианты выбора, которые отбросит | Draft | — | поверх #2307 | да |
| [#2310](https://github.com/ivanarama/onebase/pull/2310) | feat(формы): choice_dropdown: false — ссылочное поле без выпадающего списка | Draft | — | — | да |
| [#2311](https://github.com/ivanarama/onebase/pull/2311) | fix(уведомления): команды формы самому себе — только в окно-источник | Draft | — | — | да |
| [#2312](https://github.com/ivanarama/onebase/pull/2312) | fix(формы): подпись пустого пункта списка — по текущей доступности поля | Draft | — | — | да |
| [#2313](https://github.com/ivanarama/onebase/pull/2313) | feat(вложения): встроенный проигрыватель аудио-вложений | Draft | — | — | да |
| [#2314](https://github.com/ivanarama/onebase/pull/2314) | fix(формы): fit_window — панель, перенесённая под страницы, не ужимается | Draft | — | — | да, кроме поздних коммитов (5/8) |
| [#2315](https://github.com/ivanarama/onebase/pull/2315) | feat(формы): fit_window у ТабличнаяЧасть — сетка на остаток высоты окна | Draft | — | поверх #2314 | да, кроме поздних коммитов (6/9) |
| [#2316](https://github.com/ivanarama/onebase/pull/2316) | fix(обработки): параметр enum:<Перечисление> — значения и подписи перечисления на форме | Draft | — | — | да |
| [#2332](https://github.com/ivanarama/onebase/pull/2332) | feat(лаунчер): программы, запускаемые вместе с базой (по ключу от сервера) | Draft | — | — | да |
| [#2333](https://github.com/ivanarama/onebase/pull/2333) | feat(формы): Сообщить со СтатусСообщения и цвет текста поля — ошибки красным | Draft | — | — | да, кроме поздних коммитов (1/2) |
| [#2334](https://github.com/ivanarama/onebase/pull/2334) | feat(формы): важность и цвет текста у поля диалога ПоказатьВопрос (поверх #2257) | Ready | reviewed, changes-requested | поверх #2257 | да |
| [#2338](https://github.com/ivanarama/onebase/pull/2338) | fix(доступ): заглушку маски ПДн нельзя записать в объект | Ready | changes-requested | — | да, кроме поздних коммитов (1/3) |
| [#2339](https://github.com/ivanarama/onebase/pull/2339) | fix(формы): форма обработки переносит параметры без элемента между событиями | Ready | changes-requested | — | да, кроме поздних коммитов (1/3) |
| [#2343](https://github.com/ivanarama/onebase/pull/2343) | feat(dsl): Новый УникальныйИдентификатор — случайный v4 и разбор строки | Ready | needs-decision | — | да, кроме поздних коммитов (1/3) |
| [#2351](https://github.com/ivanarama/onebase/pull/2351) | feat(шлюз): dlq.retry_by_sender — сбой обработчика 503, повтор отправителя переигрывает карантин | Ready | changes-requested | — | да |
| [#2352](https://github.com/ivanarama/onebase/pull/2352) | feat(сервисы): hmac_allow_repeated_get — одинаковая подпись справочного GET не считается повтором | Ready | reviewed, changes-requested | — | да |
| [#2353](https://github.com/ivanarama/onebase/pull/2353) | fix(оболочка): ссылки верхней панели, поиск и «Назад» не уводят страницу вкладок | Ready | changes-requested | — | да |
| [#2355](https://github.com/ivanarama/onebase/pull/2355) | fix(ui): без истории ввода браузера во всех текстовых полях — ПДн | Ready | reviewed, changes-requested | — | да |
| [#2356](https://github.com/ivanarama/onebase/pull/2356) | fix(ui): текущая строка таблицы SlickGrid подсвечена постоянно | Ready | changes-requested | — | да |
| [#2357](https://github.com/ivanarama/onebase/pull/2357) | feat(ui): ширина колонок таблицы формы за пользователем | Ready | changes-requested | — | да, кроме поздних коммитов (1/7) |
| [#2364](https://github.com/ivanarama/onebase/pull/2364) | feat(auth): переименование учётки — onebase user rename и поле логина в веб-админке | Ready | changes-requested | — | да |
| [#2365](https://github.com/ivanarama/onebase/pull/2365) | feat(ui): помеченные на удаление не предлагаются в выборе ссылок при вводе | Ready | changes-requested | — | да |
| [#2366](https://github.com/ivanarama/onebase/pull/2366) | feat(ui): оформление строк выбора ссылок — choice_row_style справочника | Ready | changes-requested | — | да |
| [#2367](https://github.com/ivanarama/onebase/pull/2367) | feat(forms): hint_from — всплывающая подсказка элемента из реквизита формы | Ready | changes-requested | — | да |
| [#2397](https://github.com/ivanarama/onebase/pull/2397) | feat(forms): Оповестить и ОбработкаОповещения — оповещение открытых форм, как в 1С | Draft | — | — | да |

**Цепочки зависимостей** (вливать по порядку):
- #2257 → #2334;
- #2273 → #2275;
- #2270 → #2285 → #2286;
- #2304 → #2305 → #2306 → #2307 → #2309;
- #2314 → #2315.

Остальные PR стоят прямо на `main`.

## 4. Что уже влито в `main` (наши PR, с 01.10.2026)

Эти изменения в dev-сборке взяты в версии владельца из `main`, а не из наших
веток.

- [#2266](https://github.com/ivanarama/onebase/pull/2266) 2026-10-27 — fix(списки): поиск и отбор в иерархическом справочнике — по всему справочнику
- [#2337](https://github.com/ivanarama/onebase/pull/2337) 2026-10-27 — feat(формы): условие choice_filter запросом — допустимы ссылки из результата
- [#2354](https://github.com/ivanarama/onebase/pull/2354) 2026-10-27 — feat(ui): закрепление режима открытия форм администратором
- [#2232](https://github.com/ivanarama/onebase/pull/2232) 2026-10-27 — feat(dsl): HTTPСоединение пишет ответ в ИмяВыходногоФайла, как в 1С
- [#2180](https://github.com/ivanarama/onebase/pull/2180) 2026-10-26 — feat(ui): ширина колонки табличной части — width элемента Колонка
- [#2271](https://github.com/ivanarama/onebase/pull/2271) 2026-10-26 — perf(storage): порядок списка по умолчанию идёт по индексу
- [#2170](https://github.com/ivanarama/onebase/pull/2170) 2026-10-25 — fix(ui): закладка СтраницыФормы, скрытая hidden_when после события формы, пропадает сразу
- [#2106](https://github.com/ivanarama/onebase/pull/2106) 2026-10-25 — feat(формы): событие ПередПроведением — вопрос и отказ перед стандартным проведением
- [#2174](https://github.com/ivanarama/onebase/pull/2174) 2026-10-25 — fix(ui): записанная обработчиком дата «сейчас» не делает форму изменённой
- [#1871](https://github.com/ivanarama/onebase/pull/1871) 2026-10-24 — fix(формы): подсказки браузера в полях с маской и в окне Предприятия
- [#2278](https://github.com/ivanarama/onebase/pull/2278) 2026-10-23 — fix(формы): отказ ПриЧтенииНаСервере показывает текст исключения
- [#2272](https://github.com/ivanarama/onebase/pull/2272) 2026-10-23 — fix(dsl): Знач в списке параметров — модификатор, а не имя; запятая обязательна
- [#2269](https://github.com/ivanarama/onebase/pull/2269) 2026-10-23 — fix(админ): таблицы админки не вылезают за карточку
- [#2224](https://github.com/ivanarama/onebase/pull/2224) 2026-10-23 — feat(ui): ui.открытьФорму открывает список с отбором и сформированный отчёт с параметрами
- [#1949](https://github.com/ivanarama/onebase/pull/1949) 2026-10-23 — fix(storage): имена длиннее 63 байт — одно правило сокращения для PostgreSQL и SQLite
- [#2173](https://github.com/ivanarama/onebase/pull/2173) 2026-10-22 — fix(ui): ответ события, открывший подбор, применяет реквизиты шапки
- [#2181](https://github.com/ivanarama/onebase/pull/2181) 2026-10-22 — feat(списки): отбор формы списка запросом и снимаемый отбор по умолчанию (поверх #2172)
- [#2274](https://github.com/ivanarama/onebase/pull/2274) 2026-10-22 — feat(формы): пустое поле с маской ввода показывает её шаблон
- [#2252](https://github.com/ivanarama/onebase/pull/2252) 2026-10-21 — fix(check): предупреждение о нераспознанном типе реквизита (field.type)
- [#2237](https://github.com/ivanarama/onebase/pull/2237) 2026-10-21 — feat(формы): событие Ответ на уровне формы — ответ на вопрос из ПриОткрытии
- [#1915](https://github.com/ivanarama/onebase/pull/1915) 2026-10-21 — feat(формы): кнопка «Открыть карточку» только администратору — ref_card_button_admin_only (#1876)
- [#2133](https://github.com/ivanarama/onebase/pull/2133) 2026-10-21 — feat(ui): смена раздела без перезагрузки в режиме вкладок
- [#2184](https://github.com/ivanarama/onebase/pull/2184) 2026-10-21 — fix(списки): отбор по булеву реквизиту — выбор Да/Нет и точное сравнение
- [#2223](https://github.com/ivanarama/onebase/pull/2223) 2026-10-21 — fix(query): дата присоединённого источника и МИНИМУМ/МАКСИМУМ даты приходят из запроса датой
- [#2189](https://github.com/ivanarama/onebase/pull/2189) 2026-10-20 — fix(sheet): PDF не вычитает поле ячейки второй раз
- [#2187](https://github.com/ivanarama/onebase/pull/2187) 2026-10-20 — perf(ui): варианты ссылок табличных частей — только id и подпись, один блок на форме
- [#2186](https://github.com/ivanarama/onebase/pull/2186) 2026-10-20 — perf(storage): иерархический справочник без отбора папок не сортируется по is_folder
- [#2185](https://github.com/ivanarama/onebase/pull/2185) 2026-10-20 — fix(dsl): Формат числа как в 1С — без ЧДЦ свои знаки, ЧГ=0, округление от нуля
- [#2130](https://github.com/ivanarama/onebase/pull/2130) 2026-10-20 — feat(меню): меню раздела по ролям — nav_by_role в home_page подсистемы
- [#2112](https://github.com/ivanarama/onebase/pull/2112) 2026-10-20 — feat(подбор): выделение строк — Конфиг.ПолеВыделения
- [#2165](https://github.com/ivanarama/onebase/pull/2165) 2026-10-19 — feat(ui): создание из интерфейса — отдельное право create (interactive_create: false)
- [#2213](https://github.com/ivanarama/onebase/pull/2213) 2026-10-19 — fix(auth): страница входа — свой список пользователей вместо datalist, смена имени очищает пароль
- [#2212](https://github.com/ivanarama/onebase/pull/2212) 2026-10-19 — fix(ui): дата в простой таблице (no_grid) — поле даты, а не сырой текст
- [#2151](https://github.com/ivanarama/onebase/pull/2151) 2026-10-18 — fix(вкладки): ui.закрытьФорму закрывает форму нового объекта, записанную в обработчике
- [#2171](https://github.com/ivanarama/onebase/pull/2171) 2026-10-18 — fix(доступ): поле под маской не прячет набираемое точками
- [#1926](https://github.com/ivanarama/onebase/pull/1926) 2026-10-17 — fix(запросы): поля присоединённого регистра переводятся по его метаданным
- [#1924](https://github.com/ivanarama/onebase/pull/1924) 2026-10-17 — fix(формы): ValueTable с заданной высотой прокручивается внутри своего блока
- [#1922](https://github.com/ivanarama/onebase/pull/1922) 2026-10-17 — fix(ui): широкая таблица регистров и журнала прокручивается внутри карточки
- [#1916](https://github.com/ivanarama/onebase/pull/1916) 2026-10-17 — docs(plans): план 158 — поля диалога вопроса (#1877) и сверка со слитыми #1528/#1557
- [#2182](https://github.com/ivanarama/onebase/pull/2182) 2026-10-16 — feat(виджеты): запрос виджета знает, кто смотрит — &ТекущийПользователь и &ТекущийЛогин
- [#2168](https://github.com/ivanarama/onebase/pull/2168) 2026-10-16 — feat(ui): list_filters — основной набор полей панели «Отбор», остальные пользователь добавляет себе
- [#2179](https://github.com/ivanarama/onebase/pull/2179) 2026-10-16 — fix(ui): переключение закладки не гасит вложенный набор СтраницыФормы
- [#2176](https://github.com/ivanarama/onebase/pull/2176) 2026-10-16 — fix(ui): форма обработки с ПолеСписка отрисовывается целиком
- [#2183](https://github.com/ivanarama/onebase/pull/2183) 2026-10-16 — fix(ui): строка виджета в режиме вкладок открывает вкладку оболочки
- [#1914](https://github.com/ivanarama/onebase/pull/1914) 2026-10-15 — fix(ui): hidden_when — поля остаются в DOM, ответ события доходит до всех копий (маршрут #1759)
- [#2167](https://github.com/ivanarama/onebase/pull/2167) 2026-10-15 — feat(ui): порядок колонок таблицы формы — перетаскиванием, за пользователем
- [#2169](https://github.com/ivanarama/onebase/pull/2169) 2026-10-15 — fix(ui): ряд из одних кнопок в горизонтальной группе — без пустой полосы сверху
- [#2161](https://github.com/ivanarama/onebase/pull/2161) 2026-10-14 — feat(регистры): индексы регистра сведений — indexes: в inforegs/*.yaml
- [#2159](https://github.com/ivanarama/onebase/pull/2159) 2026-10-14 — feat(формы): событие ПриСменеСтраницы у набора страниц управляемой формы
- [#2132](https://github.com/ivanarama/onebase/pull/2132) 2026-10-13 — fix(ui): кнопка с hidden_when остаётся в разметке и показывается событием формы
- [#2118](https://github.com/ivanarama/onebase/pull/2118) 2026-10-13 — feat(подбор): поля отбора в шапке окна — Конфиг.ПоляШапки, БезВыбора
- [#2116](https://github.com/ivanarama/onebase/pull/2116) 2026-10-13 — feat(подбор): ширина окна — Конфиг.Ширина
- [#2115](https://github.com/ivanarama/onebase/pull/2115) 2026-10-12 — feat(отчёты): параметр-дата «по» — конец дня (period_end)
- [#1950](https://github.com/ivanarama/onebase/pull/1950) 2026-10-12 — fix(backup): пустая строка в нетекстовой колонке при загрузке в PostgreSQL — NULL
- [#1976](https://github.com/ivanarama/onebase/pull/1976) 2026-10-12 — fix(регистры): период со временем и секундами — в списках, движениях и форме записи
- [#2111](https://github.com/ivanarama/onebase/pull/2111) 2026-10-12 — feat(формы): «ОК» проводимого документа проводит — actions.ok.post
- [#2109](https://github.com/ivanarama/onebase/pull/2109) 2026-10-12 — feat(формы): фон кнопки управляемой формы — ключ background
- [#1917](https://github.com/ivanarama/onebase/pull/1917) 2026-10-11 — feat(формы): choice_filter по колонке табличной части выбираемого справочника (#1822)
- [#1955](https://github.com/ivanarama/onebase/pull/1955) 2026-10-09 — feat(формы): подбор — двойной щелчок и Enter в одиночном выборе, отборы по колонкам
- [#1945](https://github.com/ivanarama/onebase/pull/1945) 2026-10-09 — feat(dsl): РасшифроватьСекрет — enc:-секрет из данных базы
- [#1944](https://github.com/ivanarama/onebase/pull/1944) 2026-10-09 — fix(доступ): страница констант — только администратору
- [#1923](https://github.com/ivanarama/onebase/pull/1923) 2026-10-08 — fix(доступ): поле под маской — заполнить пустое, набранное не превращается в звёздочки
- [#1918](https://github.com/ivanarama/onebase/pull/1918) 2026-10-07 — fix(формы): нет 🔍 у reference:_users, вкладка с 404 закрывается (#1684)
- [#1872](https://github.com/ivanarama/onebase/pull/1872) 2026-10-04 — feat(формы): оператор not_in_hierarchy в choice_filter — «не в группе» (#1821)
- [#1786](https://github.com/ivanarama/onebase/pull/1786) 2026-10-03 — fix(query): разыменование до ссылочного реквизита берёт колонку _id
- [#1856](https://github.com/ivanarama/onebase/pull/1856) 2026-10-03 — feat(forms): choice_filter — постоянная ссылка ref как третий источник (#1820)
- [#1834](https://github.com/ivanarama/onebase/pull/1834) 2026-10-02 — feat(формы): parent_id в choice_filter — отбор внутри группы иерархии
- [#1846](https://github.com/ivanarama/onebase/pull/1846) 2026-10-02 — fix(формы): choice_filter в формах обработок и в onebase check
- [#1813](https://github.com/ivanarama/onebase/pull/1813) 2026-10-02 — feat(формы): строковый конец глубокого пути в choice_filter
- [#1809](https://github.com/ivanarama/onebase/pull/1809) 2026-10-01 — fix(ui): дата в колонке list-виджета — в местном времени
- [#1781](https://github.com/ivanarama/onebase/pull/1781) 2026-10-01 — feat(формы): оператор eq_or_empty в choice_filter — «равно или пусто»

## 5. Сборка и выкладка

### Сборка

```bash
# CLI + сервер (без CGo) — основной бинарь
go build -o onebase.exe ./cmd/onebase

# GUI с нативным окном (CGo + WebView2 на Windows)
go build -tags webview -ldflags="-H windowsgui" -o onebase-gui.exe ./cmd/onebase

# Linux-бинарь для сервера КЦ (как собирается dev-сборка)
GOOS=linux GOARCH=amd64 CGO_ENABLED=0 go build \
  -ldflags="-s -w -X github.com/ivantit66/onebase/internal/version.Build=dev-$(git rev-parse --short=7 HEAD)" \
  -o onebase-cc2-dev ./cmd/onebase
```

На Windows запущенный `onebase.exe` держит файл, и `go build` молча не
обновит бинарь. Перед пересборкой остановите сервер. При нехватке места на
диске сборка падает с «not enough space» — помогает `go clean -cache`.

### Выкладка на сервер КЦ

Адрес сервера, учётную запись и SSH-доступ выдаёт администратор. В
репозитории и в этом документе их нет.

На сервере две базы:
- **разработки** (стенд, порт 8444) — сюда идёт всё новое;
- **оператора** (рабочая, порт 8443) — только выпуском.

Порядок:
1. Бинарь кладётся рядом с рабочим как `bin/onebase-cc2-dev.new-<суффикс>`.
   Суффикс — дата и буква, например `1028a`.
2. На нём выполняется `onebase check --project <конфигурация разработки>`.
3. Замену бинаря на стенде, перезапуск и прогон сценариев делает
   ответственный за конфигурацию cc2. Скрипт выкладки лежит в репозитории
   конфигурации, `инструменты/выложить.sh`. Прежний бинарь сохраняется
   рядом как `.bak`.
4. Рабочая база оператора обновляется **только по решению заказчика**, одним
   пакетом и всегда вместе с конфигурацией. Порядок такой:
   - полный прогон сценариев на SQLite и на копии PostgreSQL;
   - копия базы до остановки службы;
   - закреплённая копия бинаря выпуска со сверкой sha;
   - дымовые проверки после выкладки.
5. Пока на стенде идёт полный прогон сценариев, выкладывать туда и
   перезапускать его нельзя: сценарии падают пачкой, и это не регрессия.

### Еженедельная пересборка от `main`

Сливать `main` в старую интеграционную ветку нельзя: на 21.10 отставание
было больше тысячи коммитов и десятки файлов в конфликте. Раз в неделю или
после пачки мержей наших PR:

1. Новая ветка от свежего `origin/main`.
2. Перенос только наших коммитов, которых в `main` нет, — открытые PR.
   Уже влитые PR — в версии владельца из `main`.
3. Перед пересборкой сверка «коммиты старой сборки ↔ main/PR», чтобы ни одна
   правка не потерялась. Так однажды потерялся учёт `actions.copy.visible`.
4. Сборка → стенд → прогон всех сценариев cc2 на SQLite и PostgreSQL →
   выпуск оператору по решению заказчика.

Последняя пересборка была 22.10 от `main` на 21.10, следующая уже назрела.
Ожидаемые конфликты при переносе:
- `internal/ui/handlers_events.go` — #2311 и #2397 оба меняют `hubNotifier`;
- `README.md`, `QUICKSTART.md` — счётчики функций;
- `docs/features.md`, `docs/forms.md`;
- `docs/dsl-reference.md` — перегенерировать командой
  `onebase langref --output docs/dsl-reference.md`.

## 6. Конвейер владельца upstream

Подробно — `CLAUDE.md` (раздел «Соглашения»), `CONTRIBUTING.md`,
`docs/maintenance-pipeline.md`. Коротко:

- **Маршрут контрибутора.** Ветка `feature/*` или `fix/*` от свежего `main`
  → **Draft PR** → проверка через публичную точку входа (тест идёт тем же
  путём, что пользователь) + документация → подмешать `main`, зелёный CI →
  отметить в описании «Поставка» и перевести в **Ready**. Один
  пользовательский результат на PR.
- **Режим работы в описании PR.** Галочка «Эксперимент» (остаётся Draft) или
  «Поставка» (HEAD готов к конвейеру). 27.10 по просьбе владельца 16 PR
  переведены в Ready с «Поставкой». Ответ владельцу со сводкой по всем
  Draft — [комментарий в #2357](https://github.com/ivanarama/onebase/pull/2357#issuecomment-6442378567).
- **Метки:**
  - `reviewed` / `changes-requested` — итог ревью, на правки две попытки;
  - `needs-decision` — нужна развилка владельца;
  - `ship` ставит только человек, без неё MERGE не вливает;
  - `hold` — пауза.
- **Закрытие заявок.** В теле PR — английское `Fixes #N` (`Closes` /
  `Resolves`): русское «Закрывает» GitHub не понимает.
- **Коммиты** — `тип(scope): описание` по-русски, трейлер
  `Generated-with: Claude Code`, если правку готовил ИИ-ассистент (не
  `Co-Authored-By`).
- **Обязательные проверки `main`:** `build`, `lint`, `postgres-integration`,
  `vuln`, `smoke`, `e2e`, `test-windows`, `launcher-webview-build`;
  `strict` — перед мержем подмешать свежий `main`. `bench` красным мерж не
  блокирует.
- **Чужие PR не трогать,** в ветку PR, уже вошедшего в REVIEW/FIX, не
  пушить без договорённости; ничего не форс-пушить поверх чужих веток.
- Нумерованные планы фич — `Plans/`; новые пользовательские возможности
  заносятся в `docs/features.md`.

## 7. Открытые задачи платформы

1. **#2397 `Оповестить` / `ОбработкаОповещения`** (28.10, Draft) — для
   кнопки «Вернуться в звонок».
   - Обработка поиска заявок оповещает открытую форму звонка, и та очищает
     три поля на своём несохранённом состоянии, без записи и без
     `ПередЗакрытием`.
   - Сборка 1028a передана на стенд. В конфигурации cc2 кнопку и форму
     звонка переводит ответственный за конфигурацию; затем сценарий.
   - Дальше по PR: дождаться CI, Ready.
2. **19 PR с `changes-requested`** — замечания владельца ждут правок (список
   в таблице). Сначала цепочки (#2270, #2273, #2257) и PR, нужные в рабочей
   базе.
3. **Draft-эксперименты (28 шт., #2285–#2316):**
   - обновить ветку от `main`, зелёный CI, отметить «Поставку» — по одному;
   - у многих красный `build` из-за тайм-аута пакета `internal/ui` под
     `-race`, конкретных падений нет;
   - адресные хвосты `lint`: #2311 (QF1001), #2285 (errcheck/gosec в
     тесте);
   - у #2279 был конфликт с `main`.
4. **Доделать до Ready:** #2333 — `lint` ST1018 (символы форматирования
   Unicode в строках тестов); #2397 — ждёт CI.
   **#2332 (лаунчер, программы вместе с базой):** формат
   `companions.yaml` (`env`) может ещё меняться под новый сервер телефонии.
5. **Не сделано, но известно:**
   - ширина колонок в простых таблицах `no_grid` (#2357 их не покрывает);
   - фильтры виджетов рабочего стола и помеченные на удаление (#2365) —
     решение не принято;
   - пометка удаления элемента справочника из DSL не поддерживается:
     конфигурация ставила её прямым SQL, нужен отдельный PR;
   - еженедельная пересборка от `main` (раздел 5).

## 8. Куда смотреть

- `CLAUDE.md` — сборка, тесты, архитектура, соглашения, конвейер.
- `CONTRIBUTING.md` — маршрут работы и оформление PR.
- `docs/maintenance-pipeline.md` — этапы и метки конвейера.
- `Plans/` — планы фич; `DEVELOPER.md` — форматы объектов конфигурации.
- `docs/forms.md`, `docs/dsl-reference.md`, `onebase ai-guide` — справочники
  для правок форм и языка.
- Проверка конфигурации — `onebase check --project <dir>`: компилирует и
  выполняет запросы модулей, ловит ошибки до рантайма.
