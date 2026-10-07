param([string]$OutputDirectory = 'build/rc')
$ErrorActionPreference = 'Stop'
$repoRoot = Split-Path $PSScriptRoot -Parent
$exePath = Join-Path $repoRoot 'build/bin/SubtitleDoctor.exe'
$info = [Diagnostics.FileVersionInfo]::GetVersionInfo($exePath)
if ($info.ProductVersion -ne '1.0.1' -or $info.FileVersion -ne '1.0.1') {throw 'Executable metadata must be 1.0.1'}
$out = [IO.Path]::GetFullPath((Join-Path $repoRoot $OutputDirectory))
New-Item -ItemType Directory -Path $out -Force | Out-Null
$stage = Join-Path $out ('stage-' + [guid]::NewGuid().ToString('N'))
New-Item -ItemType Directory -Path $stage | Out-Null
try {
  Copy-Item -LiteralPath $exePath -Destination (Join-Path $stage 'SubtitleDoctor.exe')
  foreach ($name in @('README.md','LICENSE','RELEASE_NOTES.md')) {Copy-Item -LiteralPath (Join-Path $repoRoot $name) -Destination (Join-Path $stage $name)}
  $zip = Join-Path $out 'SubtitleDoctor-v1.0.1-win-x64.zip'
  Compress-Archive -Path (Join-Path $stage '*') -DestinationPath $zip -Force -CompressionLevel Optimal
  $sha = (Get-FileHash -LiteralPath $zip -Algorithm SHA256).Hash.ToLowerInvariant()
  [IO.File]::WriteAllText("$zip.sha256", "$sha  $([IO.Path]::GetFileName($zip))`n", [Text.UTF8Encoding]::new($false))
  [pscustomobject]@{ZIP=$zip;SHA256=$sha}
} finally {
  if ([IO.Path]::GetFullPath($stage).StartsWith($out + [IO.Path]::DirectorySeparatorChar)) {Remove-Item -LiteralPath $stage -Recurse -Force}
}
