# ЛПЦ PLAN v2: summary-only и ledger — статус для perenos-codex / cc2

**В части platform это по-прежнему SOURCE GAP PLAN, не реализованный server ledger-gate. Ни B26, ни B29 я не выпускаю как имеющий такую защиту. Готовую реализацию и review/runtime-квитанции ledger предъявить не могу: этот этап не был реализован.** В B29+ ledger также не добавлялся; текущие B28/B29 меняют отчёты/XLSX/enum, а не исторический импорт.

## Что действительно существует

1. Предложение summary-only профиля: `C:\Users\a.sevrjukova\AppData\Local\Temp\platform-help-admin-20261111\lpc-historical-ledger-PLAN-20261114\01-source-summary-profile-PLAN.md`; SHA256 `a4d196b0f51e0907a0699308f4e8ecdbc6322f8607a37736510a1fca2449f606`. В первой строке статуса прямо PLAN/SOURCE-ONLY, «не реализация CC_V2». Это требования к coverage/counts/identity/completeness, не финальная wire-схема: нет утверждённых mode/полей запроса-ответа/численных budgets/реализованного CC_V2 SHA. Использовать как реализованный wire нельзя. Окончательный summary-only wire должен назвать и подтвердить cc2; новым ответом cc2 этот план не подменяю.
2. Неисполняемый ledger binding: `C:\Users\a.sevrjukova\AppData\Local\Temp\platform-help-admin-20261111\lpc-historical-ledger-PLAN-20261114\02-ledger-binding.PROPOSAL.json`; SHA256 `1a7d41ddeeb23e9e704e39de9b1f5aa6cda20aafcbec7327d479d279ecb7ebef`. `executable=false`, `existing_native_contract=false`, `runtime_witness=false`, реальные bindings не заполнены. DDL/admission/authority ещё проектные.
3. Источник анализа26 engine blobs: `C:\Users\a.sevrjukova\AppData\Local\Temp\platform-help-admin-20261111\lpc-historical-ledger-PLAN-20261114\03-SOURCES.json`; SHA256 `1d7b4a9ff1e7f728352075b6a0234b5f56f0f8380d18f181e27acbd2656986c9`, pin `f49a9e0bfd2eb5184ce9e13e583a5a163e967534`.
4. Полный source-gap план: `C:\Users\a.sevrjukova\AppData\Local\Temp\claude\C--Users-a-sevrjukova-onebase\cd50d987-9739-4c93-a5be-68d748529225\scratchpad\отчёты-для-analysis\20261114-2035-platform-ЛПЦ-historical-ledger-SOURCE-GAP-PLAN.md`; SHA256 `836333f0ce4c55934c5eb2a56438904f7cbea1957ed52fbbf7ba63a6e84d98cd`.
5. Manifest этих4 artifacts: `C:\Users\a.sevrjukova\AppData\Local\Temp\platform-help-admin-20261111\lpc-historical-ledger-PLAN-20261114\04-PLAN-MANIFEST.json`; SHA256 `ab84e13505c99710d75f088b07214266468416ef4b6ba87b83db38471850f7c1`. Сейчас повторно проверены bytes/SHA всех4, без HTTP/БД/экспорта.

## Что принято review

- PLAN v2 perenos: `perenos-codex/отчёты/20261114-ЛПЦ-PLAN-исходные-движения-v2.md`, SHA256 `bd1d1915803fbc9287dddfa9d698cdfa06ca42463e3e7217bf7848e106611397`.
- Принятие cc2: `C:\Users\a.sevrjukova\AppData\Local\Temp\claude\C--Users-a-sevrjukova-onebase\cd50d987-9739-4c93-a5be-68d748529225\scratchpad\отчёты-для-analysis\20261114-cc2-ЛПЦ-PLAN-v2-ПРИНЯТ-и-native-gaps.md`, SHA256 `57ee48f42334db3462ab3fe0a4178eb5cfd373471829d04310374e2a9ca07098`. Там явно: согласован PLAN; следующий результат cc2 — SOURCE summary-only wire; ledger DDL/installation/admission/manifest authority не готовая реализация, platform имел только PLAN/source scope.
- Независимое review: `C:\Users\a.sevrjukova\AppData\Local\Temp\claude\C--Users-a-sevrjukova-onebase\cd50d987-9739-4c93-a5be-68d748529225\scratchpad\отчёты-для-analysis\20261114-2036-review-ЛПЦ-PLAN-v2-движения.md`, SHA256 `785bffa0e0bfe9f7d54d562143f6a56f5dcbd54838f9eafaf4c7bc97cf1901b4`. Защита регистратора закрыта **как проектное требование**, прямо «реализации нет»; native apply остаётся блокирован этим пробелом.
- Root audit PLAN v2: `C:\Users\a.sevrjukova\AppData\Local\Temp\cc2-lpc-movement-plan-v2-root-audit.json`, SHA256 `9634be64ffb92d49a29081f707c2d7dac8c247c15c66287f1ba956dfb57857eb`. Это квитанция проверки плана/metadata/mapping, не исполнения ledger и не release gate.

## Привязка к B26 / B29

- B26 runtime `513274b7e1bc230d8c0de5fcd22a34deb672345c` (`fix/side-panel-full-height`), Linux SDK SHA256 `e977a632cfefc6c1b34575b4ac7797aeb1c057090738f470243fb03b48b67e6d`. Его handoff `20261116-1051-выпуск-claude-полоса-B26-на-8444.md` описывает B25+высоту полосы, не ledger. Историческая установка на8444 не меняет нового правила выкладки.
- B29 runtime `902e4d97e50e90f047ec6d27832e2f9d175d095d`, branch `integration/cc2-b29-report-enum-20261116`; его SDK/native38/PG312 handoff — `20261116-1403-platform-для-cc2-enum-B29-native38-SDK.md`. Ledger туда не переносился.
- Новая RO-квитанция `C:\Users\a.sevrjukova\AppData\Local\Temp\platform-help-admin-20261111\lpc-ledger-status-readonly-20261116.json`, SHA256 `d6cc79dbe9ed853acc938b2bd310e5afef15ae57626d86c2a1bb3d6c8a894a0e`: сопоставлены **Git bytes** всех26 source anchors f49 с exact B26 и B29. Совпадают B26:23/26, B29:23/26; изменённые файлы перечислены отдельно в JSON. Вывод не строится на поиске имён ledger/wire в индексе.

| Критический source путь | B26 относительно source-gap f49 | B29 |
|---|---|---|
| `internal/storage/register.go` | тот же blob | тот же blob |
| `internal/storage/movement_cleanup.go` | тот же blob | тот же blob |
| `internal/storage/crud.go` | тот же blob | тот же blob |
| `internal/entityservice/service.go` | тот же blob | тот же blob |
| `internal/ui/dsl_documents.go` | тот же blob | тот же blob |
| `internal/exchange/package.go` | тот же blob | тот же blob |

## Что ещё нужно и когда

Последний принятый этап14.11 — PLAN. Дальше: cc2 выдаёт точный readonly summary-only wire/source commit и его review; по подтверждённому profile выбирается native scope/модель. Затем отдельная реализация server gate до persistence hooks/первой записи, сериализация ordinary/import/release, storage backstops и публичная SQLite+PG матрица всех UI/REST/DSL/exchange/repair путей, включая admin. Нужны точные executable manifest/undo/CAS и отдельная runtime-квитанция. Этих реализаций/проверок сейчас у platform нет; срок не назначен, в текущей реализации отчётного UI он не скрыт. Perenos не должен считать PLAN уже доступным API или готовым B26 gate.

Новых тяжёлых действий по этому запросу нет: только точные локальные документы/Git blobs и SHA. БД/1С/HTTP/SSH/выкладки не выполнялись. Для общих стендов действует её правило: основной ЛПЦ :8448, только её слово и один накопленный пакет; B29 на8444 не ставить.

Generated-with: Codex
