param([string]$DataRoot = (Join-Path $PSScriptRoot '../..'))
$ErrorActionPreference = 'Stop'
$taskRoot = (Resolve-Path -LiteralPath $DataRoot).Path
$taskApiPath = [IO.Path]::GetFullPath((Join-Path $taskRoot 'runtime/api.exe'))
# Also stop the previous installation's Node proxy during the one-time upgrade.
$taskWebPath = Join-Path $taskRoot 'server.cjs'
foreach ($taskProcess in (Get-CimInstance Win32_Process -Filter "Name = 'api.exe' OR Name = 'node.exe'")) {
  if ($taskProcess.ExecutablePath -eq $taskApiPath -or ($taskProcess.Name -eq 'node.exe' -and $taskProcess.CommandLine -like ('*"' + $taskWebPath + '"*'))) {
    Stop-Process -Id $taskProcess.ProcessId -Force -ErrorAction SilentlyContinue
    Wait-Process -Id $taskProcess.ProcessId -Timeout 10 -ErrorAction SilentlyContinue
  }
}
$taskPg = Join-Path $taskRoot 'runtime/pgsql/bin/pg_ctl.exe'
$taskData = Join-Path $taskRoot 'data/postgres'
& $taskPg status -D $taskData *> $null
if ($LASTEXITCODE -eq 0) {
  & $taskPg stop -D $taskData -m fast -w
  if ($LASTEXITCODE -ne 0) { throw 'Database shutdown failed.' }
}
Write-Host 'Pokedextracker stopped. Your saved progress remains in data/postgres.'
