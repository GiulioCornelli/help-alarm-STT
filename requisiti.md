# help-alarm

Sto cercando di sviluppare un sistema per ascoltare da un microfono delle persone e rilevare se qualcuno prununcia la parola aiuto.


Volevo utilizzare lo Speech-to-Text (STT) (o Riconoscimento Automatico del Parlato) è un sistema che prende in ingresso un flusso audio continuo (i campioni PCM provenienti dal microfono), ne analizza le frequenze vocali e le converte in stringhe di testo leggibile tramite un modello linguistico.

Nell'approccio STT completo, il programma trascrive costantemente tutto ciò che sente in parole, consentendoti di analizzare le stringhe risultanti con un semplice controllo del testo.

Dipendenze e Pacchetti Go Necessari
Per realizzare questo flusso in Go in locale e senza servizi cloud esterni, la combinazione più efficiente prevede la libreria audio mal-go abbinata al motore STT offline Vosk, che dispone di binding nativi in Go e di un modello in lingua italiana compatto.


1. Cattura Audio dal Microfono
[github.com/gen2brain/mal-go](https://github.com/gen2brain/mal-go)
Wrapper Go per miniaudio. È leggero, multipiattaforma (Linux, Windows, macOS) e permette di aprire uno stream di input dal microfono catturando i frame audio PCM in tempo reale.

(Alternativa) [github.com/gordonklaus/portaudio](https://github.com/gordonklaus/portaudio): Binding Go per PortAudio, ideale se su Linux/Raspberry Pi usi driver ALSA/PulseAudio.

2. Trascrizione Audio (Motore STT Offline)
[github.com/alphacep/vosk-api/go](https://github.com/alphacep/vosk-api/go) (o vosk-go)
I binding Go ufficiali per Vosk. Passi il buffer di byte letto dal microfono direttamente alla funzione AcceptWaveform(buffer) e il motore restituisce un JSON contenente il testo trascritto in tempo reale.

3. Analisi del Testo (Riconoscimento "Aiuto")
strings e encoding/json (Libreria Standard Go)
Non servono pacchetti terzi. Si effettua l'unmarshal del JSON di Vosk per estrarre il campo text, si converte in minuscolo e si verifica la presenza della parola chiave:
strings.Contains(strings.ToLower(result.Text), "aiuto")
Per analizzare il testo utilizza anche lo go routine se neccessario per essere piu' veloce ed efficente.


4. Logging degli Eventi
log/slog (Libreria Standard Go 1.21+)
Il logger strutturato nativo di Go. Permette di registrare su stdout o su un file .log tutti gli eventi (avvio audio, trascrizioni e attivazione luce) con timestamp e formato JSON o testo.

(Alternativa) [github.com/rs/zerolog](https://github.com/rs/zerolog): Se desideri un logger esterno ad altissime prestazioni orientato a file di log su disco.

5. Controllo Hardware (Accensione Luce / GPIO)
[github.com/stianeikeland/go-rpio/v4](https://github.com/stianeikeland/go-rpio/v4) o periph.io/x/conn/v3/gpio
Se il progetto gira su Raspberry Pi o scheda Linux con GPIO, per settare il pin del relè/LED a HIGH.


Devi appunto realizzare il codice che prenda le aprole in ingresso da un microfono e che capisca se qualcuno dice la parola aiuto (ma possono essere anche alre e devono essere poter configurate in un file apposito o json,yml o .env classico) e se lo rileva deve stampare su terminale "aiuto rilevato, luce accesa" e scrivi in un file di log la data e l'ora con il messaggio che ti ho detto prima. in un fututro non dovra' scrivere su teminale, ma oltre che ai log dovra' accendere una luce ma non devi farlo adesso.