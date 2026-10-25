param(
  [Parameter(Mandatory)][string]$ApiUrl,
  [Parameter(Mandatory)][string]$DatabaseName,
  [string]$DataRoot=(Join-Path $PSScriptRoot '../..')
)
$ErrorActionPreference='Stop'
$taskAddress=[Uri]$ApiUrl
if($taskAddress.Host -notin @('127.0.0.1','localhost') -or $taskAddress.Port -eq 9898 -or $DatabaseName -eq 'pokedex_tracker' -or $DatabaseName -notmatch 'trial|test'){
  throw 'Use a separate loopback trial/test server and database.'
}
$taskRoot=(Resolve-Path -LiteralPath $DataRoot).Path
$taskApi=$ApiUrl.TrimEnd('/')+'/api'
$taskSecrets=Get-Content -LiteralPath "$taskRoot/private.local.json" -Raw|ConvertFrom-Json
$taskSession=Invoke-RestMethod "$taskApi/sessions" -Method Post -ContentType 'application/json' -Body (@{username='cache117';password=$taskSecrets.login_password}|ConvertTo-Json)
$taskAuth=@{Authorization='Bearer '+$taskSession.token}
$taskOriginal=Invoke-RestMethod "$taskApi/personal/catalog" -Headers $taskAuth
$taskProfile=Invoke-RestMethod "$taskApi/users/cache117"
$taskRegional=Invoke-RestMethod "$taskApi/pokemon/1?dex_type=1"
$taskFrlg=Invoke-RestMethod "$taskApi/pokemon/249?dex_type=1000002"
function Save-HomeConfig($value){
  $taskBody=@{revision=$value.revision;settings=$value.settings;overrides=$value.overrides}|ConvertTo-Json -Depth 30
  Invoke-RestMethod "$taskApi/personal/catalog" -Method Post -Headers $taskAuth -ContentType 'application/json; charset=utf-8' -Body ([Text.Encoding]::UTF8.GetBytes($taskBody))
}
$taskState=$taskOriginal|ConvertTo-Json -Depth 30|ConvertFrom-Json
try {
  $taskState.settings.home_mode='all'
  $taskState.settings.hidden_groups=@()
  $taskState.settings | Add-Member -NotePropertyName hidden_platforms -NotePropertyValue @() -Force
  $taskState.settings | Add-Member -NotePropertyName hidden_routes -NotePropertyValue @('bank') -Force
  $taskState=Save-HomeConfig $taskState
  $taskHome=Invoke-RestMethod "$taskApi/pokemon/1?dex_type=21"
  $taskBankGames=@($taskState.settings.games|Where-Object route_id -eq 'bank'|ForEach-Object id)
  if(@($taskHome.locations|Where-Object {$_.game.id -in $taskBankGames}).Count){throw 'Bank sources leaked into historical HOME mode'}
  if('scarlet' -notin $taskHome.locations.game.id){throw 'Hiding Bank also hid direct HOME sources'}
  $taskState.settings.hidden_routes=@()
  $taskState=Save-HomeConfig $taskState
  if('x' -notin (Invoke-RestMethod "$taskApi/pokemon/1?dex_type=21").locations.game.id){throw 'Unhiding Bank did not restore sources'}
  $taskState.settings.hidden_platforms=@('Switch')
  $taskState=Save-HomeConfig $taskState
  $taskHome=Invoke-RestMethod "$taskApi/pokemon/1?dex_type=21"
  $taskSwitchGames=@($taskState.settings.games|Where-Object platform -eq 'Switch'|ForEach-Object id)
  if(@($taskHome.locations|Where-Object {$_.game.id -in $taskSwitchGames}).Count -or 'x' -notin $taskHome.locations.game.id){throw 'Platform filter did not independently hide Switch sources'}
  $taskState.settings.home_mode='current'
  $taskState.settings.hidden_platforms=@('3DS','DS','Game Boy Advance','Game Boy Color','Game Boy')
  $taskState.settings.hidden_routes=@('bank')
  $taskState=Save-HomeConfig $taskState
  $taskExpected=$taskState.settings|ConvertTo-Json -Depth 30 -Compress
  $taskReload=Invoke-RestMethod "$taskApi/personal/catalog" -Headers $taskAuth
  if(($taskReload.settings|ConvertTo-Json -Depth 30 -Compress) -cne $taskExpected){throw 'Source rules did not survive save/reload'}
  $taskSource=Split-Path $PSScriptRoot -Parent
  $taskRevision=git -C $taskSource merge-base HEAD upstream/master
  if($LASTEXITCODE -ne 0){throw 'Cannot identify reference revision'}
  $env:ENVIRONMENT='local';$env:DATABASE_NAME=$DatabaseName;$env:DATABASE_PORT='9876'
  $env:DATABASE_PASSWORD=$taskSecrets.database_password;$env:JWT_SECRET=$taskSecrets.jwt_secret
  & "$taskRoot/runtime/personal-seed-import.exe" "$taskSource/seeds" $taskRevision
  if($LASTEXITCODE -ne 0){throw 'Trial reference import failed'}
  $taskReload=Invoke-RestMethod "$taskApi/personal/catalog" -Headers $taskAuth
  if(($taskReload.settings|ConvertTo-Json -Depth 30 -Compress) -cne $taskExpected){throw 'Reference import discarded source rules'}
  $taskHome=Invoke-RestMethod "$taskApi/pokemon/1?dex_type=21"
  if('scarlet' -notin $taskHome.locations.game.id -or @($taskHome.locations|Where-Object {$_.game.id -in $taskBankGames}).Count){throw 'Switch/HOME focus incorrect'}
  if(($taskRegional.locations|ConvertTo-Json -Depth 30 -Compress) -cne ((Invoke-RestMethod "$taskApi/pokemon/1?dex_type=1").locations|ConvertTo-Json -Depth 30 -Compress)){throw 'HOME filters changed X/Y encounters'}
  if(($taskFrlg.locations|ConvertTo-Json -Depth 30 -Compress) -cne ((Invoke-RestMethod "$taskApi/pokemon/249?dex_type=1000002").locations|ConvertTo-Json -Depth 30 -Compress)){throw 'HOME filters changed FR/LG encounters'}
  $taskState.settings.PSObject.Properties.Remove('hidden_platforms')
  $taskState.settings.PSObject.Properties.Remove('hidden_routes')
  $taskState=Save-HomeConfig $taskState
  if($taskState.settings.hidden_platforms -or $taskState.settings.hidden_routes){throw 'Older configuration compatibility failed'}
  Write-Output 'Hidden Bank routes, independent platform filtering, unhide, save/reload, reference import preservation, Switch/HOME focus, older configuration compatibility and game-specific encounters passed.'
} finally {
  $taskLatest=Invoke-RestMethod "$taskApi/personal/catalog" -Headers $taskAuth
  $taskOriginal.revision=$taskLatest.revision
  $null=Save-HomeConfig $taskOriginal
}
$taskAfter=Invoke-RestMethod "$taskApi/users/cache117"
foreach($taskDex in $taskProfile.dexes){
  $taskMatch=$taskAfter.dexes|Where-Object id -eq $taskDex.id
  if($taskMatch.caught -ne $taskDex.caught -or $taskMatch.total -ne $taskDex.total){throw 'Saved progress changed'}
}
Write-Output 'Saved progress preserved and original trial catalogue restored.'
