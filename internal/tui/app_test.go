package tui

import (
	"bytes"
	"context"
	"errors"
	"strings"
	"testing"
	"time"

	tea "charm.land/bubbletea/v2"
	"charm.land/lipgloss/v2"
	"github.com/charmbracelet/x/ansi"
	"github.com/charmbracelet/x/exp/teatest/v2"

	"github.com/JacobAtchley/sundial/internal/calendar"
	"github.com/JacobAtchley/sundial/internal/config"
	"github.com/JacobAtchley/sundial/internal/debuglog"
	"github.com/JacobAtchley/sundial/internal/fakesource"
	"github.com/JacobAtchley/sundial/internal/store"
	"github.com/JacobAtchley/sundial/internal/tui/shared"
)

func newTestApp(t *testing.T, w, h int) (*App, *fakesource.Source) {
	t.Helper()
	src := fakesource.Demo(shared.TestNow)
	st := store.New(src)
	ctx := context.Background()
	_ = st.LoadCalendars(ctx)
	_ = st.Load(ctx, calendar.Range{Start: shared.TestNow.AddDate(0, 0, -60), End: shared.TestNow.AddDate(0, 0, 60)})
	a := New(Options{
		Config: config.Default(), ConfigPath: "/dev/null", Store: st, Source: src,
		Now: func() time.Time { return shared.TestNow }, Log: debuglog.Discard(),
		Getenv: func(string) string { return "" },
	})
	a.Update(tea.WindowSizeMsg{Width: w, Height: h})
	return a, src
}

func key(s string) tea.KeyPressMsg {
	switch s {
	case "enter":
		return tea.KeyPressMsg{Code: tea.KeyEnter}
	case "esc":
		return tea.KeyPressMsg{Code: tea.KeyEscape}
	case "ctrl+k":
		return tea.KeyPressMsg{Code: 'k', Mod: tea.ModCtrl}
	}
	r := []rune(s)[0]
	return tea.KeyPressMsg{Code: r, Text: s}
}

func press(a *App, keys ...string) tea.Cmd {
	var last tea.Cmd
	for _, k := range keys {
		_, last = a.Update(key(k))
	}
	return last
}

func typeText(a *App, s string) {
	for _, r := range s {
		a.Update(tea.KeyPressMsg{Code: r, Text: string(r)})
	}
}

func screen(a *App) string { return ansi.Strip(a.render()) }

func TestTooSmall(t *testing.T) {
	for _, sz := range [][2]int{{39, 30}, {80, 9}, {0, 0}} {
		a, _ := newTestApp(t, sz[0], sz[1])
		if !strings.Contains(screen(a), "too small") {
			t.Errorf("%v: got %q", sz, screen(a))
		}
	}
}

func TestDefaultAgendaAndStatusBar(t *testing.T) {
	a, _ := newTestApp(t, 100, 30)
	out := screen(a)
	for _, want := range []string{"AGENDA", "Oct 1 – Oct 14", "Team Standup", "? help"} {
		if !strings.Contains(out, want) {
			t.Errorf("missing %q:\n%s", want, out)
		}
	}
}

func TestSwitchViewsKeepsFocusDate(t *testing.T) {
	a, _ := newTestApp(t, 100, 30)
	press(a, "l") // agenda: next day with events (Oct 2)
	press(a, "m")
	if a.view != shared.ViewMonth || a.month.Selected().Day() != 2 {
		t.Fatalf("month selected %v", a.month.Selected())
	}
	press(a, "w")
	if !strings.Contains(screen(a), "Sep 27 – Oct 3, 2026") {
		t.Errorf("week title missing:\n%s", screen(a))
	}
}

func TestPaletteSwitchesView(t *testing.T) {
	a, _ := newTestApp(t, 100, 30)
	press(a, "ctrl+k")
	if !a.paletteOn {
		t.Fatal("palette did not open")
	}
	typeText(a, "week view")
	press(a, "enter")
	if a.paletteOn || a.view != shared.ViewWeek {
		t.Errorf("paletteOn=%v view=%v", a.paletteOn, a.view)
	}
}

func TestPaletteGotoDate(t *testing.T) {
	a, _ := newTestApp(t, 100, 30)
	press(a, "ctrl+k")
	typeText(a, "2026-12-25")
	press(a, "enter")
	if s := a.agenda.Range().Start; s.Month() != time.December || s.Day() != 25 {
		t.Errorf("agenda start = %v", s)
	}
}

func TestPaletteTogglesCalendar(t *testing.T) {
	a, _ := newTestApp(t, 100, 30)
	press(a, "ctrl+k")
	typeText(a, "toggle calendar work")
	press(a, "enter")
	if !a.opts.Store.IsHidden("work") {
		t.Error("work calendar should be hidden")
	}
	if strings.Contains(screen(a), "Team Standup") {
		t.Error("hidden calendar still rendered")
	}
}

func TestDetailOpenClose(t *testing.T) {
	a, _ := newTestApp(t, 100, 30)
	press(a, "enter")
	if a.detail == nil || !strings.Contains(screen(a), "Calendar") {
		t.Fatalf("detail not shown:\n%s", screen(a))
	}
	press(a, "esc")
	if a.detail != nil {
		t.Error("esc should close detail")
	}
}

func TestMonthEnterJumpsToAgenda(t *testing.T) {
	a, _ := newTestApp(t, 100, 30)
	press(a, "m", "l", "enter")
	if a.view != shared.ViewAgenda || a.agenda.Range().Start.Day() != 2 {
		t.Errorf("view=%v start=%v", a.view, a.agenda.Range().Start)
	}
}

func TestHelpOverlay(t *testing.T) {
	a, _ := newTestApp(t, 100, 30)
	press(a, "?")
	if !strings.Contains(screen(a), "Go to month view") {
		t.Errorf("help missing commands:\n%s", screen(a))
	}
	press(a, "esc")
	if a.helpOn {
		t.Error("esc should close help")
	}
}

func TestFetchErrorKeepsDataAndShowsStatus(t *testing.T) {
	a, src := newTestApp(t, 100, 30)
	src.SetError(errors.New("calendar daemon unavailable"))
	_, cmd := a.Update(shared.ReloadMsg{})
	if cmd == nil {
		t.Fatal("reload should trigger a load")
	}
	a.Update(cmd())
	out := screen(a)
	if !strings.Contains(out, "✗") || !strings.Contains(out, "Team Standup") {
		t.Errorf("want error status and last good data:\n%s", out)
	}
}

func TestStoreChangeTriggersReload(t *testing.T) {
	a, _ := newTestApp(t, 100, 30)
	a.watch = make(chan struct{})
	_, cmd := a.Update(storeChangedMsg{})
	if cmd == nil || a.opts.Store.Covers(a.visibleRange()) {
		t.Error("store change should invalidate and reload")
	}
}

func TestDeniedScreen(t *testing.T) {
	a := New(Options{Config: config.Default(), Now: func() time.Time { return shared.TestNow }, Log: debuglog.Discard(), Denied: true})
	a.Update(tea.WindowSizeMsg{Width: 100, Height: 30})
	if !strings.Contains(screen(a), "needs calendar access") {
		t.Errorf("got:\n%s", screen(a))
	}
	cmd := press(a, "q")
	if cmd == nil {
		t.Fatal("q should quit")
	}
	if _, ok := cmd().(tea.QuitMsg); !ok {
		t.Error("q should return tea.Quit")
	}
}

func TestProgramSmoke(t *testing.T) {
	a, _ := newTestApp(t, 100, 30)
	tm := teatest.NewTestModel(t, a, teatest.WithInitialTermSize(100, 30))
	contains := func(s string) func([]byte) bool {
		return func(b []byte) bool { return bytes.Contains([]byte(ansi.Strip(string(b))), []byte(s)) }
	}
	teatest.WaitFor(t, tm.Output(), contains("Team Standup"), teatest.WithDuration(3*time.Second))
	tm.Send(key("m"))
	teatest.WaitFor(t, tm.Output(), contains("October 2026"), teatest.WithDuration(3*time.Second))
	tm.Send(key("q"))
	tm.WaitFinished(t, teatest.WithFinalTimeout(3*time.Second))
}

// runCmd executes cmd and, for a batch, each child, in goroutines with a 1s
// timeout so blocking commands (watch, ticks) cannot hang the test.
func runCmd(cmd tea.Cmd) []tea.Msg {
	if cmd == nil {
		return nil
	}
	exec := func(c tea.Cmd) tea.Msg {
		ch := make(chan tea.Msg, 1)
		go func() { ch <- c() }()
		select {
		case m := <-ch:
			return m
		case <-time.After(time.Second):
			return nil
		}
	}
	var out []tea.Msg
	msg := exec(cmd)
	if b, ok := msg.(tea.BatchMsg); ok {
		for _, c := range b {
			if c != nil {
				out = append(out, exec(c))
			}
		}
		return out
	}
	return append(out, msg)
}

func feed(a *App, cmd tea.Cmd) {
	for _, m := range runCmd(cmd) {
		if _, ok := m.(eventsLoadedMsg); ok {
			a.Update(m)
		}
	}
}

func TestReloadRefreshesCalendars(t *testing.T) {
	for _, msg := range []tea.Msg{shared.ReloadMsg{}, storeChangedMsg{}} {
		a, src := newTestApp(t, 100, 30)
		a.watch = make(chan struct{})
		src.SetCalendars(append(a.opts.Store.Calendars(), calendar.Calendar{ID: "new", Title: "New Cal"})...)
		_, cmd := a.Update(msg)
		feed(a, cmd)
		if _, ok := a.opts.Store.Calendar("new"); !ok {
			t.Errorf("%T: new calendar not loaded", msg)
		}
	}
}

func TestReloadCalendarsErrorReported(t *testing.T) {
	a, src := newTestApp(t, 100, 30)
	src.SetError(errors.New("boom"))
	_, cmd := a.Update(shared.ReloadMsg{})
	feed(a, cmd)
	if a.err == nil {
		t.Error("error from LoadCalendars/Load should be reported")
	}
}

func TestPaletteSearchLoadsWindow(t *testing.T) {
	src := fakesource.Demo(shared.TestNow)
	far := shared.TestNow.AddDate(0, 0, 90)
	src.AddEvents(calendar.Event{ID: "far", CalendarID: "work", Title: "Far Future Checkup", Start: far, End: far.Add(time.Hour)})
	st := store.New(src)
	ctx := context.Background()
	_ = st.LoadCalendars(ctx)
	_ = st.Load(ctx, calendar.Range{Start: shared.TestNow.AddDate(0, 0, -7), End: shared.TestNow.AddDate(0, 0, 7)})
	a := New(Options{
		Config: config.Default(), ConfigPath: "/dev/null", Store: st, Source: src,
		Now: func() time.Time { return shared.TestNow }, Log: debuglog.Discard(),
		Getenv: func(string) string { return "" },
	})
	a.Update(tea.WindowSizeMsg{Width: 100, Height: 30})
	if len(a.searchEvents("far future")) != 0 {
		t.Fatal("event should not be cached yet")
	}
	cmd := press(a, "ctrl+k")
	feed(a, cmd)
	if got := a.searchEvents("far future"); len(got) != 1 {
		t.Errorf("search results = %d, want 1", len(got))
	}
}

func TestCtrlCQuitsWhileFiltering(t *testing.T) {
	a, _ := newTestApp(t, 100, 30)
	press(a, "/")
	if !a.agenda.Filtering() {
		t.Fatal("filter not focused")
	}
	_, cmd := a.Update(tea.KeyPressMsg{Code: 'c', Mod: tea.ModCtrl})
	if cmd == nil {
		t.Fatal("ctrl+c should quit")
	}
	if _, ok := cmd().(tea.QuitMsg); !ok {
		t.Error("ctrl+c should return tea.Quit")
	}
}

func TestTooSmallTruncatesToWidth(t *testing.T) {
	a, _ := newTestApp(t, 20, 30)
	if w := lipgloss.Width(screen(a)); w > 20 {
		t.Errorf("message width %d exceeds terminal width 20", w)
	}
	a, _ = newTestApp(t, 0, 0)
	if !strings.Contains(screen(a), "at least 40×10") {
		t.Errorf("width 0 should keep full message: %q", screen(a))
	}
}
