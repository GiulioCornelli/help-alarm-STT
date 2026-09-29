package alert

import (
	"fmt"
	"io"
	"log/slog"
	"sync"
	"time"
)

// Message is the text printed and logged when a keyword is detected.
const Message = "aiuto rilevato, luce accesa"

// Cooldown suppresses repeated alarms for the same keyword within a time
// window.
//
// It is needed because Vosk reports a transcription every 100 ms: while the
// person is still talking, the same word stays inside the partial hypothesis
// and would fire once per audio chunk. Without this, a single "aiuto" produces
// a burst of identical alarm lines and the real log signal is lost.
type Cooldown struct {
	window time.Duration

	mu   sync.Mutex
	last map[string]time.Time
}

// NewCooldown returns a Cooldown with the given window. A window of zero or
// less disables the suppression and allows every match, as the naive
// behaviour.
func NewCooldown(window time.Duration) *Cooldown {
	return &Cooldown{
		window: window,
		last:   make(map[string]time.Time),
	}
}

// Allow reports whether an alarm for keyword at time now should be raised. It
// returns false when the same keyword already fired less than window ago, and
// records the new time when it returns true.
func (c *Cooldown) Allow(keyword string, now time.Time) bool {
	c.mu.Lock()
	defer c.mu.Unlock()

	if c.window <= 0 {
		return true
	}
	if prev, ok := c.last[keyword]; ok && now.Sub(prev) < c.window {
		return false
	}
	c.last[keyword] = now
	return true
}

// Event describes a detected keyword.
type Event struct {
	Keyword string
	Phrase  string
	At      time.Time
}

// Alerter reacts to a detected keyword.
//
// The GPIO light will be added later as another implementation of this
// interface: the rest of the program does not need to change.
type Alerter interface {
	// Name identifies the alerter in the logs.
	Name() string
	// Trigger is called for every detected keyword.
	Trigger(Event) error
}

// TerminalAlerter prints the event on the terminal and writes it to the log
// file with a timestamp.
type TerminalAlerter struct {
	out  io.Writer
	log  *slog.Logger
	mu   sync.Mutex
	now  func() time.Time
	name string
}

// NewTerminal builds an Alerter writing on out and logging on log.
func NewTerminal(out io.Writer, log *slog.Logger) *TerminalAlerter {
	return &TerminalAlerter{out: out, log: log, now: time.Now, name: "terminal"}
}

// Name implements Alerter.
func (t *TerminalAlerter) Name() string { return t.name }

// Trigger implements Alerter: it prints "aiuto rilevato, luce accesa" on the
// terminal and appends a timestamped line to the log file.
func (t *TerminalAlerter) Trigger(e Event) error {
	t.mu.Lock()
	defer t.mu.Unlock()

	ts := e.At
	if ts.IsZero() {
		ts = t.now()
	}

	if _, err := fmt.Fprintf(t.out, "%s %s\n", ts.Format("15:04:05"), Message); err != nil {
		return fmt.Errorf("alert: scrittura su terminale fallita: %w", err)
	}

	t.log.Info(Message,
		slog.Time("timestamp", ts),
		slog.String("keyword", e.Keyword),
		slog.String("trascrizione", e.Phrase),
	)
	return nil
}
