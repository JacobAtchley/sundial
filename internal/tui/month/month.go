// Package month renders a calendar month grid.
package month

import (
	"fmt"
	"strings"
	"time"

	"charm.land/lipgloss/v2"

	"github.com/JacobAtchley/sundial/internal/calendar"
	"github.com/JacobAtchley/sundial/internal/datemath"
	"github.com/JacobAtchley/sundial/internal/tui/shared"
)

type Model struct {
	selected time.Time
}

func New(d time.Time) Model { return Model{selected: calendar.StartOfDay(d)} }

func (m Model) Selected() time.Time { return m.selected }

func (m Model) SetDate(d time.Time) Model { return New(d) }

func (m Model) Title() string { return m.selected.Format("January 2006") }

func (m Model) grid(ws time.Weekday) [][]time.Time {
	return datemath.MonthGrid(m.selected.Year(), m.selected.Month(), ws, m.selected.Location())
}

func (m Model) Range(ws time.Weekday) calendar.Range {
	g := m.grid(ws)
	last := g[len(g)-1][6]
	return calendar.Range{Start: g[0][0], End: last.AddDate(0, 0, 1)}
}

func (m Model) Nav(n shared.Nav) Model {
	switch n {
	case shared.NavLeft:
		m.selected = m.selected.AddDate(0, 0, -1)
	case shared.NavRight:
		m.selected = m.selected.AddDate(0, 0, 1)
	case shared.NavUp:
		m.selected = m.selected.AddDate(0, 0, -7)
	case shared.NavDown:
		m.selected = m.selected.AddDate(0, 0, 7)
	case shared.NavPrev:
		m.selected = datemath.AddMonthsClamped(m.selected, -1)
	case shared.NavNext:
		m.selected = datemath.AddMonthsClamped(m.selected, 1)
	}
	return m
}

func (m Model) View(ctx shared.Context) string {
	if ctx.Width < 7 || ctx.Height < 3 {
		return ""
	}
	th := ctx.Theme
	grid := m.grid(ctx.WeekStart)
	rows := len(grid)
	cellW := max((ctx.Width-6)/7, 1)
	totalW := cellW*7 + 6
	avail := ctx.Height - 2 // title + weekday header
	cellH := max((avail-(rows-1))/rows, 1)
	sep := th.Border.Render("│")
	today := calendar.StartOfDay(ctx.Now)

	var out []string
	out = append(out, lipgloss.PlaceHorizontal(totalW, lipgloss.Center, th.Title.Render(m.Title())))

	var head []string
	for _, d := range grid[0] {
		head = append(head, th.DayHeader.Render(shared.PadRight(" "+d.Format("Mon"), cellW)))
	}
	out = append(out, strings.Join(head, " "))

	rule := th.Border.Render(strings.Repeat("─", cellW))
	for r, week := range grid {
		if r > 0 {
			out = append(out, strings.Join(repeat(rule, 7), th.Border.Render("┼")))
		}
		cells := make([][]string, 7)
		for i, day := range week {
			cells[i] = m.cell(ctx, day, today, cellW, cellH)
		}
		for line := range cellH {
			parts := make([]string, 7)
			for i := range 7 {
				parts[i] = cells[i][line]
			}
			out = append(out, strings.Join(parts, sep))
		}
	}
	return strings.Join(out, "\n")
}

func (m Model) cell(ctx shared.Context, day, today time.Time, w, h int) []string {
	th := ctx.Theme
	lines := make([]string, h)
	num := fmt.Sprintf("%2d", day.Day())
	switch {
	case day.Equal(m.selected):
		num = th.Selected.Render(num)
	case day.Equal(today):
		num = th.Today.Render(num)
	case day.Month() != m.selected.Month():
		num = th.Muted.Render(num)
	}
	lines[0] = shared.PadRight(num, w)

	evs := ctx.Store.EventsOn(day)
	slots := h - 1
	show := evs
	more := 0
	if len(evs) > slots {
		show, more = evs[:max(slots-1, 0)], len(evs)-max(slots-1, 0)
	}
	for i, e := range show {
		c, _ := ctx.Store.Calendar(e.CalendarID)
		text := th.Dot(c.Color) + " " + shared.Truncate(shared.OneLine(e.Title), w-2)
		if day.Month() != m.selected.Month() {
			text = th.Dot(c.Color) + " " + th.Muted.Render(shared.Truncate(shared.OneLine(e.Title), w-2))
		}
		lines[1+i] = shared.PadRight(text, w)
	}
	if more > 0 && slots > 0 {
		lines[h-1] = shared.PadRight(th.Muted.Render(fmt.Sprintf("+%d more", more)), w)
	}
	for i := range lines {
		if lines[i] == "" {
			lines[i] = strings.Repeat(" ", w)
		}
	}
	return lines
}

func repeat(s string, n int) []string {
	out := make([]string, n)
	for i := range out {
		out[i] = s
	}
	return out
}
