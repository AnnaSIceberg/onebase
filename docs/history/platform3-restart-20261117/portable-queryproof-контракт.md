# ENG12 → portable strict query proof: exact API и предел публикации

Автор/owner SQL lineage platform3. Portable candidate e1feede5be3c761f7ae89cc6e30390d942cc8df0, parent independently accepted codec v2 2ab7c1ebf21e49f008c9585070b6dc0eadb6f0d2 (base cb2e0acee). Fork AnnaSIceberg/onebase, branch feature/platform3-report-query-proof-20261116, push и ls-remote exact подтверждены. Работа остаётся source-only под существующим Draft2641, не Ready. Отдельной полной копии WT не создано: использован приватный Git index, canonical/active indexes и файлы не менялись. Go/build/test/compile/link/bin/SDK/DB/native0, single allocation lane ENG31 не использовалась.

ВАЖНЫЙ ТЕКУЩИЙ LIMIT: independently accepted portable SQL-proof exact пока НЕ выдан. Новый e1 — кандидат на bounded repeat. Ранее CODE ONLY приняты constituents foundation970, codec2ab и v2wire421; эти ACK автоматически не превращаются в ACK нового base/carry e1. До repeat не считать его accepted dependency/runtime permission. Fullprivacy421 и scopebd9 переносить для этого API не нужно.

Срез минимальный: 10files +1471/-0 поверх2ab. internal/query/report_lineage.go побайтно равен970 (Git blob2634d85d8dccf9398fc1541f5abb4c664ffb48da); report_lineage_digest.go побайтно равен421 wrapper (c36bebad62e4052f9674808660eb57423cc22b6f). Digest/reserved/group-review regression tests взяты exact970 без изменений; один matrix adapter заменяет недоступный на base main access.ReportOutputQueryMaskPlan(nil,...) на существующий access.QueryMaskPlanFor(&auth.User{},...) + importauth/comment. Эта проверка теперь явно про ordinary role masking, НЕ про mandatory fullprivacy. query.go/projection.go/metadata/access/UI/API/runtime/rowsource не изменены. Parent2ab содержит общий provenance package, не новый codec. git diff --check PASS. Нового compile/PASS не заявляю: Go lane закрыта; после END требуется назначенное окно и freshfree>=2GiB.

## Точный server API, package internal/query

```go
func CompileReportLineage(src string, opts CompileOpts) (Result, *ReportLineage, error)
func (p *ReportLineage) Outputs() []ReportLineageOutput
func (p *ReportLineage) Uses() []ReportLineageUse
func (p *ReportLineage) Matches(res Result, opts CompileOpts) bool

type ReportFieldOrigin struct {
    Occurrence int
    Source SourceRef
    Field string
    Type metadata.FieldType
    Path []string
}
type ReportExpression struct {
    Op string
    Type metadata.FieldType
    Parameter string
    LiteralDigest [32]byte
    Origins []ReportFieldOrigin
    Args []ReportExpression
}
type ReportLineageOutput struct { Name string; Expression ReportExpression }
type ReportLineageUse struct { Clause string; Expression ReportExpression }
```

Это strict foundation970: один SELECT с physical JOIN и доказанными выражениями. В этом minimum нет поля Uses.Scope и нет derived/UNION/VT/FK-predicate/calendar extensions. Unknown source/type/function/field, ambiguity, reserved output aliases, bare reference presentation и unsupported syntax STOP без partial Result/receipt. Literal1 описывается как typed expression, но это само по себе не DSL admission. Native DateDiff/A004/A005 originals и KPI VT этот минимум не решает. Поздние accepted FK/calendarcf не скрыто включены в e1, их возможный portable carry будет отдельной дельтой.

Outputs/Uses — detached inspection. Проверять ВСЕ Origins, включая protected owning/root FK path, не только terminal id/тип UUID. ВСЕ Uses являются control/membership dependencies ВСЕХ выходов, в т.ч. COUNT. Выбранная колонка/alias/type/равное значение/inspection JSON не authority. SQL/args, literal fingerprints и inspection не передавать клиенту/preset/snapshot как доказательство.

Matches — только integrity typed SQL/args/schema/source metadata/Params/RowFilters. Использует codec FormatVersion2/domainv2; старые fingerprints invalid/recompute. Это не proof actual Query.Text staticness, host accepted user/default/context kind, current actor/session/roles/read/RLS, registry/module generation, DB snapshot, successful rows/producer seal или permission. Host должен получить свежий авторизованный контекст, сам сформировать/проверить object/field/RLS/output boundary для всех физических и навигационных источников и связать proof с текущим execution frame. SQL receipt не переписывает ваш lexical/runtime Query.Text ledger, missing controls или user-input как fixed-literal.

ENG12 frame/userkind/textstaticness остаются отдельными host proofs. Masked native originals + minimal DSL subset, successful producer/final filter schema/effectiveFilters/exports/presets/background/snapshot/admission этим API не включаются. Нет generic all-native-result allow, PII/native passthrough или новых исключений для Addr/bareRef. Full ENG06/ENG16 compatibility/runtime/native/SDK HOLD OPEN.

Следующий review: bounded e1 portable assembly/dependency cut + one test adapter; не повторять970/701/scope/codec/cf. До новой Go lane возможен code/Git blob review, затем tiny compile/regression по назначенному окну. Автор не reviewer. cc2 маршрутизирует очередь.

Generated-with: Codex
## Независимый bounded ACK нового portable exact

Review принял 2ab..e1feede5be3c761f7ae89cc6e30390d942cc8df0 как static assembly CODE ONLY ACK. Receipt scratchpad/perenos-codex/отчёты/review-platform3-ENG12-portable-e1feede5-20261116.md: пять carry blobs exact970/421, codec и protected8trees unchanged, единственный ordinary-role testadapter прочитан. Это заменяет прежний статус НЕACK только в указанной bounded границе. Compile/Go/suites0, новых PASS на e1 нет. Constituent scopes не переаудированы. Virtual/infoslice WIP исключён; full ENG06/16/12 compatibility/runtime/native/SDK HOLD OPEN. Tinycompile потребует отдельного plan/окна perenos и freshfree; никаких новых allocations из-за этого ACK не запускается. Actor/session/dialect/timezone/current snapshot/registry/module/staticness/user-kind по-прежнему обязан связать host frame, Matches этого не заменяет.
