param(
  [Parameter(Mandatory)][string]$ApiUrl,
  [Parameter(Mandatory)][string]$DatabaseName,
  [string]$DataRoot=(Join-Path $PSScriptRoot '../..'),
  [string]$ImporterExe=''
)
$ErrorActionPreference='Stop'
$taskAddress=[Uri]$ApiUrl
if($taskAddress.Host -notin @('127.0.0.1','localhost') -or $taskAddress.Port -eq 9898 -or $DatabaseName -eq 'pokedex_tracker' -or $DatabaseName -notmatch 'trial|test'){
  throw 'Use a separate loopback trial/test server and database.'
}
$taskRoot=(Resolve-Path $DataRoot).Path
$taskSource=Split-Path $PSScriptRoot -Parent
if(-not $ImporterExe){$ImporterExe=Join-Path $taskRoot 'runtime/personal-seed-import.exe'}
$taskApi=$ApiUrl.TrimEnd('/')+'/api'
$taskData=Get-Content "$taskSource/cmd/personal-seed-import/fire_red_leaf_green_locations.json" -Raw|ConvertFrom-Json
$taskProfile=Invoke-RestMethod "$taskApi/users/cache117"
foreach($taskRow in $taskData){
  $taskRaw=Invoke-RestMethod "$taskApi/personal/catalog/encounters/$($taskRow.national_id)"
  foreach($taskGame in @('fire_red','leaf_green')){
    $taskMatch=@($taskRaw|Where-Object game_id -eq $taskGame)
    if($taskMatch.Count -ne 1 -or ($taskMatch[0].values -join "`n") -cne ($taskRow.$taskGame -join "`n")){throw "Imported baseline differs for $taskGame/$($taskRow.national_id)"}
  }
}
foreach($taskID in @(1,23,37,65,151,179,186,249,250,251,252,385,386)){
  $taskDetail=Invoke-RestMethod "$taskApi/pokemon/$($taskID)?dex_type=1000002"
  foreach($taskGame in @('fire_red','leaf_green')){
    if($taskGame -notin $taskDetail.locations.game.id){throw "Location missing from National API: $taskGame/$taskID"}
  }
  if($taskID -le 151){
    $taskRegional=Invoke-RestMethod "$taskApi/pokemon/$($taskID)?dex_type=1000001"
    if(@($taskRegional.locations).Count -ne 2){throw 'Regional scope incorrect'}
  }
}
$taskSecrets=Get-Content "$taskRoot/private.local.json" -Raw|ConvertFrom-Json
$taskSession=Invoke-RestMethod "$taskApi/sessions" -Method Post -ContentType 'application/json' -Body (@{username='cache117';password=$taskSecrets.login_password}|ConvertTo-Json)
$taskAuth=@{Authorization='Bearer '+$taskSession.token}
$taskOriginal=Invoke-RestMethod "$taskApi/personal/catalog" -Headers $taskAuth
function Save-LocationConfig($config){
  $taskBody=@{revision=$config.revision;settings=$config.settings;overrides=$config.overrides}|ConvertTo-Json -Depth 30
  Invoke-RestMethod "$taskApi/personal/catalog" -Method Post -Headers $taskAuth -ContentType 'application/json; charset=utf-8' -Body ([Text.Encoding]::UTF8.GetBytes($taskBody))
}
try{
  $taskState=$taskOriginal|ConvertTo-Json -Depth 30|ConvertFrom-Json
  $taskState.overrides=@($taskState.overrides|Where-Object {-not ($_.pokemon_id -eq 1 -and $_.game_id -eq 'fire_red')})+@(@{pokemon_id=1;game_id='fire_red';values=@('FR/LG override preservation test')})
  $null=Save-LocationConfig $taskState
  $env:ENVIRONMENT='local';$env:DATABASE_NAME=$DatabaseName;$env:DATABASE_PORT='9876'
  $env:DATABASE_PASSWORD=$taskSecrets.database_password;$env:JWT_SECRET=$taskSecrets.jwt_secret
  $taskRevision=git -C $taskSource merge-base HEAD upstream/master
  & $ImporterExe "$taskSource/seeds" $taskRevision
  if($LASTEXITCODE -ne 0){throw 'Repeat import failed'}
  $taskDetail=Invoke-RestMethod "$taskApi/pokemon/1?dex_type=1000001"
  if(($taskDetail.locations|Where-Object {$_.game.id -eq 'fire_red'}).value[0] -ne 'FR/LG override preservation test'){throw 'Import overwrote the owner correction'}
  $taskRaw=Invoke-RestMethod "$taskApi/personal/catalog/encounters/1"
  if(($taskRaw|Where-Object game_id -eq 'fire_red').values[0] -notmatch 'Pallet Town'){throw 'Override changed the imported baseline'}
}finally{
  $taskLatest=Invoke-RestMethod "$taskApi/personal/catalog" -Headers $taskAuth
  $taskOriginal.revision=$taskLatest.revision
  $null=Save-LocationConfig $taskOriginal
}
$taskAfter=Invoke-RestMethod "$taskApi/users/cache117"
if(($taskProfile.dexes|ConvertTo-Json -Depth 20 -Compress) -cne ($taskAfter.dexes|ConvertTo-Json -Depth 20 -Compress)){throw 'Existing owner progress changed'}
Write-Output 'All 772 baselines, Regional/National API visibility, repeat import and owner correction preservation passed; existing collections unchanged.'
