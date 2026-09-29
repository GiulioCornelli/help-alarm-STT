package alert

import (
	"bytes"
	"encoding/json"
	"log/slog"
	"strings"
	"testing"
	"time"
)

func newTestLogger() (*slog.Logger, *bytes.Buffer) {
	buf := &bytes.Buffer{}
	return slog.New(slog.NewJSONHandler(buf, nil)), buf
}

func TestTriggerPrintsAndLogs(t *testing.T) {
	log, logBuf := newTestLogger()
	out := &bytes.Buffer{}
	terminal := NewTerminal(out, log)

	at := time.Date(2026, 9, 29, 14, 5, 6, 0, time.UTC)
	err := terminal.Trigger(Event{Keyword: "aiuto", Phrase: "aiuto per favore", At: at})
	if err != nil {
		t.Fatalf("Trigger: %v", err)
	}

	const want = "aiuto rilevato, luce accesa"

	if !strings.Contains(out.String(), want) {
		t.Errorf("il terminale non contiene %q, output: %q", want, out.String())
	}
	if !strings.Contains(out.String(), "14:05:06") {
		t.Errorf("il terminale non mostra l'orario, output: %q", out.String())
	}

	var rec map[string]any
	if err := json.Unmarshal(logBuf.Bytes(), &rec); err != nil {
		t.Fatalf("record di log non è JSON valido: %v (%s)", err, logBuf.String())
	}
	if rec["msg"] != want {
		t.Errorf("msg = %v, atteso %q", rec["msg"], want)
	}
	if rec["keyword"] != "aiuto" {
		t.Errorf("keyword = %v, atteso \"aiuto\"", rec["keyword"])
	}
	if rec["trascrizione"] != "aiuto per favore" {
		t.Errorf("trascrizione = %v", rec["trascrizione"])
	}
	if _, ok := rec["time"]; !ok {
		t.Error("il record di log non ha il timestamp")
	}
	if ts, ok := rec["timestamp"].(string); !ok || !strings.HasPrefix(ts, "2026-09-29") {
		t.Errorf("timestamp = %v, atteso il 2026-09-29", rec["timestamp"])
	}
}

func TestTriggerDefaultsToNow(t *testing.T) {
	log, _ := newTestLogger()
	terminal := NewTerminal(&bytes.Buffer{}, log)
	terminal.now = func() time.Time {
		return time.Date(2000, 1, 2, 3, 4, 5, 0, time.UTC)
	}

	if err := terminal.Trigger(Event{Keyword: "aiuto"}); err != nil {
		t.Fatalf("Trigger: %v", err)
	}
}

// TestAlerterIsAnInterface documenta il punto di estensione: la luce con GPIO
// entrerà come seconda implementazione di Alerter, senza toccare chi chiama.
func TestAlerterIsAnInterface(t *testing.T) {
	log, _ := newTestLogger()
	var a Alerter = NewTerminal(&bytes.Buffer{}, log)
	if a.Name() != "terminal" {
		t.Errorf("Name = %q", a.Name())
	}
}
