param([string]$GoExe='go', [string]$NodeExe='node', [string]$PnpmScript='')
$ErrorActionPreference='Stop'
$taskSource=Split-Path $PSScriptRoot -Parent
Push-Location $taskSource
try {
  $taskRevision=git rev-parse --short HEAD
  if($LASTEXITCODE -ne 0){throw 'Cannot identify source revision'}
  $env:PATH=(Split-Path (Get-Command $NodeExe -ErrorAction Stop).Source -Parent)+';'+$env:PATH
  $env:NODE_ENV='production'
  $env:VERSION='personal-'+$taskRevision
  # Keep the upstream pnpm lockfile and embedded-frontend build order.
  if($PnpmScript){& $NodeExe $PnpmScript install --frozen-lockfile --ignore-scripts}
  else {pnpm install --frozen-lockfile --ignore-scripts}
  if($LASTEXITCODE -ne 0){throw 'Pinned dependency installation failed'}
  & $NodeExe node_modules/typescript/bin/tsc -b --noEmit
  if($LASTEXITCODE -ne 0){throw 'TypeScript check failed'}
  # The normal installation keeps the plain token key, preserving its login.
  $env:POKEDEXTRACKER_TOKEN_NAMESPACE=''
  & $NodeExe node_modules/vite/bin/vite.js build
  $taskBuildResult=$LASTEXITCODE
  [IO.File]::WriteAllText((Join-Path $taskSource 'internal/webapp/dist/.gitkeep'),'')
  if($taskBuildResult -ne 0){throw 'Frontend build failed'}
  [IO.File]::WriteAllText((Join-Path $taskSource 'internal/webapp/dist/version'),$env:VERSION)
  & $GoExe build -ldflags "-X github.com/pokedextracker/pokedextracker.com/pkg/config.Version=$($env:VERSION)" -o ../runtime/api-next.exe ./cmd/api
  if($LASTEXITCODE -ne 0){throw 'Unified server build failed'}
  & $GoExe build -o ../runtime/personal-seed-import.exe ./cmd/personal-seed-import
  if($LASTEXITCODE -ne 0){throw 'Reference importer build failed'}
  & $GoExe build -o ../runtime/migrations.exe ./cmd/migrations
  if($LASTEXITCODE -ne 0){throw 'Migration build failed'}
  Write-Output 'Build ready. Stop the tracker, replace runtime/api.exe with runtime/api-next.exe, then start it again.'
} finally {Pop-Location}
