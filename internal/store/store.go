// Package store caches calendar events per local day on top of a
// calendar.Source. Views read only from the store.
package store

import (
	"context"
	"slices"
	"sync"
	"time"

	"github.com/JacobAtchley/sundial/internal/calendar"
)

type Store struct {
	src calendar.Source

	mu          sync.RWMutex
	days        map[int64][]calendar.Event // keyed by local-midnight Unix seconds
	fresh       map[int64]bool
	calendars   []calendar.Calendar
	hiddenNames []string
	hidden      map[string]bool // calendar IDs
}

func New(src calendar.Source) *Store {
	return &Store{
		src:    src,
		days:   map[int64][]calendar.Event{},
		fresh:  map[int64]bool{},
		hidden: map[string]bool{},
	}
}

func dayKey(t time.Time) int64 { return calendar.StartOfDay(t).Unix() }

// wholeDays widens r to local midnight boundaries.
func wholeDays(r calendar.Range) calendar.Range {
	end := calendar.StartOfDay(r.End)
	if end.Before(r.End) {
		end = end.AddDate(0, 0, 1)
	}
	return calendar.Range{Start: calendar.StartOfDay(r.Start), End: end}
}

func (s *Store) LoadCalendars(ctx context.Context) error {
	cals, err := s.src.Calendars(ctx)
	if err != nil {
		return err
	}
	s.mu.Lock()
	defer s.mu.Unlock()
	s.calendars = cals
	s.resolveHiddenLocked()
	return nil
}

func (s *Store) Calendars() []calendar.Calendar {
	s.mu.RLock()
	defer s.mu.RUnlock()
	return slices.Clone(s.calendars)
}

func (s *Store) Calendar(id string) (calendar.Calendar, bool) {
	s.mu.RLock()
	defer s.mu.RUnlock()
	for _, c := range s.calendars {
		if c.ID == id {
			return c, true
		}
	}
	return calendar.Calendar{}, false
}

// SetHidden hides calendars whose title or ID is listed. Names that match
// no known calendar are kept and resolved when calendars load.
func (s *Store) SetHidden(titlesOrIDs []string) {
	s.mu.Lock()
	defer s.mu.Unlock()
	s.hiddenNames = slices.Clone(titlesOrIDs)
	s.hidden = map[string]bool{}
	s.resolveHiddenLocked()
}

func (s *Store) resolveHiddenLocked() {
	for _, name := range s.hiddenNames {
		for _, c := range s.calendars {
			if c.ID == name || c.Title == name {
				s.hidden[c.ID] = true
			}
		}
	}
}

func (s *Store) ToggleHidden(id string) {
	s.mu.Lock()
	defer s.mu.Unlock()
	if s.hidden[id] {
		delete(s.hidden, id)
	} else {
		s.hidden[id] = true
	}
}

func (s *Store) IsHidden(id string) bool {
	s.mu.RLock()
	defer s.mu.RUnlock()
	return s.hidden[id]
}

// Load fetches r (widened to whole days) and replaces those day buckets.
// On error the previous data stays in place.
func (s *Store) Load(ctx context.Context, r calendar.Range) error {
	r = wholeDays(r)
	evs, err := s.src.Events(ctx, r)
	if err != nil {
		return err
	}
	days := r.Days()
	buckets := make(map[int64][]calendar.Event, len(days))
	for _, d := range days {
		buckets[d.Unix()] = nil
	}
	for _, e := range evs {
		for _, d := range days {
			if e.OccursOn(d) {
				buckets[d.Unix()] = append(buckets[d.Unix()], e)
			}
		}
	}
	s.mu.Lock()
	defer s.mu.Unlock()
	for k, v := range buckets {
		calendar.SortEvents(v)
		s.days[k] = v
		s.fresh[k] = true
	}
	return nil
}

// Covers reports whether every day in r has fresh data.
func (s *Store) Covers(r calendar.Range) bool {
	s.mu.RLock()
	defer s.mu.RUnlock()
	for _, d := range wholeDays(r).Days() {
		if !s.fresh[d.Unix()] {
			return false
		}
	}
	return true
}

// Invalidate marks everything stale but keeps it for display.
func (s *Store) Invalidate() {
	s.mu.Lock()
	defer s.mu.Unlock()
	clear(s.fresh)
}

func (s *Store) visibleLocked(evs []calendar.Event) []calendar.Event {
	out := make([]calendar.Event, 0, len(evs))
	for _, e := range evs {
		if !s.hidden[e.CalendarID] {
			out = append(out, e)
		}
	}
	return out
}

// EventsOn returns visible events on day's local date, sorted.
func (s *Store) EventsOn(day time.Time) []calendar.Event {
	s.mu.RLock()
	defer s.mu.RUnlock()
	return s.visibleLocked(s.days[dayKey(day)])
}

// EventsIn returns visible events touching r, deduplicated and sorted.
func (s *Store) EventsIn(r calendar.Range) []calendar.Event {
	s.mu.RLock()
	defer s.mu.RUnlock()
	var keys []int64
	for _, d := range wholeDays(r).Days() {
		keys = append(keys, d.Unix())
	}
	return s.collectLocked(keys, &r)
}

// All returns every cached visible event, deduplicated and sorted.
func (s *Store) All() []calendar.Event {
	s.mu.RLock()
	defer s.mu.RUnlock()
	keys := make([]int64, 0, len(s.days))
	for k := range s.days {
		keys = append(keys, k)
	}
	slices.Sort(keys)
	return s.collectLocked(keys, nil)
}

func (s *Store) collectLocked(keys []int64, within *calendar.Range) []calendar.Event {
	seen := map[string]bool{}
	var out []calendar.Event
	for _, k := range keys {
		for _, e := range s.visibleLocked(s.days[k]) {
			if seen[e.Key()] {
				continue
			}
			if within != nil && !e.Range().Overlaps(*within) && !within.Contains(e.Start) {
				continue
			}
			seen[e.Key()] = true
			out = append(out, e)
		}
	}
	calendar.SortEvents(out)
	return out
}
