$ErrorActionPreference = 'Stop'
$root = Split-Path -Parent $PSScriptRoot
$dist = Join-Path $root 'dist'
New-Item -ItemType Directory -Force -Path $dist | Out-Null
Push-Location $root
try {
    $windres = Get-Command windres -ErrorAction SilentlyContinue
    if ($null -ne $windres) {
        & $windres.Source --input 'cmd\xint-lanpulse\windows.rc' --output 'cmd\xint-lanpulse\rsrc_windows_amd64.syso' --output-format coff --target pe-x86-64
        if ($LASTEXITCODE -ne 0) { throw 'Windows 图标资源编译失败。' }
    }
    go build -trimpath -tags 'desktop,production' -ldflags '-s -w -H=windowsgui' -o (Join-Path $dist 'XinT-LanPulse.exe') ./cmd/xint-lanpulse
    if ($LASTEXITCODE -ne 0) { throw 'Windows 程序构建失败。' }
    Copy-Item -LiteralPath (Join-Path $root 'assets\icon.ico') -Destination (Join-Path $dist 'icon.ico') -Force
}
finally { Pop-Location }
