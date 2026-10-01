package month

import (
	"strings"
	"testing"
	"time"

	"charm.land/lipgloss/v2"
	"github.com/charmbracelet/x/ansi"
	"github.com/charmbracelet/x/exp/golden"

	"github.com/JacobAtchley/sundial/internal/fakesource"
	"github.com/JacobAtchley/sundial/internal/store"
	"github.com/JacobAtchley/sundial/internal/tui/shared"
)

func TestMonthDefault(t *testing.T) {
	ctx := shared.TestContext(80, 30)
	golden.RequireEqual(t, ansi.Strip(New(shared.TestNow).View(ctx)))
}

func TestMonthMondayStart(t *testing.T) {
	ctx := shared.TestContext(80, 30)
	ctx.WeekStart = time.Monday
	out := ansi.Strip(New(shared.TestNow).View(ctx))
	header := strings.Split(out, "\n")[1]
	if !strings.HasPrefix(strings.TrimSpace(header), "Mon") {
		t.Errorf("header = %q", header)
	}
}

func TestMonthNav(t *testing.T) {
	m := New(shared.TestNow)
	if got := m.Nav(shared.NavRight).Selected().Day(); got != 2 {
		t.Errorf("right = %d", got)
	}
	if got := m.Nav(shared.NavDown).Selected().Day(); got != 8 {
		t.Errorf("down = %d", got)
	}
	if got := m.Nav(shared.NavUp).Selected(); got.Month() != time.September || got.Day() != 24 {
		t.Errorf("up = %v", got)
	}
	jan31 := New(time.Date(2026, 1, 31, 0, 0, 0, 0, time.UTC))
	if got := jan31.Nav(shared.NavNext).Selected(); got.Month() != time.February || got.Day() != 28 {
		t.Errorf("next from Jan 31 = %v", got)
	}
}

func TestMonthMultiDayEventOnEachDay(t *testing.T) {
	ctx := shared.TestContext(120, 40)
	out := ansi.Strip(New(shared.TestNow).View(ctx))
	if got := strings.Count(out, "Team Offsite"); got != 2 {
		t.Errorf("Team Offsite appears %d times, want 2 (Oct 6 and Oct 7)", got)
	}
}

func TestMonthOverflowShowsMore(t *testing.T) {
	ctx := shared.TestContext(80, 16) // tiny cells: 1 event line each
	out := ansi.Strip(New(shared.TestNow).View(ctx))
	if !strings.Contains(out, "+") {
		t.Errorf("expected a +N more marker:\n%s", out)
	}
}

func TestMonthFitsWidthEmptyAndZero(t *testing.T) {
	for _, w := range []int{30, 80, 133} {
		ctx := shared.TestContext(w, 30)
		for _, line := range strings.Split(New(shared.TestNow).View(ctx), "\n") {
			if lipgloss.Width(line) > w {
				t.Fatalf("width %d: line width %d", w, lipgloss.Width(line))
			}
		}
	}
	ctx := shared.TestContext(80, 30)
	ctx.Store = store.New(fakesource.New())
	if out := New(shared.TestNow).View(ctx); !strings.Contains(ansi.Strip(out), "October 2026") {
		t.Error("empty month should still render the grid")
	}
	_ = New(shared.TestNow).View(shared.TestContext(0, 0))
}
