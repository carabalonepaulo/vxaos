# Launches a client with the test-input bridge enabled.
#
# The bridge is part of VXAOS.dll and only starts when VXAOS_TEST_INPUT is set
# for the child process. It listens on 127.0.0.1 and never moves the physical
# cursor or sends global input. Use bin/vxaos-input.exe to drive it.
param(
  [int]$Port = 5001,
  [string]$Game = (Join-Path $PSScriptRoot '..\..\Client\Game.exe'),
  [string]$Work = (Join-Path $PSScriptRoot '..\..\Client')
)

$ErrorActionPreference = 'Stop'
$serverRoot = (Resolve-Path (Join-Path $PSScriptRoot '..')).Path
$cli = Join-Path $serverRoot 'bin\vxaos-input.exe'
if (-not (Test-Path $cli)) {
  Push-Location $serverRoot
  try {
    & go build -o bin/vxaos-input.exe ./cmd/vxaos-input
    if ($LASTEXITCODE -ne 0) {
      throw "go build exited with $LASTEXITCODE"
    }
  } finally {
    Pop-Location
  }
}

$env:VXAOS_TEST_INPUT = [string]$Port
try {
  $process = Start-Process -FilePath $Game -WorkingDirectory $Work -PassThru
} finally {
  Remove-Item Env:VXAOS_TEST_INPUT -ErrorAction SilentlyContinue
}

Write-Host "started Game.exe pid=$($process.Id) input=127.0.0.1:$Port"
Write-Host "drive it with: $cli -addr 127.0.0.1:$Port <command>"
