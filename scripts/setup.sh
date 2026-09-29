#!/usr/bin/env bash
# Scarica la libreria nativa Vosk (C) e il modello linguistico italiano
# dentro il progetto, senza toccare il sistema e senza sudo.
#
#   libreria : third_party/vosk/{libvosk.so,vosk_api.h}
#   modello  : models/vosk-model-small-it-0.22/
set -euo pipefail

cd "$(dirname "$0")/.."

VOSK_VERSION="0.3.45"
MODEL_NAME="vosk-model-small-it-0.22"

mkdir -p third_party/vosk models

# --- libreria nativa -------------------------------------------------------
# L'archivio ufficiale "vosk-linux-x86" contiene la libreria a 32 bit, quindi
# per x86_64 si prende quella dentro la wheel Python, che è la stessa
# libreria compilata per l'architettura corrente. L'header C è indipendente
# dall'architettura.
if [[ -f third_party/vosk/libvosk.so && -f third_party/vosk/vosk_api.h ]]; then
	echo "[ok] libreria Vosk già presente"
else
	echo "[..] scarico libreria Vosk ${VOSK_VERSION}"
	tmp=$(mktemp -d)
	trap 'rm -rf "$tmp"' EXIT

	case "$(uname -m)" in
	x86_64)
		asset="vosk-${VOSK_VERSION}-py3-none-linux_x86_64.whl"
		;;
	aarch64 | arm64)
		asset="vosk-${VOSK_VERSION}-py3-none-manylinux2014_aarch64.whl"
		;;
	*)
		echo "Architettura $(uname -m) non supportata da questo script."
		echo "Scarica libvosk.so da https://github.com/alphacep/vosk-api/releases"
		exit 1
		;;
	esac

	curl -fsSL -o "$tmp/vosk.whl" \
		"https://github.com/alphacep/vosk-api/releases/download/v${VOSK_VERSION}/${asset}"
	curl -fsSL -o "$tmp/vosk-src.zip" \
		"https://github.com/alphacep/vosk-api/releases/download/v${VOSK_VERSION}/vosk-linux-x86-${VOSK_VERSION}.zip"

	unzip -q "$tmp/vosk.whl" -d "$tmp/whl"
	unzip -q "$tmp/vosk-src.zip" -d "$tmp/src"

	cp "$tmp"/whl/vosk/libvosk.so third_party/vosk/libvosk.so
	cp "$tmp"/src/*/vosk_api.h third_party/vosk/vosk_api.h
	chmod +x third_party/vosk/libvosk.so
	echo "[ok] libreria Vosk installata in third_party/vosk"
fi

# --- modello linguistico ---------------------------------------------------
if [[ -f "models/${MODEL_NAME}/am/final.mdl" ]]; then
	echo "[ok] modello ${MODEL_NAME} già presente"
else
	echo "[..] scarico modello ${MODEL_NAME} (48 MB)"
	tmp=$(mktemp -d)
	curl -fsSL -o "$tmp/model.zip" \
		"https://alphacephei.com/vosk/models/${MODEL_NAME}.zip"
	unzip -q "$tmp/model.zip" -d models/
	rm -rf "$tmp"
	echo "[ok] modello installato in models/${MODEL_NAME}"
fi

echo
echo "Fatto. Compila con: make run   (oppure ./build.sh)"
