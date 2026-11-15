$ErrorActionPreference = 'Stop'
$taskRoot = 'C:\Users\a.sevrjukova\AppData\Local\Temp\platform-help-admin-20261111'
$taskProtected = @('onebase-1115b18','onebase-1115b19','reports-kpi-p1-b18','reports-kpi-p1-b19','onebase-1113b7','onebase-1114b8')
$taskCandidates = @(Get-ChildItem -LiteralPath $taskRoot -File | Where-Object {
    $_.Name -match '^onebase-111[345]b1[0-7]-' -or
    $_.Name -match '^reports-.*-b1[67]-.*\.test$' -or
    $_.Name -in @('audio-dirty-stage2-ff0518fd-linux-amd64.test','audio-diag-f49-stage1-linux.test','current-reference-b15-linux-amd64.test','reference-dirty-b14-linux-amd64.test','form-dirty-defaults-b13-linux-amd64.test','audio-viewstate-1114b10-final-ui-linux.test')
})
# The other named directories contain old source worktrees and receipts.
# Remove only untracked executable outputs there, preserving all source/history.
foreach ($taskExtra in @('C:\Users\a.sevrjukova\AppData\Local\Temp\onebase-2558-codex','C:\Users\a.sevrjukova\AppData\Local\Temp\platform-result-form-20261111')) {
    if (-not (Test-Path -LiteralPath $taskExtra)) { continue }
    foreach ($taskFile in @(Get-ChildItem -LiteralPath $taskExtra -Recurse -File | Where-Object { $_.Length -gt 8000000 -and $_.FullName -notmatch '[\\/]\.git[\\/]' })) {
        $taskStream = [IO.File]::OpenRead($taskFile.FullName)
        try { $taskMagic=New-Object byte[] 4; $null=$taskStream.Read($taskMagic,0,4) } finally { $taskStream.Dispose() }
        $taskIsExecutable = ($taskMagic[0] -eq 77 -and $taskMagic[1] -eq 90) -or ($taskMagic[0] -eq 127 -and $taskMagic[1] -eq 69 -and $taskMagic[2] -eq 76 -and $taskMagic[3] -eq 70)
        if (-not $taskIsExecutable) { continue }
        $taskParent=$taskFile.Directory.FullName
        $taskRepo=''
        while ($taskParent.StartsWith($taskExtra,[StringComparison]::OrdinalIgnoreCase)) {
            if (Test-Path -LiteralPath (Join-Path $taskParent '.git')) { $taskRepo=$taskParent; break }
            $taskParent=[IO.Path]::GetDirectoryName($taskParent)
        }
        if ($taskRepo) {
            $taskRelative=$taskFile.FullName.Substring($taskRepo.Length+1).Replace('\','/')
            $null=& git -C $taskRepo ls-files --error-unmatch -- $taskRelative 2>$null
            if ($LASTEXITCODE -eq 0) { continue }
        }
        $taskCandidates += $taskFile
    }
}
$taskProcesses=@(Get-CimInstance Win32_Process)
$taskPlan=@()
$taskRoots=@($taskRoot,'C:\Users\a.sevrjukova\AppData\Local\Temp\onebase-2558-codex','C:\Users\a.sevrjukova\AppData\Local\Temp\platform-result-form-20261111')
foreach ($taskFile in @($taskCandidates | Sort-Object FullName -Unique)) {
    $taskAbsolute=(Resolve-Path -LiteralPath $taskFile.FullName).Path
    $taskOwnerRoot=@($taskRoots | Where-Object { $taskAbsolute.StartsWith($_+'\',[StringComparison]::OrdinalIgnoreCase) })
    if ($taskOwnerRoot.Count -ne 1) { throw 'STOP: resolved candidate outside exact authorized roots' }
    if (@($taskProtected | Where-Object { $taskFile.Name.StartsWith($_,[StringComparison]::OrdinalIgnoreCase) }).Count -ne 0) { throw 'STOP: protected SDK in candidates' }
    $taskCheck=$taskAbsolute
    while ($taskCheck.Length -ge $taskOwnerRoot[0].Length) {
        if (((Get-Item -LiteralPath $taskCheck -Force).Attributes -band [IO.FileAttributes]::ReparsePoint) -ne 0) { throw 'STOP: reparse point in candidate path' }
        $taskCheck=[IO.Path]::GetDirectoryName($taskCheck)
    }
    if (@($taskProcesses | Where-Object { ($_.ExecutablePath -and $_.ExecutablePath.Equals($taskAbsolute,[StringComparison]::OrdinalIgnoreCase)) -or ($_.CommandLine -and $_.CommandLine.IndexOf($taskAbsolute,[StringComparison]::OrdinalIgnoreCase) -ge 0) }).Count -ne 0) { throw 'STOP: active process references old artifact' }
    $taskPlan += [ordered]@{path=$taskAbsolute; bytes=$taskFile.Length; sha256=(Get-FileHash -LiteralPath $taskAbsolute -Algorithm SHA256).Hash.ToLowerInvariant()}
}
$taskReceiptPath=Join-Path $taskRoot '20261115-platform-b10-b17-cleanup.json'
if (Test-Path -LiteralPath $taskReceiptPath) { throw 'STOP: cleanup receipt already exists' }
$taskBefore=(Get-PSDrive C).Free
$taskPlan | ConvertTo-Json -Depth 8 | Set-Content -Encoding UTF8 -LiteralPath (Join-Path $taskRoot '20261115-platform-b10-b17-delete-plan.json')
foreach ($taskEntry in $taskPlan) { Remove-Item -LiteralPath $taskEntry.path -Force }
$taskRemoved=[int64]0
foreach ($taskEntry in $taskPlan) { $taskRemoved += [int64]$taskEntry['bytes'] }
$taskReceipt=[ordered]@{utc=[DateTime]::UtcNow.ToString('o'); removed_count=$taskPlan.Count; removed_file_bytes=$taskRemoved; free_C_before=$taskBefore; free_C_after=(Get-PSDrive C).Free; files=$taskPlan; source_and_receipts_preserved=$true; protected_B7_B8_B18_B19_preserved=$true; production_bb0269_and_6efa_outside_selection=$true; extra_directories_retained='source worktrees/receipts retained; only untracked executable outputs removed'; recursive_delete_performed=$false; no_process_stopped=$true}
$taskReceipt | ConvertTo-Json -Depth 8 | Set-Content -Encoding UTF8 -LiteralPath $taskReceiptPath
[ordered]@{receipt=$taskReceiptPath; count=$taskPlan.Count; removed_file_bytes=$taskRemoved; free_C_after=$taskReceipt.free_C_after} | ConvertTo-Json
