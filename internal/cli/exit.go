package cli

import (
	"errors"

	"github.com/JacobAtchley/sundial/internal/calendar"
)

// ExitError carries a specific process exit code.
type ExitError struct {
	Code int
	Err  error
}

func (e *ExitError) Error() string { return e.Err.Error() }
func (e *ExitError) Unwrap() error { return e.Err }

// ExitCode maps an error to sundial's exit codes: 0 ok, 1 general error,
// 2 calendar access denied.
func ExitCode(err error) int {
	if err == nil {
		return 0
	}
	var e *ExitError
	if errors.As(err, &e) {
		return e.Code
	}
	if errors.Is(err, calendar.ErrAccessDenied) {
		return 2
	}
	return 1
}
