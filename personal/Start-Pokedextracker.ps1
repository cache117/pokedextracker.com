param([switch]$NoBrowser, [string]$DataRoot = (Join-Path $PSScriptRoot '../..'))
$ErrorActionPreference = 'Stop'
$taskRoot = (Resolve-Path -LiteralPath $DataRoot).Path
$taskSource = Split-Path $PSScriptRoot -Parent
$taskSecrets = Get-Content -LiteralPath (Join-Path $taskRoot 'private.local.json') -Raw | ConvertFrom-Json
$env:ENVIRONMENT = 'local'
$env:DATABASE_PORT = '9876'
$env:DATABASE_NAME = 'pokedex_tracker'
$env:DATABASE_PASSWORD = $taskSecrets.database_password
$env:JWT_SECRET = $taskSecrets.jwt_secret
$env:PORT = '9898'
$taskPg = Join-Path $taskRoot 'runtime/pgsql/bin/pg_ctl.exe'
$taskData = Join-Path $taskRoot 'data/postgres'
& $taskPg status -D $taskData *> $null
if ($LASTEXITCODE -ne 0) {
  & $taskPg start -D $taskData -l (Join-Path $taskRoot 'logs/postgres.log') -w
  if ($LASTEXITCODE -ne 0) { throw 'PostgreSQL could not start. See logs/postgres.log.' }
}
function Test-LocalEndpoint([string]$Url) {
  try { return (Invoke-WebRequest -Uri $Url -TimeoutSec 2 -UseBasicParsing).StatusCode -eq 200 } catch { return $false }
}
if (-not (Test-LocalEndpoint 'http://127.0.0.1:9898/api/health')) {
  $taskApi = Start-Process -FilePath (Join-Path $taskRoot 'runtime/api.exe') -WorkingDirectory $taskSource -WindowStyle Hidden -PassThru -RedirectStandardOutput (Join-Path $taskRoot 'logs/api.log') -RedirectStandardError (Join-Path $taskRoot 'logs/api-error.log')
  @{api=$taskApi.Id} | ConvertTo-Json | Set-Content -LiteralPath (Join-Path $taskRoot 'data/processes.json') -Encoding utf8
}
for ($taskAttempt = 0; $taskAttempt -lt 30; $taskAttempt++) {
  if ((Test-LocalEndpoint 'http://127.0.0.1:9898/api/games') -and (Test-LocalEndpoint 'http://127.0.0.1:9898/')) {
    Write-Host 'Pokedextracker is ready at http://localhost:9898'
    if (-not $NoBrowser) { Start-Process 'http://localhost:9898/u/cache117' }
    return
  }
  Start-Sleep -Seconds 1
}
throw 'Pokedextracker did not become ready. Check the logs folder.'
