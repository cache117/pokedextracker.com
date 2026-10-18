$ErrorActionPreference='Stop'
$taskSource=Split-Path $PSScriptRoot -Parent
$taskRoot=Split-Path $taskSource -Parent
$taskRevision=git -C $taskSource rev-parse upstream/master
if($LASTEXITCODE -ne 0){throw 'Cannot identify upstream revision'}
git -C $taskSource diff --quiet $taskRevision -- seeds
if($LASTEXITCODE -ne 0){throw 'Seed files differ from upstream/master. Review those changes before importing.'}
$taskSecrets=Get-Content -LiteralPath "$taskRoot/private.local.json" -Raw|ConvertFrom-Json
$env:ENVIRONMENT='local';$env:DATABASE_NAME='pokedex_tracker';$env:DATABASE_PORT='9876'
$env:DATABASE_PASSWORD=$taskSecrets.database_password;$env:JWT_SECRET=$taskSecrets.jwt_secret
$taskBackup=Join-Path $taskRoot ('backups/before-upstream-seeds-'+(Get-Date -Format 'yyyyMMdd-HHmmss')+'.dump')
$env:PGPASSWORD=$taskSecrets.database_password
try {
  & "$taskRoot/runtime/pgsql/bin/pg_dump.exe" -h 127.0.0.1 -p 9876 -U pokedex_tracker_admin -d pokedex_tracker -Fc -f $taskBackup
  if($LASTEXITCODE -ne 0){throw 'Database backup failed; nothing imported'}
} finally {Remove-Item Env:PGPASSWORD}
& "$taskRoot/runtime/personal-seed-import.exe" "$taskSource/seeds" $taskRevision
if($LASTEXITCODE -ne 0){throw 'Import failed and its transaction rolled back; see the error above'}
Write-Output "Database backup: $taskBackup"
