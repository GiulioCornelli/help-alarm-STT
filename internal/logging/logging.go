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
	// Path is the file with the ordinary messages. Its parent directory is
	// created if missing.
	Path string
	// AlertPath is the file that receives only the alarm records. When it
	// matches Path the two loggers share the same file handle.
	AlertPath string
	// Level is debug, info, warn or error.
	Level string
	// Stdout, when not nil, receives the human readable stream in addition
	// to the structured one written to Path.
	Stdout io.Writer
}

// Logger exposes the three sinks separately, because they must not be mixed:
// Console is for the operator, File holds the running record and Alerts holds
// only the events that triggered the alarm, so that log file can be watched or
// shipped on its own.
type Logger struct {
	// Console writes to both Stdout and Path.
	Console *slog.Logger
	// File writes to Path only.
	File *slog.Logger
	// Alerts writes to AlertPath only.
	Alerts *slog.Logger

	closers []io.Closer
}

// Close releases the log files.
func (l *Logger) Close() error {
	var firstErr error
	for _, c := range l.closers {
		if err := c.Close(); err != nil && firstErr == nil {
			firstErr = err
		}
	}
	return firstErr
}

// New opens the log files and returns the console, file and alert loggers.
// The ordinary file receives JSON records with a timestamp; the console
// receives text records prefixed by their own timestamp.
func New(opts Options) (*Logger, error) {
	level, err := parseLevel(opts.Level)
	if err != nil {
		return nil, err
	}

	logFile, err := open(opts.Path)
	if err != nil {
		return nil, err
	}

	l := &Logger{closers: []io.Closer{logFile}}
	l.File = slog.New(jsonHandler(logFile, level))

	alertFile := logFile
	if opts.AlertPath != "" && opts.AlertPath != opts.Path {
		alertFile, err = open(opts.AlertPath)
		if err != nil {
			_ = logFile.Close()
			return nil, err
		}
		l.closers = append(l.closers, alertFile)
	}
	l.Alerts = slog.New(jsonHandler(alertFile, level))

	l.Console = slog.New(&fanout{handlers: []slog.Handler{jsonHandler(logFile, level)}})
	if opts.Stdout != nil {
		l.Console = slog.New(&fanout{handlers: []slog.Handler{
			jsonHandler(logFile, level),
			slog.NewTextHandler(
				&lockedWriter{w: opts.Stdout},
				&slog.HandlerOptions{Level: level, ReplaceAttr: withoutTime},
			),
		}})
	}

	return l, nil
}

func jsonHandler(w io.Writer, level slog.Level) slog.Handler {
	return slog.NewJSONHandler(&lockedWriter{w: w}, &slog.HandlerOptions{Level: level})
}

// open creates the parent directory of path and opens the file in append mode.
func open(path string) (*os.File, error) {
	if dir := filepath.Dir(path); dir != "" {
		if err := os.MkdirAll(dir, 0o755); err != nil {
			return nil, fmt.Errorf("logging: creazione directory %s fallita: %w", dir, err)
		}
	}
	f, err := os.OpenFile(path, os.O_CREATE|os.O_WRONLY|os.O_APPEND, 0o644)
	if err != nil {
		return nil, fmt.Errorf("logging: apertura %s fallita: %w", path, err)
	}
	return f, nil
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
