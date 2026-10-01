package shared

import (
	"context"
	"time"

	"github.com/JacobAtchley/sundial/internal/calendar"
	"github.com/JacobAtchley/sundial/internal/config"
	"github.com/JacobAtchley/sundial/internal/fakesource"
	"github.com/JacobAtchley/sundial/internal/store"
	"github.com/JacobAtchley/sundial/internal/tui/theme"
)

// TestNow is the fixed clock for golden tests: Thursday 2026-10-01 08:00 UTC.
var TestNow = time.Date(2026, 10, 1, 8, 0, 0, 0, time.UTC)

// TestContext returns a Context over fictional demo data with the store
// preloaded 60 days either side of TestNow. For tests only.
func TestContext(width, height int) Context {
	st := store.New(fakesource.Demo(TestNow))
	ctx := context.Background()
	_ = st.LoadCalendars(ctx)
	_ = st.Load(ctx, calendar.Range{Start: TestNow.AddDate(0, 0, -60), End: TestNow.AddDate(0, 0, 60)})
	return Context{
		Store:     st,
		Theme:     theme.New(config.Default().Theme, true),
		Now:       TestNow,
		WeekStart: time.Sunday,
		DayStart:  8,
		Width:     width,
		Height:    height,
	}
}
