//go:build !darwin

package eventkit

import (
	"context"
	"errors"

	"github.com/JacobAtchley/sundial/internal/calendar"
)

func Status() AuthStatus { return StatusDenied }

func RequestAccess() error { return errors.ErrUnsupported }

type Source struct{}

func New() *Source { return &Source{} }

func (s *Source) Calendars(context.Context) ([]calendar.Calendar, error) {
	return nil, errors.ErrUnsupported
}

func (s *Source) Events(context.Context, calendar.Range) ([]calendar.Event, error) {
	return nil, errors.ErrUnsupported
}

func (s *Source) Watch(context.Context) (<-chan struct{}, error) { return nil, errors.ErrUnsupported }
