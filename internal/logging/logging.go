package logging

import (
	"context"
	"fmt"
	"io"
	"log/slog"
	"os"
	"path/filepath"
	"strings"
	"sync"
)

// Options configures the logger.
type Options struct {
	// Path is the log file. Its parent directory is created if missing.
	Path string
	// Level is debug, info, warn or error.
	Level string
	// Stdout, when not nil, receives the human readable stream in addition
	// to the structured one written to Path.
	Stdout io.Writer
}

// Logger exposes the two sinks separately: Console for the operator, File for
// the durable record. They are distinct because the alarm prints its own line
// on the terminal and must not be duplicated by the console handler.
type Logger struct {
	Console *slog.Logger
	File    *slog.Logger

	closer io.Closer
}

// Close releases the log file.
func (l *Logger) Close() error {
	if l.closer == nil {
		return nil
	}
	return l.closer.Close()
}

// New opens the log file and returns the console and file loggers. The file
// receives JSON records with a timestamp; the console receives text records
// prefixed by their own timestamp.
func New(opts Options) (*Logger, error) {
	level, err := parseLevel(opts.Level)
	if err != nil {
		return nil, err
	}

	if dir := filepath.Dir(opts.Path); dir != "" {
		if err := os.MkdirAll(dir, 0o755); err != nil {
			return nil, fmt.Errorf("logging: creazione directory %s fallita: %w", dir, err)
		}
	}

	file, err := os.OpenFile(opts.Path, os.O_CREATE|os.O_WRONLY|os.O_APPEND, 0o644)
	if err != nil {
		return nil, fmt.Errorf("logging: apertura %s fallita: %w", opts.Path, err)
	}

	l := &Logger{
		File: slog.New(&fanout{handlers: []slog.Handler{
			slog.NewJSONHandler(&lockedWriter{w: file}, &slog.HandlerOptions{Level: level}),
		}}),
		closer: file,
	}

	consoleHandlers := []slog.Handler{slog.NewJSONHandler(&lockedWriter{w: file}, &slog.HandlerOptions{Level: level})}
	if opts.Stdout != nil {
		consoleHandlers = append(consoleHandlers, slog.NewTextHandler(
			&lockedWriter{w: opts.Stdout},
			&slog.HandlerOptions{Level: level, ReplaceAttr: withoutTime},
		))
	}
	l.Console = slog.New(&fanout{handlers: consoleHandlers})

	return l, nil
}

func parseLevel(s string) (slog.Level, error) {
	switch strings.ToLower(strings.TrimSpace(s)) {
	case "debug":
		return slog.LevelDebug, nil
	case "", "info":
		return slog.LevelInfo, nil
	case "warn":
		return slog.LevelWarn, nil
	case "error":
		return slog.LevelError, nil
	default:
		return 0, fmt.Errorf("logging: livello %q non ammesso (debug, info, warn, error)", s)
	}
}

// withoutTime drops the duplicated "time" key from the console handler, the
// console already prints it.
func withoutTime(groups []string, a slog.Attr) slog.Attr {
	if len(groups) == 0 && a.Key == slog.TimeKey {
		return slog.Attr{}
	}
	return a
}

// fanout sends every record to a set of handlers.
type fanout struct {
	handlers []slog.Handler
}

func (f *fanout) Enabled(ctx context.Context, level slog.Level) bool {
	for _, h := range f.handlers {
		if h.Enabled(ctx, level) {
			return true
		}
	}
	return false
}

func (f *fanout) Handle(ctx context.Context, r slog.Record) error {
	var firstErr error
	for _, h := range f.handlers {
		if err := h.Handle(ctx, r.Clone()); err != nil && firstErr == nil {
			firstErr = err
		}
	}
	return firstErr
}

func (f *fanout) WithAttrs(attrs []slog.Attr) slog.Handler {
	next := make([]slog.Handler, 0, len(f.handlers))
	for _, h := range f.handlers {
		next = append(next, h.WithAttrs(attrs))
	}
	return &fanout{handlers: next}
}

func (f *fanout) WithGroup(name string) slog.Handler {
	next := make([]slog.Handler, 0, len(f.handlers))
	for _, h := range f.handlers {
		next = append(next, h.WithGroup(name))
	}
	return &fanout{handlers: next}
}

// lockedWriter serializes writes, since the handlers run from several
// goroutines and os.File is not safe for interleaved writes.
type lockedWriter struct {
	mu sync.Mutex
	w  io.Writer
}

func (l *lockedWriter) Write(p []byte) (int, error) {
	l.mu.Lock()
	defer l.mu.Unlock()
	return l.w.Write(p)
}
