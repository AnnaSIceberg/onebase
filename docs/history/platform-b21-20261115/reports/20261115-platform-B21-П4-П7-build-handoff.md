# B21: П4 + П7, DEV build handoff

Runtime/source integration tree 0d77e91557b12bada4668fd7bfa5f1b13a37de08, exactB20 parent e6cb9a375b0bae15a430ae93b0913c02938bb45e; upstream inputs П4cf5cf100 / П702c62ecd / B20 source d232 отдельно. 33changedfiles, branch pushed. Механические .go конфликты решены с сохранением B20 choice_show_deleted/P2/P3/P1; build./.../i18nen PASS. Native DB UUID user_card проверен с реальным reader и типом uuid.UUID, через UI/API/exports, доqueryless/query result_proc. П4 адаптирован к detail_cells/rowWithDetail; flat12+row actions, настоящие поля/ссылки/guards не исключались.

Local383leafPASS,14localPGrootSKIP отдельно. Independent PG по точным API12/UI166 именам пока PENDING; sourcePG18/14 не подмена. Первый FAIL только новых test adapters сохранён; конечные pinned gates GREEN. Native Chrome ещё НЕ запускался.

Linux SDK 881d78ad9d6d37916350b6bca099a8e7b9208f14a35b8f742f7e3776c92f300a (C:\Users\a.sevrjukova\AppData\Local\Temp\platform-help-admin-20261111\onebase-1115b21-report-context-drilldown-linux-amd64); Windows SDK e9bbbce60ce5b67e1867c0087eda77897b8e6ec540638bd8164814127bf6eb06 (C:\Users\a.sevrjukova\AppData\Local\Temp\platform-help-admin-20261111\onebase-1115b21-report-context-drilldown-windows-amd64.exe). Оба Go1.26.11/CGO0, serialbuild. API prebuilt 8f3025dc0df9e43923761fda9f8c995399ce0745d302b0b2daacaa402b728bc9, UI prebuilt e07da9bd2419585c4c94b4303b49e5ca1632570db5322d375b93f7b80da74f66; expectednames/regex в manifest.

Manifest C:\Users\a.sevrjukova\AppData\Local\Temp\platform-help-admin-20261111\manifest-1115b21-report-context-drilldown-DEV.json SHA 2de91957641e98920f0b19a1da7007b3521e3dc97858e6634c2988e326b2a5bb. Explicit GO END count0; ждём отдельную native очередь cc2. FrozenB20 SHA проверены, shared :8444/B18 не менялся; DBcopies0. Выпуск только prepare, execution по её вечернему слову. Source Draft2629/2630 остаются Draft, fullCI/production readiness не объявлены.

Generated-with: Codex
