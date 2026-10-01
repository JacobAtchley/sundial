package shared

import (
	"strings"
	"testing"
	"time"

	"charm.land/lipgloss/v2"

	"github.com/JacobAtchley/sundial/internal/calendar"
)

func TestOneLineFlattensWhitespace(t *testing.T) {
	if got := OneLine("a\nb\tc\r\n  d"); got != "a b c d" {
		t.Errorf("got %q", got)
	}
}

func TestOneLineRemovesEscapeSequences(t *testing.T) {
	// OSC sequence
	if got := OneLine("a\x1b]0;evil\x07b"); !noEscapeIn(got, "\x1b", "\x07") {
		t.Errorf("got %q, contains escape chars", got)
	}
	// SGR sequence
	if got := OneLine("x\x1b[31mred\x1b[0m"); got != "xred" {
		t.Errorf("got %q, want %q", got, "xred")
	}
}

func TestCleanTextPreservesNewlines(t *testing.T) {
	if got := CleanText("line1\nline2\x00\x9b"); got != "line1\nline2" {
		t.Errorf("got %q, want %q", got, "line1\nline2")
	}
}

func noEscapeIn(s string, patterns ...string) bool {
	for _, p := range patterns {
		if strings.Contains(s, p) {
			return false
		}
	}
	return true
}

func TestTruncateByDisplayWidth(t *testing.T) {
	cases := []struct {
		in string
		w  int
	}{
		{"Quarterly planning sync 🚀", 10},
		{"日本語のタイトルです", 7},
		{"short", 10},
		{"anything", 0},
		{"anything", 1},
	}
	for _, tc := range cases {
		got := Truncate(tc.in, tc.w)
		if lipgloss.Width(got) > tc.w {
			t.Errorf("Truncate(%q, %d) = %q, width %d", tc.in, tc.w, got, lipgloss.Width(got))
		}
	}
	if got := Truncate("short", 10); got != "short" {
		t.Errorf("short string changed: %q", got)
	}
}

func TestPadRightExactWidth(t *testing.T) {
	for _, s := range []string{"", "abc", "日本語", "a very long string that overflows"} {
		if got := PadRight(s, 8); lipgloss.Width(got) != 8 {
			t.Errorf("PadRight(%q) width = %d", s, lipgloss.Width(got))
		}
	}
}

func TestFormatClock(t *testing.T) {
	tm := time.Date(2026, 10, 1, 13, 5, 0, 0, time.UTC)
	if got := FormatClock(tm, false); got != "1:05pm" {
		t.Errorf("12h = %q", got)
	}
	if got := FormatClock(tm, true); got != "13:05" {
		t.Errorf("24h = %q", got)
	}
	if got := FormatHour(0, false); got != "12am" {
		t.Errorf("hour 0 = %q", got)
	}
	if got := FormatHour(12, false); got != "12pm" {
		t.Errorf("hour 12 = %q", got)
	}
	if got := FormatHour(9, true); got != "09:00" {
		t.Errorf("hour 9 24h = %q", got)
	}
}

func TestFormatSpan(t *testing.T) {
	st := time.Date(2026, 10, 1, 9, 30, 0, 0, time.UTC)
	e := calendar.Event{Start: st, End: st.Add(15 * time.Minute)}
	if got := FormatSpan(e, false); got != "9:30am–9:45am" {
		t.Errorf("span = %q", got)
	}
	e.AllDay = true
	if got := FormatSpan(e, false); got != "all day" {
		t.Errorf("all-day span = %q", got)
	}
}

func TestTestContextLoadsDemo(t *testing.T) {
	ctx := TestContext(80, 24)
	if len(ctx.Store.EventsOn(TestNow)) == 0 {
		t.Fatal("test context has no events on the anchor day")
	}
}
