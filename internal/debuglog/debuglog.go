// Package debuglog writes an opt-in debug log that redacts event content
// unless verbose mode is on.
package debuglog

import (
	"fmt"
	"io"
	"log/slog"
	"os"
	"path/filepath"
)

type Logger struct {
	*slog.Logger
	Verbose bool
}

// Redact hides event content unless verbose logging was requested.
func (l Logger) Redact(s string) string {
	if l.Verbose {
		return s
	}
	return "[redacted]"
}

func Discard() Logger { return Logger{Logger: slog.New(slog.NewTextHandler(io.Discard, nil))} }

// Path returns $XDG_STATE_HOME/sundial/debug.log (fallback ~/.local/state).
func Path() (string, error) {
	if dir := os.Getenv("XDG_STATE_HOME"); dir != "" {
		return filepath.Join(dir, "sundial", "debug.log"), nil
	}
	home, err := os.UserHomeDir()
	if err != nil {
		return "", fmt.Errorf("find home directory: %w", err)
	}
	return filepath.Join(home, ".local", "state", "sundial", "debug.log"), nil
}

func Open(verbose bool) (Logger, func() error, error) {
	p, err := Path()
	if err != nil {
		return Discard(), func() error { return nil }, err
	}
	if err := os.MkdirAll(filepath.Dir(p), 0o700); err != nil {
		return Discard(), func() error { return nil }, fmt.Errorf("create log directory: %w", err)
	}
	f, err := os.OpenFile(p, os.O_CREATE|os.O_WRONLY|os.O_APPEND, 0o600)
	if err != nil {
		return Discard(), func() error { return nil }, fmt.Errorf("open debug log: %w", err)
	}
	h := slog.NewTextHandler(f, &slog.HandlerOptions{Level: slog.LevelDebug})
	return Logger{Logger: slog.New(h), Verbose: verbose}, f.Close, nil
}
