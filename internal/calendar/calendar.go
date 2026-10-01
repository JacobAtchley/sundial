// Package calendar defines sundial's calendar domain types and the Source
// interface that calendar backends implement.
package calendar

import (
	"context"
	"errors"
	"fmt"
	"sort"
	"time"
)

// ErrAccessDenied means the user has not granted calendar access.
var ErrAccessDenied = errors.New("calendar access denied")

// Calendar is one calendar from any account macOS syncs.
type Calendar struct {
	ID       string `json:"id"`
	Title    string `json:"title"`
	Color    string `json:"color"`
	Source   string `json:"source"`
	ReadOnly bool   `json:"readOnly"`
}

// Status mirrors EKEventStatus.
type Status int

const (
	StatusNone Status = iota
	StatusConfirmed
	StatusTentative
	StatusCanceled
)

var statusNames = []string{"", "confirmed", "tentative", "canceled"}

func (s Status) String() string {
	if s < 0 || int(s) >= len(statusNames) {
		return ""
	}
	return statusNames[s]
}

func (s Status) MarshalText() ([]byte, error) { return []byte(s.String()), nil }

func (s *Status) UnmarshalText(b []byte) error {
	for i, n := range statusNames {
		if n == string(b) {
			*s = Status(i)
			return nil
		}
	}
	return fmt.Errorf("unknown event status %q", b)
}

// Event is one occurrence. Recurring events share ID across occurrences;
// use Key to tell occurrences apart.
type Event struct {
	ID         string    `json:"id"`
	CalendarID string    `json:"calendarId"`
	Title      string    `json:"title"`
	Start      time.Time `json:"start"`
	End        time.Time `json:"end"`
	AllDay     bool      `json:"allDay"`
	Location   string    `json:"location,omitempty"`
	Notes      string    `json:"notes,omitempty"`
	URL        string    `json:"url,omitempty"`
	Attendees  []string  `json:"attendees,omitempty"`
	Status     Status    `json:"status"`
}

// Key identifies a single occurrence.
func (e Event) Key() string { return fmt.Sprintf("%s@%d", e.ID, e.Start.Unix()) }

// Range returns the event's [Start, End) span.
func (e Event) Range() Range { return Range{Start: e.Start, End: e.End} }

// OccursOn reports whether any part of the event falls on day's local date.
func (e Event) OccursOn(day time.Time) bool {
	d := DayRange(day)
	if !e.End.After(e.Start) {
		return d.Contains(e.Start)
	}
	return e.Range().Overlaps(d)
}

// Range is a half-open time interval [Start, End).
type Range struct {
	Start time.Time
	End   time.Time
}

func (r Range) Contains(t time.Time) bool { return !t.Before(r.Start) && t.Before(r.End) }

func (r Range) Overlaps(o Range) bool { return r.Start.Before(o.End) && o.Start.Before(r.End) }

// Days returns the local midnight of every day the range touches.
func (r Range) Days() []time.Time {
	var out []time.Time
	for d := StartOfDay(r.Start); d.Before(r.End); d = d.AddDate(0, 0, 1) {
		out = append(out, d)
	}
	return out
}

// StartOfDay returns local midnight of t's date in t's location.
func StartOfDay(t time.Time) time.Time {
	y, m, d := t.Date()
	return time.Date(y, m, d, 0, 0, 0, 0, t.Location())
}

// DayRange returns the full local day containing t. It is 23 or 25 hours
// long on DST transition days.
func DayRange(t time.Time) Range {
	s := StartOfDay(t)
	return Range{Start: s, End: s.AddDate(0, 0, 1)}
}

// SortEvents orders all-day events first, then by start, end, and title.
func SortEvents(evs []Event) {
	sort.SliceStable(evs, func(i, j int) bool {
		a, b := evs[i], evs[j]
		if a.AllDay != b.AllDay {
			return a.AllDay
		}
		if !a.Start.Equal(b.Start) {
			return a.Start.Before(b.Start)
		}
		if !a.End.Equal(b.End) {
			return a.End.Before(b.End)
		}
		return a.Title < b.Title
	})
}

// Source is a read-only calendar backend. Write support will live in a
// separate Writer interface so readers are unaffected.
type Source interface {
	Calendars(ctx context.Context) ([]Calendar, error)
	Events(ctx context.Context, r Range) ([]Event, error)
	// Watch emits when calendar data may have changed. The channel closes
	// when ctx is done.
	Watch(ctx context.Context) (<-chan struct{}, error)
}
