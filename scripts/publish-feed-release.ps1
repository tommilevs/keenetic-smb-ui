$ErrorActionPreference = 'Stop'

# Take the GitHub token from the machine's git credential store instead of
# hardcoding it: the previous PAT was revoked and scripts must not carry secrets.
$env:GCM_INTERACTIVE = 'never'
$env:GIT_TERMINAL_PROMPT = '0'
$credLines = "protocol=https`nhost=github.com`n`n" | git credential fill 2>$null
$token = ($credLines -split "`n" | Where-Object { $_ -like 'password=*' }) -replace 'password=', ''
if (-not $token) { throw 'no GitHub credential in the store' }

$headers = @{ Authorization = "token $token"; 'User-Agent' = 'dsh'; Accept = 'application/vnd.github+json' }
$repo = 'tommilevs/keenetic-smb-ui'
$rel = 'C:\workspace\dsh\projects\keenetic-smb-ui\dist\feed\release'
$notes = [IO.File]::ReadAllText('C:\workspace\dsh\projects\keenetic-smb-ui\dist\feed\release-notes.md', [Text.Encoding]::UTF8)

$payload = @{ tag_name = 'v1.5.0'; name = 'v1.5.0 - vendored dependencies (opkg feed)'; body = $notes; draft = $false; prerelease = $false } | ConvertTo-Json -Compress
$relObj = Invoke-RestMethod "https://api.github.com/repos/$repo/releases" -Method Post -Headers $headers -Body ([Text.Encoding]::UTF8.GetBytes($payload)) -ContentType 'application/json; charset=utf-8' -TimeoutSec 30
Write-Host "release: $($relObj.tag_name) id=$($relObj.id)"

$files = Get-ChildItem $rel -File | Sort-Object Name
$i = 0
$failed = 0
foreach ($f in $files) {
  $i++
  $url = "https://uploads.github.com/repos/$repo/releases/$($relObj.id)/assets?name=$([uri]::EscapeDataString($f.Name))"
  try {
    Invoke-RestMethod $url -Method Post -Headers $headers -ContentType 'application/octet-stream' -InFile $f.FullName -TimeoutSec 300 | Out-Null
    if (($i % 20) -eq 0 -or $f.Name -like 'Packages*' -or $f.Name -like 'keenetic-smb-ui_*') { Write-Host "[$i/$($files.Count)] $($f.Name)" }
  }
  catch {
    $failed++
    Write-Host "FAIL $($f.Name): $($_.Exception.Message)"
  }
}
Write-Host "uploaded $i files, failed $failed"
