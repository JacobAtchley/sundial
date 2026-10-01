//go:build integration && darwin

package eventkit

import (
	"context"
	"testing"
	"time"

	"github.com/JacobAtchley/sundial/internal/calendar"
)

// Logs counts only. Never log titles, notes, locations, or attendees.
func TestIntegrationReadsCalendars(t *testing.T) {
	if err := RequestAccess(); err != nil {
		t.Skipf("calendar access not granted: %v", err)
	}
	ctx := context.Background()
	src := New()
	cals, err := src.Calendars(ctx)
	if err != nil {
		t.Fatal(err)
	}
	t.Logf("calendars: %d", len(cals))
	now := time.Now()
	evs, err := src.Events(ctx, calendar.Range{Start: now.AddDate(0, 0, -7), End: now.AddDate(0, 0, 7)})
	if err != nil {
		t.Fatal(err)
	}
	t.Logf("events in ±7 days: %d", len(evs))
	for i, e := range evs {
		if e.End.Before(e.Start) {
			t.Errorf("event #%d ends before it starts", i)
		}
		if e.AllDay && (e.Start.Hour() != 0 || e.End.Hour() != 0) {
			t.Errorf("all-day event #%d is not midnight-aligned", i)
		}
	}
}
