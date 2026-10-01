// Package week renders a 7-day (or 3-day when narrow) hour grid.
package week

import (
	"strconv"
	"strings"
	"time"

	"charm.land/lipgloss/v2"

	"github.com/JacobAtchley/sundial/internal/calendar"
	"github.com/JacobAtchley/sundial/internal/datemath"
	"github.com/JacobAtchley/sundial/internal/tui/shared"
)

const (
	gutterW   = 6 // "12pm " / "13:00 "
	minColW   = 12
	maxLanes  = 3
	allDayMax = 2
)

type Model struct {
	selected time.Time
	cursor   int // index into the selected day's timed events; -1 = none
	scroll   int // first visible half-hour row
}

func New(d time.Time, dayStart int) Model {
	return Model{selected: calendar.StartOfDay(d), cursor: -1, scroll: dayStart * 2}
}

func (m Model) Selected() time.Time { return m.selected }

func (m Model) SetDate(d time.Time) Model {
	m.selected, m.cursor = calendar.StartOfDay(d), -1
	return m
}

func narrow(ctx shared.Context) bool { return (ctx.Width-gutterW)/7-1 < minColW }

func (m Model) Days(ctx shared.Context) []time.Time {
	if narrow(ctx) {
		return datemath.Days(m.selected.AddDate(0, 0, -1), 3)
	}
	return datemath.Days(datemath.StartOfWeek(m.selected, ctx.WeekStart), 7)
}

func (m Model) Range(ctx shared.Context) calendar.Range {
	days := m.Days(ctx)
	return calendar.Range{Start: days[0], End: days[len(days)-1].AddDate(0, 0, 1)}
}

func (m Model) Title(ctx shared.Context) string {
	days := m.Days(ctx)
	first, last := days[0], days[len(days)-1]
	return first.Format("Jan 2") + " – " + last.Format("Jan 2, 2006")
}

func timed(evs []calendar.Event) []calendar.Event {
	var out []calendar.Event
	for _, e := range evs {
		if !e.AllDay {
			out = append(out, e)
		}
	}
	return out
}

func allDay(evs []calendar.Event) []calendar.Event {
	var out []calendar.Event
	for _, e := range evs {
		if e.AllDay {
			out = append(out, e)
		}
	}
	return out
}

func (m Model) SelectedEvent(ctx shared.Context) (calendar.Event, bool) {
	evs := timed(ctx.Store.EventsOn(m.selected))
	if m.cursor < 0 || m.cursor >= len(evs) {
		return calendar.Event{}, false
	}
	return evs[m.cursor], true
}

func (m Model) Nav(n shared.Nav, ctx shared.Context) Model {
	evs := timed(ctx.Store.EventsOn(m.selected))
	switch n {
	case shared.NavLeft:
		m.selected, m.cursor = m.selected.AddDate(0, 0, -1), -1
	case shared.NavRight:
		m.selected, m.cursor = m.selected.AddDate(0, 0, 1), -1
	case shared.NavPrev:
		m.selected, m.cursor = m.selected.AddDate(0, 0, -7), -1
	case shared.NavNext:
		m.selected, m.cursor = m.selected.AddDate(0, 0, 7), -1
	case shared.NavDown:
		if len(evs) == 0 || m.cursor >= len(evs)-1 {
			// Past the last event: no selection (cursor == len(evs) is a
			// "beyond" sentinel so further downs keep scrolling).
			m.cursor = len(evs)
			if len(evs) == 0 {
				m.cursor = -1
			}
			m.scroll += 2
		} else {
			m.cursor++
		}
	case shared.NavUp:
		if m.cursor <= 0 {
			m.cursor = -1
			m.scroll -= 2
		} else {
			m.cursor--
		}
	}
	if e, ok := m.SelectedEvent(ctx); ok {
		s, en := rowSpan(e, m.selected)
		vis := m.visibleRows(ctx)
		if s < m.scroll {
			m.scroll = s
		}
		if en > m.scroll+vis {
			m.scroll = max(en-vis, s-vis+1, 0)
			if s < m.scroll {
				m.scroll = s
			}
		}
	}
	m.scroll = min(max(m.scroll, 0), max(rowsPerDay-m.visibleRows(ctx), 0))
	return m
}

func (m Model) allDayLines(ctx shared.Context) int {
	n := 0
	for _, d := range m.Days(ctx) {
		n = max(n, len(allDay(ctx.Store.EventsOn(d))))
	}
	return min(n, allDayMax)
}

// visibleRows is the grid height after title, day header, all-day strip,
// and the separator.
func (m Model) visibleRows(ctx shared.Context) int {
	return max(ctx.Height-2-m.allDayLines(ctx)-1, 0)
}

func (m Model) View(ctx shared.Context) string {
	th := ctx.Theme
	days := m.Days(ctx)
	// Minimum: gutter + 3-char columns with separators; title + header +
	// all-day lines + separator + at least one grid row.
	if ctx.Width < gutterW+4*len(days) || ctx.Height < 4+m.allDayLines(ctx) {
		if ctx.Width <= 0 || ctx.Height <= 0 {
			return ""
		}
		return th.Muted.Render(shared.Truncate("Too small for week view", ctx.Width))
	}
	colW := max((ctx.Width-gutterW)/len(days)-1, 3)
	totalW := gutterW + (colW+1)*len(days)
	today := calendar.StartOfDay(ctx.Now)
	sep := th.Border.Render("│")
	scroll := min(max(m.scroll, 0), max(rowsPerDay-m.visibleRows(ctx), 0))

	var out []string
	out = append(out, lipgloss.PlaceHorizontal(totalW, lipgloss.Center, th.Title.Render(shared.Truncate(m.Title(ctx), totalW))))

	header := strings.Repeat(" ", gutterW)
	for _, d := range days {
		label := shared.PadRight(" "+d.Format("Mon 2"), colW)
		switch {
		case d.Equal(m.selected):
			label = th.Selected.Render(label)
		case d.Equal(today):
			label = th.Today.Render(label)
		default:
			label = th.DayHeader.Render(label)
		}
		header += sep + label
	}
	out = append(out, header)

	for line := range m.allDayLines(ctx) {
		row := th.Muted.Render(shared.PadRight("all", gutterW))
		if line > 0 {
			row = strings.Repeat(" ", gutterW)
		}
		for _, d := range days {
			evs := allDay(ctx.Store.EventsOn(d))
			cell := strings.Repeat(" ", colW)
			switch {
			case line == allDayMax-1 && len(evs) > allDayMax:
				cell = th.Muted.Render(shared.PadRight(" +"+strconv.Itoa(len(evs)-allDayMax+1)+" more", colW))
			case line < len(evs):
				c, _ := ctx.Store.Calendar(evs[line].CalendarID)
				cell = th.Chip(c.Color).Render(shared.PadRight(" "+shared.OneLine(evs[line].Title), colW))
			}
			row += sep + cell
		}
		out = append(out, row)
	}
	out = append(out, th.Border.Render(strings.Repeat("─", totalW)))

	cols := make([][]string, len(days))
	for i, d := range days {
		cols[i] = m.column(ctx, d, colW, d.Equal(today))
	}
	for r := scroll; r < min(scroll+m.visibleRows(ctx), rowsPerDay); r++ {
		gutter := strings.Repeat(" ", gutterW)
		if r%2 == 0 {
			gutter = th.Muted.Render(shared.PadRight(shared.FormatHour(r/2, ctx.Use24h), gutterW))
		}
		row := gutter
		for i := range days {
			row += sep + cols[i][r]
		}
		out = append(out, row)
	}
	return strings.Join(out, "\n")
}

// column renders all 48 rows for one day.
func (m Model) column(ctx shared.Context, day time.Time, colW int, isToday bool) []string {
	th := ctx.Theme
	evs := timed(ctx.Store.EventsOn(day))
	lanes, n := assignLanes(evs, day)
	n = max(min(n, maxLanes), 1)
	laneW := colW / n
	widths := make([]int, n)
	for i := range widths {
		widths[i] = laneW
	}
	widths[n-1] += colW - laneW*n

	cells := make([][]string, rowsPerDay)
	for r := range cells {
		cells[r] = make([]string, n)
		for l := range n {
			cells[r][l] = strings.Repeat(" ", widths[l])
		}
	}
	if isToday {
		nowRow := (ctx.Now.Hour()*60 + ctx.Now.Minute()) / 30
		for l := range n {
			cells[nowRow][l] = th.Accent.Render(strings.Repeat("─", widths[l]))
		}
	}
	selected := -1
	if day.Equal(m.selected) {
		selected = m.cursor
	}
	// Events in hidden lanes (>= maxLanes) show a muted "+N" marker on their
	// start row in the last lane, N being the hidden events starting there.
	hiddenAt := map[int]int{}
	draw := func(i, l int) {
		e := evs[i]
		c, _ := ctx.Store.Calendar(e.CalendarID)
		style := th.Chip(c.Color)
		if i == selected {
			style = th.Selected
		}
		s, en := rowSpan(e, day)
		for r := s; r < en; r++ {
			// Title first so narrow lanes still show it; time on the second row.
			text := ""
			switch r {
			case s:
				text = " " + shared.OneLine(e.Title)
			case s + 1:
				text = " " + shared.FormatClock(e.Start, ctx.Use24h)
			}
			cells[r][l] = style.Render(shared.PadRight(text, widths[l]))
		}
	}
	for i, e := range evs {
		if lanes[i] >= maxLanes {
			s, _ := rowSpan(e, day)
			hiddenAt[s]++
			continue
		}
		draw(i, lanes[i])
	}
	for r, cnt := range hiddenAt {
		cells[r][n-1] = th.Muted.Render(shared.PadRight(" +"+strconv.Itoa(cnt), widths[n-1]))
	}
	if selected >= 0 && selected < len(evs) && lanes[selected] >= maxLanes {
		draw(selected, n-1)
	}
	out := make([]string, rowsPerDay)
	for r := range out {
		out[r] = strings.Join(cells[r], "")
	}
	return out
}
