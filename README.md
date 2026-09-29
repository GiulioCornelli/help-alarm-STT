# help-alarm

Sistema in Go che ascolta il microfono, trascrive in tempo reale ciò che sente
usando il riconoscimento automatico del parlato (STT) **interamente in locale**
e segnala la presenza di una parola chiave, per esempio **"aiuto"**.

Quando la parola viene riconosciuta il programma stampa sul terminale:

```
15:42:07 aiuto rilevato, luce accesa
```

e scrive lo stesso messaggio in un file di log JSON con data e ora.

Nessun dato audio lascia il computer: modello linguistico e libreria STT sono
locali, non c'è alcuna chiamata di rete.

## Come funziona

Il programma è una catena di tre stadi, ognuno in un package separato:

```
  microfono                testo                          allarme
 ┌──────────┐  PCM s16   ┌──────────┐   frasi   ┌──────────┐
 │  audio   │──────────▶ │   stt    │──────────▶ │ keyword  │──▶ alert
 └──────────┘   16 kHz   │  (Vosk)  │  Go       └──────────┘
    monofon.   mono       └──────────┘  routine
```

1. **`internal/audio`** apre il microfono con [malgo](https://github.com/gen2brain/malgo)
   (wrapper Go di miniaudio) e produce flussi di campioni PCM grezzi.
   miniaudio si occupa di convertire quello che espone il microfono nel formato
   che Vosk richiede.

2. **`internal/stt`** usa [Vosk](https://alphacephei.com/vosk/) tramite i binding Go
   ufficiali (`github.com/alphacep/vosk-api/go`). Ogni 100 ms di audio viene
   passato a `AcceptWaveform`, che restituisce una trascrizione: una **ipotesi
   parziale** mentre la persona sta ancora parlando, il **testo definitivo** quando
   Vosk riconosce la fine della frase. Il JSON di risposta viene decodificato e
   ridotto al solo campo testuale.

3. **`internal/keyword`** confronta il testo con le parole chiave di
   `config.json` e **`internal/alert`** reagisce a ogni occorrenza trovata.

Ogni stadio gira nella propria goroutine collegata all'altro da un canale: la
trascrizione non blocca mai la cattura audio, e viceversa.

### Il thread audio non deve bloccarsi

miniaudio chiama la callback di cattura dal proprio thread audio a bassa
priorità: bloccarla provoca un "glitch", cioè un suono che si interrompe. Per
questo la callback di `internal/audio` copia il buffer e fa una `select` **non
bloccante** sul canale. Se chi legge è in ritardo, il frame viene scartato
invece di fermare l'audio — un frame perso (100 ms) è preferibile a un suono
interrotto.

## Struttura delle cartelle

```
help-alarm-STT/
├── main.go                  cablaggio: config → audio → stt → keyword → alert
├── config.json              configurazione utente
├── Makefile                 build, test e run con i flag cgo già impostati
├── scripts/setup.sh         scarica libreria nativa Vosk + modello italiano
├── TASKS.md                 elenco delle attività del progetto
│
├── internal/
│   ├── config/              lettura e validazione di config.json
│   ├── audio/               apertura del microfono e callback PCM
│   ├── stt/                 wrapper attorno a Vosk, parsing del JSON
│   ├── keyword/             normalizzazione del testo e ricerca parole chiave
│   ├── alert/               interfaccia Alerter + implementazione su terminale
│   └── logging/             log/slog su file JSON e su console testuale
│
├── models/                  modello Vosk (scaricato, non nel repository)
├── third_party/vosk/        libvosk.so e vosk_api.h (scaricati, non nel repository)
└── logs/                    log di esecuzione
```

## Requisiti

- Go 1.21 o superiore (sviluppato con Go 1.23)
- un compilatore C (`gcc` o `clang`): sia malgo sia Vosk usano cgo
- Linux con ALSA oppure PulseAudio
- un microfono

## Installazione

```bash
cd help-alarm-STT
./scripts/setup.sh
```

Lo script scarica due cose dentro il progetto, senza usare `sudo` e senza
installare nulla a sistema:

- **`third_party/vosk/libvosk.so`** e **`vosk_api.h`** — la libreria nativa C di
  Vosk, necessaria perché i binding Go la chiamano tramite cgo;
- **`models/vosk-model-small-it-0.22/`** (~48 MB) — il modello linguistico
  italiano, l'unico supportato per ora dal progetto.

Se lo script non è adatto alla tua architettura, scarica `libvosk.so` da
<https://github.com/alphacep/vosk-api/releases> e mettilo in `third_party/vosk/`.

## Build ed esecuzione

```bash
make build     # compila in ./help-alarm
make run       # compila e avvia con config.json
make test      # esegue i test
make devices   # elenca i microfoni disponibili
```

Il `Makefile` è necessario perché i binding Go di Vosk hanno bisogno dei flag
`CGO_CFLAGS` e `CGO_LDFLAGS` per trovare la libreria nativa. Se preferisci usare
`go` direttamente, esportali a mano:

```bash
export CGO_CFLAGS="-I$PWD/third_party/vosk"
export CGO_LDFLAGS="-L$PWD/third_party/vosk -lvosk -Wl,-rpath,$PWD/third_party/vosk"
go build -o help-alarm .
```

Se `go` non è nel `PATH`, come nel caso di un'installazione manuale in
`/usr/local/go/bin`, anteggi `export PATH=$PATH:/usr/local/go/bin`.

### Opzioni da riga di comando

```
-config percorso   file di configurazione da usare (default: config.json)
-devices           elenca i microfoni disponibili ed esci
```

## Configurazione

Tutto si configura da `config.json`:

```json
{
  "keywords": [
    "aiuto",
    "soccorso",
    "emergenza"
  ],
  "model_path": "models/vosk-model-small-it-0.22",
  "device_id": -1,
  "sample_rate": 16000,
  "log_file": "logs/help-alarm.log",
  "log_level": "info",
  "log_transcriptions": true
}
```

| Campo | Significato |
| --- | --- |
| `keywords` | parole o frasi da cercare nella trascrizione |
| `model_path` | cartella del modello Vosk |
| `device_id` | indice del microfono, `-1` = quello predefinito di sistema |
| `sample_rate` | deve essere `16000`, è il valore che Vosk richiede |
| `log_file` | percorso del file di log |
| `log_level` | `debug`, `info`, `warn` o `error` |
| `log_transcriptions` | se `true`, logga ogni frase riconosciuta |

Le chiavi omesse prendono il valore di default mostrato sopra. I percorsi
relativi sono risolti rispetto alla cartella che contiene il file di
configurazione, quindi si può lanciare il programma da qualsiasi directory.
Un file di configurazione errato fa fallire l'avvio con un messaggio esplicito,
non in silenzio.

### Parole chiave

Le parole chiave sono normalizzate allo stesso modo del testo trascritto:
minuscole, senza punteggiatura. `Aiuto`, `aiuto` e `AIUTO!` sono la stessa
chiave, e i duplicati vengono eliminati.

Il confronto avviene **a confine di parola**, non con una ricerca di sottostringa:
`aiuto` scatta su "per favore aiuto" ma non su "aiutami" né su "inaiuto".

Sono supportate anche le frasi di più parole, che scattano solo se le parole
compaiono consecutive:

```json
{ "keywords": ["aiuto", "per favore aiuto", "sono in pericolo"] }
```

### Scelta del microfono

```bash
make devices
```

```
dispositivi di cattura (indice da usare in config.json -> device_id):
    0   Discard all samples (playback) or generate zero samples (capture)
  * 1   Default Audio Device
    2   USB Device 0x46d:0x823, USB Audio
  * = dispositivo predefinito di sistema (corrisponde a device_id: -1)
```

Metti l'indice indicato in `device_id`. Utile se il predefinito di sistema
corrisponde a un output HDMI senza microfono utile, o se su un Raspberry Pi si
vuole scegliere esplicitamente il GPIO.

## Log

Il programma scrive su due destinazioni diverse:

- **`logs/help-alarm.log`** — record JSON, uno per riga, con timestamp. È la
  traccia durevole, pensata per essere letta da un programma esterno.
- **console** — righe di testo leggibili, per chi guarda il terminale.

```json
{"time":"2026-09-29T15:42:07.118Z","level":"INFO","msg":"trascrizione","testo":"per favore aiuto"}
{"time":"2026-09-29T15:42:07.120Z","level":"INFO","msg":"aiuto rilevato, luce accesa","timestamp":"2026-09-29T15:42:07.119Z","keyword":"aiuto","trascrizione":"per favore aiuto"}
```

Ogni evento di allarme riporta `timestamp` (l'istante esatto in cui è stato
rilevato), `keyword` (la parola configurata che ha scattato) e `trascrizione`
(la frase in cui è stata trovata).

Per non ripetere la riga sul terminale, l'allarme scrive il messaggio sul
terminale con una `fmt.Fprintf` e sul file con il logger: le due destinazioni
hanno logger separati proprio per questo.

## Comportamento della rilevazione

Le parole chiave vengono cercate sia nelle **ipotesi parziali** sia nelle frasi
definitive. Questo permette di reagire mentre la persona sta ancora parlando,
senza aspettare che finisca la frase, che è il comportamento desiderato in un
contesto di emergenza.

La conseguenza è che una parola chiave detta in una frase lunga può generare
più di un evento: "aiuto" viene riconosciuto come ipotesi parziale, poi di nuovo
quando Vosk conferma la frase. **Non c'è nessun filtro né tempo di attesa**: ogni
match emette un evento, come richiesto.

A ogni avvio e a ogni arresto, il programmo scrive cosa ha sentito:

```
INFO avvio help-alarm keywords="[aiuto soccorso emergenza]" modello=...
INFO microfono in ascolto dispositivo="Default Audio Device (predefinito) — 16000 Hz, s16 mono, periodo 100 ms"
INFO trascrizione testo="per favore aiuto"
INFO arresto in corso
INFO arresto completato
```

## Arresto

`Ctrl+C` (SIGINT) o `SIGTERM` fermano il programma in modo ordinato: lo stream
del microfono viene chiuso, la frase eventualmente rimasta a mezz'aria viene
trascritta e analizzata, e solo dopo vengono liberate le risorse native.

## Note tecniche

**Formato audio.** Vosk accetta solo PCM signed 16 bit little endian, mono, a
16 000 Hz. `internal/audio` chiede esattamente questo a miniaudio, che si fa
carico della conversione se il microfono espone altro (per esempio 48 kHz
float). Dopo l'apertura il programma **verifica** che il dispositivo sia
davvero ai valori richiesti e, se il driver ha rifiutato la conversione, si
ferma con un errore: continuare produrrebbe solo silenzio e silenzio sembrerebbe
"nessuna parola riconosciuta", che è il peggiore risultato possibile per un
sistema di allarme.

**Modello linguistico.** `vosk-model-small-it-0.22` è scelto perché gira su
Raspberry Pi e su computer modesti (~300 MB di memoria). Esiste anche
`vosk-model-it-0.22` (1.2 GB, accuratezza quasi doppia ma pensato per server):
basta scaricarlo e cambiare `model_path`.

**Libreria nativa.** I binding Go di Vosk chiamano `libvosk.so` tramite cgo,
quindi servono header e libreria al momento della compilazione. Il `Makefile`
imposta `-I`, `-L` e `-rpath` in modo che il binario trovi la libreria anche a
runtime senza configurare `LD_LIBRARY_PATH`.

## Sviluppo futuro: la luce

Il GPIO non è ancora implementato, come previsto dai requisiti. La base è però
già pronta: `internal/alert` espone l'interfaccia

```go
type Alerter interface {
    Name() string
    Trigger(Event) error
}
```

per ora implementata solo da `TerminalAlerter`. Per accendere una luce basterà
aggiungere una seconda implementazione (con `go-rpio` o `periph.io`) e passarla
a `trigger` in `main.go`, senza modificare nessun altro package. Allo stesso
modo, un attivatore che accenda un relè potrà sostituire la `fmt.Fprintf` sul
terminale.

## Test

```bash
make test
```

I test coprono la normalizzazione e il riconoscimento a confine di parola delle
parole chiave, il caricamento e la validazione della configurazione, il
formattatore del log e la struttura del JSON di Vosk. Il riconoscimento della
voce in sé richiede un microfono reale e si verifica lanciando il programma.

## Note sui requisiti originali

- Il pacchetto indicato nei requisiti, `github.com/gen2brain/mal-go`, non esiste
  più: il progetto si chiama `github.com/gen2brain/malgo` ed è quello usato qui.
- Il logging usa `log/slog` della libreria standard, come richiesto.
- L'analisi del testo non usa `strings.Contains` grezzo ma un confronto a
  confine di parola, che evita i falsi positivi; il requisito è rispettato nello
  spirone e reso più robusto.
