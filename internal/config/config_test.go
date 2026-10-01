package config

import (
	"errors"
	"os"
	"path/filepath"
	"reflect"
	"strings"
	"testing"
	"time"
)

func TestLoadMissingFileReturnsDefaults(t *testing.T) {
	cfg, err := Load(filepath.Join(t.TempDir(), "nope.toml"))
	if err != nil {
		t.Fatal(err)
	}
	if !reflect.DeepEqual(cfg, Default()) {
		t.Errorf("got %+v, want defaults", cfg)
	}
}

func TestParsePartialOverride(t *testing.T) {
	cfg, err := Parse("c.toml", []byte("week_start = \"monday\"\n[theme]\nheader = \"#112233\"\n"))
	if err != nil {
		t.Fatal(err)
	}
	if cfg.WeekStartDay() != time.Monday {
		t.Error("week_start not applied")
	}
	if cfg.Theme.Header != "#112233" || !cfg.Theme.Pastelize {
		t.Errorf("theme = %+v; header should be set and pastelize keep its default", cfg.Theme)
	}
	if cfg.DefaultView != "agenda" || cfg.Agenda.Days != 14 {
		t.Error("unset keys must keep defaults")
	}
}

func requireConfigError(t *testing.T, err error, line int, key, msgPart string) {
	t.Helper()
	var ce *Error
	if !errors.As(err, &ce) {
		t.Fatalf("error %v is not *config.Error", err)
	}
	if ce.Line != line || ce.Key != key || !strings.Contains(ce.Msg, msgPart) {
		t.Errorf("got line=%d key=%q msg=%q; want line=%d key=%q msg containing %q", ce.Line, ce.Key, ce.Msg, line, key, msgPart)
	}
	if !strings.HasPrefix(err.Error(), "c.toml:") {
		t.Errorf("error %q should start with the path", err)
	}
}

func TestParseErrors(t *testing.T) {
	cases := []struct {
		name, src, key, msg string
		line                int
	}{
		{"unknown top-level key", "default_view = \"week\"\ncolour = \"red\"\n", "colour", "unknown key", 2},
		{"unknown nested key", "[theme]\npastelize = true\nheadr = \"#ffffff\"\n", "theme.headr", "unknown key", 3},
		{"bad enum", "\ndefault_view = \"year\"\n", "default_view", "month, week, agenda", 2},
		{"bad week start", "week_start = \"friday\"\n", "week_start", "sunday, monday", 1},
		{"bad time format", "time_format = \"am\"\n", "time_format", "12h, 24h", 1},
		{"day start range", "day_start = 24\n", "day_start", "0 and 23", 1},
		{"agenda days range", "[agenda]\ndays = 0\n", "agenda.days", "1 and 365", 2},
		{"bad color", "[theme]\nheader = \"lavender\"\n", "theme.header", "hex color", 2},
		{"unknown action", "[keys]\nexplode = [\"x\"]\n", "keys.explode", "unknown action", 2},
		{"empty key list", "[keys]\nquit = []\n", "keys.quit", "at least one key", 2},
	}
	for _, tc := range cases {
		t.Run(tc.name, func(t *testing.T) {
			_, err := Parse("c.toml", []byte(tc.src))
			requireConfigError(t, err, tc.line, tc.key, tc.msg)
		})
	}
}

func TestParseSyntaxErrorHasLine(t *testing.T) {
	_, err := Parse("c.toml", []byte("default_view = \"week\"\nweek_start = \n"))
	var ce *Error
	if !errors.As(err, &ce) || ce.Line != 2 {
		t.Fatalf("got %v, want syntax error on line 2", err)
	}
}

func TestStarterMatchesDefaults(t *testing.T) {
	cfg, err := Parse("starter.toml", []byte(Starter))
	if err != nil {
		t.Fatal(err)
	}
	if !reflect.DeepEqual(cfg, Default()) {
		t.Errorf("starter parses to %+v, want defaults", cfg)
	}
}

func TestPathRespectsXDG(t *testing.T) {
	dir := t.TempDir()
	t.Setenv("XDG_CONFIG_HOME", dir)
	p, err := Path()
	if err != nil {
		t.Fatal(err)
	}
	if p != filepath.Join(dir, "sundial", "config.toml") {
		t.Errorf("path = %s", p)
	}
	t.Setenv("XDG_CONFIG_HOME", "")
	t.Setenv("HOME", dir)
	p, _ = Path()
	if p != filepath.Join(dir, ".config", "sundial", "config.toml") {
		t.Errorf("fallback path = %s", p)
	}
}

func TestInitRefusesOverwrite(t *testing.T) {
	p := filepath.Join(t.TempDir(), "sundial", "config.toml")
	if err := Init(p); err != nil {
		t.Fatal(err)
	}
	b, _ := os.ReadFile(p)
	if string(b) != Starter {
		t.Error("init did not write the starter file")
	}
	if err := os.WriteFile(p, []byte("week_start = \"monday\"\n"), 0o644); err != nil {
		t.Fatal(err)
	}
	if err := Init(p); err == nil {
		t.Fatal("init must refuse to overwrite")
	}
	b, _ = os.ReadFile(p)
	if !strings.Contains(string(b), "monday") {
		t.Error("existing file was modified")
	}
}

func TestKeysForOverride(t *testing.T) {
	cfg := Default()
	if got := cfg.KeysFor("palette"); !reflect.DeepEqual(got, []string{"ctrl+k", ":"}) {
		t.Errorf("default palette keys = %v", got)
	}
	cfg.Keys = map[string][]string{"palette": {"ctrl+p"}}
	if got := cfg.KeysFor("palette"); !reflect.DeepEqual(got, []string{"ctrl+p"}) {
		t.Errorf("override palette keys = %v", got)
	}
}
