// Package palette implements the fuzzy command palette.
package palette

import (
	"strings"
	"time"

	"charm.land/bubbles/v2/textinput"
	tea "charm.land/bubbletea/v2"
	"charm.land/lipgloss/v2"
	"github.com/sahilm/fuzzy"

	"github.com/JacobAtchley/sundial/internal/datemath"
	"github.com/JacobAtchley/sundial/internal/tui/shared"
	"github.com/JacobAtchley/sundial/internal/tui/theme"
)

// Command is one palette entry. Run returns the message the app handles.
type Command struct {
	ID    string
	Title string
	Keys  string // display only
	Run   func() tea.Msg
}

// Result tells the app the palette closed and which message (if any) to
// dispatch.
type Result struct {
	Done bool
	Msg  tea.Msg
}

const (
	maxResults = 10
	maxWidth   = 72
)

type Model struct {
	input    textinput.Model
	commands []Command
	search   func(q string) []Command
	now      time.Time
	matches  []Command
	cursor   int
}

func New() Model {
	in := textinput.New()
	in.Prompt = "› "
	in.Placeholder = "command, date (fri, +3d, 2026-10-15), or event"
	return Model{input: in}
}

func (m Model) Open(cmds []Command, search func(q string) []Command, now time.Time) (Model, tea.Cmd) {
	m.commands, m.search, m.now, m.cursor = cmds, search, now, 0
	m.input.SetValue("")
	m.matches = m.compute()
	return m, m.input.Focus()
}

func (m Model) Matches() []Command { return m.matches }

func (m Model) Update(msg tea.Msg) (Model, tea.Cmd, Result) {
	if k, ok := msg.(tea.KeyPressMsg); ok {
		switch k.String() {
		case "esc", "ctrl+c":
			m.input.Blur()
			return m, nil, Result{Done: true}
		case "enter":
			m.input.Blur()
			if m.cursor < len(m.matches) && m.matches[m.cursor].Run != nil {
				return m, nil, Result{Done: true, Msg: m.matches[m.cursor].Run()}
			}
			return m, nil, Result{Done: true}
		case "up", "ctrl+p":
			if m.cursor > 0 {
				m.cursor--
			}
			return m, nil, Result{}
		case "down", "ctrl+n":
			if m.cursor < len(m.matches)-1 {
				m.cursor++
			}
			return m, nil, Result{}
		}
	}
	before := m.input.Value()
	var cmd tea.Cmd
	m.input, cmd = m.input.Update(msg)
	if m.input.Value() != before {
		m.matches, m.cursor = m.compute(), 0
	}
	return m, cmd, Result{}
}

func (m Model) compute() []Command {
	q := strings.TrimSpace(m.input.Value())
	var out []Command
	if t, ok := datemath.ParseDateExpr(q, m.now); ok {
		date := t
		out = append(out, Command{
			ID:    "goto.date",
			Title: "Go to " + date.Format("Mon, Jan 2 2006"),
			Run:   func() tea.Msg { return shared.GotoDateMsg{Date: date} },
		})
	}
	if q == "" {
		return append(out, m.commands...)
	}
	titles := make([]string, len(m.commands))
	for i, c := range m.commands {
		titles[i] = c.Title
	}
	for _, match := range fuzzy.Find(q, titles) {
		out = append(out, m.commands[match.Index])
	}
	if m.search != nil && len(q) >= 2 {
		out = append(out, m.search(q)...)
	}
	return out
}

func (m Model) View(th theme.Theme, width int) string {
	w := min(maxWidth, width-4)
	if w < 20 {
		w = max(width, 6)
	}
	inner := w - 4
	m.input.SetWidth(max(inner-3, 1))
	lines := []string{shared.Truncate(m.input.View(), inner), th.Border.Render(strings.Repeat("─", inner))}
	if len(m.matches) == 0 {
		lines = append(lines, th.Muted.Render("No matches"))
	}
	start := 0
	if m.cursor >= maxResults {
		start = m.cursor - maxResults + 1
	}
	for i := start; i < min(len(m.matches), start+maxResults); i++ {
		c := m.matches[i]
		keys := shared.Truncate(shared.OneLine(c.Keys), max(inner/3, 0))
		titleW := max(inner-lipgloss.Width(keys)-1, 1)
		title := shared.PadRight(" "+shared.OneLine(c.Title), titleW)
		if i == m.cursor {
			title = th.Selected.Render(title)
		}
		lines = append(lines, title+" "+th.Muted.Render(keys))
	}
	return lipgloss.NewStyle().
		Border(lipgloss.RoundedBorder()).
		BorderForeground(th.P.Header).
		Padding(0, 1).
		Width(w).
		Render(strings.Join(lines, "\n"))
}
