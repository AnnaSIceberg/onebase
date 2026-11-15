$ErrorActionPreference = 'Stop'
$taskPinnedGo = 'C:/Users/a.sevrjukova/go/pkg/mod/golang.org/toolchain@v0.0.1-go1.26.11.windows-amd64/bin/go.exe'
$taskCompilerNames = @('go.exe','compile.exe','link.exe','asm.exe','cgo.exe')
$taskActiveGo = @(Get-CimInstance Win32_Process | Where-Object { $_.Name -in $taskCompilerNames })
if ($taskActiveGo.Count -ne 0) { throw 'STOP: Go processes exist; cache cleanup not allowed' }
$env:GOTOOLCHAIN = 'go1.26.11'
$taskCachePath = (& $taskPinnedGo env GOCACHE).Trim()
if ($LASTEXITCODE -ne 0) { throw 'go env GOCACHE failed' }
$taskExpectedCache = 'C:\Users\a.sevrjukova\AppData\Local\go-build'
$taskResolvedCache = (Resolve-Path -LiteralPath $taskCachePath).Path
if ($taskResolvedCache -ne $taskExpectedCache) { throw 'STOP: unexpected build cache path' }
if (((Get-Item -LiteralPath $taskResolvedCache).Attributes -band [IO.FileAttributes]::ReparsePoint) -ne 0) { throw 'STOP: cache is a reparse point' }
$taskCacheBytesBefore = (Get-ChildItem -LiteralPath $taskResolvedCache -Recurse -File | Measure-Object -Property Length -Sum).Sum
$taskFreeBefore = (Get-PSDrive C).Free
$taskStart = [DateTime]::UtcNow.ToString('o')
& $taskPinnedGo clean -cache
if ($LASTEXITCODE -ne 0) { throw 'go clean -cache failed' }
$taskCacheBytesAfter = (Get-ChildItem -LiteralPath $taskResolvedCache -Recurse -File | Measure-Object -Property Length -Sum).Sum
$taskActiveAfter = @(Get-CimInstance Win32_Process | Where-Object { $_.Name -in $taskCompilerNames })
if ($taskActiveAfter.Count -ne 0) { throw 'STOP: Go processes appeared; report not clean END' }
$taskReceipt = [ordered]@{
    kind='authorized reproducible build cache cleanup only'; start_utc=$taskStart; end_utc=[DateTime]::UtcNow.ToString('o');
    pinned_go=$taskPinnedGo; toolchain='go1.26.11'; command='go clean -cache'; cache_path=$taskResolvedCache;
    go_before=0; go_after=0; cache_bytes_before=$taskCacheBytesBefore; cache_bytes_after=$taskCacheBytesAfter;
    removed_file_bytes=($taskCacheBytesBefore-$taskCacheBytesAfter); free_C_before=$taskFreeBefore; free_C_after=(Get-PSDrive C).Free;
    modcache_touched=$false; sdk_touched=$false; sources_touched=$false; databases_touched=$false; receipts_touched=$false
}
$taskReceipt | ConvertTo-Json | Set-Content -Encoding UTF8 -LiteralPath 'C:/Users/a.sevrjukova/AppData/Local/Temp/platform-help-admin-20261111/20261115-platform-build-cache-cleanup-b19.json'
$taskReceipt | ConvertTo-Json
