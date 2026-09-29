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
