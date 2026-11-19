# Builds the 32-bit VXAOS.dll used by the VX Ace client.
#
# Requires Zig (tested with 0.17.0) on PATH. The client is a 32-bit process, so
# every object must target x86-windows-gnu.
param(
  [string]$Out = (Join-Path $PSScriptRoot '..\..\Client\System\VXAOS.dll')
)

$ErrorActionPreference = 'Stop'
if (-not (Get-Command zig -ErrorAction SilentlyContinue)) {
  throw 'zig was not found on PATH'
}

$sources = @(
  'main.cpp',
  'md5.c',
  'vxace_input.c',
  'vxace_mouse.c',
  'vxace_sdk.c',
  'vxace_socket.c',
  'vxaos_crypto.c',
  'vxaos_input_test.c'
)

Push-Location $PSScriptRoot
try {
  & zig c++ -target x86-windows-gnu -shared -O2 -DBUILD_DLL `
    -Wno-incompatible-function-pointer-types `
    -o $Out @sources -lws2_32 -lgdi32 -luser32
  if ($LASTEXITCODE -ne 0) {
    throw "zig exited with $LASTEXITCODE"
  }
} finally {
  Pop-Location
}

$out = (Resolve-Path $Out).Path
Write-Host "built $out"

# Fail the build if a required export disappeared.
$dump = & objdump -p $out
$required = @(
  'Crypto__sha256',
  'Crypto__hmac_sha256',
  'Crypto__pbkdf2_sha256',
  'Input__UTF8String',
  'Input__setup',
  'Input__update',
  'VxaosInputTest__mousePos',
  'SocketLib__send',
  'SocketLib__recv_non_block'
)
foreach ($name in $required) {
  if (-not ($dump -match [regex]::Escape($name))) {
    throw "missing export: $name"
  }
}
Write-Host "exports ok ($($required.Count) checked)"
