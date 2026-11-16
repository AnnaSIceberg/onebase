$ErrorActionPreference = 'Stop'
[Console]::OutputEncoding = New-Object Text.UTF8Encoding($false)
$OutputEncoding = [Console]::OutputEncoding
$base = '2ab7c1ebf21e49f008c9585070b6dc0eadb6f0d2'
$source = '9700083bd451d29564708785cfd42fd5dfb6b451'
$wire = '421440ff5a5d4335416bacfb0d42f8edaec0446b'
$branch = 'refs/heads/feature/platform3-report-query-proof-20261116'
$area = 'C:/Users/a.sevrjukova/AppData/Local/Temp/platform3-engine-probes-20261116'
$env:GIT_INDEX_FILE = $area + '/portable-queryproof-' + [Guid]::NewGuid().ToString('N') + '.index'
function GitOK { if ($LASTEXITCODE -ne 0) { throw 'git command failed' } }
function ContentAt($revision, $path) {
    $lines = @(git show ($revision + ':' + $path)); GitOK
    return ($lines -join "`n") + "`n"
}
function StageText($path, $text) {
    $file = $area + '/portable-queryproof-' + ($path.Replace('/','_'))
    [IO.File]::WriteAllText($file, $text, (New-Object Text.UTF8Encoding($false)))
    $blob = git hash-object -w -- $file; GitOK
    git update-index --add --cacheinfo ('100644,' + $blob + ',' + $path); GitOK
}
git show-ref --verify --quiet $branch
if ($LASTEXITCODE -eq 0) { throw 'portable branch already exists' }
git read-tree $base; GitOK
$paths = @(git ls-tree -r --name-only $source -- internal/query | Where-Object { $_.StartsWith('internal/query/report_lineage') -and $_.EndsWith('.go') }); GitOK
foreach ($path in $paths) {
    $revision = $source
    if ($path -eq 'internal/query/report_lineage_digest.go') { $revision = $wire }
    if ($path -eq 'internal/query/report_lineage_matrix_test.go') {
        $text = ContentAt $revision $path
        if (-not $text.Contains('access.ReportOutputQueryMaskPlan(nil, res,')) { throw 'unexpected test adapter source' }
        $text = $text.Replace('access.ReportOutputQueryMaskPlan(nil, res,', 'access.QueryMaskPlanFor(&auth.User{}, res,')
        $text = $text.Replace('"github.com/ivantit66/onebase/internal/access"', '"github.com/ivantit66/onebase/internal/access"' + "`n`t" + '"github.com/ivantit66/onebase/internal/auth"')
        $text = $text.Replace('// A receipt proves syntax/provenance only. It cannot relax the mandatory' + "`n`t`t" + '// output policy by itself; the separate ENG16 authorization is not wired.', '// A receipt grants no rights. Ordinary role masking still rejects this' + "`n`t`t" + '// protected comparison; mandatory reporting policy belongs to the host.')
        StageText $path $text
    } else {
        $blob = git rev-parse ($revision + ':' + $path); GitOK
        git update-index --add --cacheinfo ('100644,' + $blob + ',' + $path); GitOK
    }
}
$doc = @'
# Portable strict SQL provenance profile

Source API (package internal/query):

```go
func CompileReportLineage(src string, opts CompileOpts) (Result, *ReportLineage, error)
func (p *ReportLineage) Outputs() []ReportLineageOutput
func (p *ReportLineage) Uses() []ReportLineageUse
func (p *ReportLineage) Matches(res Result, opts CompileOpts) bool
```

This portable slice contains the strict single-SELECT foundation only. Its parser
is identical to accepted source970; the digest wrapper is identical to wire421
and uses provenance.FormatVersion2, supplied by the parent portable2ab commit.
It does not carry report privacy policy, handlers, report context, derived/UNION
scopes, FK predicate extensions, calendar extensions, virtual tables or runtime
admission. Ordinary Compile/query.go and metadata definitions are unchanged.

Outputs returns detached expression trees with Op/Type/Parameter/LiteralDigest,
Origins and Args. Each origin retains SourceRef, FROM occurrence, canonical field,
type and every reference hop in Path. Uses returns Clause+Expression for all
JOIN/WHERE/GROUP/HAVING/ORDER dependencies. This strict version has no Scope field.
Every Use is a control/membership dependency of every output, including COUNT.
Inspecting Outputs alone cannot authorize an aggregate or result filter.

Known physical sources/JOINs, scalar expressions, CASE, limited aggregates/string
functions and literals are represented. Unsupported syntax/types, ambiguous
sources/fields/aliases, bare reference presentations, reserved output aliases,
derived/UNION/virtual sources and unknown functions return no partial receipt.
This version does not solve native DateDiff, A004/A005 passthrough or KPI VT inputs.

Matches checks typed SQL/args/projection/source metadata/params/RLS fingerprints.
Old-format receipts must be invalidated and recomputed with FormatVersion2.
Matches does not prove actor authorization, current session/roles, snapshot,
registry/module generation, trusted Query.Text origin, parameter origin kind or
permission to read/declassify protected data. A user-input value equal to a literal
remains user-input. Host frames and runtime data/control taint are separate proofs.

These APIs are server inspection only. Result SQL/args, literal fingerprints and
inspection JSON must not become client/preset/snapshot evidence. Keep the immutable
receipt private; detached views and zero/deserialized handles cannot mint authority.
No execution/SQL reads, mask bypass, final filter schema or DSL variable is enabled.
The host must still enforce object/field/RLS/output policy before any execution and
bind every input, source, operation and output to a successful host trace.

Publication is a portable candidate: independently accepted constituent blobs do
not automatically accept this new base/carry exact. One test adapter uses ordinary
role masking available on base main instead of importing full report privacy policy.
New compile/test runs wait for the assigned Go allocation lane and fresh disk barrier.
ENG06/full ENG16 runtime/native/SDK compatibility remains HOLD.
'@
StageText 'docs/report-query-lineage.md' ($doc + "`n")
StageText '.gitignore' ((ContentAt $base '.gitignore') + "`n!docs/report-query-lineage.md`n")
$changelog = ContentAt $base 'CHANGELOG.md'
$changelog = $changelog.Replace('Изменения после v0.14.0 будут перечисляться здесь.', 'Изменения после v0.14.0 будут перечисляться здесь.' + "`n`n" + 'Добавлен переносимый strict SQL provenance API для одного SELECT: сохраняет происхождение выходов и условий, но не разрешает чтение ПДн и не подключён к runtime отчётов. Неизвестные/неподдержанные источники возвращают STOP без partial receipt; fingerprints используют codec FormatVersion2.')
StageText 'CHANGELOG.md' $changelog
$features = ContentAt $base 'docs/features.md'
$features += "`n## SQL provenance: portable strict source API`n<!-- status: testing -->`n<!-- date: 2026-11-16 -->`n`nCompileReportLineage/Outputs/Uses/Matches дают серверу проверяемое происхождение`nодного SELECT с JOIN. Это source foundation без runtime/allow и без изменения`nобычных запросов/карточек. Контракт и STOP-границы: docs/report-query-lineage.md.`n"
StageText 'docs/features.md' $features
git diff --cached $base --check; GitOK
$tree = git write-tree; GitOK
$message = 'feat(query): переносимый strict SQL provenance profile' + "`n`n" + 'Source-foundation: ' + $source + "`n" + 'Source-codec-wire: ' + $wire + "`n" + 'Source-codec: ' + $base + "`n`n" + 'Generated-with: Codex'
$commit = $message | git commit-tree $tree -p $base; GitOK
git update-ref $branch $commit ('0' * 40); GitOK
Write-Output ('PORTABLE_COMMIT=' + $commit)
git show --stat --oneline $commit; GitOK
