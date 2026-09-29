<#
.SYNOPSIS
    Prepara il progetto su Windows: scarica la libreria nativa Vosk e il
    modello linguistico italiano.

.DESCRIPTION
    Il binario Go usa cgo, quindi su Windows non si può compilare con il
    compilatore di Visual Studio: serve gcc, che in MSYS2 si chiama mingw-w64.
    Ecco perché questo progetto si builda dentro MSYS2 (o WSL) e non con
    "Developer Command Prompt".

    Questo script scarica:

      libreria : third_party/vosk/{libvosk.dll,libvosk.lib,vosk_api.h}
      runtime  : third_party/vosk/bin/{libstdc++-6,libwinpthread-1,libgcc_s_seh-1}.dll
      modello  : models/vosk-model-small-it-0.22/

    Le tre DLL di runtime servono accanto a help-alarm.exe: senza di esse
    Windows apre l'allarme con "libvosk.dll non trovata" e basta.

.EXAMPLE
    powershell -ExecutionPolicy Bypass -File scripts\setup.ps1
#>

[CmdletBinding()]
param()

$ErrorActionPreference = 'Stop'

# Su Windows PowerShell 5.1 la TLS 1.2 non è il default: senza questa riga
# il download del modello fallisce con un errore di protocollo poco chiaro.
[Net.ServicePointManager]::SecurityProtocol = [Net.SecurityProtocolType]::Tls12

$VoskVersion = '0.3.45'
$ModelName   = 'vosk-model-small-it-0.22'

$ProjectRoot = Split-Path -Parent $PSScriptRoot
$VoskDir     = Join-Path $ProjectRoot 'third_party\vosk'
$RuntimeDir  = Join-Path $VoskDir 'bin'
$ModelsDir   = Join-Path $ProjectRoot 'models'
$TempDir     = Join-Path ([IO.Path]::GetTempPath()) ('help-alarm-' + [guid]::NewGuid())

function Write-Step($Text) { Write-Host "==> $Text" -ForegroundColor Cyan }
function Write-Ok($Text)   { Write-Host "    [ok] $Text" -ForegroundColor Green }

New-Item -ItemType Directory -Force -Path $VoskDir, $RuntimeDir, $ModelsDir | Out-Null

try {
    # --- libreria nativa ----------------------------------------------------
    if ((Test-Path (Join-Path $VoskDir 'libvosk.dll')) -and
        (Test-Path (Join-Path $VoskDir 'libvosk.lib')) -and
        (Test-Path (Join-Path $VoskDir 'vosk_api.h'))) {
        Write-Ok 'libreria Vosk già presente'
    }
    else {
        Write-Step "scarico libreria Vosk $VoskVersion"

        # vosk-win64 è l'unico archivio Windows ufficiale e contiene già
        # l'header C, la libreria di import e le DLL di runtime.
        $url = "https://github.com/alphacep/vosk-api/releases/download/v$VoskVersion/vosk-win64-$VoskVersion.zip"
        $zip = Join-Path $TempDir 'vosk.zip'

        Invoke-WebRequest -Uri $url -OutFile $zip -UseBasicParsing
        Expand-Archive -Path $zip -DestinationPath $TempDir -Force

        $root = Join-Path $TempDir "vosk-win64-$VoskVersion"
        Copy-Item (Join-Path $root 'vosk_api.h')  $VoskDir
        Copy-Item (Join-Path $root 'libvosk.lib') $VoskDir
        Copy-Item (Join-Path $root 'libvosk.dll') $VoskDir

        # Le DLL di cui libvosk.dll ha bisogno vanno in bin/ e poi, in fase di
        # build, vengono copiate accanto all'eseguibile.
        foreach ($dll in 'libstdc++-6.dll', 'libwinpthread-1.dll', 'libgcc_s_seh-1.dll') {
            Copy-Item (Join-Path $root $dll) $RuntimeDir
        }
        Write-Ok 'libreria Vosk installata in third_party\vosk'
    }

    # --- modello linguistico ------------------------------------------------
    if (Test-Path (Join-Path $ModelsDir "$ModelName\am\final.mdl")) {
        Write-Ok "modello $ModelName già presente"
    }
    else {
        Write-Step "scarico modello $ModelName (48 MB)"

        $modelZip = Join-Path $TempDir 'model.zip'
        Invoke-WebRequest -Uri "https://alphacephei.com/vosk/models/$ModelName.zip" `
                          -OutFile $modelZip -UseBasicParsing
        Expand-Archive -Path $modelZip -DestinationPath $ModelsDir -Force
        Write-Ok "modello installato in models\$ModelName"
    }
}
finally {
    if (Test-Path $TempDir) { Remove-Item $TempDir -Recurse -Force -ErrorAction SilentlyContinue }
}

Write-Host ''
Write-Host 'Fatto. Ora, da Git Bash di MSYS2:' -ForegroundColor Green
Write-Host '  make run'
Write-Host ''
Write-Host 'Oppure senza make:' -ForegroundColor Green
Write-Host '  powershell -ExecutionPolicy Bypass -File scripts\build.ps1'
Write-Host '  powershell -ExecutionPolicy Bypass -File scripts\run.ps1'
