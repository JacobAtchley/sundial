//go:build darwin

package eventkit

/*
#cgo CFLAGS: -fobjc-arc
#cgo LDFLAGS: -framework EventKit -framework Foundation -framework CoreGraphics
#include <stdlib.h>
#include "bridge.h"
*/
import "C"

import (
	"context"
	"errors"
	"time"

	"github.com/JacobAtchley/sundial/internal/calendar"
)

func Status() AuthStatus { return AuthStatus(C.sd_auth_status()) }

// RequestAccess prompts on first run. It returns calendar.ErrAccessDenied
// unless sundial ends up with full read access.
func RequestAccess() error {
	switch Status() {
	case StatusFullAccess:
		return nil
	case StatusNotDetermined:
		if C.sd_request_access() == 1 && Status() == StatusFullAccess {
			return nil
		}
	}
	return calendar.ErrAccessDenied
}

const (
	// pollInterval backs up change notifications, which were not observed
	// reliably during the feasibility spike.
	pollInterval  = 60 * time.Second
	debounceDelay = 500 * time.Millisecond
)

type Source struct{}

func New() *Source { return &Source{} }

func ready(ctx context.Context) error {
	if err := ctx.Err(); err != nil {
		return err
	}
	if Status() != StatusFullAccess {
		return calendar.ErrAccessDenied
	}
	return nil
}

func (s *Source) Calendars(ctx context.Context) ([]calendar.Calendar, error) {
	if err := ready(ctx); err != nil {
		return nil, err
	}
	raw := C.sd_calendars()
	if raw == nil {
		return nil, errors.New("eventkit: could not encode calendars")
	}
	defer C.sd_free(raw)
	return decodeCalendars([]byte(C.GoString(raw)))
}

func (s *Source) Events(ctx context.Context, r calendar.Range) ([]calendar.Event, error) {
	if err := ready(ctx); err != nil {
		return nil, err
	}
	start := float64(r.Start.UnixNano()) / 1e9
	end := float64(r.End.UnixNano()) / 1e9
	raw := C.sd_events(C.double(start), C.double(end))
	if raw == nil {
		return nil, errors.New("eventkit: could not encode events")
	}
	defer C.sd_free(raw)
	return decodeEvents([]byte(C.GoString(raw)), time.Local)
}

// Watch supports one active watcher per process.
func (s *Source) Watch(ctx context.Context) (<-chan struct{}, error) {
	if err := ready(ctx); err != nil {
		return nil, err
	}
	C.sd_watch_start()
	go func() {
		<-ctx.Done()
		C.sd_watch_stop()
	}()
	return debounce(ctx, withPoll(ctx, changes, pollInterval), debounceDelay), nil
}
