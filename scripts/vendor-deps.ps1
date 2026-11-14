$ErrorActionPreference = 'Stop'

# Entware arches that real Keenetic devices use, plus the package Architecture
# strings each feed reports (used when generating our own Packages index).
$Arches = @(
  'aarch64-k3.10',
  'armv7sf-k3.2',
  'mipselsf-k3.4',
  'mipssf-k3.4'
)
$OutDir = 'C:\workspace\dsh\projects\keenetic-smb-ui\dist\feed'
New-Item -ItemType Directory -Force -Path $OutDir | Out-Null

$report = @()

function Parse-Index([string]$arch) {
  $tmp = Join-Path $env:TEMP "P-$arch.gz"
  Remove-Item $tmp -Force -ErrorAction SilentlyContinue
  curl.exe -sS -L -o $tmp "https://bin.entware.net/$arch/Packages.gz"
  if (-not (Test-Path $tmp) -or (Get-Item $tmp).Length -lt 1000) { throw "index download failed for $arch" }
  $fs = [IO.File]::OpenRead($tmp)
  $gz = New-Object IO.Compression.GZipStream($fs, [System.IO.Compression.CompressionMode]::Decompress)
  $sr = New-Object IO.StreamReader($gz)
  $txt = $sr.ReadToEnd(); $sr.Dispose()
  $pkgs = @{}
  $cur = $null; $fields = @{}
  foreach ($line in ($txt -split "`n")) {
    $line = $line.TrimEnd("`r")
    if ($line -match '^([A-Za-z][A-Za-z0-9-]*):\s?(.*)$') {
      $k = $Matches[1]; $v = $Matches[2]
      if ($k -eq 'Package') { if ($cur) { $pkgs[$cur] = $fields }; $cur = $v; $fields = @{} }
      else { $fields[$k] = $v }
    }
    elseif ($line -eq '' -and $cur) { $pkgs[$cur] = $fields; $cur = $null; $fields = @{} }
  }
  if ($cur) { $pkgs[$cur] = $fields }
  return $pkgs
}

foreach ($arch in $Arches) {
  $pkgs = Parse-Index $arch
  $roots = @('samba4-server', 'samba4-client')
  $need = New-Object System.Collections.Generic.HashSet[string]
  $stack = New-Object System.Collections.Generic.Stack[string]
  foreach ($r in $roots) {
    if ($pkgs.ContainsKey($r)) { [void]$need.Add($r); $stack.Push($r) }
    else { Write-Host "  missing root $r in $arch" }
  }
  while ($stack.Count) {
    $n = $stack.Pop()
    $deps = $pkgs[$n]['Depends']
    if (-not $deps) { continue }
    foreach ($dep in ($deps -split ',')) {
      $name = (($dep.Trim() -split ' ')[0])
      if (-not $name) { continue }
      if ($name -eq 'libc') { continue }
      if ($pkgs.ContainsKey($name) -and -not $need.Contains($name)) { [void]$need.Add($name); $stack.Push($name) }
    }
  }
  $list = @($need | Sort-Object)
  $archDir = Join-Path $OutDir $arch
  New-Item -ItemType Directory -Force -Path $archDir | Out-Null
  $rows = @(); $total = 0
  foreach ($name in $list) {
    $p = $pkgs[$name]
    $fn = $p['Filename']
    $fileName = Split-Path $fn -Leaf
    $dest = Join-Path $archDir $fileName
    if (-not (Test-Path $dest)) { curl.exe -sS -L -o $dest "https://bin.entware.net/$arch/$fn" }
    $size = (Get-Item $dest).Length
    $total += $size
    $rows += [pscustomobject]@{
      Package      = $name
      Version      = $p['Version']
      Depends      = $p['Depends']
      Architecture = $p['Architecture']
      FileName     = $fileName
      Size         = $size
    }
  }
  $report += [pscustomobject]@{ arch = $arch; packages = $rows.Count; totalMB = [math]::Round($total / 1MB, 1); rows = $rows }
  Write-Host "=== $arch : $($rows.Count) packages, $([math]::Round($total/1MB,1)) MB"
}

$report | ConvertTo-Json -Depth 5 | Set-Content -Encoding utf8 (Join-Path $OutDir 'closure.json')
Write-Host "closure saved"
