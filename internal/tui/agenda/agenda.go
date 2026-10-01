// Package agenda renders a scrolling, day-grouped list of events.
package agenda

import (
	"fmt"
	"strings"
	"time"

	"charm.land/bubbles/v2/textinput"
	tea "charm.land/bubbletea/v2"
	"charm.land/lipgloss/v2"

	"github.com/JacobAtchley/sundial/internal/calendar"
	"github.com/JacobAtchley/sundial/internal/tui/shared"
)

type Model struct {
	start  time.Time
	days   int
	step   int // days added when scrolling past the end
	cursor int // index into the visible event list
	offset int // first visible line

	query     string
	filtering bool
	input     textinput.Model
}

func New(start time.Time, days int) Model {
	in := textinput.New()
	in.Prompt = "/ "
	in.Placeholder = "filter by title or location"
	days = max(days, 1)
	return Model{start: calendar.StartOfDay(start), days: days, step: days, input: in}
}

func (m Model) Range() calendar.Range {
	return calendar.Range{Start: m.start, End: m.start.AddDate(0, 0, m.days)}
}

func (m Model) SetDate(d time.Time) Model {
	m.start, m.cursor, m.offset, m.days = calendar.StartOfDay(d), 0, 0, m.step
	return m
}

func (m Model) Title() string {
	end := m.start.AddDate(0, 0, m.days-1)
	return m.start.Format("Jan 2") + " – " + end.Format("Jan 2")
}

type item struct {
	day   time.Time
	event calendar.Event
}

func (m Model) items(ctx shared.Context) []item {
	var out []item
	q := strings.ToLower(m.query)
	for _, day := range m.Range().Days() {
		for _, e := range ctx.Store.EventsOn(day) {
			if q != "" && !strings.Contains(strings.ToLower(e.Title+" "+e.Location), q) {
				continue
			}
			out = append(out, item{day: day, event: e})
		}
	}
	return out
}

func (m Model) Selected(ctx shared.Context) (calendar.Event, bool) {
	items := m.items(ctx)
	if m.cursor < 0 || m.cursor >= len(items) {
		return calendar.Event{}, false
	}
	return items[m.cursor].event, true
}

func (m Model) SelectedDay(ctx shared.Context) time.Time {
	items := m.items(ctx)
	if m.cursor < 0 || m.cursor >= len(items) {
		return m.start
	}
	return items[m.cursor].day
}

func (m Model) Nav(n shared.Nav, ctx shared.Context) Model {
	items := m.items(ctx)
	switch n {
	case shared.NavDown:
		if m.cursor < len(items)-1 {
			m.cursor++
		} else {
			// Past the end: grow the window; the app loads the new days.
			m.days = min(m.days+m.step, 365)
		}
	case shared.NavUp:
		if m.cursor > 0 {
			m.cursor--
		}
	case shared.NavRight:
		for i := m.cursor + 1; i < len(items); i++ {
			if !items[i].day.Equal(items[m.cursor].day) {
				m.cursor = i
				break
			}
		}
	case shared.NavLeft:
		if m.cursor > 0 && m.cursor < len(items) {
			cur := items[m.cursor].day
			i := m.cursor - 1
			for i > 0 && items[i].day.Equal(cur) {
				i--
			}
			for i > 0 && items[i-1].day.Equal(items[i].day) {
				i--
			}
			m.cursor = i
		}
	case shared.NavNext:
		return m.SetDate(m.start.AddDate(0, 0, m.step))
	case shared.NavPrev:
		return m.SetDate(m.start.AddDate(0, 0, -m.step))
	}
	return m.scrollTo(ctx)
}

func (m Model) Filtering() bool { return m.filtering }

func (m Model) StartFilter() (Model, tea.Cmd) {
	m.filtering = true
	m.input.SetValue(m.query)
	return m, m.input.Focus()
}

// UpdateFilter handles keys while the filter input is focused: enter
// applies, esc clears.
func (m Model) UpdateFilter(msg tea.Msg) (Model, tea.Cmd) {
	if k, ok := msg.(tea.KeyPressMsg); ok {
		switch k.String() {
		case "enter":
			m.query, m.filtering, m.cursor, m.offset = strings.TrimSpace(m.input.Value()), false, 0, 0
			m.input.Blur()
			return m, nil
		case "esc":
			m.query, m.filtering, m.cursor, m.offset = "", false, 0, 0
			m.input.Blur()
			return m, nil
		}
	}
	var cmd tea.Cmd
	m.input, cmd = m.input.Update(msg)
	return m, cmd
}

// lines renders every line and reports the line index of the cursor.
func (m Model) lines(ctx shared.Context) ([]string, int) {
	th := ctx.Theme
	items := m.items(ctx)
	if len(items) == 0 {
		msg := fmt.Sprintf("No events in the next %d days.", m.days)
		if m.query != "" {
			msg = fmt.Sprintf("No events match %q.", m.query)
		}
		return []string{th.Muted.Render(shared.Truncate(msg, ctx.Width))}, -1
	}
	const spanW = 16
	var out []string
	selLine := -1
	var lastDay time.Time
	for i, it := range items {
		if !it.day.Equal(lastDay) {
			if len(out) > 0 {
				out = append(out, "")
			}
			header := th.DayHeader.Render(it.day.Format("Monday, January 2"))
			if it.day.Equal(calendar.StartOfDay(ctx.Now)) {
				header += th.Today.Render(" · Today")
			}
			out = append(out, shared.Truncate(header, ctx.Width))
			lastDay = it.day
		}
		e := it.event
		c, _ := ctx.Store.Calendar(e.CalendarID)
		marker := "  "
		title := shared.OneLine(e.Title)
		if i == m.cursor {
			marker = th.Selected.Render("▌") + " "
			selLine = len(out)
		}
		prefix := marker + th.Muted.Render(shared.PadRight(shared.FormatSpan(e, ctx.Use24h), spanW)) + " " + th.Dot(c.Color) + " "
		room := ctx.Width - 2 - spanW - 3
		text := shared.Truncate(title, room)
		if i == m.cursor {
			text = th.Bold.Render(text)
		}
		if loc := shared.OneLine(e.Location); loc != "" {
			if left := room - lipgloss.Width(title) - 3; left > 3 {
				text += th.Muted.Render(" · " + shared.Truncate(loc, left))
			}
		}
		out = append(out, shared.Truncate(prefix+text, ctx.Width))
	}
	return out, selLine
}

func (m Model) scrollTo(ctx shared.Context) Model {
	_, sel := m.lines(ctx)
	h := m.bodyHeight(ctx)
	if sel < 0 || h <= 0 {
		m.offset = 0
		return m
	}
	if sel < m.offset {
		m.offset = max(sel-1, 0) // keep the day header visible when possible
	}
	if sel >= m.offset+h {
		m.offset = sel - h + 1
	}
	return m
}

func (m Model) bodyHeight(ctx shared.Context) int {
	if m.filtering {
		return ctx.Height - 1
	}
	return ctx.Height
}

func (m Model) View(ctx shared.Context) string {
	if ctx.Width <= 0 || ctx.Height <= 0 {
		return ""
	}
	lines, _ := m.lines(ctx)
	h := m.bodyHeight(ctx)
	start := min(m.offset, max(len(lines)-1, 0))
	end := min(start+max(h, 0), len(lines))
	visible := lines[start:end]
	if m.filtering {
		visible = append(visible, shared.Truncate(m.input.View(), ctx.Width))
	}
	return strings.Join(visible, "\n")
}
