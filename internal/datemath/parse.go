package datemath

import (
	"regexp"
	"strconv"
	"strings"
	"time"

	"github.com/JacobAtchley/sundial/internal/calendar"
)

var relativeRe = regexp.MustCompile(`^([+-])(\d{1,4})([dwm])$`)

var weekdays = map[string]time.Weekday{
	"sun": time.Sunday, "sunday": time.Sunday,
	"mon": time.Monday, "monday": time.Monday,
	"tue": time.Tuesday, "tues": time.Tuesday, "tuesday": time.Tuesday,
	"wed": time.Wednesday, "wednesday": time.Wednesday,
	"thu": time.Thursday, "thur": time.Thursday, "thurs": time.Thursday, "thursday": time.Thursday,
	"fri": time.Friday, "friday": time.Friday,
	"sat": time.Saturday, "saturday": time.Saturday,
}

// ParseDateExpr parses palette date input relative to now. Accepted forms:
// today, tomorrow, yesterday, +Nd/-Nd/+Nw/+Nm, YYYY-MM-DD, M/D, a weekday
// name (next occurrence, today included), and "next <weekday>" (strictly
// after today). It returns local midnight in now's location.
func ParseDateExpr(s string, now time.Time) (time.Time, bool) {
	s = strings.ToLower(strings.TrimSpace(s))
	loc := now.Location()
	today := calendar.StartOfDay(now)

	switch s {
	case "":
		return time.Time{}, false
	case "today":
		return today, true
	case "tomorrow", "tmr":
		return today.AddDate(0, 0, 1), true
	case "yesterday":
		return today.AddDate(0, 0, -1), true
	}

	if m := relativeRe.FindStringSubmatch(s); m != nil {
		n, _ := strconv.Atoi(m[2])
		if m[1] == "-" {
			n = -n
		}
		switch m[3] {
		case "d":
			return today.AddDate(0, 0, n), true
		case "w":
			return today.AddDate(0, 0, 7*n), true
		default:
			return AddMonthsClamped(today, n), true
		}
	}

	if t, err := time.ParseInLocation("2006-01-02", s, loc); err == nil {
		return t, true
	}
	if t, err := time.ParseInLocation("1/2", s, loc); err == nil {
		out := time.Date(now.Year(), t.Month(), t.Day(), 0, 0, 0, 0, loc)
		if out.Month() != t.Month() {
			return time.Time{}, false
		}
		return out, true
	}

	next := false
	if rest, ok := strings.CutPrefix(s, "next "); ok {
		next, s = true, strings.TrimSpace(rest)
	}
	if wd, ok := weekdays[s]; ok {
		diff := (int(wd) - int(today.Weekday()) + 7) % 7
		if next && diff == 0 {
			diff = 7
		}
		return today.AddDate(0, 0, diff), true
	}
	return time.Time{}, false
}
