package week

import (
	"testing"
	"time"

	"github.com/JacobAtchley/sundial/internal/calendar"
)

func at(day time.Time, h, m int) time.Time {
	return time.Date(day.Year(), day.Month(), day.Day(), h, m, 0, 0, day.Location())
}

func TestRowSpan(t *testing.T) {
	day := time.Date(2026, 10, 1, 0, 0, 0, 0, time.UTC)
	cases := []struct {
		name       string
		start, end time.Time
		s, e       int
	}{
		{"half hour aligned", at(day, 9, 30), at(day, 10, 0), 19, 20},
		{"rounds out", at(day, 9, 40), at(day, 9, 50), 19, 20},
		{"from previous day", at(day.AddDate(0, 0, -1), 22, 0), at(day, 0, 30), 0, 1},
		{"into next day", at(day, 23, 0), at(day.AddDate(0, 0, 1), 1, 0), 46, 48},
		{"zero length", at(day, 9, 0), at(day, 9, 0), 18, 19},
	}
	for _, tc := range cases {
		s, e := rowSpan(calendar.Event{Start: tc.start, End: tc.end}, day)
		if s != tc.s || e != tc.e {
			t.Errorf("%s: got %d..%d want %d..%d", tc.name, s, e, tc.s, tc.e)
		}
	}
}

func TestRowSpanDSTUsesWallClock(t *testing.T) {
	ny, err := time.LoadLocation("America/New_York")
	if err != nil {
		t.Skip("tzdata unavailable")
	}
	day := time.Date(2026, 3, 8, 0, 0, 0, 0, ny)
	s, e := rowSpan(calendar.Event{Start: at(day, 1, 0), End: at(day, 4, 0)}, day)
	if s != 2 || e != 8 {
		t.Errorf("DST span = %d..%d, want 2..8", s, e)
	}
}

func TestAssignLanes(t *testing.T) {
	day := time.Date(2026, 10, 1, 0, 0, 0, 0, time.UTC)
	evs := []calendar.Event{
		{Title: "focus", Start: at(day, 13, 0), End: at(day, 15, 0)},
		{Title: "1:1", Start: at(day, 14, 0), End: at(day, 14, 30)},
		{Title: "later", Start: at(day, 15, 0), End: at(day, 16, 0)},
	}
	lanes, n := assignLanes(evs, day)
	if n != 2 || lanes[0] != 0 || lanes[1] != 1 || lanes[2] != 0 {
		t.Errorf("lanes = %v n = %d", lanes, n)
	}
}
