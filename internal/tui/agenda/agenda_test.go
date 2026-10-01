package agenda

import (
	"strings"
	"testing"

	tea "charm.land/bubbletea/v2"
	"charm.land/lipgloss/v2"
	"github.com/charmbracelet/x/ansi"
	"github.com/charmbracelet/x/exp/golden"

	"github.com/JacobAtchley/sundial/internal/calendar"
	"github.com/JacobAtchley/sundial/internal/fakesource"
	"github.com/JacobAtchley/sundial/internal/store"
	"github.com/JacobAtchley/sundial/internal/tui/shared"
)

func render(m Model, ctx shared.Context) string { return ansi.Strip(m.View(ctx)) }

func TestAgendaDefault(t *testing.T) {
	ctx := shared.TestContext(80, 20)
	golden.RequireEqual(t, render(New(shared.TestNow, 14), ctx))
}

func TestAgendaNavigationMovesSelection(t *testing.T) {
	ctx := shared.TestContext(80, 20)
	m := New(shared.TestNow, 14)
	first, _ := m.Selected(ctx)
	m = m.Nav(shared.NavDown, ctx).Nav(shared.NavDown, ctx)
	third, _ := m.Selected(ctx)
	if first.Title != "Team Standup" || third.Title != "Focus Time" {
		t.Errorf("selection %q -> %q", first.Title, third.Title)
	}
	m = m.Nav(shared.NavRight, ctx)
	if d := m.SelectedDay(ctx); d.Day() != 2 {
		t.Errorf("right should jump to next day with events, got %v", d)
	}
	m = m.Nav(shared.NavLeft, ctx)
	if d := m.SelectedDay(ctx); d.Day() != 1 {
		t.Errorf("left should jump back, got %v", d)
	}
}

func TestAgendaScrollsToKeepSelectionVisible(t *testing.T) {
	ctx := shared.TestContext(80, 8)
	m := New(shared.TestNow, 14)
	for range 12 {
		m = m.Nav(shared.NavDown, ctx)
	}
	sel, _ := m.Selected(ctx)
	if !strings.Contains(render(m, ctx), "▌") {
		t.Errorf("selected event %q scrolled out of view:\n%s", sel.Title, render(m, ctx))
	}
}

func TestAgendaNextExtendsAndPrevMovesBack(t *testing.T) {
	ctx := shared.TestContext(80, 20)
	m := New(shared.TestNow, 14).Nav(shared.NavNext, ctx)
	if got := m.Range().Start.Day(); got != 15 {
		t.Errorf("next start = %d", got)
	}
	m = m.Nav(shared.NavPrev, ctx)
	if !m.Range().Start.Equal(calendar.StartOfDay(shared.TestNow)) {
		t.Errorf("prev start = %v", m.Range().Start)
	}
}

func TestAgendaNavAfterGrowthUsesConfiguredStep(t *testing.T) {
	ctx := shared.TestContext(80, 20)
	m := New(shared.TestNow, 14)
	m.days = 42 // grown by scrolling
	next := m.Nav(shared.NavNext, ctx)
	if got := next.Range().Start.Day(); got != 15 {
		t.Errorf("next start = %d, want 15", got)
	}
	if d := next.Range().End.Sub(next.Range().Start).Hours() / 24; d != 14 {
		t.Errorf("SetDate should reset days to 14, got %.0f", d)
	}
	prev := m.Nav(shared.NavPrev, ctx)
	if got := prev.Range().Start; !got.Equal(calendar.StartOfDay(shared.TestNow).AddDate(0, 0, -14)) {
		t.Errorf("prev start = %v", got)
	}
}

func TestAgendaExtendsAtEnd(t *testing.T) {
	ctx := shared.TestContext(80, 20)
	m := New(shared.TestNow, 14)
	for range 200 {
		m = m.Nav(shared.NavDown, ctx)
	}
	if days := m.Range().End.Sub(m.Range().Start).Hours() / 24; days <= 14 {
		t.Errorf("range did not grow past 14 days: %.0f", days)
	}
}

func TestAgendaFilter(t *testing.T) {
	ctx := shared.TestContext(80, 20)
	m, _ := New(shared.TestNow, 14).StartFilter()
	for _, r := range "yoga" {
		m, _ = m.UpdateFilter(tea.KeyPressMsg{Code: r, Text: string(r)})
	}
	m, _ = m.UpdateFilter(tea.KeyPressMsg{Code: tea.KeyEnter})
	out := render(m, ctx)
	if !strings.Contains(out, "Yoga") || strings.Contains(out, "Team Standup") {
		t.Errorf("filter not applied:\n%s", out)
	}
	m, _ = m.StartFilter()
	m, _ = m.UpdateFilter(tea.KeyPressMsg{Code: tea.KeyEscape})
	if !strings.Contains(render(m, ctx), "Team Standup") {
		t.Error("esc should clear the filter")
	}
}

func TestAgendaEmpty(t *testing.T) {
	ctx := shared.TestContext(80, 20)
	ctx.Store = store.New(fakesource.New())
	out := render(New(shared.TestNow, 14), ctx)
	if !strings.Contains(out, "No events in the next 14 days") {
		t.Errorf("got:\n%s", out)
	}
	if _, ok := New(shared.TestNow, 14).Nav(shared.NavDown, ctx).Selected(ctx); ok {
		t.Error("nothing should be selected in an empty agenda")
	}
}

func TestAgendaLinesFitWidthAndZeroSize(t *testing.T) {
	for _, w := range []int{20, 40, 80} {
		ctx := shared.TestContext(w, 30)
		for _, line := range strings.Split(New(shared.TestNow, 14).View(ctx), "\n") {
			if lipgloss.Width(line) > w {
				t.Fatalf("width %d: line too wide (%d): %q", w, lipgloss.Width(line), ansi.Strip(line))
			}
		}
	}
	_ = New(shared.TestNow, 14).View(shared.TestContext(0, 0)) // must not panic
}
