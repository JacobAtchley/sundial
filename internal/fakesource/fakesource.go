// Package fakesource is an in-memory calendar.Source for tests and demos.
// It only ever holds fictional data.
package fakesource

import (
	"context"
	"slices"
	"sync"

	"github.com/JacobAtchley/sundial/internal/calendar"
)

type Source struct {
	mu        sync.Mutex
	calendars []calendar.Calendar
	events    []calendar.Event
	err       error
	calls     int
	watchers  []chan struct{}
}

func New() *Source { return &Source{} }

func (s *Source) SetCalendars(c ...calendar.Calendar) {
	s.mu.Lock()
	defer s.mu.Unlock()
	s.calendars = slices.Clone(c)
}

func (s *Source) AddEvents(e ...calendar.Event) {
	s.mu.Lock()
	defer s.mu.Unlock()
	s.events = append(s.events, e...)
}

// SetError makes every subsequent call fail with err (nil clears it).
func (s *Source) SetError(err error) {
	s.mu.Lock()
	defer s.mu.Unlock()
	s.err = err
}

// EventCalls reports how many times Events has been called.
func (s *Source) EventCalls() int {
	s.mu.Lock()
	defer s.mu.Unlock()
	return s.calls
}

// Trigger notifies every active watcher.
func (s *Source) Trigger() {
	s.mu.Lock()
	defer s.mu.Unlock()
	for _, ch := range s.watchers {
		select {
		case ch <- struct{}{}:
		default:
		}
	}
}

func (s *Source) Calendars(context.Context) ([]calendar.Calendar, error) {
	s.mu.Lock()
	defer s.mu.Unlock()
	if s.err != nil {
		return nil, s.err
	}
	return slices.Clone(s.calendars), nil
}

func (s *Source) Events(_ context.Context, r calendar.Range) ([]calendar.Event, error) {
	s.mu.Lock()
	defer s.mu.Unlock()
	s.calls++
	if s.err != nil {
		return nil, s.err
	}
	var out []calendar.Event
	for _, e := range s.events {
		if e.Range().Overlaps(r) || (!e.End.After(e.Start) && r.Contains(e.Start)) {
			out = append(out, e)
		}
	}
	calendar.SortEvents(out)
	return out, nil
}

func (s *Source) Watch(ctx context.Context) (<-chan struct{}, error) {
	ch := make(chan struct{}, 1)
	s.mu.Lock()
	s.watchers = append(s.watchers, ch)
	s.mu.Unlock()
	go func() {
		<-ctx.Done()
		s.mu.Lock()
		defer s.mu.Unlock()
		s.watchers = slices.DeleteFunc(s.watchers, func(c chan struct{}) bool { return c == ch })
		close(ch)
	}()
	return ch, nil
}
