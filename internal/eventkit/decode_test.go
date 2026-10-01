package eventkit

import (
	"strconv"
	"testing"
	"time"

	"github.com/JacobAtchley/sundial/internal/calendar"
)

func TestDecodeCalendars(t *testing.T) {
	cals, err := decodeCalendars([]byte(`[{"id":"c1","title":"Work","color":"#1BADF8","source":"iCloud","readOnly":false}]`))
	if err != nil {
		t.Fatal(err)
	}
	if len(cals) != 1 || cals[0].Title != "Work" || cals[0].Color != "#1BADF8" {
		t.Errorf("got %+v", cals)
	}
}

func TestDecodeTimedEvent(t *testing.T) {
	ny, err := time.LoadLocation("America/New_York")
	if err != nil {
		t.Skip("tzdata unavailable")
	}
	start := time.Date(2026, 10, 1, 9, 30, 0, 0, ny)
	raw := `[{"id":"e1","calendarId":"c1","title":"Standup","start":` + unix(start) + `,"end":` + unix(start.Add(15*time.Minute)) +
		`,"allDay":false,"location":"Zoom","notes":"","url":"https://example.com","attendees":["A"],"status":2}]`
	evs, err := decodeEvents([]byte(raw), ny)
	if err != nil {
		t.Fatal(err)
	}
	e := evs[0]
	if !e.Start.Equal(start) || e.Start.Location() != ny || e.End.Sub(e.Start) != 15*time.Minute {
		t.Errorf("times = %v..%v", e.Start, e.End)
	}
	if e.Status != calendar.StatusTentative || e.Location != "Zoom" || len(e.Attendees) != 1 {
		t.Errorf("fields = %+v", e)
	}
}

func TestDecodeAllDayUsesDates(t *testing.T) {
	ny, err := time.LoadLocation("America/New_York")
	if err != nil {
		t.Skip("tzdata unavailable")
	}
	// start/end seconds are deliberately in a different zone's midnight; the
	// day strings must win so the event never shifts.
	raw := `[{"id":"e2","calendarId":"c1","title":"Offsite","start":0,"end":0,"allDay":true,"startDay":"2026-10-06","endDay":"2026-10-08","status":1}]`
	evs, err := decodeEvents([]byte(raw), ny)
	if err != nil {
		t.Fatal(err)
	}
	e := evs[0]
	if !e.Start.Equal(time.Date(2026, 10, 6, 0, 0, 0, 0, ny)) || !e.End.Equal(time.Date(2026, 10, 8, 0, 0, 0, 0, ny)) {
		t.Errorf("all-day = %v..%v", e.Start, e.End)
	}
}

func TestDecodeClampsUnknownStatus(t *testing.T) {
	evs, err := decodeEvents([]byte(`[{"id":"x","start":0,"end":60,"status":9}]`), time.UTC)
	if err != nil {
		t.Fatal(err)
	}
	if evs[0].Status != calendar.StatusNone {
		t.Errorf("status = %v", evs[0].Status)
	}
}

func TestDecodeRejectsBadJSON(t *testing.T) {
	if _, err := decodeEvents([]byte(`{`), time.UTC); err == nil {
		t.Error("expected error")
	}
}

func unix(t time.Time) string { return strconv.FormatInt(t.Unix(), 10) }
