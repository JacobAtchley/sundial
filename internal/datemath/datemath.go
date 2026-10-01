// Package datemath holds calendar grid and navigation arithmetic. All
// functions work on local wall-clock dates so DST days stay correct.
package datemath

import (
	"time"

	"github.com/JacobAtchley/sundial/internal/calendar"
)

// StartOfWeek returns local midnight of the first day of t's week.
func StartOfWeek(t time.Time, weekStart time.Weekday) time.Time {
	day := calendar.StartOfDay(t)
	diff := (int(day.Weekday()) - int(weekStart) + 7) % 7
	return day.AddDate(0, 0, -diff)
}

// Days returns n consecutive local midnights starting at start's date.
func Days(start time.Time, n int) []time.Time {
	out := make([]time.Time, n)
	day := calendar.StartOfDay(start)
	for i := range out {
		out[i] = day.AddDate(0, 0, i)
	}
	return out
}

// MonthGrid returns the weeks covering month, each a row of 7 days.
func MonthGrid(year int, month time.Month, weekStart time.Weekday, loc *time.Location) [][]time.Time {
	first := time.Date(year, month, 1, 0, 0, 0, 0, loc)
	last := first.AddDate(0, 1, -1)
	var rows [][]time.Time
	for day := StartOfWeek(first, weekStart); !day.After(last); day = day.AddDate(0, 0, 7) {
		rows = append(rows, Days(day, 7))
	}
	return rows
}

// AddMonthsClamped adds n months, clamping the day to the target month's
// length (Jan 31 + 1 month = Feb 28).
func AddMonthsClamped(t time.Time, n int) time.Time {
	y, m, day := t.Date()
	first := time.Date(y, m+time.Month(n), 1, 0, 0, 0, 0, t.Location())
	if last := first.AddDate(0, 1, -1).Day(); day > last {
		day = last
	}
	return time.Date(first.Year(), first.Month(), day, t.Hour(), t.Minute(), t.Second(), t.Nanosecond(), t.Location())
}
