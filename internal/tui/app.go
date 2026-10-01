// Package tui is sundial's interactive Bubble Tea app.
package tui

import (
	"context"
	"os"
	"time"

	tea "charm.land/bubbletea/v2"

	"github.com/JacobAtchley/sundial/internal/calendar"
	"github.com/JacobAtchley/sundial/internal/config"
	"github.com/JacobAtchley/sundial/internal/debuglog"
	"github.com/JacobAtchley/sundial/internal/eventkit"
	"github.com/JacobAtchley/sundial/internal/store"
	"github.com/JacobAtchley/sundial/internal/tui/agenda"
	"github.com/JacobAtchley/sundial/internal/tui/detail"
	"github.com/JacobAtchley/sundial/internal/tui/month"
	"github.com/JacobAtchley/sundial/internal/tui/palette"
	"github.com/JacobAtchley/sundial/internal/tui/shared"
	"github.com/JacobAtchley/sundial/internal/tui/theme"
	"github.com/JacobAtchley/sundial/internal/tui/week"
)

type Options struct {
	Config     config.Config
	ConfigPath string
	Store      *store.Store    // nil when Denied
	Source     calendar.Source // nil when Denied
	Now        func() time.Time
	Log        debuglog.Logger
	Denied     bool
	Getenv     func(string) string
}

type App struct {
	opts  Options
	keys  keyMap
	theme theme.Theme

	width, height int
	now           time.Time

	view   shared.ViewID
	month  month.Model
	week   week.Model
	agenda agenda.Model

	detail    *calendar.Event
	palette   palette.Model
	paletteOn bool
	helpOn    bool

	loading bool
	err     error
	live    bool
	notice  string
	watch   <-chan struct{}

	ctx    context.Context
	cancel context.CancelFunc
}

// Internal messages.
type (
	eventsLoadedMsg struct {
		r   calendar.Range
		err error
	}
	watchStartedMsg struct {
		ch  <-chan struct{}
		err error
	}
	storeChangedMsg struct{}
	tickMsg         time.Time
	editorDoneMsg   struct{ err error }
)

func New(opts Options) *App {
	if opts.Getenv == nil {
		opts.Getenv = os.Getenv
	}
	now := opts.Now()
	ctx, cancel := context.WithCancel(context.Background())
	return &App{
		opts:    opts,
		keys:    newKeyMap(opts.Config),
		theme:   theme.New(opts.Config.Theme, true),
		now:     now,
		view:    shared.ParseView(opts.Config.DefaultView),
		month:   month.New(now),
		week:    week.New(now, opts.Config.DayStart),
		agenda:  agenda.New(now, opts.Config.Agenda.Days),
		palette: palette.New(),
		ctx:     ctx,
		cancel:  cancel,
	}
}

// Run starts the interactive program and blocks until it exits.
func Run(opts Options) error {
	a := New(opts)
	defer a.cancel()
	_, err := tea.NewProgram(a).Run()
	return err
}

func (a *App) Init() tea.Cmd {
	if a.opts.Denied {
		return tea.RequestBackgroundColor
	}
	return tea.Batch(tea.RequestBackgroundColor, a.load(), a.startWatch(), tick())
}

func tick() tea.Cmd {
	return tea.Tick(time.Minute, func(t time.Time) tea.Msg { return tickMsg(t) })
}

func (a *App) viewCtx() shared.Context {
	return shared.Context{
		Store:     a.opts.Store,
		Theme:     a.theme,
		Now:       a.now,
		WeekStart: a.opts.Config.WeekStartDay(),
		Use24h:    a.opts.Config.Use24h(),
		DayStart:  a.opts.Config.DayStart,
		Width:     a.width,
		Height:    max(a.height-1, 0), // status bar
	}
}

// visibleRange is what the current view shows plus a week of buffer.
func (a *App) visibleRange() calendar.Range {
	ctx := a.viewCtx()
	var r calendar.Range
	switch a.view {
	case shared.ViewMonth:
		r = a.month.Range(ctx.WeekStart)
	case shared.ViewWeek:
		r = a.week.Range(ctx)
	default:
		r = a.agenda.Range()
	}
	return calendar.Range{Start: r.Start.AddDate(0, 0, -7), End: r.End.AddDate(0, 0, 7)}
}

func (a *App) load() tea.Cmd {
	if a.opts.Store == nil {
		return nil
	}
	r := a.visibleRange()
	if a.opts.Store.Covers(r) {
		return nil
	}
	a.loading = true
	st, ctx := a.opts.Store, a.ctx
	return func() tea.Msg { return eventsLoadedMsg{r: r, err: st.Load(ctx, r)} }
}

func (a *App) startWatch() tea.Cmd {
	src, ctx := a.opts.Source, a.ctx
	return func() tea.Msg {
		ch, err := src.Watch(ctx)
		return watchStartedMsg{ch: ch, err: err}
	}
}

func waitChange(ch <-chan struct{}) tea.Cmd {
	if ch == nil {
		return nil
	}
	return func() tea.Msg {
		if _, ok := <-ch; !ok {
			return nil
		}
		return storeChangedMsg{}
	}
}

func (a *App) Update(msg tea.Msg) (tea.Model, tea.Cmd) {
	switch msg := msg.(type) {
	case tea.WindowSizeMsg:
		a.width, a.height = msg.Width, msg.Height
		return a, a.load()
	case tea.BackgroundColorMsg:
		a.theme = theme.New(a.opts.Config.Theme, msg.IsDark())
		return a, nil
	case tickMsg:
		a.now = a.opts.Now()
		return a, tea.Batch(tick(), a.load())
	case eventsLoadedMsg:
		a.loading, a.err = false, msg.err
		if msg.err != nil {
			a.opts.Log.Warn("load failed", "err", msg.err)
		} else {
			a.opts.Log.Debug("events loaded", "from", msg.r.Start, "to", msg.r.End)
		}
		return a, nil
	case watchStartedMsg:
		if msg.err != nil {
			a.opts.Log.Warn("watch unavailable", "err", msg.err)
			return a, nil
		}
		a.live, a.watch = true, msg.ch
		return a, waitChange(msg.ch)
	case storeChangedMsg:
		a.opts.Log.Debug("calendar store changed")
		a.opts.Store.Invalidate()
		return a, tea.Batch(a.load(), waitChange(a.watch))
	case editorDoneMsg:
		if msg.err != nil {
			a.err = msg.err
		} else {
			a.notice = "config saved · restart sundial to apply"
		}
		return a, nil
	case detail.OpenFailedMsg:
		a.err = msg.Err
		return a, nil
	case tea.KeyPressMsg:
		return a.handleKey(msg)
	}
	if a.paletteOn {
		var cmd tea.Cmd
		a.palette, cmd, _ = a.palette.Update(msg)
		return a, cmd
	}
	if a.agenda.Filtering() {
		var cmd tea.Cmd
		a.agenda, cmd = a.agenda.UpdateFilter(msg)
		return a, cmd
	}
	return a.handleAction(msg)
}

func (a *App) handleAction(msg tea.Msg) (tea.Model, tea.Cmd) {
	switch msg := msg.(type) {
	case shared.SwitchViewMsg:
		a.switchView(msg.View)
		return a, a.load()
	case shared.GotoDateMsg:
		a.gotoDate(msg.Date)
		return a, a.load()
	case shared.ToggleCalendarMsg:
		a.opts.Store.ToggleHidden(msg.ID)
	case shared.ReloadMsg:
		a.err, a.notice = nil, ""
		a.opts.Store.Invalidate()
		return a, a.load()
	case shared.EditConfigMsg:
		return a, a.editConfig()
	case shared.ShowHelpMsg:
		a.helpOn = true
	case shared.QuitMsg:
		return a.quit()
	case shared.ShowEventMsg:
		e := msg.Event
		a.view = shared.ViewAgenda
		a.agenda = a.agenda.SetDate(e.Start)
		a.detail = &e
		return a, a.load()
	}
	return a, nil
}

func (a *App) quit() (tea.Model, tea.Cmd) {
	a.cancel()
	return a, tea.Quit
}

func (a *App) handleKey(k tea.KeyPressMsg) (tea.Model, tea.Cmd) {
	if a.opts.Denied {
		switch {
		case a.keys.is(k, "quit"):
			return a.quit()
		case a.keys.is(k, "open_url"):
			return a, detail.OpenURLCmd(eventkit.SettingsURL)
		}
		return a, nil
	}
	if a.paletteOn {
		var cmd tea.Cmd
		var res palette.Result
		a.palette, cmd, res = a.palette.Update(k)
		if !res.Done {
			return a, cmd
		}
		a.paletteOn = false
		if res.Msg == nil {
			return a, cmd
		}
		_, next := a.Update(res.Msg)
		return a, tea.Batch(cmd, next)
	}
	if a.agenda.Filtering() {
		var cmd tea.Cmd
		a.agenda, cmd = a.agenda.UpdateFilter(k)
		return a, cmd
	}
	if a.helpOn {
		if a.keys.is(k, "close") || a.keys.is(k, "help") || a.keys.is(k, "quit") {
			a.helpOn = false
		}
		return a, nil
	}
	if a.detail != nil {
		switch {
		case a.keys.is(k, "close"):
			a.detail = nil
		case a.keys.is(k, "open_url"):
			if u := detail.MeetingURL(*a.detail); u != "" {
				return a, detail.OpenURLCmd(u)
			}
		case a.keys.is(k, "quit"):
			return a.quit()
		}
		return a, nil
	}

	ctx := a.viewCtx()
	switch {
	case a.keys.is(k, "quit"):
		return a.quit()
	case a.keys.is(k, "palette"):
		a.paletteOn = true
		var cmd tea.Cmd
		a.palette, cmd = a.palette.Open(a.commands(), a.searchEvents, a.now)
		return a, cmd
	case a.keys.is(k, "help"):
		a.helpOn = true
	case a.keys.is(k, "month"):
		a.switchView(shared.ViewMonth)
		return a, a.load()
	case a.keys.is(k, "week"):
		a.switchView(shared.ViewWeek)
		return a, a.load()
	case a.keys.is(k, "agenda"):
		a.switchView(shared.ViewAgenda)
		return a, a.load()
	case a.keys.is(k, "today"):
		a.gotoDate(a.now)
		return a, a.load()
	case a.keys.is(k, "filter") && a.view == shared.ViewAgenda:
		var cmd tea.Cmd
		a.agenda, cmd = a.agenda.StartFilter()
		return a, cmd
	case a.keys.is(k, "open"):
		return a, a.open(ctx)
	default:
		for _, n := range navActions {
			if a.keys.is(k, n.action) {
				a.nav(n.nav, ctx)
				return a, a.load()
			}
		}
	}
	return a, nil
}

func (a *App) nav(n shared.Nav, ctx shared.Context) {
	switch a.view {
	case shared.ViewMonth:
		a.month = a.month.Nav(n)
	case shared.ViewWeek:
		a.week = a.week.Nav(n, ctx)
	default:
		a.agenda = a.agenda.Nav(n, ctx)
	}
}

func (a *App) focusDate() time.Time {
	switch a.view {
	case shared.ViewMonth:
		return a.month.Selected()
	case shared.ViewWeek:
		return a.week.Selected()
	default:
		return a.agenda.SelectedDay(a.viewCtx())
	}
}

func (a *App) switchView(v shared.ViewID) {
	d := a.focusDate()
	a.view = v
	a.gotoDate(d)
}

func (a *App) gotoDate(d time.Time) {
	switch a.view {
	case shared.ViewMonth:
		a.month = a.month.SetDate(d)
	case shared.ViewWeek:
		a.week = a.week.SetDate(d)
	default:
		a.agenda = a.agenda.SetDate(d)
	}
}

func (a *App) open(ctx shared.Context) tea.Cmd {
	switch a.view {
	case shared.ViewMonth:
		d := a.month.Selected()
		a.view = shared.ViewAgenda
		a.agenda = a.agenda.SetDate(d)
		return a.load()
	case shared.ViewWeek:
		if e, ok := a.week.SelectedEvent(ctx); ok {
			a.detail = &e
		}
	default:
		if e, ok := a.agenda.Selected(ctx); ok {
			a.detail = &e
		}
	}
	return nil
}

func (a *App) editConfig() tea.Cmd {
	path := a.opts.ConfigPath
	if _, err := os.Stat(path); os.IsNotExist(err) {
		if err := config.Init(path); err != nil {
			return func() tea.Msg { return editorDoneMsg{err: err} }
		}
	}
	c := config.EditorCommand(a.opts.Getenv, path)
	return tea.ExecProcess(c, func(err error) tea.Msg { return editorDoneMsg{err: err} })
}

func (a *App) View() tea.View {
	v := tea.NewView(a.render())
	v.AltScreen = true
	return v
}
