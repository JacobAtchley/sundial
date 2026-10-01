package cli

import (
	"errors"
	"fmt"
	"os"
	"time"

	"github.com/JacobAtchley/sundial/internal/calendar"
	"github.com/JacobAtchley/sundial/internal/eventkit"
	"github.com/JacobAtchley/sundial/internal/fakesource"
)

// Options carries dependencies the commands need. Tests replace them.
type Options struct {
	Now        func() time.Time
	OpenSource func(name string) (calendar.Source, error)
	Getenv     func(string) string
	// LaunchTUI runs the interactive app. Nil means "print help".
	LaunchTUI func(Env) error
}

func DefaultOptions() Options {
	return Options{Now: time.Now, OpenSource: openSource, Getenv: os.Getenv}
}

// openSource picks the backend: "fake" for demo data, "" or "eventkit" for
// the real calendar (prompting for access on first run).
func openSource(name string) (calendar.Source, error) {
	switch name {
	case "fake":
		return fakesource.Demo(time.Now()), nil
	case "", "eventkit":
		if err := eventkit.RequestAccess(); err != nil {
			return nil, err
		}
		return eventkit.New(), nil
	default:
		return nil, fmt.Errorf("unknown source %q (want eventkit or fake)", name)
	}
}

var errDeniedHelp = errors.New("calendar access denied: grant access in System Settings → Privacy & Security → Calendars, then run sundial again")
