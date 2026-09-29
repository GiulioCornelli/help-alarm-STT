# TASKS — help-alarm-STT

Sistema in Go che ascolta il microfono, trascrive l'audio con Vosk (STT offline)
e segnala la parola chiave "aiuto" (o altre configurabili).

## Fase 0 — Setup
- [x] 0.1 Inizializzare il modulo Go (`go mod init`) e la struttura delle cartelle
- [x] 0.2 Scaricare il modello `vosk-model-small-it-0.22` (48 MB) in `models/`
- [x] 0.3 Aggiungere le dipendenze: `github.com/gen2brain/malgo`, `github.com/alphacep/vosk-api/go`
- [x] 0.4 Installare la libreria nativa `libvosk.so` + `vosk_api.h` in `third_party/vosk/`
      e cablare i flag cgo nel Makefile

## Fase 1 — Configurazione
- [x] 1.1 Creare `config.json` con: keywords, model_path, device_id, sample_rate,
      log_file, log_level, log_transcriptions
- [x] 1.2 Implementare `internal/config`: caricamento JSON, default, validazione errori espliciti

## Fase 2 — Cattura audio
- [x] 2.1 Implementare `internal/audio`: apertura microfono via malgo (ALSA + PulseAudio)
- [x] 2.2 Forzare formato PCM s16 mono 16000 Hz (richiesto da Vosk)
- [x] 2.3 Inoltrare i frame PCM su un canale, senza bloccare la callback audio
- [x] 2.4 Gestire chiusura pulita dello stream (context + Stop al termine)
- [x] 2.5 Aggiungere `-devices` per elencare i microfoni

## Fase 3 — Trascrizione STT
- [x] 3.1 Implementare `internal/stt`: wrapper sottile su Vosk (NewModel / NewRecognizer)
- [x] 3.2 Goroutine dedicata: legge dal canale audio, chiama `AcceptWaveform`, fa l'unmarshal del JSON
- [x] 3.3 Estrarre `text` da risultati finali e `partial` dalle ipotesi, inoltrare al matcher
- [x] 3.4 Liberare correttamente recognizer e model allo shutdown (con `Flush` finale)

## Fase 4 — Analisi del testo
- [x] 4.1 Implementare `internal/keyword`: normalizzazione (lowercase, rimozione punteggiatura, spazi)
- [x] 4.2 Matching a confine di parola, anche per frasi multi-parola
- [x] 4.3 Goroutine dedicata + canale per non rallentare la trascrizione

## Fase 5 — Allarme e logging
- [x] 5.1 `internal/alert`: interfaccia `Alerter` — oggi solo `TerminalAlerter`, domani GPIO
- [x] 5.2 Su match: stampa a terminale `aiuto rilevato, luce accesa`
- [x] 5.3 `internal/logging`: `log/slog` su **due** file (messaggi normali e soli
      allarmi) + stream su console (testo)
- [x] 5.4 Log di tutti gli eventi: avvio, microfono, match, trascrizioni, arresti, errori

## Fase 6 — Integrazione
- [x] 6.1 `main.go`: cablaggio config → audio → stt → keyword → alert → log
- [x] 6.2 Gestione segnali SIGINT/SIGTERM per spegnere tutto senza perdere il log
- [x] 6.3 Flag CLI: `-config` (default `config.json`), `-devices`
- [x] 6.4 Cooldown per parola chiave (`cooldown_seconds`, default 3) per evitare
      la raffica di allarmi ripetuti

## Fase 7 — Verifica
- [x] 7.1 `gofmt` + `go vet` puliti
- [x] 7.2 Build del binario
- [x] 7.3 Test automatici: keyword, config, alert, parsing JSON Vosk
- [x] 7.4 Avvio reale col microfono: apertura del device, streaming, trascrizione funzionanti
- [ ] 7.5 Verificare end-to-end che la parola "aiuto" faccia scattare l'allarme
- [x] 7.6 `README.md` con istruzioni di build, configurazione ed esempi di keyword

## Bug trovati e corretti durante i test
- Il pacchetto `github.com/gen2brain/mal-go` indicato nei requisiti non esiste
  più: il progetto corretto è `github.com/gen2brain/malgo`.
- Vosk restituisce `partial` come **stringa** (`{"partial": "..."}`), non come
  oggetto: il parsing iniziale falliva e riempiva il log di errori.
- Il dispositivo veniva aperto alla **frequenza nativa (32 kHz)** invece dei
  16 kHz richiesti, perché `SampleRate` non veniva impostato esplicitamente.
  Ora viene imposto e **verificato** dopo l'apertura, per non produrre silenzio
  spacciato in caso di driver che rifiutano la conversione.

## Note
- Le parole chiave sono cercate sia nelle ipotesi parziali sia nelle frasi
  definitive; il `cooldown` (default 3 s) limita a uno ogni 3 secondi gli
  allarmi della stessa parola chiave. Con `cooldown_seconds: 0` si torna al
  comportamento senza filtro.
- I log sono separati: `logs/help-alarm.log` per i messaggi normali,
  `logs/help-alarm-alerts.log` solo per gli allarmi.
- Solo microfono: nessuna modalità da file WAV.
- Il GPIO/luce **non** viene implementato ora, solo l'interfaccia `Alerter`
  perché sia un'estensione di poche righe.
