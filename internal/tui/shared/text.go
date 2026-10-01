package shared

import (
	"fmt"
	"strings"
	"time"

	"charm.land/lipgloss/v2"
	"github.com/charmbracelet/x/ansi"

	"github.com/JacobAtchley/sundial/internal/calendar"
)

// OneLine collapses newlines, tabs, and runs of spaces so a title can
// never break a row.
func OneLine(s string) string { return strings.Join(strings.Fields(s), " ") }

// Truncate cuts s to at most w display cells, adding an ellipsis.
func Truncate(s string, w int) string {
	if w <= 0 {
		return ""
	}
	if lipgloss.Width(s) <= w {
		return s
	}
	return ansi.Truncate(s, w, "…")
}

// PadRight truncates or pads s to exactly w display cells.
func PadRight(s string, w int) string {
	s = Truncate(s, w)
	if pad := w - lipgloss.Width(s); pad > 0 {
		s += strings.Repeat(" ", pad)
	}
	return s
}

func FormatClock(t time.Time, use24h bool) string {
	if use24h {
		return t.Format("15:04")
	}
	return t.Format("3:04pm")
}

func FormatHour(h int, use24h bool) string {
	if use24h {
		return fmt.Sprintf("%02d:00", h)
	}
	switch {
	case h == 0:
		return "12am"
	case h < 12:
		return fmt.Sprintf("%dam", h)
	case h == 12:
		return "12pm"
	default:
		return fmt.Sprintf("%dpm", h-12)
	}
}

func FormatSpan(e calendar.Event, use24h bool) string {
	if e.AllDay {
		return "all day"
	}
	return FormatClock(e.Start, use24h) + "–" + FormatClock(e.End, use24h)
}
