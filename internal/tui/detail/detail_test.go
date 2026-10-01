package detail

import (
	"strings"
	"testing"
	"time"

	"charm.land/lipgloss/v2"
	"github.com/charmbracelet/x/ansi"
	"github.com/charmbracelet/x/exp/golden"

	"github.com/JacobAtchley/sundial/internal/calendar"
	"github.com/JacobAtchley/sundial/internal/tui/shared"
)

var day = time.Date(2026, 10, 5, 0, 0, 0, 0, time.UTC)

func planning() calendar.Event {
	return calendar.Event{
		ID: "planning", CalendarID: "work", Title: "Sprint Planning",
		Start: day.Add(10 * time.Hour), End: day.Add(11 * time.Hour),
		Location: "Room 4", URL: "https://example.com/planning",
		Attendees: []string{"Alex Rivera", "Sam Chen", "Priya Patel"},
		Notes:     "Review the backlog and pick sprint goals.",
		Status:    calendar.StatusTentative,
	}
}

func TestDetailGolden(t *testing.T) {
	ctx := shared.TestContext(80, 30)
	cal, _ := ctx.Store.Calendar("work")
	golden.RequireEqual(t, ansi.Strip(View(planning(), cal, ctx)))
}

func TestDetailFitsNarrowTerminal(t *testing.T) {
	ctx := shared.TestContext(30, 30)
	cal, _ := ctx.Store.Calendar("work")
	e := planning()
	e.Notes = strings.Repeat("very long notes ", 40)
	for _, line := range strings.Split(View(e, cal, ctx), "\n") {
		if lipgloss.Width(line) > 30 {
			t.Fatalf("line width %d > 30", lipgloss.Width(line))
		}
	}
}

func TestWhen(t *testing.T) {
	timed := calendar.Event{Start: day.Add(9*time.Hour + 30*time.Minute), End: day.Add(9*time.Hour + 45*time.Minute)}
	if got := When(timed, false); got != "Mon, Oct 5 · 9:30am–9:45am" {
		t.Errorf("timed = %q", got)
	}
	one := calendar.Event{AllDay: true, Start: day, End: day.AddDate(0, 0, 1)}
	if got := When(one, false); got != "Mon, Oct 5 · all day" {
		t.Errorf("one day = %q", got)
	}
	multi := calendar.Event{AllDay: true, Start: day.AddDate(0, 0, 1), End: day.AddDate(0, 0, 3)}
	if got := When(multi, false); got != "Tue, Oct 6 – Wed, Oct 7 · all day" {
		t.Errorf("multi = %q", got)
	}
	overnight := calendar.Event{Start: day.Add(22 * time.Hour), End: day.Add(24*time.Hour + 30*time.Minute)}
	if got := When(overnight, false); got != "Mon, Oct 5 10:00pm – Tue, Oct 6 12:30am" {
		t.Errorf("overnight = %q", got)
	}
}

func TestMeetingURL(t *testing.T) {
	cases := []struct {
		e    calendar.Event
		want string
	}{
		{calendar.Event{URL: "https://example.com/a"}, "https://example.com/a"},
		{calendar.Event{URL: "file:///etc/passwd", Location: "https://example.com/zoom"}, "https://example.com/zoom"},
		{calendar.Event{Notes: "Join: https://example.com/meet?id=1 thanks"}, "https://example.com/meet?id=1"},
		{calendar.Event{URL: "zoommtg://example.com/join?confno=1"}, "zoommtg://example.com/join?confno=1"},
		{calendar.Event{URL: "javascript:alert(1)"}, ""},
		{calendar.Event{}, ""},
	}
	for _, tc := range cases {
		if got := MeetingURL(tc.e); got != tc.want {
			t.Errorf("MeetingURL(%+v) = %q, want %q", tc.e, got, tc.want)
		}
	}
}

func TestDetailSanitizesEscapeSequences(t *testing.T) {
	ctx := shared.TestContext(80, 30)
	cal, _ := ctx.Store.Calendar("work")
	e := planning()
	// Inject escape sequences into multiple fields
	e.Title = "Evil\x1b[2JTitle"
	e.Notes = "Notes\x1b]52;c;ZXZpbA==\x07Evil"
	e.Attendees = []string{"User\x1b[31m", "Normal\x1b]52;c;test\x07"}

	view := View(e, cal, ctx)

	// Check that escape sequences are not present in the output
	if strings.Contains(view, "\x1b[2J") {
		t.Error("View output contains \\x1b[2J escape sequence")
	}
	if strings.Contains(view, "]52;") {
		t.Error("View output contains ]52; OSC sequence")
	}
}
