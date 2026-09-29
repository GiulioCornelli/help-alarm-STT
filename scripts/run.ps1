<#
.SYNOPSIS
    Compila e avvia help-alarm su Windows.

.DESCRIPTION
    Wrapper di build.ps1 pensato per l'uso quotidiano: un comando e il
    programma parte con il microfono in ascolto.

    Da usare dentro MSYS2 (Git Bash o MINGW64), non da PowerShell di sistema:
    cgo richiede gcc.
#>

[CmdletBinding()]
param(
    [string]$Config = 'config.json',
    [switch]$Devices
)

$ErrorActionPreference = 'Stop'

$ProjectRoot = Split-Path -Parent $PSScriptRoot
$DistDir     = Join-Path $ProjectRoot 'dist'
$Exe         = Join-Path $DistDir 'help-alarm.exe'

# Ricompila solo se il sorgente è più recente dell'eseguibile: altrimenti
# l'avvio quotidiano non aspetterebbe la compilazione.
$needsBuild = $true
if (Test-Path $Exe) {
    $exeTime  = (Get-Item $Exe).LastWriteTime
    $newer    = Get-ChildItem -Path $ProjectRoot -Recurse -Include *.go,go.mod,go.sum |
                Where-Object { $_.FullName -notlike '*\dist\*' } |
                Where-Object { $_.LastWriteTime -gt $exeTime }
    $needsBuild = [bool]$newer
}

if ($needsBuild) {
    & (Join-Path $PSScriptRoot 'build.ps1')
}

if (-not (Test-Path $Exe)) { throw "build fallito: $Exe non trovato" }

$configPath = if ([IO.Path]::IsPathRooted($Config)) { $Config } else { Join-Path $ProjectRoot $Config }

$arguments = if ($Devices) { @('-devices') } else { @('-config', $configPath) }

Write-Host "==> avvio: $Exe $($arguments -join ' ')" -ForegroundColor Cyan
Push-Location $ProjectRoot
try {
    # LastExitCode viene ereditato: Ctrl+C e l'uscita pulita restano visibili.
    & $Exe @arguments
    exit $LastExitCode
}
finally {
    Pop-Location
}
