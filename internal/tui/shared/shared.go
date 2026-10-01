// Package shared holds types every TUI view uses: the render context,
// navigation intents, cross-component messages, and text helpers.
package shared

import (
	"time"

	"github.com/JacobAtchley/sundial/internal/calendar"
	"github.com/JacobAtchley/sundial/internal/store"
	"github.com/JacobAtchley/sundial/internal/tui/theme"
)

// Context is everything a view needs to render. Width and Height are the
// space available to the view (status bar excluded).
type Context struct {
	Store     *store.Store
	Theme     theme.Theme
	Now       time.Time
	WeekStart time.Weekday
	Use24h    bool
	DayStart  int
	Width     int
	Height    int
}

// Nav is a navigation intent, decoupled from concrete keys.
type Nav int

const (
	NavLeft Nav = iota
	NavRight
	NavUp
	NavDown
	NavPrev
	NavNext
)

type ViewID int

const (
	ViewMonth ViewID = iota
	ViewWeek
	ViewAgenda
)

func (v ViewID) String() string {
	switch v {
	case ViewMonth:
		return "month"
	case ViewWeek:
		return "week"
	default:
		return "agenda"
	}
}

// ParseView maps a config value to a ViewID (unknown means agenda).
func ParseView(s string) ViewID {
	switch s {
	case "month":
		return ViewMonth
	case "week":
		return ViewWeek
	default:
		return ViewAgenda
	}
}

// Messages produced by the palette and handled by the root app.
type (
	SwitchViewMsg     struct{ View ViewID }
	GotoDateMsg       struct{ Date time.Time }
	ToggleCalendarMsg struct{ ID string }
	ReloadMsg         struct{}
	EditConfigMsg     struct{}
	ShowHelpMsg       struct{}
	QuitMsg           struct{}
	ShowEventMsg      struct{ Event calendar.Event }
)
