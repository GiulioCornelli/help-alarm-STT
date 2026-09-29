# Build di help-alarm.
#
# I binding Go di Vosk chiamano la libreria nativa C tramite cgo, quindi
#servono i flag di compilazione e linkaggio. La libreria vive dentro il
# progetto (third_party/vosk), perciò non serve installare nulla a sistema.

MODULE      := github.com/giulio/help-alarm-STT
VOSK_DIR    := $(CURDIR)/third_party/vosk
BIN         := help-alarm

CGO_CFLAGS  := -I$(VOSK_DIR)
CGO_LDFLAGS := -L$(VOSK_DIR) -lvosk -Wl,-rpath,$(VOSK_DIR)

export CGO_CFLAGS
export CGO_LDFLAGS

.PHONY: all setup build run devices test fmt vet clean

all: build

setup:
	./scripts/setup.sh

build:
	@test -f $(VOSK_DIR)/libvosk.so || { echo "libreria Vosk mancante: esegui 'make setup'"; exit 1; }
	go build -o $(BIN) $(MODULE)

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
	rm -f $(BIN)
