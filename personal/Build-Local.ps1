param([string]$GoExe='go')
$ErrorActionPreference='Stop'
$taskSource=Split-Path $PSScriptRoot -Parent
$taskRoot=Split-Path $taskSource -Parent
Push-Location $taskSource
try {
  $taskRevision=git merge-base HEAD upstream/master
  if($taskRevision){$taskRevision=$taskRevision.Substring(0,7)}
  if($LASTEXITCODE -ne 0){throw 'Cannot identify upstream revision'}
  $env:NODE_ENV='production'
  $env:VERSION='personal-'+$taskRevision
  & $GoExe build -o ../runtime/api-next.exe ./cmd/api
  if($LASTEXITCODE -ne 0){throw 'API build failed'}
  & $GoExe build -o ../runtime/personal-seed-import.exe ./cmd/personal-seed-import
  if($LASTEXITCODE -ne 0){throw 'Reference importer build failed'}
  & $GoExe build -o ../runtime/migrations.exe ./cmd/migrations
  if($LASTEXITCODE -ne 0){throw 'Migration build failed'}
  node node_modules/typescript/bin/tsc --noEmit
  if($LASTEXITCODE -ne 0){throw 'TypeScript check failed'}
  node node_modules/webpack/bin/webpack.js
  if($LASTEXITCODE -ne 0){throw 'Frontend build failed'}
  Copy-Item -Path public/* -Destination build/web -Recurse -Force
  [IO.File]::WriteAllText((Join-Path $taskSource 'build/web/version'),$env:VERSION)
  Write-Output 'Build ready. Stop the tracker, replace runtime/api.exe with runtime/api-next.exe, then start it again.'
} finally {Pop-Location}
