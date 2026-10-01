package palette

import (
	"strings"
	"testing"
	"time"

	tea "charm.land/bubbletea/v2"
	"charm.land/lipgloss/v2"
	"github.com/charmbracelet/x/ansi"

	"github.com/JacobAtchley/sundial/internal/config"
	"github.com/JacobAtchley/sundial/internal/tui/shared"
	"github.com/JacobAtchley/sundial/internal/tui/theme"
)

var now = time.Date(2026, 10, 1, 8, 0, 0, 0, time.UTC)

func msg(m tea.Msg) func() tea.Msg { return func() tea.Msg { return m } }

func commands() []Command {
	return []Command{
		{ID: "view.month", Title: "Go to month view", Keys: "m", Run: msg(shared.SwitchViewMsg{View: shared.ViewMonth})},
		{ID: "view.week", Title: "Go to week view", Keys: "w", Run: msg(shared.SwitchViewMsg{View: shared.ViewWeek})},
		{ID: "app.quit", Title: "Quit", Keys: "q", Run: msg(shared.QuitMsg{})},
	}
}

func open(search func(string) []Command) Model {
	m, _ := New().Open(commands(), search, now)
	return m
}

func typeText(m Model, s string) Model {
	for _, r := range s {
		m, _, _ = m.Update(tea.KeyPressMsg{Code: r, Text: string(r)})
	}
	return m
}

func press(m Model, code rune) (Model, Result) {
	m, _, res := m.Update(tea.KeyPressMsg{Code: code})
	return m, res
}

func TestEmptyQueryListsAll(t *testing.T) {
	if got := len(open(nil).Matches()); got != 3 {
		t.Errorf("matches = %d", got)
	}
}

func TestFuzzyMatch(t *testing.T) {
	m := typeText(open(nil), "week")
	if got := m.Matches(); len(got) == 0 || got[0].ID != "view.week" {
		t.Fatalf("matches = %+v", got)
	}
	_, res := press(m, tea.KeyEnter)
	if !res.Done || res.Msg != (shared.SwitchViewMsg{View: shared.ViewWeek}) {
		t.Errorf("result = %+v", res)
	}
}

func TestDateQueryIsTopResult(t *testing.T) {
	m := typeText(open(nil), "2026-10-15")
	_, res := press(m, tea.KeyEnter)
	g, ok := res.Msg.(shared.GotoDateMsg)
	if !ok || !g.Date.Equal(time.Date(2026, 10, 15, 0, 0, 0, 0, time.UTC)) {
		t.Errorf("result = %+v", res)
	}
	m = typeText(open(nil), "mon")
	if got := m.Matches(); len(got) < 2 || !strings.HasPrefix(got[0].Title, "Go to Mon, Oct 5") || got[1].ID != "view.month" {
		t.Errorf("'mon' should offer the date first, then month view: %+v", got)
	}
}

func TestCursorAndEscape(t *testing.T) {
	m := open(nil)
	m, _ = press(m, tea.KeyDown)
	m, _ = press(m, tea.KeyDown)
	m, _ = press(m, tea.KeyDown) // clamps at last
	_, res := press(m, tea.KeyEnter)
	if res.Msg != (shared.QuitMsg{}) {
		t.Errorf("third item = %+v", res)
	}
	_, res = press(open(nil), tea.KeyEscape)
	if !res.Done || res.Msg != nil {
		t.Errorf("esc result = %+v", res)
	}
}

func TestSearchResultsAppended(t *testing.T) {
	search := func(q string) []Command {
		if strings.Contains(q, "dent") {
			return []Command{{ID: "event.dentist", Title: "Event: Dentist — Sun, Oct 4"}}
		}
		return nil
	}
	m := typeText(open(search), "dent")
	got := m.Matches()
	if len(got) == 0 || got[len(got)-1].ID != "event.dentist" {
		t.Errorf("matches = %+v", got)
	}
}

func TestNoMatchesAndWidth(t *testing.T) {
	th := theme.New(config.Default().Theme, true)
	m := typeText(open(nil), "zzzzzz")
	out := m.View(th, 80)
	if !strings.Contains(ansi.Strip(out), "No matches") {
		t.Errorf("got:\n%s", ansi.Strip(out))
	}
	_, res := press(m, tea.KeyEnter)
	if !res.Done || res.Msg != nil {
		t.Errorf("enter with no matches = %+v", res)
	}
	for _, w := range []int{24, 80, 200} {
		for _, line := range strings.Split(open(nil).View(th, w), "\n") {
			if lipgloss.Width(line) > w {
				t.Fatalf("width %d: line %d", w, lipgloss.Width(line))
			}
		}
	}
}
