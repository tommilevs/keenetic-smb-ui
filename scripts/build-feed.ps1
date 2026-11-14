$ErrorActionPreference = 'Stop'
$root = 'C:\workspace\dsh\projects\keenetic-smb-ui'
$dist = Join-Path $root 'dist'
$feedSrc = Join-Path $dist 'feed'
$rel = Join-Path $feedSrc 'release'
New-Item -ItemType Directory -Force -Path $rel | Out-Null
Get-ChildItem $rel -File | Remove-Item -Force

$version = '1.5.0'
$appArches = @('aarch64-3.10', 'armv7-3.2', 'mipsel-3.4', 'mips-3.4', 'x64-3.2', 'x86-2.6')

# 1. copy app packages
foreach ($a in $appArches) {
  $src = Join-Path $dist "keenetic-smb-ui_${version}_${a}.ipk"
  if (Test-Path $src) { Copy-Item $src (Join-Path $rel (Split-Path $src -Leaf)) }
}

# 2. copy vendored dependencies
$closure = Get-Content (Join-Path $feedSrc 'closure.json') -Raw | ConvertFrom-Json
$depRows = @()
foreach ($arch in $closure) {
  $dir = Join-Path $feedSrc $arch.arch
  foreach ($row in $arch.rows) {
    $src = Join-Path $dir $row.FileName
    if (Test-Path $src) {
      Copy-Item $src (Join-Path $rel $row.FileName) -Force
      $depRows += $row
    }
  }
}

# 3. build Packages index (each stanza separated by a blank line)
function Stanza([string[]]$lines) { return (($lines -join "`n") + "`n`n") }

$sb = New-Object System.Text.StringBuilder
foreach ($r in $depRows) {
  $full = Join-Path $rel $r.FileName
  $sha = (Get-FileHash $full -Algorithm SHA256).Hash.ToLower()
  $dep = if ($r.Depends) { $r.Depends } else { 'libc' }
  $stanza = Stanza @(
    "Package: $($r.Package)",
    "Version: $($r.Version)",
    "Depends: $dep",
    "Architecture: $($r.Architecture)",
    "Filename: $($r.FileName)",
    "Installed-Size: $(([math]::Round((Get-Item $full).Length / 1024)))",
    "Size: $((Get-Item $full).Length)",
    "SHA256sum: $sha",
    "Section: net",
    "License: AGPL-3.0-or-later",
    "Description: vendored dependency for keenetic-smb-ui"
  )
  [void]$sb.Append($stanza)
}
foreach ($a in $appArches) {
  $file = "keenetic-smb-ui_${version}_${a}.ipk"
  $full = Join-Path $rel $file
  if (-not (Test-Path $full)) { continue }
  $sha = (Get-FileHash $full -Algorithm SHA256).Hash.ToLower()
  $stanza = Stanza @(
    "Package: keenetic-smb-ui",
    "Version: $version",
    "Depends: libc, samba4-server, samba4-client",
    "Architecture: $a",
    "Filename: $file",
    "Installed-Size: $(([math]::Round((Get-Item $full).Length / 1024)))",
    "Size: $((Get-Item $full).Length)",
    "SHA256sum: $sha",
    "Section: net",
    "License: AGPL-3.0-or-later",
    "Description: SMB server manager with a web UI for Keenetic/Entware routers"
  )
  [void]$sb.Append($stanza)
}

$pkgPath = Join-Path $rel 'Packages'
[IO.File]::WriteAllText($pkgPath, $sb.ToString(), (New-Object Text.UTF8Encoding($false)))
$bytes = [IO.File]::ReadAllBytes($pkgPath)
$out = New-Object IO.MemoryStream
$gz = New-Object IO.Compression.GZipStream($out, [System.IO.Compression.CompressionMode]::Compress)
$gz.Write($bytes, 0, $bytes.Length); $gz.Close()
[IO.File]::WriteAllBytes((Join-Path $rel 'Packages.gz'), $out.ToArray())

$all = Get-ChildItem $rel -File
"release dir: {0} files, {1} MB" -f $all.Count, [math]::Round((($all | Measure-Object Length -Sum).Sum) / 1MB, 1)
$stanzas = ($sb.ToString() -split "`n`n" | Where-Object { $_.Trim() }).Count
"stanzas: $stanzas"
$blankSeparated = ($sb.ToString() -match "Description: [^\n]+\n\nPackage: ")
"blank-line separated: $blankSeparated"
