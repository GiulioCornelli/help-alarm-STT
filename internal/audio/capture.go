package audio

import (
	"fmt"
	"io"
	"sync"

	"github.com/gen2brain/malgo"
)

// Config describes which capture device to open and in which PCM format.
type Config struct {
	// DeviceID selects the capture device. A negative value means "system
	// default"; otherwise it is the index returned by ListDevices.
	DeviceID int
	// SampleRate in Hz. Vosk requires 16000.
	SampleRate uint32
}

// Capture streams raw PCM frames from a microphone.
//
// The frames are produced by miniaudio on its own audio thread: the callback
// only copies the buffer and hands it over to a channel, it never blocks, so
// the audio thread is never stalled by the downstream pipeline.
type Capture struct {
	cfg    Config
	ctx    *malgo.AllocatedContext
	device *malgo.Device
	name   string

	ch   chan []byte
	once sync.Once
}

// backendsForPlatform lists the audio backends to try, in order. ALSA comes
// first so that a plain Linux box without a sound server still works.
func backendsForPlatform() []malgo.Backend {
	return []malgo.Backend{malgo.BackendAlsa, malgo.BackendPulseaudio}
}

// ListDevices writes the available capture devices to w, with the index to put
// in "device_id" in the configuration file.
func ListDevices(w io.Writer) error {
	backends := backendsForPlatform()
	allocCtx, err := malgo.InitContext(backends, malgo.ContextConfig{}, func(string) {})
	if err != nil {
		return fmt.Errorf("audio: inizializzazione contesto fallita: %w", err)
	}
	defer allocCtx.Free()

	devices, err := allocCtx.Devices(malgo.Capture)
	if err != nil {
		return fmt.Errorf("audio: elenco dispositivi fallito: %w", err)
	}

	if len(devices) == 0 {
		fmt.Fprintln(w, "nessun microfono trovato")
		return nil
	}

	fmt.Fprintln(w, "dispositivi di cattura (indice da usare in config.json -> device_id):")
	for i, d := range devices {
		marker := " "
		if d.IsDefault != 0 {
			marker = "*"
		}
		fmt.Fprintf(w, " %s %d  %s  [%s]\n", marker, i, d.Name(), d.ID.String())
	}
	fmt.Fprintln(w, "  * = dispositivo predefinito di sistema (corrisponde a device_id: -1)")
	return nil
}

// New opens the capture device but does not start streaming yet.
func New(cfg Config) (*Capture, error) {
	if cfg.SampleRate == 0 {
		return nil, fmt.Errorf("audio: sample rate mancante")
	}

	backends := backendsForPlatform()
	allocCtx, err := malgo.InitContext(backends, malgo.ContextConfig{}, func(string) {})
	if err != nil {
		return nil, fmt.Errorf("audio: inizializzazione contesto fallita: %w", err)
	}

	c := &Capture{
		cfg: cfg,
		ctx: allocCtx,
		// 8 slot: abbastanza margine per non perdere frame se la
		// trascrizione rallenta per un istante.
		ch: make(chan []byte, 8),
	}

	deviceCfg := malgo.DefaultDeviceConfig(malgo.Capture)
	// Senza questo, miniaudio apre il dispositivo alla sua frequenza nativa
	// (spesso 32000 o 48000 Hz) e le trasformiamo in silenzio.
	deviceCfg.SampleRate = cfg.SampleRate
	deviceCfg.PeriodSizeInMilliseconds = 100
	deviceCfg.Periods = 3
	// Vosk accetta solo PCM s16 little endian, mono, 16 kHz: miniaudio
	// converte per noi se il microfono espone altro.
	deviceCfg.Capture.Format = malgo.FormatS16
	deviceCfg.Capture.Channels = 1
	deviceCfg.Capture.ShareMode = malgo.Shared
	// Speex non è compilato in miniaudio, il resampling lineare sì.
	deviceCfg.Resampling = malgo.ResampleConfig{Algorithm: malgo.ResampleAlgorithmLinear}

	if err := c.applyDeviceID(&deviceCfg); err != nil {
		allocCtx.Free()
		return nil, err
	}

	device, err := malgo.InitDevice(allocCtx.Context, deviceCfg, malgo.DeviceCallbacks{Data: c.onData})
	if err != nil {
		allocCtx.Free()
		return nil, fmt.Errorf("audio: apertura microfono fallita: %w", err)
	}
	c.device = device
	c.name = c.resolveName()

	if err := c.verifyFormat(); err != nil {
		c.Stop()
		return nil, err
	}

	return c, nil
}

// verifyFormat makes sure miniaudio really handed us the PCM Vosk expects. If
// the driver refused the conversion the transcription would be garbage, so it
// is better to stop here than to silently report "nessun testo riconosciuto".
func (c *Capture) verifyFormat() error {
	if got := c.device.SampleRate(); got != c.cfg.SampleRate {
		return fmt.Errorf("audio: il dispositivo è a %d Hz invece dei %d Hz richiesti", got, c.cfg.SampleRate)
	}
	if got := c.device.CaptureFormat(); got != malgo.FormatS16 {
		return fmt.Errorf("audio: il dispositivo restituisce il formato %v invece di s16", got)
	}
	if got := c.device.CaptureChannels(); got != 1 {
		return fmt.Errorf("audio: il dispositivo restituisce %d canali invece di mono", got)
	}
	return nil
}

// applyDeviceID selects an explicit capture device, or the system default when
// cfg.DeviceID is negative.
func (c *Capture) applyDeviceID(deviceCfg *malgo.DeviceConfig) error {
	c.name = "dispositivo predefinito"
	if c.cfg.DeviceID < 0 {
		return nil
	}

	devices, err := c.ctx.Devices(malgo.Capture)
	if err != nil {
		return fmt.Errorf("audio: elenco dispositivi fallito: %w", err)
	}
	if c.cfg.DeviceID >= len(devices) {
		return fmt.Errorf("audio: device_id %d non valido, trovati %d dispositivi di cattura", c.cfg.DeviceID, len(devices))
	}
	deviceCfg.Capture.DeviceID = devices[c.cfg.DeviceID].ID.Pointer()
	c.name = devices[c.cfg.DeviceID].Name()
	return nil
}

func (c *Capture) resolveName() string {
	devices, err := c.ctx.Devices(malgo.Capture)
	if err != nil {
		return "sconosciuto"
	}
	if c.cfg.DeviceID >= 0 && c.cfg.DeviceID < len(devices) {
		return devices[c.cfg.DeviceID].Name()
	}
	for _, d := range devices {
		if d.IsDefault != 0 {
			return d.Name() + " (predefinito)"
		}
	}
	return "predefinito"
}

// onData is the miniaudio callback. It must never block.
func (c *Capture) onData(_, input []byte, _ uint32) {
	if len(input) == 0 {
		return
	}
	frame := make([]byte, len(input))
	copy(frame, input)

	select {
	case c.ch <- frame:
	default:
		// Il consumer è indietro: meglio scartare il frame che bloccare il
		// thread audio, per miniaudio un blocco qui significa un glitch.
	}
}

// Start begins streaming. Frames become readable on the channel returned by
// Frames.
func (c *Capture) Start() error {
	if err := c.device.Start(); err != nil {
		return fmt.Errorf("audio: avvio stream fallito: %w", err)
	}
	return nil
}

// Frames returns the channel of PCM chunks. It is closed by Stop.
func (c *Capture) Frames() <-chan []byte { return c.ch }

// Stop halts the stream and releases the device and the context.
func (c *Capture) Stop() {
	c.once.Do(func() {
		if c.device != nil {
			_ = c.device.Stop()
			c.device.Uninit()
		}
		if c.ctx != nil {
			c.ctx.Free()
		}
		close(c.ch)
	})
}

// Description returns a human readable label of the device in use.
func (c *Capture) Description() string {
	return fmt.Sprintf("%s — %d Hz, s16 mono, periodo 100 ms", c.name, c.device.SampleRate())
}
