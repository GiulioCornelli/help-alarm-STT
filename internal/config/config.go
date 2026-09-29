package config

import (
	"encoding/json"
	"fmt"
	"os"
	"path/filepath"
	"strings"
)

// Config holds every tunable parameter of the application.
type Config struct {
	// Keywords to detect in the transcription. Single words and multi-word
	// phrases are both supported.
	Keywords []string `json:"keywords"`
	// ModelPath is the directory of the Vosk acoustic model.
	ModelPath string `json:"model_path"`
	// DeviceID selects the capture device. -1 means "system default".
	DeviceID int `json:"device_id"`
	// SampleRate in Hz. Vosk requires 16000.
	SampleRate int `json:"sample_rate"`
	// LogFile is the path of the structured log file.
	LogFile string `json:"log_file"`
	// LogLevel is one of: debug, info, warn, error.
	LogLevel string `json:"log_level"`
	// LogTranscriptions enables logging of every recognized phrase.
	LogTranscriptions bool `json:"log_transcriptions"`

	// dir is the directory holding the config file, used to resolve the
	// relative paths of ModelPath and LogFile.
	dir string
}

const voskSampleRate = 16000

// Load reads and validates the configuration from path.
func Load(path string) (*Config, error) {
	raw, err := os.ReadFile(path)
	if err != nil {
		return nil, fmt.Errorf("lettura config: %w", err)
	}

	cfg := &Config{
		ModelPath:         "models/vosk-model-small-it-0.22",
		DeviceID:          -1,
		SampleRate:        voskSampleRate,
		LogFile:           "logs/help-alarm.log",
		LogLevel:          "info",
		LogTranscriptions: true,
	}
	if err := json.Unmarshal(raw, cfg); err != nil {
		return nil, fmt.Errorf("config non valido %s: %w", path, err)
	}

	abs, err := filepath.Abs(path)
	if err != nil {
		return nil, fmt.Errorf("percorso config: %w", err)
	}
	cfg.dir = filepath.Dir(abs)

	if err := cfg.normalize(); err != nil {
		return nil, err
	}
	return cfg, nil
}

func (c *Config) normalize() error {
	keywords := make([]string, 0, len(c.Keywords))
	seen := make(map[string]struct{}, len(c.Keywords))
	for _, k := range c.Keywords {
		k = strings.TrimSpace(strings.ToLower(k))
		if k == "" {
			continue
		}
		if _, dup := seen[k]; dup {
			continue
		}
		seen[k] = struct{}{}
		keywords = append(keywords, k)
	}
	if len(keywords) == 0 {
		return fmt.Errorf("config non valido: \"keywords\" deve contenere almeno una parola chiave")
	}
	c.Keywords = keywords

	if c.SampleRate != voskSampleRate {
		return fmt.Errorf("config non valido: sample_rate deve essere %d (richiesto da Vosk), trovato %d", voskSampleRate, c.SampleRate)
	}

	switch c.LogLevel {
	case "debug", "info", "warn", "error":
	default:
		return fmt.Errorf("config non valido: log_level %q non ammesso (debug, info, warn, error)", c.LogLevel)
	}

	if strings.TrimSpace(c.ModelPath) == "" {
		return fmt.Errorf("config non valido: \"model_path\" è obbligatorio")
	}
	if strings.TrimSpace(c.LogFile) == "" {
		return fmt.Errorf("config non valido: \"log_file\" è obbligatorio")
	}

	c.ModelPath = c.resolve(c.ModelPath)
	c.LogFile = c.resolve(c.LogFile)
	return nil
}

// resolve turns a possibly relative path into an absolute one, relative to the
// directory containing the config file.
func (c *Config) resolve(p string) string {
	if filepath.IsAbs(p) {
		return filepath.Clean(p)
	}
	return filepath.Clean(filepath.Join(c.dir, p))
}

// AbsModelPath returns the absolute path of the Vosk model directory.
func (c *Config) AbsModelPath() string { return c.ModelPath }

// AbsLogPath returns the absolute path of the log file.
func (c *Config) AbsLogPath() string { return c.LogFile }
