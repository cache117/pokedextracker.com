param(
  [Parameter(Mandatory)][string]$ApiUrl,
  [Parameter(Mandatory)][string]$DatabaseName,
  [string]$DataRoot=(Join-Path $PSScriptRoot '../..')
)
$ErrorActionPreference='Stop'
$taskAddress=[Uri]$ApiUrl
if($taskAddress.Host -notin @('127.0.0.1','localhost') -or $taskAddress.Port -eq 9898 -or $DatabaseName -eq 'pokedex_tracker' -or $DatabaseName -notmatch 'trial|test'){
  throw 'Run this check only against a separate loopback trial/test server and database.'
}
$taskRoot=(Resolve-Path $DataRoot).Path
$taskApi=$ApiUrl.TrimEnd('/')+'/api'
$taskSecrets=Get-Content "$taskRoot/private.local.json" -Raw|ConvertFrom-Json
$taskSession=Invoke-RestMethod "$taskApi/sessions" -Method Post -ContentType 'application/json' -Body (@{username='cache117';password=$taskSecrets.login_password}|ConvertTo-Json)
$taskAuth=@{Authorization='Bearer '+$taskSession.token}
$taskOriginal=Invoke-RestMethod "$taskApi/personal/catalog" -Headers $taskAuth
$taskProfile=Invoke-RestMethod "$taskApi/users/cache117"
if(-not $taskOriginal.is_owner){throw 'Local owner login failed'}
function Copy-Config($value){$value|ConvertTo-Json -Depth 30|ConvertFrom-Json}
function Save-Config($value){
  $taskBody=@{revision=$value.revision;settings=$value.settings;overrides=$value.overrides}|ConvertTo-Json -Depth 30
  Invoke-RestMethod "$taskApi/personal/catalog" -Method Post -Headers $taskAuth -ContentType 'application/json; charset=utf-8' -Body ([Text.Encoding]::UTF8.GetBytes($taskBody))
}
function Expect-Rejection($headers,$body,$expectedStatus=0){
  try{$null=Invoke-RestMethod "$taskApi/personal/catalog" -Method Post -Headers $headers -ContentType 'application/json' -Body $body}
  catch{if($_.Exception.Response.StatusCode.value__ -ge 400){if($expectedStatus -and $_.Exception.Response.StatusCode.value__ -ne $expectedStatus){throw 'Unexpected rejection status'};return};throw}
  throw 'A forbidden/invalid edit was accepted'
}
$taskState=Copy-Config $taskOriginal
try{
  $taskReadOnly=Invoke-RestMethod "$taskApi/personal/catalog"
  if($taskReadOnly.is_owner){throw 'Anonymous browsing grants owner rights'}
  Expect-Rejection @{} '{}'
  Expect-Rejection @{Authorization='Bearer invalid-token'} '{}'
  $taskOther=Invoke-RestMethod "$taskApi/users" -Method Post -ContentType 'application/json' -Body (@{username=('trial_'+[Guid]::NewGuid().ToString('N').Substring(0,12));password='trial-only-password';title='Trial catalogue permissions';slug='trial-catalogue';game='legends_za';dex_type=23;shiny=$false}|ConvertTo-Json)
  $taskOtherAuth=@{Authorization='Bearer '+$taskOther.token}
  if((Invoke-RestMethod "$taskApi/personal/catalog" -Headers $taskOtherAuth).is_owner){throw 'Another account acquired owner privileges'}
  Expect-Rejection $taskOtherAuth '{}' 403
  $taskBefore=Invoke-RestMethod "$taskApi/personal/catalog/encounters/1"
  $taskState.settings.home_mode='all'
  $taskState.settings.hidden_groups=@('3DS')
  $taskState=Save-Config $taskState
  $taskHome=Invoke-RestMethod "$taskApi/pokemon/1?dex_type=21"
  if($taskHome.locations.game.id -contains 'x'){throw 'Hidden group remains in HOME sources'}
  $taskRegional=Invoke-RestMethod "$taskApi/pokemon/1?dex_type=1"
  if($taskRegional.locations.game.id -notcontains 'x'){throw 'HOME filters affected regional sources'}
  $taskState.settings.hidden_groups=@()
  $taskState.overrides=@(@{pokemon_id=1;game_id='x';values=@('Personal regression check')})
  $taskState=Save-Config $taskState
  $taskRegional=Invoke-RestMethod "$taskApi/pokemon/1?dex_type=1"
  if(($taskRegional.locations|Where-Object {$_.game.id -eq 'x'}).value[0] -ne 'Personal regression check'){throw 'Override not applied'}
  $taskRaw=Invoke-RestMethod "$taskApi/personal/catalog/encounters/1"
  if(($taskBefore|ConvertTo-Json -Depth 10 -Compress) -cne ($taskRaw|ConvertTo-Json -Depth 10 -Compress)){throw 'Imported baseline changed'}
  Expect-Rejection $taskAuth (@{revision=-1;settings=$taskState.settings;overrides=$taskState.overrides}|ConvertTo-Json -Depth 30)
  $taskInvalid=Copy-Config $taskState
  $taskInvalid.overrides=@(@{pokemon_id=2147483647;game_id='x';values=@('Invalid')})
  Expect-Rejection $taskAuth (@{revision=$taskInvalid.revision;settings=$taskInvalid.settings;overrides=$taskInvalid.overrides}|ConvertTo-Json -Depth 30)
  $taskDetail=Invoke-RestMethod "$taskApi/pokemon/133?dex_type=21"
  if(-not $taskDetail.reference_details_imported -or $taskDetail.evolution_family.pokemon[1].Count -lt 8){throw 'Branched evolution details missing'}
  $taskTypes=Invoke-RestMethod "$taskApi/dex-types"
  foreach($taskType in @(23,24,25)){if($taskType -notin $taskTypes.id){throw 'Z-A template missing'}}
  $env:ENVIRONMENT='local';$env:DATABASE_NAME=$DatabaseName;$env:DATABASE_PORT='9876'
  $env:DATABASE_PASSWORD=$taskSecrets.database_password;$env:JWT_SECRET=$taskSecrets.jwt_secret
  $taskSource=Split-Path $PSScriptRoot -Parent
  $taskSeedRevision=git -C $taskSource rev-parse --verify --quiet MERGE_HEAD
  if($LASTEXITCODE -ne 0){$taskSeedRevision=git -C $taskSource merge-base HEAD upstream/master}
  & "$taskRoot/runtime/personal-seed-import.exe" "$taskSource/seeds" $taskSeedRevision
  if($LASTEXITCODE -ne 0){throw 'Repeat seed import failed'}
  $taskRegional=Invoke-RestMethod "$taskApi/pokemon/1?dex_type=1"
  if(($taskRegional.locations|Where-Object {$_.game.id -eq 'x'}).value[0] -ne 'Personal regression check'){throw 'Seed refresh discarded override'}
  Write-Output 'Owner authentication, anonymous/invalid/stale edits, HOME filtering, regional sources, encounter overlays, seed refresh and branched evolutions passed.'
}finally{
  $taskLatest=Invoke-RestMethod "$taskApi/personal/catalog" -Headers $taskAuth
  $taskOriginal.revision=$taskLatest.revision
  $null=Save-Config $taskOriginal
}
$taskAfter=Invoke-RestMethod "$taskApi/users/cache117"
foreach($taskDex in $taskProfile.dexes){$taskMatch=$taskAfter.dexes|Where-Object id -eq $taskDex.id;if($taskMatch.caught -ne $taskDex.caught -or $taskMatch.total -ne $taskDex.total){throw 'Saved progress changed'}}
Write-Output 'All 24 existing dex counts preserved; test configuration restored.'
