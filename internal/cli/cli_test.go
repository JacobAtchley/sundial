package cli

import (
	"bytes"
	"encoding/json"
	"errors"
	"os"
	"path/filepath"
	"strings"
	"testing"
	"time"

	"github.com/charmbracelet/x/exp/golden"

	"github.com/JacobAtchley/sundial/internal/calendar"
	"github.com/JacobAtchley/sundial/internal/fakesource"
)

var now = time.Date(2026, 10, 1, 8, 0, 0, 0, time.UTC)

func testOptions(t *testing.T) Options {
	t.Helper()
	t.Setenv("XDG_CONFIG_HOME", t.TempDir())
	return Options{
		Now:        func() time.Time { return now },
		OpenSource: func(string) (calendar.Source, error) { return fakesource.Demo(now), nil },
		Getenv:     func(string) string { return "" },
	}
}

func run(t *testing.T, opts Options, args ...string) (string, error) {
	t.Helper()
	cmd := NewRootCommand(opts)
	var out bytes.Buffer
	cmd.SetOut(&out)
	cmd.SetErr(&out)
	cmd.SetArgs(args)
	err := cmd.Execute()
	return out.String(), err
}

func TestAgendaPlain(t *testing.T) {
	out, err := run(t, testOptions(t), "agenda", "--from", "2026-10-01", "--days", "2")
	if err != nil {
		t.Fatal(err)
	}
	golden.RequireEqual(t, out)
}

func TestTodayPlain(t *testing.T) {
	out, err := run(t, testOptions(t), "today")
	if err != nil {
		t.Fatal(err)
	}
	if !strings.Contains(out, "Team Standup") || !strings.Contains(out, "Lunch with Sam") {
		t.Errorf("today output missing events:\n%s", out)
	}
	if strings.Contains(out, "\x1b[") {
		t.Error("non-TTY output must not contain ANSI escapes")
	}
}

func TestAgendaJSON(t *testing.T) {
	out, err := run(t, testOptions(t), "agenda", "--from", "2026-10-01", "--days", "1", "--json")
	if err != nil {
		t.Fatal(err)
	}
	var evs []struct {
		Title    string `json:"title"`
		Calendar string `json:"calendar"`
		Status   string `json:"status"`
	}
	if err := json.Unmarshal([]byte(out), &evs); err != nil {
		t.Fatalf("invalid JSON: %v\n%s", err, out)
	}
	if len(evs) == 0 || evs[0].Calendar == "" {
		t.Errorf("events = %+v", evs)
	}
}

func TestAgendaEmptyRange(t *testing.T) {
	opts := testOptions(t)
	opts.OpenSource = func(string) (calendar.Source, error) { return fakesource.New(), nil }
	out, err := run(t, opts, "agenda", "--days", "3")
	if err != nil {
		t.Fatal(err)
	}
	if !strings.Contains(out, "No events in the next 3 days.") {
		t.Errorf("got %q", out)
	}
}

func TestAgendaRespectsHiddenCalendars(t *testing.T) {
	opts := testOptions(t)
	p := filepath.Join(os.Getenv("XDG_CONFIG_HOME"), "sundial", "config.toml")
	_ = os.MkdirAll(filepath.Dir(p), 0o755)
	_ = os.WriteFile(p, []byte("[calendars]\nhidden = [\"Work\"]\n"), 0o644)
	out, err := run(t, opts, "today")
	if err != nil {
		t.Fatal(err)
	}
	if strings.Contains(out, "Team Standup") {
		t.Error("hidden calendar events printed")
	}
}

func TestCalendarsList(t *testing.T) {
	out, err := run(t, testOptions(t), "calendars")
	if err != nil {
		t.Fatal(err)
	}
	for _, name := range []string{"Work", "Personal", "Family", "Holidays"} {
		if !strings.Contains(out, name) {
			t.Errorf("missing %s in:\n%s", name, out)
		}
	}
}

func TestBadFromDate(t *testing.T) {
	_, err := run(t, testOptions(t), "agenda", "--from", "someday")
	if err == nil || ExitCode(err) != 1 {
		t.Errorf("err = %v", err)
	}
}

func TestAccessDeniedExitCode(t *testing.T) {
	opts := testOptions(t)
	opts.OpenSource = func(string) (calendar.Source, error) { return nil, calendar.ErrAccessDenied }
	out, err := run(t, opts, "today")
	if ExitCode(err) != 2 {
		t.Fatalf("exit code = %d (err %v)", ExitCode(err), err)
	}
	if !strings.Contains(err.Error(), "Privacy & Security") {
		t.Errorf("denied message should explain how to grant access: %v / %s", err, out)
	}
}

func TestInvalidConfigFails(t *testing.T) {
	opts := testOptions(t)
	p := filepath.Join(os.Getenv("XDG_CONFIG_HOME"), "sundial", "config.toml")
	_ = os.MkdirAll(filepath.Dir(p), 0o755)
	_ = os.WriteFile(p, []byte("default_view = \"year\"\n"), 0o644)
	_, err := run(t, opts, "today")
	if err == nil || !strings.Contains(err.Error(), "config.toml:1") {
		t.Errorf("err = %v", err)
	}
}

func TestConfigInitAndPath(t *testing.T) {
	opts := testOptions(t)
	out, err := run(t, opts, "config", "path")
	if err != nil {
		t.Fatal(err)
	}
	p := strings.TrimSpace(out)
	if _, err := run(t, opts, "config", "init"); err != nil {
		t.Fatal(err)
	}
	if _, err := os.Stat(p); err != nil {
		t.Fatalf("config not created at %s", p)
	}
	if _, err := run(t, opts, "config", "init"); err == nil {
		t.Error("second init must fail")
	}
}

func TestUnknownSource(t *testing.T) {
	_, err := openSource("bogus")
	if err == nil {
		t.Fatal("expected error")
	}
	var e *ExitError
	if errors.As(err, &e) {
		t.Fatal("unknown source is a general error")
	}
}
