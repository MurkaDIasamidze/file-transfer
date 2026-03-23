// logger/logger.go
//
// Two outputs:
//   1. Terminal  — human-readable with ANSI colors per level
//   2. logs/app.json — newline-delimited JSON, one record per line
//
// Usage (in main.go):
//
//	slog.SetDefault(logger.New("logs/app.json"))
package logger

import (
	"context"
	"fmt"
	"io"
	"log/slog"
	"os"
	"path/filepath"
	"sync"
	"time"
)

// ── ANSI color codes ──────────────────────────────────────────────────────────

const (
	reset  = "\033[0m"
	bold   = "\033[1m"

	colorDebug = "\033[36m"  // cyan
	colorInfo  = "\033[32m"  // green
	colorWarn  = "\033[33m"  // yellow
	colorError = "\033[31m"  // red
	colorFatal = "\033[35m"  // magenta

	colorKey   = "\033[34m"  // blue  — attribute keys
	colorTime  = "\033[90m"  // dark gray — timestamp
	colorID    = "\033[96m"  // bright cyan — file/upload IDs
)

// ── Console handler (colored) ─────────────────────────────────────────────────

type consoleHandler struct {
	mu  sync.Mutex
	out io.Writer
	lvl slog.Level
}

func (h *consoleHandler) Enabled(_ context.Context, lvl slog.Level) bool {
	return lvl >= h.lvl
}

func (h *consoleHandler) WithAttrs(attrs []slog.Attr) slog.Handler  { return h }
func (h *consoleHandler) WithGroup(name string) slog.Handler         { return h }

func (h *consoleHandler) Handle(_ context.Context, r slog.Record) error {
	// Level badge
	var levelStr string
	switch {
	case r.Level >= slog.LevelError:
		levelStr = colorError + bold + " ERROR " + reset
	case r.Level >= slog.LevelWarn:
		levelStr = colorWarn + bold + "  WARN " + reset
	case r.Level >= slog.LevelInfo:
		levelStr = colorInfo + bold + "  INFO " + reset
	default:
		levelStr = colorDebug + bold + " DEBUG " + reset
	}

	// Timestamp
	ts := colorTime + r.Time.Format("2006/01/02 15:04:05") + reset

	// Message
	msg := r.Message

	// Attributes
	attrs := ""
	r.Attrs(func(a slog.Attr) bool {
		key := colorKey + a.Key + reset + "="
		val := fmt.Sprintf("%v", a.Value.Any())
		// Highlight numeric IDs in bright cyan
		if a.Key == "file" || a.Key == "id" || a.Key == "file_id" || a.Key == "upload_id" {
			val = colorID + bold + val + reset
		}
		attrs += "  " + key + val
		return true
	})

	h.mu.Lock()
	defer h.mu.Unlock()
	fmt.Fprintf(h.out, "%s %s  %s%s\n", ts, levelStr, msg, attrs)
	return nil
}

// ── JSON file handler ─────────────────────────────────────────────────────────
// Uses the standard slog JSONHandler writing to a file.

func newJSONFileHandler(path string, lvl slog.Level) (slog.Handler, error) {
	if err := os.MkdirAll(filepath.Dir(path), 0o755); err != nil {
		return nil, fmt.Errorf("logger: mkdir %s: %w", filepath.Dir(path), err)
	}
	f, err := os.OpenFile(path, os.O_CREATE|os.O_APPEND|os.O_WRONLY, 0o644)
	if err != nil {
		return nil, fmt.Errorf("logger: open %s: %w", path, err)
	}
	return slog.NewJSONHandler(f, &slog.HandlerOptions{
		Level:     lvl,
		AddSource: false,
		ReplaceAttr: func(_ []string, a slog.Attr) slog.Attr {
			// Rename "time" → "ts" for shorter JSON keys
			if a.Key == slog.TimeKey {
				a.Key = "ts"
				a.Value = slog.StringValue(a.Value.Time().Format(time.RFC3339))
			}
			return a
		},
	}), nil
}

// ── Fan-out handler — writes to multiple handlers ────────────────────────────

type multiHandler struct {
	handlers []slog.Handler
}

func (m *multiHandler) Enabled(ctx context.Context, lvl slog.Level) bool {
	for _, h := range m.handlers {
		if h.Enabled(ctx, lvl) {
			return true
		}
	}
	return false
}

func (m *multiHandler) Handle(ctx context.Context, r slog.Record) error {
	for _, h := range m.handlers {
		if h.Enabled(ctx, r.Level) {
			if err := h.Handle(ctx, r.Clone()); err != nil {
				return err
			}
		}
	}
	return nil
}

func (m *multiHandler) WithAttrs(attrs []slog.Attr) slog.Handler {
	hs := make([]slog.Handler, len(m.handlers))
	for i, h := range m.handlers {
		hs[i] = h.WithAttrs(attrs)
	}
	return &multiHandler{hs}
}

func (m *multiHandler) WithGroup(name string) slog.Handler {
	hs := make([]slog.Handler, len(m.handlers))
	for i, h := range m.handlers {
		hs[i] = h.WithGroup(name)
	}
	return &multiHandler{hs}
}

// ── Public constructor ────────────────────────────────────────────────────────

// New returns a *slog.Logger that:
//   - prints colored output to stderr
//   - appends JSON records to logFile (e.g. "logs/app.json")
//
// Pass logFile = "" to disable file logging.
func New(logFile string) *slog.Logger {
	console := &consoleHandler{out: os.Stderr, lvl: slog.LevelDebug}

	if logFile == "" {
		return slog.New(console)
	}

	jsonH, err := newJSONFileHandler(logFile, slog.LevelDebug)
	if err != nil {
		// Fall back to console-only and warn
		l := slog.New(console)
		l.Warn("could not open log file, console only", "err", err)
		return l
	}

	return slog.New(&multiHandler{[]slog.Handler{console, jsonH}})
}