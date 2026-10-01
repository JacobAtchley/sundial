package config

import (
	"bytes"
	_ "embed"
	"errors"
	"fmt"
	"io/fs"
	"maps"
	"os"
	"path/filepath"
	"regexp"
	"slices"
	"strings"

	"github.com/pelletier/go-toml/v2"
)

//go:embed starter.toml
var Starter string

// Error describes a problem in the config file.
type Error struct {
	Path string
	Line int
	Key  string
	Msg  string
}

func (e *Error) Error() string {
	loc := e.Path
	if e.Line > 0 {
		loc = fmt.Sprintf("%s:%d", e.Path, e.Line)
	}
	if e.Key != "" {
		return fmt.Sprintf("%s: %s: %s", loc, e.Key, e.Msg)
	}
	return loc + ": " + e.Msg
}

// Path returns the config file location, honoring XDG_CONFIG_HOME.
func Path() (string, error) {
	if dir := os.Getenv("XDG_CONFIG_HOME"); dir != "" {
		return filepath.Join(dir, "sundial", "config.toml"), nil
	}
	home, err := os.UserHomeDir()
	if err != nil {
		return "", fmt.Errorf("find home directory: %w", err)
	}
	return filepath.Join(home, ".config", "sundial", "config.toml"), nil
}

// Load reads path. A missing file yields Default() and no error.
func Load(path string) (Config, error) {
	raw, err := os.ReadFile(path)
	if errors.Is(err, fs.ErrNotExist) {
		return Default(), nil
	}
	if err != nil {
		return Default(), fmt.Errorf("read config: %w", err)
	}
	return Parse(path, raw)
}

// Parse decodes and validates raw TOML. path is used only in errors.
func Parse(path string, raw []byte) (Config, error) {
	cfg := Default()
	dec := toml.NewDecoder(bytes.NewReader(raw)).DisallowUnknownFields()
	if err := dec.Decode(&cfg); err != nil {
		var missing *toml.StrictMissingError
		if errors.As(err, &missing) && len(missing.Errors) > 0 {
			first := missing.Errors[0]
			row, _ := first.Position()
			return Default(), &Error{Path: path, Line: row, Key: strings.Join(first.Key(), "."), Msg: "unknown key"}
		}
		var de *toml.DecodeError
		if errors.As(err, &de) {
			row, _ := de.Position()
			return Default(), &Error{Path: path, Line: row, Msg: strings.TrimPrefix(de.Error(), "toml: ")}
		}
		return Default(), &Error{Path: path, Msg: err.Error()}
	}
	if err := cfg.validate(raw); err != nil {
		err.Path = path
		return Default(), err
	}
	return cfg, nil
}

var hexRe = regexp.MustCompile(`^#[0-9A-Fa-f]{6}$`)

func (c Config) validate(raw []byte) *Error {
	fail := func(key, format string, args ...any) *Error {
		return &Error{Line: lineOf(raw, key), Key: key, Msg: fmt.Sprintf(format, args...)}
	}
	if !slices.Contains([]string{"month", "week", "agenda"}, c.DefaultView) {
		return fail("default_view", "must be one of month, week, agenda (got %q)", c.DefaultView)
	}
	if !slices.Contains([]string{"sunday", "monday"}, c.WeekStart) {
		return fail("week_start", "must be one of sunday, monday (got %q)", c.WeekStart)
	}
	if !slices.Contains([]string{"12h", "24h"}, c.TimeFormat) {
		return fail("time_format", "must be one of 12h, 24h (got %q)", c.TimeFormat)
	}
	if c.DayStart < 0 || c.DayStart > 23 {
		return fail("day_start", "must be between 0 and 23 (got %d)", c.DayStart)
	}
	if c.Agenda.Days < 1 || c.Agenda.Days > 365 {
		return fail("agenda.days", "must be between 1 and 365 (got %d)", c.Agenda.Days)
	}
	for _, o := range c.Theme.Overrides() {
		if o.Hex != "" && !hexRe.MatchString(o.Hex) {
			return fail("theme."+o.Name, "must be a hex color like #C6A0F6 (got %q)", o.Hex)
		}
	}
	actions := slices.Sorted(maps.Keys(DefaultKeys))
	for _, action := range slices.Sorted(maps.Keys(c.Keys)) {
		if !slices.Contains(actions, action) {
			return fail("keys."+action, "unknown action; valid actions: %s", strings.Join(actions, ", "))
		}
		if len(c.Keys[action]) == 0 {
			return fail("keys."+action, "must list at least one key")
		}
	}
	return nil
}

// lineOf finds the 1-based line where dotted key is assigned, or 0.
func lineOf(raw []byte, key string) int {
	table, name := "", key
	if i := strings.LastIndex(key, "."); i >= 0 {
		table, name = key[:i], key[i+1:]
	}
	current := ""
	for i, line := range strings.Split(string(raw), "\n") {
		t := strings.TrimSpace(line)
		if strings.HasPrefix(t, "[") && strings.HasSuffix(t, "]") {
			current = strings.TrimSpace(strings.Trim(t, "[]"))
			continue
		}
		if current != table {
			continue
		}
		if k, _, ok := strings.Cut(t, "="); ok && strings.TrimSpace(k) == name {
			return i + 1
		}
	}
	return 0
}

// Init writes the starter config to path. It never overwrites.
func Init(path string) error {
	if err := os.MkdirAll(filepath.Dir(path), 0o755); err != nil {
		return fmt.Errorf("create config directory: %w", err)
	}
	f, err := os.OpenFile(path, os.O_WRONLY|os.O_CREATE|os.O_EXCL, 0o644)
	if errors.Is(err, fs.ErrExist) {
		return fmt.Errorf("config already exists at %s", path)
	}
	if err != nil {
		return fmt.Errorf("create config: %w", err)
	}
	if _, err := f.WriteString(Starter); err != nil {
		_ = f.Close()
		return fmt.Errorf("write config: %w", err)
	}
	return f.Close()
}
