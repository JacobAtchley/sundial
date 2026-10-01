package fakesource

import (
	"context"
	"errors"
	"testing"
	"time"

	"github.com/JacobAtchley/sundial/internal/calendar"
)

var anchor = time.Date(2026, 10, 1, 8, 0, 0, 0, time.UTC)

func TestEventsFiltersByRange(t *testing.T) {
	s := New()
	in := calendar.Event{ID: "in", Start: anchor, End: anchor.Add(time.Hour)}
	out := calendar.Event{ID: "out", Start: anchor.AddDate(0, 0, 5), End: anchor.AddDate(0, 0, 5).Add(time.Hour)}
	s.AddEvents(out, in)
	got, err := s.Events(context.Background(), calendar.DayRange(anchor))
	if err != nil {
		t.Fatal(err)
	}
	if len(got) != 1 || got[0].ID != "in" {
		t.Errorf("got %v", got)
	}
	if s.EventCalls() != 1 {
		t.Errorf("calls = %d", s.EventCalls())
	}
}

func TestSetError(t *testing.T) {
	s := New()
	boom := errors.New("boom")
	s.SetError(boom)
	if _, err := s.Events(context.Background(), calendar.DayRange(anchor)); !errors.Is(err, boom) {
		t.Errorf("err = %v", err)
	}
	if _, err := s.Calendars(context.Background()); !errors.Is(err, boom) {
		t.Errorf("err = %v", err)
	}
}

func TestWatchTriggerAndClose(t *testing.T) {
	s := New()
	ctx, cancel := context.WithCancel(context.Background())
	ch, err := s.Watch(ctx)
	if err != nil {
		t.Fatal(err)
	}
	s.Trigger()
	select {
	case <-ch:
	case <-time.After(time.Second):
		t.Fatal("no notification")
	}
	cancel()
	select {
	case _, ok := <-ch:
		if ok {
			t.Fatal("expected closed channel")
		}
	case <-time.After(time.Second):
		t.Fatal("channel not closed after cancel")
	}
}

func TestDemoIsDeterministicAndFictional(t *testing.T) {
	a, b := Demo(anchor), Demo(anchor)
	r := calendar.Range{Start: anchor.AddDate(0, 0, -7), End: anchor.AddDate(0, 0, 14)}
	ea, _ := a.Events(context.Background(), r)
	eb, _ := b.Events(context.Background(), r)
	if len(ea) == 0 || len(ea) != len(eb) {
		t.Fatalf("demo sizes %d vs %d", len(ea), len(eb))
	}
	for i := range ea {
		if ea[i].Key() != eb[i].Key() {
			t.Fatal("demo data is not deterministic")
		}
	}
	cals, _ := a.Calendars(context.Background())
	if len(cals) != 4 {
		t.Errorf("demo calendars = %d", len(cals))
	}
	day, _ := a.Events(context.Background(), calendar.DayRange(anchor))
	want := map[string]bool{"Team Standup": false, "Lunch with Sam": false, "1:1 with Manager": false, "Focus Time": false}
	for _, e := range day {
		if _, ok := want[e.Title]; ok {
			want[e.Title] = true
		}
	}
	for title, seen := range want {
		if !seen {
			t.Errorf("anchor day missing %q", title)
		}
	}
}
