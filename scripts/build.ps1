<#
.SYNOPSIS
    Compila help-alarm per Windows senza bisogno di GNU Make.

.DESCRIPTION
    Usare dentro MSYS2 (Git Bash o MINGW64), perché serve gcc per cgo:
    il compilatore di Visual Studio non va bene con il cgo di Go.

    Produce dist\help-alarm.exe con le DLL di Vosk accanto, così l'eseguibile
    parte con un doppio clic.
#>

[CmdletBinding()]
param(
    [switch]$Tests
)

$ErrorActionPreference = 'Stop'

$ProjectRoot = Split-Path -Parent $PSScriptRoot
$VoskDir     = Join-Path $ProjectRoot 'third_party\vosk'
$RuntimeDir  = Join-Path $VoskDir 'bin'
$DistDir     = Join-Path $ProjectRoot 'dist'

if (-not (Test-Path (Join-Path $VoskDir 'libvosk.dll'))) {
    throw "libreria Vosk mancante: esegui prima scripts\setup.ps1"
}

# cgo su Windows usa gcc, non MSVC: senza questa variabile Go non riesce a
# linkare la libreria nativa.
$env:CGO_ENABLED  = '1'
$env:CC           = 'gcc'
$env:CGO_CFLAGS   = "-I$(($VoskDir -replace '\\','/'))"
$env:CGO_LDFLAGS  = "-L$(($VoskDir -replace '\\','/')) -lvosk"

Push-Location $ProjectRoot
try {
    if ($Tests) {
        Write-Host '==> go test' -ForegroundColor Cyan
        go test ./...
        return
    }

    Write-Host '==> go build' -ForegroundColor Cyan
    New-Item -ItemType Directory -Force -Path $DistDir | Out-Null

    # Senza -trimpath il percorso di compilazione finisce dentro l'eseguibile.
    go build -trimpath -ldflags '-s -w' -o (Join-Path $DistDir 'help-alarm.exe') .

    # Le DLL devono stare accanto all'exe: Windows le cerca lì prima del PATH.
    Write-Host '==> copio le DLL di Vosk in dist' -ForegroundColor Cyan
    Copy-Item (Join-Path $VoskDir 'libvosk.dll')    $DistDir
    Copy-Item (Join-Path $RuntimeDir '*.dll')       $DistDir

    Write-Host ''
    Write-Host "Binario pronto: dist\help-alarm.exe" -ForegroundColor Green
}
finally {
    Pop-Location
}
