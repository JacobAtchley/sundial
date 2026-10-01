package calendar

import (
	"encoding/json"
	"testing"
	"time"
)

func date(y int, m time.Month, d, h, min int, loc *time.Location) time.Time {
	return time.Date(y, m, d, h, min, 0, 0, loc)
}

func TestRangeContainsIsHalfOpen(t *testing.T) {
	r := Range{date(2026, 10, 1, 0, 0, time.UTC), date(2026, 10, 2, 0, 0, time.UTC)}
	if !r.Contains(r.Start) {
		t.Error("start should be contained")
	}
	if r.Contains(r.End) {
		t.Error("end should not be contained")
	}
}

func TestRangeOverlaps(t *testing.T) {
	a := Range{date(2026, 10, 1, 9, 0, time.UTC), date(2026, 10, 1, 10, 0, time.UTC)}
	touching := Range{a.End, a.End.Add(time.Hour)}
	inside := Range{a.Start.Add(10 * time.Minute), a.Start.Add(20 * time.Minute)}
	if a.Overlaps(touching) {
		t.Error("touching ranges must not overlap")
	}
	if !a.Overlaps(inside) || !inside.Overlaps(a) {
		t.Error("nested ranges must overlap both ways")
	}
}

func TestOccursOnMultiDayAllDay(t *testing.T) {
	// All-day Oct 1 through Oct 2, exclusive end Oct 3.
	e := Event{AllDay: true, Start: date(2026, 10, 1, 0, 0, time.UTC), End: date(2026, 10, 3, 0, 0, time.UTC)}
	for _, tc := range []struct {
		day  int
		want bool
	}{{30, false}, {1, true}, {2, true}, {3, false}} {
		m := time.October
		if tc.day == 30 {
			m = time.September
		}
		if got := e.OccursOn(date(2026, m, tc.day, 12, 0, time.UTC)); got != tc.want {
			t.Errorf("day %d: got %v want %v", tc.day, got, tc.want)
		}
	}
}

func TestOccursOnCrossesMidnight(t *testing.T) {
	e := Event{Start: date(2026, 9, 29, 22, 0, time.UTC), End: date(2026, 9, 30, 0, 30, time.UTC)}
	if !e.OccursOn(date(2026, 9, 29, 0, 0, time.UTC)) || !e.OccursOn(date(2026, 9, 30, 0, 0, time.UTC)) {
		t.Error("event crossing midnight must occur on both days")
	}
	endsAtMidnight := Event{Start: date(2026, 9, 29, 22, 0, time.UTC), End: date(2026, 9, 30, 0, 0, time.UTC)}
	if endsAtMidnight.OccursOn(date(2026, 9, 30, 0, 0, time.UTC)) {
		t.Error("event ending exactly at midnight must not occur on the next day")
	}
}

func TestOccursOnZeroDuration(t *testing.T) {
	e := Event{Start: date(2026, 10, 1, 9, 0, time.UTC), End: date(2026, 10, 1, 9, 0, time.UTC)}
	if !e.OccursOn(date(2026, 10, 1, 0, 0, time.UTC)) {
		t.Error("zero-duration event must occur on its start day")
	}
}

func TestDayRangeAcrossDST(t *testing.T) {
	ny, err := time.LoadLocation("America/New_York")
	if err != nil {
		t.Skip("tzdata unavailable")
	}
	spring := DayRange(date(2026, 3, 8, 15, 0, ny))
	if got := spring.End.Sub(spring.Start); got != 23*time.Hour {
		t.Errorf("spring-forward day length = %v, want 23h", got)
	}
	fall := DayRange(date(2026, 11, 1, 15, 0, ny))
	if got := fall.End.Sub(fall.Start); got != 25*time.Hour {
		t.Errorf("fall-back day length = %v, want 25h", got)
	}
	days := Range{date(2026, 3, 7, 0, 0, ny), date(2026, 3, 10, 0, 0, ny)}.Days()
	if len(days) != 3 {
		t.Fatalf("got %d days, want 3", len(days))
	}
	for _, d := range days {
		if d.Hour() != 0 || d.Minute() != 0 {
			t.Errorf("day %v is not local midnight", d)
		}
	}
}

func TestRangeDaysIncludesPartialLastDay(t *testing.T) {
	r := Range{date(2026, 10, 1, 9, 0, time.UTC), date(2026, 10, 3, 1, 0, time.UTC)}
	if got := len(r.Days()); got != 3 {
		t.Errorf("got %d days, want 3", got)
	}
}

func TestSortEvents(t *testing.T) {
	evs := []Event{
		{Title: "b", Start: date(2026, 10, 1, 10, 0, time.UTC), End: date(2026, 10, 1, 11, 0, time.UTC)},
		{Title: "a", Start: date(2026, 10, 1, 10, 0, time.UTC), End: date(2026, 10, 1, 11, 0, time.UTC)},
		{Title: "early", Start: date(2026, 10, 1, 8, 0, time.UTC), End: date(2026, 10, 1, 9, 0, time.UTC)},
		{Title: "allday", AllDay: true, Start: date(2026, 10, 1, 0, 0, time.UTC), End: date(2026, 10, 2, 0, 0, time.UTC)},
	}
	SortEvents(evs)
	want := []string{"allday", "early", "a", "b"}
	for i, w := range want {
		if evs[i].Title != w {
			t.Fatalf("order = %v, want %v", titles(evs), want)
		}
	}
}

func titles(evs []Event) []string {
	out := make([]string, len(evs))
	for i, e := range evs {
		out[i] = e.Title
	}
	return out
}

func TestKeyDistinguishesRecurrences(t *testing.T) {
	a := Event{ID: "x", Start: date(2026, 10, 1, 9, 0, time.UTC)}
	b := Event{ID: "x", Start: date(2026, 10, 2, 9, 0, time.UTC)}
	if a.Key() == b.Key() {
		t.Error("occurrences with the same ID must have different keys")
	}
}

func TestStatusJSON(t *testing.T) {
	b, err := json.Marshal(Event{Status: StatusTentative})
	if err != nil {
		t.Fatal(err)
	}
	var back struct {
		Status string `json:"status"`
	}
	if err := json.Unmarshal(b, &back); err != nil {
		t.Fatal(err)
	}
	if back.Status != "tentative" {
		t.Errorf("status = %q, want tentative", back.Status)
	}
	var e Event
	if err := json.Unmarshal(b, &e); err != nil || e.Status != StatusTentative {
		t.Errorf("round trip status = %v, err %v", e.Status, err)
	}
}
