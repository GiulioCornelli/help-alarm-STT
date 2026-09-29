# Build di help-alarm, multipiattaforma.
#
# I binding Go di Vosk chiamano la libreria nativa C tramite cgo, quindi
# servono i flag di compilazione e linkaggio. La libreria vive dentro il
# progetto (third_party/vosk), perciò non serve installare nulla a sistema.
#
# Su Windows (OS=Windows_NT) questo Makefile va usato dentro MSYS2, non dal
# Developer Command Prompt: cgo richiede gcc, che MSVC non sostituisce.

MODULE      := github.com/giulio/help-alarm-STT
VOSK_DIR    := $(CURDIR)/third_party/vosk
VOSK_BIN    := $(VOSK_DIR)/bin
DIST        := $(CURDIR)/dist

export CGO_ENABLED := 1

# --- piattaforma ------------------------------------------------------------
ifeq ($(OS),Windows_NT)

  BIN     := $(DIST)/help-alarm.exe
  # I separatori di percorso di MSYS2 (/c/Users/...) non vanno bene per gcc,
  # che su Windows vuole i percorsi nativi.
  WIN_DIR := $(subst /,\,$(VOSK_DIR))
  # -lvosk cerca libvosk.lib nell'elenco delle librerie di import.
  CGO_CFLAGS  := -I$(WIN_DIR)
  CGO_LDFLAGS := -L$(WIN_DIR) -lvosk
  # gcc di MSYS2 sta nel PATH del processo make, ma Go lo cerca anche da solo.
  export CC := gcc

  # Le DLL di Vosk devono trovarsi accanto all'exe: Windows le cerca lì prima
  # del PATH, e senza di queste l'avvio muore con "libvosk.dll non trovata".
  VOSK_DLLS := $(VOSK_DIR)/libvosk.dll \
               $(VOSK_BIN)/libstdc++-6.dll \
               $(VOSK_BIN)/libwinpthread-1.dll \
               $(VOSK_BIN)/libgcc_s_seh-1.dll

  SETUP_CMD  := powershell -ExecutionPolicy Bypass -File scripts/setup.ps1
  RM_BIN     := rm -f $(BIN) $(addprefix $(DIST)/,*.dll)

else

  BIN         := help-alarm
  CGO_CFLAGS  := -I$(VOSK_DIR)
  # -rpath evita di dover esportare LD_LIBRARY_PATH a ogni avvio.
  CGO_LDFLAGS := -L$(VOSK_DIR) -lvosk -Wl,-rpath,$(VOSK_DIR)

  SETUP_CMD  := ./scripts/setup.sh
  RM_BIN     := rm -f $(BIN)

endif

export CGO_CFLAGS
export CGO_LDFLAGS

.PHONY: all setup build run devices test fmt vet clean \
	docker-build docker-up docker-down docker-logs

all: build

setup:
	$(SETUP_CMD)

build:
	@test -f $(VOSK_DIR)/vosk_api.h || { echo "libreria Vosk mancante: esegui 'make setup'"; exit 1; }
	@mkdir -p $(DIST)
	@mkdir -p $(VOSK_BIN)
	go build -trimpath -ldflags "-s -w" -o $(BIN) .
ifdef OS
	@cp -f $(VOSK_DLLS) $(DIST)/
	@echo "[ok] eseguibile in $(BIN) con le DLL di Vosk"
else
	@echo "[ok] eseguibile in $(BIN)"
endif

run: build
	./$(BIN) -config config.json

devices: build
	./$(BIN) -devices

test:
	go test ./...

fmt:
	gofmt -w .

vet:
	go vet ./...

clean:
	$(RM_BIN)

# --- Docker -----------------------------------------------------------------
# Percorso senza dipendenze locali: Go, gcc, libreria Vosk e modello sono
# dentro l'immagine. Serve solo Docker.
#
# Attenzione: su Windows l'audio non funziona in Docker, perché il container
# gira dentro WSL2 dove non esiste /dev/snd. Vedi la sezione Windows del README.

docker-build:
	docker compose build

docker-up:
	docker compose up -d
	docker compose logs -f

docker-down:
	docker compose down

docker-logs:
	docker compose logs -f
