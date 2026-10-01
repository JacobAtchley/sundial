package store

import (
	"context"
	"errors"
	"testing"
	"time"

	"github.com/JacobAtchley/sundial/internal/calendar"
	"github.com/JacobAtchley/sundial/internal/fakesource"
)

var day0 = time.Date(2026, 10, 1, 0, 0, 0, 0, time.UTC)

func week() calendar.Range { return calendar.Range{Start: day0, End: day0.AddDate(0, 0, 7)} }

func newStore(t *testing.T, evs ...calendar.Event) (*Store, *fakesource.Source) {
	t.Helper()
	src := fakesource.New()
	src.SetCalendars(
		calendar.Calendar{ID: "work", Title: "Work"},
		calendar.Calendar{ID: "home", Title: "Home"},
	)
	src.AddEvents(evs...)
	s := New(src)
	if err := s.LoadCalendars(context.Background()); err != nil {
		t.Fatal(err)
	}
	return s, src
}

func timed(id, cal string, dayOffset, h int) calendar.Event {
	st := day0.AddDate(0, 0, dayOffset).Add(time.Duration(h) * time.Hour)
	return calendar.Event{ID: id, CalendarID: cal, Title: id, Start: st, End: st.Add(time.Hour)}
}

func TestLoadAndEventsOn(t *testing.T) {
	s, _ := newStore(t, timed("a", "work", 0, 9), timed("b", "home", 1, 9))
	if s.Covers(week()) {
		t.Fatal("empty store must not cover")
	}
	if err := s.Load(context.Background(), week()); err != nil {
		t.Fatal(err)
	}
	if !s.Covers(week()) {
		t.Fatal("store should cover loaded range")
	}
	if got := s.EventsOn(day0); len(got) != 1 || got[0].ID != "a" {
		t.Errorf("day0 = %v", got)
	}
	if got := s.EventsOn(day0.AddDate(0, 0, 3)); len(got) != 0 {
		t.Errorf("empty day = %v", got)
	}
}

func TestMultiDayEventInEveryBucketDedupedInRange(t *testing.T) {
	multi := calendar.Event{ID: "trip", CalendarID: "home", AllDay: true, Start: day0, End: day0.AddDate(0, 0, 3)}
	s, _ := newStore(t, multi)
	_ = s.Load(context.Background(), week())
	for i := range 3 {
		if got := s.EventsOn(day0.AddDate(0, 0, i)); len(got) != 1 {
			t.Errorf("day %d has %d events", i, len(got))
		}
	}
	if got := s.EventsOn(day0.AddDate(0, 0, 3)); len(got) != 0 {
		t.Errorf("exclusive end day has %d events", len(got))
	}
	if got := s.EventsIn(week()); len(got) != 1 {
		t.Errorf("EventsIn = %d, want 1 (deduped)", len(got))
	}
}

func TestRecurringOccurrencesStayDistinct(t *testing.T) {
	a, b := timed("standup", "work", 0, 9), timed("standup", "work", 1, 9)
	s, _ := newStore(t, a, b)
	_ = s.Load(context.Background(), week())
	if got := s.EventsIn(week()); len(got) != 2 {
		t.Errorf("got %d occurrences, want 2", len(got))
	}
}

func TestHiddenByTitleOrID(t *testing.T) {
	s, _ := newStore(t, timed("a", "work", 0, 9), timed("b", "home", 0, 10))
	_ = s.Load(context.Background(), week())
	s.SetHidden([]string{"Work"})
	if got := s.EventsOn(day0); len(got) != 1 || got[0].ID != "b" {
		t.Errorf("hidden by title: %v", got)
	}
	s.SetHidden([]string{"home"})
	if got := s.EventsOn(day0); len(got) != 1 || got[0].ID != "a" {
		t.Errorf("hidden by id: %v", got)
	}
	s.ToggleHidden("work")
	if !s.IsHidden("work") || len(s.EventsOn(day0)) != 0 {
		t.Error("toggle should hide work too")
	}
	s.ToggleHidden("work")
	if s.IsHidden("work") {
		t.Error("second toggle should show work")
	}
}

func TestHiddenNamesResolveAfterCalendarsLoad(t *testing.T) {
	src := fakesource.New()
	src.SetCalendars(calendar.Calendar{ID: "work", Title: "Work"})
	s := New(src)
	s.SetHidden([]string{"Work"})
	_ = s.LoadCalendars(context.Background())
	if !s.IsHidden("work") {
		t.Error("names set before calendars load must resolve on load")
	}
}

func TestInvalidateKeepsDataAndFailedLoadKeepsData(t *testing.T) {
	s, src := newStore(t, timed("a", "work", 0, 9))
	_ = s.Load(context.Background(), week())
	s.Invalidate()
	if s.Covers(week()) {
		t.Error("invalidated store must not cover")
	}
	if len(s.EventsOn(day0)) != 1 {
		t.Error("invalidate must keep last good data")
	}
	src.SetError(errors.New("offline"))
	if err := s.Load(context.Background(), week()); err == nil {
		t.Fatal("expected error")
	}
	if len(s.EventsOn(day0)) != 1 {
		t.Error("failed load must keep last good data")
	}
}

func TestLoadRoundsToWholeDays(t *testing.T) {
	s, src := newStore(t, timed("late", "work", 0, 22))
	partial := calendar.Range{Start: day0.Add(9 * time.Hour), End: day0.Add(10 * time.Hour)}
	_ = s.Load(context.Background(), partial)
	if len(s.EventsOn(day0)) != 1 {
		t.Error("load must fetch whole days")
	}
	if !s.Covers(calendar.DayRange(day0)) {
		t.Error("whole day should be covered")
	}
	if src.EventCalls() != 1 {
		t.Errorf("calls = %d", src.EventCalls())
	}
}

func TestAllReturnsDedupedVisible(t *testing.T) {
	s, _ := newStore(t, timed("a", "work", 0, 9), timed("b", "home", 2, 9))
	_ = s.Load(context.Background(), week())
	s.SetHidden([]string{"home"})
	if got := s.All(); len(got) != 1 || got[0].ID != "a" {
		t.Errorf("All = %v", got)
	}
}

func TestTogglePersistsAcrossReloads(t *testing.T) {
	src := fakesource.New()
	src.SetCalendars(calendar.Calendar{ID: "work", Title: "Work"})
	s := New(src)
	s.SetHidden([]string{"Work"})
	_ = s.LoadCalendars(context.Background())
	if !s.IsHidden("work") {
		t.Error("initially hidden by config")
	}
	s.ToggleHidden("work")
	if s.IsHidden("work") {
		t.Error("after toggle, should be shown")
	}
	_ = s.LoadCalendars(context.Background())
	if s.IsHidden("work") {
		t.Error("after reload, toggle should persist, still shown")
	}
}
