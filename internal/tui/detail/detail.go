// Package detail renders the event detail pane.
package detail

import (
	"net/url"
	"os/exec"
	"regexp"
	"slices"
	"strings"

	tea "charm.land/bubbletea/v2"
	"charm.land/lipgloss/v2"

	"github.com/JacobAtchley/sundial/internal/calendar"
	"github.com/JacobAtchley/sundial/internal/tui/shared"
)

const (
	maxWidth  = 64
	labelW    = 10
	maxNotesH = 10
)

func View(e calendar.Event, cal calendar.Calendar, ctx shared.Context) string {
	th := ctx.Theme
	w := min(maxWidth, ctx.Width-4)
	if w < 20 {
		w = max(ctx.Width, 6)
	}
	inner := w - 4 // rounded border (2) + horizontal padding (2)
	wrap := func(s string, width int) string { return lipgloss.NewStyle().Width(max(width, 1)).Render(s) }

	var rows []string
	rows = append(rows, th.Title.Render(wrap(shared.OneLine(e.Title), inner)), "")
	field := func(name, value string) {
		label := th.DayHeader.Render(shared.PadRight(name, labelW))
		rows = append(rows, lipgloss.JoinHorizontal(lipgloss.Top, label, wrap(value, inner-labelW)))
	}
	field("When", When(e, ctx.Use24h))
	field("Calendar", th.Dot(cal.Color)+" "+shared.OneLine(cal.Title))
	if loc := shared.OneLine(e.Location); loc != "" {
		field("Where", loc)
	}
	link := MeetingURL(e)
	if link != "" {
		field("Link", shared.OneLine(link))
	}
	if e.Status == calendar.StatusTentative || e.Status == calendar.StatusCanceled {
		field("Status", th.Warning.Render(e.Status.String()))
	}
	if len(e.Attendees) > 0 {
		cleanAttendees := make([]string, len(e.Attendees))
		for i, a := range e.Attendees {
			cleanAttendees[i] = shared.OneLine(a)
		}
		field("People", strings.Join(cleanAttendees, ", "))
	}
	if notes := strings.TrimSpace(shared.CleanText(e.Notes)); notes != "" {
		lines := strings.Split(wrap(notes, inner), "\n")
		if len(lines) > maxNotesH {
			lines = append(lines[:maxNotesH-1], th.Muted.Render("…"))
		}
		rows = append(rows, "")
		rows = append(rows, lines...)
	}
	hint := "esc close"
	if link != "" {
		hint = "o open link · " + hint
	}
	rows = append(rows, "", th.Muted.Render(hint))

	return lipgloss.NewStyle().
		Border(lipgloss.RoundedBorder()).
		BorderForeground(th.P.Header).
		Padding(0, 1).
		Width(w).
		Render(strings.Join(rows, "\n"))
}

// When formats the event's date and time for the detail pane.
func When(e calendar.Event, use24h bool) string {
	const dayFmt = "Mon, Jan 2"
	if e.AllDay {
		last := e.End.AddDate(0, 0, -1)
		if !last.After(e.Start) {
			return e.Start.Format(dayFmt) + " · all day"
		}
		return e.Start.Format(dayFmt) + " – " + last.Format(dayFmt) + " · all day"
	}
	// Same day, or ending exactly at the following midnight.
	sameDay := calendar.StartOfDay(e.Start).Equal(calendar.StartOfDay(e.End)) ||
		e.End.Equal(calendar.StartOfDay(e.Start).AddDate(0, 0, 1))
	if sameDay {
		return e.Start.Format(dayFmt) + " · " + shared.FormatSpan(e, use24h)
	}
	return e.Start.Format(dayFmt) + " " + shared.FormatClock(e.Start, use24h) + " – " + e.End.Format(dayFmt) + " " + shared.FormatClock(e.End, use24h)
}

var (
	urlRe          = regexp.MustCompile(`[a-zA-Z][a-zA-Z0-9+.-]*://[^\s<>"]+`)
	allowedSchemes = []string{"http", "https", "zoommtg", "msteams", "webex"}
)

func safeURL(raw string) bool {
	u, err := url.Parse(raw)
	return err == nil && u.Host != "" && slices.Contains(allowedSchemes, strings.ToLower(u.Scheme))
}

// MeetingURL returns the event URL, or the first safe link found in the
// location or notes. Only allow-listed schemes are returned, so `open`
// can never be pointed at a file or script.
func MeetingURL(e calendar.Event) string {
	if safeURL(e.URL) {
		return e.URL
	}
	for _, text := range []string{e.Location, e.Notes} {
		for _, m := range urlRe.FindAllString(text, -1) {
			m = strings.TrimRight(m, ".,;:!?)]}'\"")
			if safeURL(m) {
				return m
			}
		}
	}
	return ""
}

type OpenFailedMsg struct{ Err error }

// OpenURLCmd opens u with macOS `open`. Callers pass MeetingURL output or
// sundial's own constants only.
func OpenURLCmd(u string) tea.Cmd {
	return func() tea.Msg {
		if err := exec.Command("open", u).Run(); err != nil {
			return OpenFailedMsg{Err: err}
		}
		return nil
	}
}
