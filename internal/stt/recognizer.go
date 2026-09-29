package stt

import (
	"encoding/json"
	"fmt"
	"os"
	"path/filepath"

	vosk "github.com/alphacep/vosk-api/go"
)

// Phrase is one transcription emitted by the recognizer.
type Phrase struct {
	Text string
	// Partial is true when the text is only the current hypothesis and can
	// still change with the following audio.
	Partial bool
}

// Result is the subset of the Vosk JSON payload that we care about.
// A final result looks like {"text": "..."} while a partial hypothesis looks
// like {"partial": "..."}: in both cases the transcription is a plain string.
type Result struct {
	Text    string `json:"text"`
	Partial string `json:"partial"`
}

// Recognizer wraps a Vosk model and turns raw PCM into Phrases.
// It is not safe for concurrent use: a single goroutine owns it.
type Recognizer struct {
	model *vosk.VoskModel
	rec   *vosk.VoskRecognizer
}

// Open loads the Vosk model located at modelPath. The directory must contain
// the files produced by the vosk-model-small-it-* archives (am/, conf/, graph/).
func Open(modelPath string) (*Recognizer, error) {
	if err := checkModel(modelPath); err != nil {
		return nil, err
	}

	// -1 silenzia il Verboso nativo di Vosk, che altrimenti scrive
	// direttamente su stdout e mescola il proprio output con il nostro.
	vosk.SetLogLevel(-1)

	model, err := vosk.NewModel(modelPath)
	if err != nil {
		return nil, fmt.Errorf("stt: caricamento modello Vosk fallito: %w", err)
	}

	rec, err := vosk.NewRecognizer(model, 16000.0)
	if err != nil {
		model.Free()
		return nil, fmt.Errorf("stt: creazione recognizer fallita: %w", err)
	}

	return &Recognizer{model: model, rec: rec}, nil
}

func checkModel(modelPath string) error {
	info, err := os.Stat(modelPath)
	if err != nil {
		return fmt.Errorf("stt: modello Vosk non trovato in %s: %w", modelPath, err)
	}
	if !info.IsDir() {
		return fmt.Errorf("stt: %s non è una directory", modelPath)
	}
	if _, err := os.Stat(filepath.Join(modelPath, "am", "final.mdl")); err != nil {
		return fmt.Errorf("stt: %s non è un modello Vosk valido (manca am/final.mdl): %w", modelPath, err)
	}
	return nil
}

// Accept feeds a PCM chunk (s16, mono, 16 kHz) to the recognizer.
//
// When final is true Vosk detected the end of an utterance and Phrase.Text is
// the complete transcription. Otherwise the returned Phrase is the current
// hypothesis and can still be revised by the following calls.
func (r *Recognizer) Accept(pcm []byte) (Phrase, bool, error) {
	if r.rec.AcceptWaveform(pcm) != 0 {
		text, err := parseText(r.rec.Result())
		if err != nil {
			return Phrase{}, false, err
		}
		return Phrase{Text: text}, true, nil
	}

	text, err := parseText(r.rec.PartialResult())
	if err != nil {
		return Phrase{}, false, err
	}
	return Phrase{Text: text, Partial: true}, false, nil
}

// Flush asks Vosk to emit the utterance still in flight without waiting for
// silence. It returns an empty Phrase when there was nothing pending.
func (r *Recognizer) Flush() (Phrase, error) {
	if r.rec == nil {
		return Phrase{}, nil
	}
	text, err := parseText(r.rec.FinalResult())
	if err != nil {
		return Phrase{}, err
	}
	return Phrase{Text: text}, nil
}

// Close frees the native resources. It is safe to call more than once.
func (r *Recognizer) Close() {
	if r.rec != nil {
		r.rec.Free()
		r.rec = nil
	}
	if r.model != nil {
		r.model.Free()
		r.model = nil
	}
}

// parseText extracts the transcription from a Vosk JSON payload, whether it is
// a final result ({"text": ...}) or a partial one ({"partial": {"text": ...}}).
func parseText(payload string) (string, error) {
	var res Result
	if err := json.Unmarshal([]byte(payload), &res); err != nil {
		return "", fmt.Errorf("stt: JSON Vosk non valido %q: %w", payload, err)
	}
	if res.Text != "" {
		return res.Text, nil
	}
	return res.Partial, nil
}
