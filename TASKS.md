# TASKS — help-alarm-STT

Sistema in Go che ascolta il microfono, trascrive l'audio con Vosk (STT offline)
e segnala la parola chiave "aiuto" (o altre configurabili).

## Fase 0 — Setup
- [ ] 0.1 Inizializzare il modulo Go (`go mod init`) e la struttura delle cartelle
- [ ] 0.2 Scaricare il modello `vosk-model-small-it-0.22` (48 MB) in `models/`
- [ ] 0.3 Aggiungere le dipendenze: `github.com/gen2brain/mal-go`, `github.com/alphacep/vosk-api/go`

## Fase 1 — Configurazione
- [ ] 1.1 Creare `config.json` con: keywords, model_path, device, log_file, sample_rate
- [ ] 1.2 Implementare `internal/config`: caricamento JSON, default, validazione errori espliciti

## Fase 2 — Cattura audio
- [ ] 2.1 Implementare `internal/audio`: apertura microfono via mal-go (backend ALSA)
- [ ] 2.2 Forzare formato PCM s16 mono 16000 Hz (richiesto da Vosk)
- [ ] 2.3 Inoltrare i frame PCM su un canale, senza bloccare la callback audio
- [ ] 2.4 Gestire chiusura pulita dello stream (context + Close al termine)

## Fase 3 — Trascrizione STT
- [ ] 3.1 Implementare `internal/stt`: wrapper sottile su Vosk (NewModel / NewRecognizer)
- [ ] 3.2 Goroutine dedicata: legge dal canale audio, chiama `AcceptWaveform`, fa l'unmarshal del JSON
- [ ] 3.3 Estrai `text` e `is_final` da ogni risultato, inoltra al matcher
- [ ] 3.4 Liberare correttamente recognizer e model allo shutdown

## Fase 4 — Analisi del testo
- [ ] 4.1 Implementare `internal/keyword`: normalizzazione (lowercase, rimozione punteggiatura, spazi)
- [ ] 4.2 Matching a confine di parola (non `strings.Contains` grezzo), anche per frasi multi-parola
- [ ] 4.3 Goroutine dedicata + canale per non rallentare la trascrizione (come da requisiti)

## Fase 5 — Allarme e logging
- [ ] 5.1 `internal/alert`: interfaccia `Alerter` — oggi solo `TerminalAlerter`, domani GPIO
- [ ] 5.2 Su match: stampa a terminale `aiuto rilevato, luce accesa`
- [ ] 5.3 `internal/logging`: `log/slog` su file (JSON, timestamp, data/ora) + stream su stdout
- [ ] 5.4 Log di tutti gli eventi: avvio, match, trascrizioni (in debug), errori

## Fase 6 — Integrazione
- [ ] 6.1 `main.go`: cablaggio config → audio → stt → keyword → alert → log
- [ ] 6.2 Gestione segnali SIGINT/SIGTERM per spegnere tutto senza perdere il log
- [ ] 6.3 Flag CLI: `-config` (default `config.json`)

## Fase 7 — Verifica
- [ ] 7.1 `gofmt` + `go vet` puliti
- [ ] 7.2 Build del binario
- [ ] 7.3 Avvio reale con microfono: verificare che "aiuto" venga rilevato
- [ ] 7.4 Verificare il file di log (data, ora, messaggio)
- [ ] 7.5 `README.md` con istruzioni di build, configurazione ed esempi di keyword

## Note
- Evento emesso **a ogni match** (nessun cooldown), come richiesto.
- Solo microfono: nessuna modalità da file WAV.
- Il GPIO/luce **non** viene implementato ora, solo l'interfaccia per estenderlo dopo.
