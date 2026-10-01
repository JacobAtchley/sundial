package tui

import (
	"strings"

	"charm.land/lipgloss/v2"

	"github.com/JacobAtchley/sundial/internal/tui/detail"
	"github.com/JacobAtchley/sundial/internal/tui/shared"
)

const (
	minWidth  = 40
	minHeight = 10
)

func (a *App) render() string {
	if a.width < minWidth || a.height < minHeight {
		msg := "Terminal too small: sundial needs at least 40×10."
		return shared.Truncate(msg, max(a.width, len(msg)))
	}
	if a.opts.Denied {
		return clipLines(a.deniedView(), a.height)
	}
	ctx := a.viewCtx()
	var body string
	switch a.view {
	case shared.ViewMonth:
		body = a.month.View(ctx)
	case shared.ViewWeek:
		body = a.week.View(ctx)
	default:
		body = a.agenda.View(ctx)
	}
	body = clipLines(body, ctx.Height)
	screen := lipgloss.Place(a.width, ctx.Height, lipgloss.Left, lipgloss.Top, body) + "\n" + a.statusBar()

	switch {
	case a.paletteOn:
		screen = overlay(screen, a.palette.View(a.theme, a.width), a.width, a.height, 2)
	case a.helpOn:
		screen = overlay(screen, a.helpView(), a.width, a.height, -1)
	case a.detail != nil:
		cal, _ := a.opts.Store.Calendar(a.detail.CalendarID)
		screen = overlay(screen, detail.View(*a.detail, cal, ctx), a.width, a.height, -1)
	}
	return screen
}

// overlay draws top over base, centered horizontally; y < 0 centers
// vertically too. top is clipped so it never covers the status bar (the
// last of the h lines) or extends the screen.
func overlay(base, top string, w, h, y int) string {
	avail := max(h-1, 0)
	if y < 0 {
		y = max((avail-lipgloss.Height(top))/2, 0)
	}
	y = min(y, avail)
	top = clipLines(top, avail-y)
	tw := lipgloss.Width(top)
	x := max((w-tw)/2, 0)
	return lipgloss.NewCompositor(
		lipgloss.NewLayer(base),
		lipgloss.NewLayer(top).X(x).Y(y).Z(1),
	).Render()
}

// clipLines keeps at most n lines of s.
func clipLines(s string, n int) string {
	lines := strings.Split(s, "\n")
	if len(lines) <= n {
		return s
	}
	return strings.Join(lines[:max(n, 0)], "\n")
}

func (a *App) viewTitle() string {
	switch a.view {
	case shared.ViewMonth:
		return a.month.Title()
	case shared.ViewWeek:
		return a.week.Title(a.viewCtx())
	default:
		return a.agenda.Title()
	}
}

func (a *App) statusBar() string {
	th := a.theme
	name := lipgloss.NewStyle().Background(th.P.Header).Foreground(th.P.Ink).Bold(true).Padding(0, 1).
		Render(strings.ToUpper(a.view.String()))
	left := name + th.Title.Render(" "+a.viewTitle())

	var right string
	switch {
	case a.err != nil:
		right = th.Warning.Render("✗ " + shared.OneLine(a.err.Error()))
	case a.loading:
		right = th.Muted.Render("⟳ syncing")
	case a.notice != "":
		right = th.Accent.Render(a.notice)
	case a.live:
		right = th.Accent.Render("● live")
	}
	hints := th.Muted.Render("  ? help · " + a.keys.label("palette") + " commands")

	used := lipgloss.Width(left) + lipgloss.Width(right) + lipgloss.Width(hints)
	if used >= a.width {
		hints = ""
		used = lipgloss.Width(left) + lipgloss.Width(right)
	}
	if used >= a.width {
		right = shared.Truncate(right, max(a.width-lipgloss.Width(left)-1, 0))
		used = lipgloss.Width(left) + lipgloss.Width(right)
	}
	gap := strings.Repeat(" ", max(a.width-used, 1))
	return shared.Truncate(left+gap+right+hints, a.width)
}

func (a *App) helpView() string {
	th := a.theme
	k := a.keys.label
	nav := [][2]string{
		{k("left") + " " + k("right"), "move left / right"},
		{k("up") + " " + k("down"), "move up / down"},
		{k("prev") + " " + k("next"), "previous / next period"},
		{k("open"), "open event (month: agenda for day)"},
		{k("close"), "close pane"},
		{k("filter"), "filter agenda"},
		{k("open_url"), "open event link"},
	}
	var rows []string
	rows = append(rows, th.Title.Render("Keyboard shortcuts"), "")
	keyW := 18
	for _, r := range nav {
		rows = append(rows, th.DayHeader.Render(shared.PadRight(r[0], keyW))+r[1])
	}
	rows = append(rows, "", th.Title.Render("Commands"), "")
	for _, c := range a.commands() {
		if c.Keys == "" || strings.HasPrefix(c.ID, "cal.") {
			continue
		}
		rows = append(rows, th.DayHeader.Render(shared.PadRight(c.Keys, keyW))+c.Title)
	}
	rows = append(rows, "", th.Muted.Render("All commands: "+k("palette")+" · close: "+k("close")))
	w := min(60, a.width-2)
	return lipgloss.NewStyle().Border(lipgloss.RoundedBorder()).BorderForeground(th.P.Header).
		Padding(0, 1).Width(w).Render(strings.Join(rows, "\n"))
}

func (a *App) deniedView() string {
	th := a.theme
	body := strings.Join([]string{
		th.Title.Render("sundial needs calendar access"),
		"",
		"macOS hasn't given sundial access to your calendars.",
		"Open System Settings → Privacy & Security → Calendars,",
		"turn on access for your terminal app, then restart sundial.",
		"",
		th.Muted.Render(a.keys.label("open_url") + " open System Settings · " + a.keys.label("quit") + " quit"),
	}, "\n")
	box := lipgloss.NewStyle().Border(lipgloss.RoundedBorder()).BorderForeground(th.P.Warning).
		Padding(1, 2).Width(min(66, a.width)).Render(body)
	return lipgloss.Place(a.width, a.height, lipgloss.Center, lipgloss.Center, box)
}
