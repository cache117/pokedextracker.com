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
$taskRoot=(Resolve-Path $DataRoot).Path
$taskSource=Split-Path $PSScriptRoot -Parent
$taskApi=$ApiUrl.TrimEnd('/')+'/api'
$taskBefore=Invoke-RestMethod "$taskApi/users/cache117"
$taskGames=Invoke-RestMethod "$taskApi/games"
foreach($taskGame in @('fire_red','leaf_green')){
  if($taskGame -notin $taskGames.id){throw "Missing published game: $taskGame"}
}
$taskTypes=@((Invoke-RestMethod "$taskApi/dex-types") | Where-Object game_family_id -eq 'fire_red_leaf_green')
if($taskTypes.Count -ne 2){throw 'Expected exactly two FireRed/LeafGreen templates'}
$taskUser='frlg_'+[Guid]::NewGuid().ToString('N').Substring(0,12)
$taskSession=Invoke-RestMethod "$taskApi/users" -Method Post -ContentType 'application/json' -Body (@{
  username=$taskUser;password='trial-only-password';title='FireRed Kanto';slug='fire-red-kanto';game='fire_red';dex_type=1000001;shiny=$false
}|ConvertTo-Json)
$taskAuth=@{Authorization='Bearer '+$taskSession.token}
$taskDexes=@(Invoke-RestMethod "$taskApi/users/$taskUser/dexes/fire-red-kanto")
foreach($taskSpec in @(
  @{game='leaf_green';dex_type=1000001;slug='leaf-green-kanto'},
  @{game='fire_red';dex_type=1000002;slug='fire-red-national'},
  @{game='leaf_green';dex_type=1000002;slug='leaf-green-national'}
)){
  $taskDexes+=Invoke-RestMethod "$taskApi/users/$taskUser/dexes" -Method Post -Headers $taskAuth -ContentType 'application/json' -Body (@{
    title=$taskSpec.slug;slug=$taskSpec.slug;game=$taskSpec.game;dex_type=$taskSpec.dex_type;shiny=$true
  }|ConvertTo-Json)
}
foreach($taskDex in $taskDexes){
  $taskExpected=if($taskDex.dex_type.id -eq 1000001){151}else{386}
  $taskCaptures=Invoke-RestMethod "$taskApi/users/$taskUser/dexes/$($taskDex.slug)/captures"
  if($taskDex.total -ne $taskExpected -or $taskCaptures.Count -ne $taskExpected){throw 'Incorrect dex size'}
  for($taskIndex=0;$taskIndex -lt $taskExpected;$taskIndex++){
    $taskMon=$taskCaptures[$taskIndex].pokemon
    if($taskMon.national_id -ne ($taskIndex+1) -or $taskMon.dex_number -ne ($taskIndex+1) -or $taskMon.form -or $taskMon.id -ne ($taskIndex+1)){
      throw 'Unexpected ordering, duplicate identity or later-generation form'
    }
  }
  $null=Invoke-RestMethod "$taskApi/captures" -Method Post -Headers $taskAuth -ContentType 'application/json' -Body (@{dex=$taskDex.id;pokemon=@(1,$taskExpected)}|ConvertTo-Json)
}
$taskEevee=Invoke-RestMethod "$taskApi/pokemon/133?dex_type=1000002"
if(@($taskEevee.evolution_family.pokemon[1]).Count -ne 5){throw 'Expected only five Generation III Eevee evolutions'}
$taskRegional=Invoke-RestMethod "$taskApi/pokemon/41?dex_type=1000001"
if($taskRegional.evolution_family.pokemon.id -contains 169){throw 'Crobat leaked into the Kanto list'}
$taskNational=Invoke-RestMethod "$taskApi/pokemon/41?dex_type=1000002"
if($taskNational.evolution_family.pokemon.id -notcontains 169){throw 'Crobat missing from National evolution line'}
$taskSecrets=Get-Content "$taskRoot/private.local.json" -Raw|ConvertFrom-Json
$env:ENVIRONMENT='local';$env:DATABASE_NAME=$DatabaseName;$env:DATABASE_PORT='9876'
$env:DATABASE_PASSWORD=$taskSecrets.database_password;$env:JWT_SECRET=$taskSecrets.jwt_secret
$taskRevision=git -C $taskSource merge-base HEAD upstream/master
& "$taskRoot/runtime/personal-seed-import.exe" "$taskSource/seeds" $taskRevision
if($LASTEXITCODE -ne 0){throw 'Repeat import failed'}
foreach($taskDex in $taskDexes){
  $taskAfter=Invoke-RestMethod "$taskApi/users/$taskUser/dexes/$($taskDex.slug)"
  if($taskAfter.caught -ne 2 -or $taskAfter.total -ne $taskDex.total -or $taskAfter.dex_type.id -ne $taskDex.dex_type.id){throw 'Import changed personal dex identity or progress'}
}
if('fire_red' -notin (Invoke-RestMethod "$taskApi/games").id){throw 'Import hid FireRed again'}
$taskAfterOwner=Invoke-RestMethod "$taskApi/users/cache117"
if(($taskBefore.dexes|ConvertTo-Json -Depth 15 -Compress) -cne ($taskAfterOwner.dexes|ConvertTo-Json -Depth 15 -Compress)){throw 'Existing owner collections changed'}
Write-Output 'Both games, four dex combinations, complete ordered memberships, generation-scoped evolutions and progress across repeat imports passed.'
