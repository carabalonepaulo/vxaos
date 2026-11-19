param(
    [string]$Version = ""
)

$ErrorActionPreference = "Stop"

$root = Split-Path $PSScriptRoot -Parent
Set-Location $root

if ([string]::IsNullOrWhiteSpace($Version)) {
    $Version = Get-Date -Format "yyyyMMdd-HHmmss"
}

$name = "vxaos-server-$Version"
$distRoot = Join-Path $root "dist"
$out = Join-Path $distRoot $name
$archive = Join-Path $distRoot "$name.7z"

if (-not (Test-Path (Join-Path $root "bin\vxaos-server.exe"))) {
    throw "bin\vxaos-server.exe not found; run 'just build' first"
}
if (-not (Test-Path (Join-Path $root "bin\vxaos-admin.exe"))) {
    throw "bin\vxaos-admin.exe not found; run 'just build' first"
}

New-Item -ItemType Directory -Force -Path $distRoot | Out-Null
if (Test-Path $out) {
    Remove-Item $out -Recurse -Force
}
New-Item -ItemType Directory -Force -Path $out | Out-Null

Copy-Item "bin\vxaos-server.exe" $out
Copy-Item "bin\vxaos-admin.exe" $out
Copy-Item "packaging\config.toml" (Join-Path $out "config.toml")
Copy-Item "packaging\README.txt" (Join-Path $out "README.txt")
Copy-Item "packaging\start-server.bat" (Join-Path $out "start-server.bat")
Copy-Item "content" (Join-Path $out "content") -Recurse

$gameData = Join-Path $out "game-data"
New-Item -ItemType Directory -Force -Path $gameData | Out-Null
Copy-Item "..\client\Data\*.rvdata2" $gameData

$dbDir = Join-Path $out "data"
New-Item -ItemType Directory -Force -Path $dbDir | Out-Null
Copy-Item "packaging\data-README.txt" (Join-Path $dbDir "README.txt")

$sevenZip = (Get-Command 7z -ErrorAction SilentlyContinue).Source
if (-not $sevenZip) {
    throw "7z.exe not found in PATH"
}

if (Test-Path $archive) {
    Remove-Item $archive -Force
}
Push-Location $distRoot
try {
    & $sevenZip a -t7z -mx=9 $archive $name | Out-Null
    if ($LASTEXITCODE -ne 0) {
        throw "7z failed with exit code $LASTEXITCODE"
    }
} finally {
    Pop-Location
}

$size = (Get-Item $archive).Length
Write-Host ("created {0} ({1:N1} MB)" -f $archive, ($size / 1MB))
