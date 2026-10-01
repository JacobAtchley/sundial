package tui

import (
	"fmt"
	"strings"

	tea "charm.land/bubbletea/v2"

	"github.com/JacobAtchley/sundial/internal/calendar"
	"github.com/JacobAtchley/sundial/internal/tui/palette"
	"github.com/JacobAtchley/sundial/internal/tui/shared"
)

func send(m tea.Msg) func() tea.Msg { return func() tea.Msg { return m } }

// commands is the single registry behind both the palette and help.
func (a *App) commands() []palette.Command {
	k := a.keys.label
	cmds := []palette.Command{
		{ID: "view.month", Title: "Go to month view", Keys: k("month"), Run: send(shared.SwitchViewMsg{View: shared.ViewMonth})},
		{ID: "view.week", Title: "Go to week view", Keys: k("week"), Run: send(shared.SwitchViewMsg{View: shared.ViewWeek})},
		{ID: "view.agenda", Title: "Go to agenda view", Keys: k("agenda"), Run: send(shared.SwitchViewMsg{View: shared.ViewAgenda})},
		{ID: "nav.today", Title: "Go to today", Keys: k("today"), Run: func() tea.Msg { return shared.GotoDateMsg{Date: a.now} }},
		{ID: "app.reload", Title: "Reload calendars", Run: send(shared.ReloadMsg{})},
		{ID: "app.config", Title: "Open config in $EDITOR", Run: send(shared.EditConfigMsg{})},
		{ID: "app.help", Title: "Show keyboard shortcuts", Keys: k("help"), Run: send(shared.ShowHelpMsg{})},
		{ID: "app.quit", Title: "Quit", Keys: k("quit"), Run: send(shared.QuitMsg{})},
	}
	if a.opts.Store != nil {
		for _, c := range a.opts.Store.Calendars() {
			state := "shown"
			if a.opts.Store.IsHidden(c.ID) {
				state = "hidden"
			}
			cmds = append(cmds, palette.Command{
				ID:    "cal.toggle." + c.ID,
				Title: fmt.Sprintf("Toggle calendar %s (%s)", shared.OneLine(c.Title), state),
				Run:   send(shared.ToggleCalendarMsg{ID: c.ID}),
			})
		}
	}
	return cmds
}

const maxSearchResults = 20

// searchEvents matches cached event titles, upcoming events first.
func (a *App) searchEvents(q string) []palette.Command {
	q = strings.ToLower(q)
	var upcoming, past []palette.Command
	for _, e := range a.opts.Store.All() {
		if !strings.Contains(strings.ToLower(e.Title), q) {
			continue
		}
		cmd := eventCommand(e)
		if e.End.After(a.now) {
			upcoming = append(upcoming, cmd)
		} else {
			past = append(past, cmd)
		}
	}
	out := append(upcoming, past...)
	return out[:min(len(out), maxSearchResults)]
}

func eventCommand(e calendar.Event) palette.Command {
	return palette.Command{
		ID:    "event." + e.Key(),
		Title: "Event: " + shared.OneLine(e.Title) + " — " + e.Start.Format("Mon, Jan 2"),
		Run:   send(shared.ShowEventMsg{Event: e}),
	}
}
