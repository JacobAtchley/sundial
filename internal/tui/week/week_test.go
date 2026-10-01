package week

import (
	"strings"
	"testing"

	"charm.land/lipgloss/v2"
	"github.com/charmbracelet/x/ansi"
	"github.com/charmbracelet/x/exp/golden"

	"github.com/JacobAtchley/sundial/internal/fakesource"
	"github.com/JacobAtchley/sundial/internal/store"
	"github.com/JacobAtchley/sundial/internal/tui/shared"
)

func TestWeekDefault(t *testing.T) {
	ctx := shared.TestContext(110, 30)
	golden.RequireEqual(t, ansi.Strip(New(shared.TestNow, 8).View(ctx)))
}

func TestWeekNarrowFallsBackToThreeDays(t *testing.T) {
	ctx := shared.TestContext(60, 30)
	m := New(shared.TestNow, 8)
	if got := len(m.Days(ctx)); got != 3 {
		t.Fatalf("days = %d, want 3", got)
	}
	if d := m.Days(ctx); d[1].Day() != 1 {
		t.Errorf("narrow view should center the selected day, got %v", d)
	}
}

func TestWeekSelectEventsAndScroll(t *testing.T) {
	ctx := shared.TestContext(110, 14)
	m := New(shared.TestNow, 8).Nav(shared.NavDown, ctx)
	e, ok := m.SelectedEvent(ctx)
	if !ok || e.Title != "Team Standup" {
		t.Fatalf("first down selects %q", e.Title)
	}
	for range 3 {
		m = m.Nav(shared.NavDown, ctx)
	}
	e, _ = m.SelectedEvent(ctx)
	if e.Title != "1:1 with Manager" {
		t.Errorf("fourth event = %q", e.Title)
	}
	if out := ansi.Strip(m.View(ctx)); !strings.Contains(out, "1:1") {
		t.Errorf("selected event scrolled out of view:\n%s", out)
	}
	m = m.Nav(shared.NavRight, ctx)
	if _, ok := m.SelectedEvent(ctx); ok {
		t.Error("changing day should clear the event selection")
	}
	if m.Selected().Day() != 2 {
		t.Errorf("selected day = %v", m.Selected())
	}
}

func TestWeekNavPeriod(t *testing.T) {
	ctx := shared.TestContext(110, 30)
	if got := New(shared.TestNow, 8).Nav(shared.NavNext, ctx).Selected().Day(); got != 8 {
		t.Errorf("next = %d", got)
	}
}

func TestWeekShowsAllDayStripAndOverlaps(t *testing.T) {
	ctx := shared.TestContext(110, 40)
	m := New(shared.TestNow.AddDate(0, 0, 5), 8) // week containing the offsite
	out := ansi.Strip(m.View(ctx))
	if !strings.Contains(out, "Team Offsite") {
		t.Errorf("all-day strip missing:\n%s", out)
	}
	wide := shared.TestContext(140, 40) // lanes wide enough to show both titles
	m = New(shared.TestNow, 12)
	out = ansi.Strip(m.View(wide))
	if !strings.Contains(out, "Focus") || !strings.Contains(out, "1:1") {
		t.Errorf("overlapping events not both visible:\n%s", out)
	}
}

func TestWeekFitsWidthEmptyAndZero(t *testing.T) {
	for _, w := range []int{40, 60, 110, 200} {
		ctx := shared.TestContext(w, 30)
		for _, line := range strings.Split(New(shared.TestNow, 8).View(ctx), "\n") {
			if lipgloss.Width(line) > w {
				t.Fatalf("width %d: line width %d", w, lipgloss.Width(line))
			}
		}
	}
	ctx := shared.TestContext(110, 30)
	ctx.Store = store.New(fakesource.New())
	_ = New(shared.TestNow, 8).Nav(shared.NavDown, ctx).View(ctx)
	_ = New(shared.TestNow, 8).View(shared.TestContext(0, 0))
}

func TestWeekNeverExceedsBounds(t *testing.T) {
	for _, w := range []int{10, 20, 40, 60, 110, 200} {
		for _, h := range []int{3, 5, 8, 14, 30} {
			ctx := shared.TestContext(w, h)
			out := New(shared.TestNow, 8).View(ctx)
			lines := strings.Split(out, "\n")
			if len(lines) > h {
				t.Errorf("%dx%d: %d lines", w, h, len(lines))
			}
			for _, line := range lines {
				if lipgloss.Width(line) > w {
					t.Errorf("%dx%d: line width %d", w, h, lipgloss.Width(line))
				}
			}
		}
	}
}
