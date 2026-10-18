param([string]$DataRoot = (Join-Path $PSScriptRoot '../..'))
$ErrorActionPreference = 'Stop'
$taskRoot = (Resolve-Path -LiteralPath $DataRoot).Path
# A previous launch can leave a stale PID record. Match this installation's
# exact executable/command as well, never another app with the same name.
$taskApiPath = [IO.Path]::GetFullPath((Join-Path $taskRoot 'runtime/api.exe'))
$taskWebPath = Join-Path $taskRoot 'server.cjs'
foreach ($taskProcess in (Get-CimInstance Win32_Process -Filter "Name = 'api.exe' OR Name = 'node.exe'")) {
  if ($taskProcess.ExecutablePath -eq $taskApiPath -or ($taskProcess.Name -eq 'node.exe' -and $taskProcess.CommandLine -like ('*"' + $taskWebPath + '"*'))) {
    Stop-Process -Id $taskProcess.ProcessId -Force -ErrorAction SilentlyContinue
    Wait-Process -Id $taskProcess.ProcessId -Timeout 10 -ErrorAction SilentlyContinue
  }
}
$taskPidFile = Join-Path $taskRoot 'data/processes.json'
if (Test-Path -LiteralPath $taskPidFile) {
  $taskIds = Get-Content -LiteralPath $taskPidFile -Raw | ConvertFrom-Json
  foreach ($taskKind in @('web','api')) {
    $taskProcessId = $taskIds.$taskKind
    if (-not $taskProcessId) { continue }
    $taskProcess = Get-CimInstance Win32_Process -Filter "ProcessId = $taskProcessId"
    if (-not $taskProcess) { continue }
    $taskOwned = if ($taskKind -eq 'api') { $taskProcess.ExecutablePath -eq (Join-Path $taskRoot 'runtime/api.exe') } else { $taskProcess.CommandLine -like ('*' + (Join-Path $taskRoot 'server.cjs') + '*') }
    if ($taskOwned) { Stop-Process -Id $taskProcessId -ErrorAction SilentlyContinue }
  }
}
& (Join-Path $taskRoot 'runtime/pgsql/bin/pg_ctl.exe') status -D (Join-Path $taskRoot 'data/postgres') *> $null
if ($LASTEXITCODE -eq 0) {
  & (Join-Path $taskRoot 'runtime/pgsql/bin/pg_ctl.exe') stop -D (Join-Path $taskRoot 'data/postgres') -m fast -w
  if ($LASTEXITCODE -ne 0) { throw 'Database shutdown failed.' }
}
Write-Host 'Pokedextracker stopped. Your saved progress remains in data/postgres.'
