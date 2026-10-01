package week

import (
	"time"

	"github.com/JacobAtchley/sundial/internal/calendar"
)

const rowsPerDay = 48 // half-hour rows

// rowSpan maps an event onto day's half-hour rows using wall-clock fields,
// so DST days keep a 48-row grid. end is exclusive and always > start.
func rowSpan(e calendar.Event, day time.Time) (int, int) {
	dayStart := calendar.StartOfDay(day)
	dayEnd := dayStart.AddDate(0, 0, 1)
	start := 0
	if !e.Start.Before(dayStart) {
		start = (e.Start.Hour()*60 + e.Start.Minute()) / 30
	}
	end := rowsPerDay
	if e.End.Before(dayEnd) {
		mins := e.End.Hour()*60 + e.End.Minute()
		end = (mins + 29) / 30
	}
	if end <= start {
		end = start + 1
	}
	return start, min(end, rowsPerDay)
}

// assignLanes gives each event (in order) the first lane free at its start.
func assignLanes(evs []calendar.Event, day time.Time) ([]int, int) {
	lanes := make([]int, len(evs))
	var laneEnds []int
	for i, e := range evs {
		s, en := rowSpan(e, day)
		placed := false
		for l, end := range laneEnds {
			if end <= s {
				lanes[i], laneEnds[l], placed = l, en, true
				break
			}
		}
		if !placed {
			lanes[i] = len(laneEnds)
			laneEnds = append(laneEnds, en)
		}
	}
	return lanes, len(laneEnds)
}
