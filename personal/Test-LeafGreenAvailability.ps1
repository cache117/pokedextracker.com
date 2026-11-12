param([Parameter(Mandatory)][string]$ApiUrl,[Parameter(Mandatory)][string]$DatabaseName,[string]$DataRoot=(Join-Path $PSScriptRoot '../..'))
$ErrorActionPreference='Stop'
$taskAddress=[Uri]$ApiUrl
if($taskAddress.Host -notin @('127.0.0.1','localhost') -or $taskAddress.Port -eq 9898 -or $DatabaseName -notmatch 'trial|test' -or $DatabaseName -eq 'pokedex_tracker'){throw 'Use a separate loopback trial/test server and database.'}
$taskRoot=(Resolve-Path $DataRoot).Path
$taskApi=$ApiUrl.TrimEnd('/')+'/api'
$taskSecrets=Get-Content "$taskRoot/private.local.json" -Raw|ConvertFrom-Json
$taskSession=Invoke-RestMethod "$taskApi/sessions" -Method Post -ContentType 'application/json' -Body (@{username='cache117';password=$taskSecrets.login_password}|ConvertTo-Json)
$taskAuth=@{Authorization='Bearer '+$taskSession.token}
$taskOriginal=Invoke-RestMethod "$taskApi/personal/catalog" -Headers $taskAuth
$taskProfile=Invoke-RestMethod "$taskApi/users/cache117"
$taskDex=$taskProfile.dexes|Where-Object {$_.game.id -eq 'leaf_green' -and $_.dex_type.tags -contains 'full national'}|Select-Object -First 1
if(-not $taskDex){throw 'LeafGreen National collection missing'}
$taskUrl="$taskApi/personal/catalog/leaf-green/$($taskDex.id)"
function Save-LeafGreen($state,$ids,$hide=$true){
 $taskBody=@{revision=$state.revision;hide_unavailable=$hide;unavailable_pokemon_ids=@($ids)}|ConvertTo-Json -Depth 10
 Invoke-RestMethod $taskUrl -Method Post -Headers $taskAuth -ContentType 'application/json' -Body $taskBody
}
function Expect-AvailabilityRejection($url,$headers,$body){
 try{$null=Invoke-RestMethod $url -Method Post -Headers $headers -ContentType 'application/json' -Body $body}catch{if($_.Exception.Response.StatusCode.value__ -ge 400){return};throw}
 throw 'Invalid availability edit was accepted'
}
try {
 $taskState=Invoke-RestMethod $taskUrl -Headers $taskAuth
 if(@($taskState.default_unavailable_pokemon_ids).Count -ne 171 -or -not $taskState.is_owner){throw 'Reviewed starting list or owner permissions incorrect'}
 if(@($taskState.default_unavailable_pokemon_ids|Where-Object {$_ -in @(237,249,250,386)}).Count){throw 'Obtainable breeding/ticket species were hidden'}
 Expect-AvailabilityRejection $taskUrl @{} '{}'
 Expect-AvailabilityRejection $taskUrl $taskAuth (@{revision=$taskState.revision;hide_unavailable=$true;unavailable_pokemon_ids=@(999999)}|ConvertTo-Json)
 Expect-AvailabilityRejection $taskUrl $taskAuth (@{revision=$taskState.revision;hide_unavailable=$true;unavailable_pokemon_ids=@(151,151)}|ConvertTo-Json)
 $taskOther=$taskProfile.dexes|Where-Object {$_.game.id -ne 'leaf_green'}|Select-Object -First 1
 Expect-AvailabilityRejection "$taskApi/personal/catalog/leaf-green/$($taskOther.id)" $taskAuth (@{revision=$taskState.revision;hide_unavailable=$true;unavailable_pokemon_ids=@()}|ConvertTo-Json)
 $taskOldState=$taskState
 $taskState=Save-LeafGreen $taskState $taskState.default_unavailable_pokemon_ids
 Expect-AvailabilityRejection $taskUrl $taskAuth (@{revision=$taskOldState.revision;hide_unavailable=$false;unavailable_pokemon_ids=@()}|ConvertTo-Json)
 $taskState=Save-LeafGreen $taskState @($taskState.unavailable_pokemon_ids|Where-Object {$_ -ne 179})
 $taskState=Invoke-RestMethod $taskUrl -Headers $taskAuth
 if(@($taskState.unavailable_pokemon_ids).Count -ne 170 -or 179 -in $taskState.unavailable_pokemon_ids){throw 'Edited list did not persist'}
 $taskState=Save-LeafGreen $taskState @()
 if($taskState.unavailable_pokemon_ids){throw 'An explicitly empty list must not restore defaults'}
 $taskState=Save-LeafGreen $taskState $taskState.default_unavailable_pokemon_ids
 $taskSource=Split-Path $PSScriptRoot -Parent
 $taskRevision=git -C $taskSource merge-base HEAD upstream/master
 $env:ENVIRONMENT='local';$env:DATABASE_NAME=$DatabaseName;$env:DATABASE_PORT='9876';$env:DATABASE_PASSWORD=$taskSecrets.database_password;$env:JWT_SECRET=$taskSecrets.jwt_secret
 & "$taskRoot/runtime/personal-seed-import.exe" "$taskSource/seeds" $taskRevision
 if($LASTEXITCODE -ne 0){throw 'Trial reference import failed'}
 $taskReload=Invoke-RestMethod $taskUrl -Headers $taskAuth
 if(-not $taskReload.hide_unavailable -or @($taskReload.unavailable_pokemon_ids).Count -ne 171){throw 'Reference import discarded availability rules'}
 $taskExport=Invoke-RestMethod "$taskApi/personal/catalog" -Headers $taskAuth
 $taskExportBody=@{revision=$taskExport.revision;settings=$taskExport.settings;overrides=$taskExport.overrides}|ConvertTo-Json -Depth 40
 $null=Invoke-RestMethod "$taskApi/personal/catalog" -Method Post -Headers $taskAuth -ContentType 'application/json; charset=utf-8' -Body ([Text.Encoding]::UTF8.GetBytes($taskExportBody))
 if(@((Invoke-RestMethod $taskUrl).unavailable_pokemon_ids).Count -ne 171){throw 'Configuration export/import lost unavailable list'}
 $taskAfter=Invoke-RestMethod "$taskApi/users/cache117"
 foreach($taskBeforeDex in $taskProfile.dexes){$taskMatch=$taskAfter.dexes|Where-Object id -eq $taskBeforeDex.id;if($taskMatch.caught -ne $taskBeforeDex.caught -or $taskMatch.total -ne $taskBeforeDex.total){throw 'Collection progress changed'}}
 Write-Output 'LeafGreen-only scope, reviewed 171-species list, obtainable ticket/breeding species, invalid and stale edits, edited/empty lists, import preservation and unchanged progress passed.'
}finally{
 $taskLatest=Invoke-RestMethod "$taskApi/personal/catalog" -Headers $taskAuth
 $taskOriginal.revision=$taskLatest.revision
 $taskRestore=@{revision=$taskOriginal.revision;settings=$taskOriginal.settings;overrides=$taskOriginal.overrides}|ConvertTo-Json -Depth 40
 $null=Invoke-RestMethod "$taskApi/personal/catalog" -Method Post -Headers $taskAuth -ContentType 'application/json; charset=utf-8' -Body ([Text.Encoding]::UTF8.GetBytes($taskRestore))
}
