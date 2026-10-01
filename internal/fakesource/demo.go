package fakesource

import (
	"fmt"
	"time"

	"github.com/JacobAtchley/sundial/internal/calendar"
)

// Demo returns a Source with fictional events spread six weeks either side
// of anchor. Used by `just demo`, golden tests, and screenshots so real
// calendar data never appears in the repo.
func Demo(anchor time.Time) *Source {
	s := New()
	s.SetCalendars(
		calendar.Calendar{ID: "work", Title: "Work", Color: "#1BADF8", Source: "Example"},
		calendar.Calendar{ID: "personal", Title: "Personal", Color: "#63DA38", Source: "Example"},
		calendar.Calendar{ID: "family", Title: "Family", Color: "#FF2968", Source: "Example"},
		calendar.Calendar{ID: "holidays", Title: "Holidays", Color: "#CC73E1", Source: "Example", ReadOnly: true},
	)
	loc := anchor.Location()
	base := calendar.StartOfDay(anchor)
	at := func(day time.Time, h, m int) time.Time {
		return time.Date(day.Year(), day.Month(), day.Day(), h, m, 0, 0, loc)
	}
	ev := func(id, cal, title string, start, end time.Time) calendar.Event {
		return calendar.Event{ID: id, CalendarID: cal, Title: title, Start: start, End: end, Status: calendar.StatusConfirmed}
	}

	for i := -42; i <= 42; i++ {
		day := base.AddDate(0, 0, i)
		switch day.Weekday() {
		case time.Saturday, time.Sunday:
		default:
			e := ev("standup", "work", "Team Standup", at(day, 9, 30), at(day, 9, 45))
			e.Location, e.URL = "Zoom", "https://example.com/standup"
			s.AddEvents(e)
		}
		switch day.Weekday() {
		case time.Monday:
			e := ev("planning", "work", "Sprint Planning", at(day, 10, 0), at(day, 11, 0))
			e.Attendees = []string{"Alex Rivera", "Sam Chen", "Priya Patel"}
			e.Notes = "Review the backlog and pick sprint goals."
			s.AddEvents(e)
		case time.Tuesday, time.Thursday:
			s.AddEvents(ev("focus", "work", "Focus Time", at(day, 13, 0), at(day, 15, 0)))
		case time.Wednesday:
			e := ev("yoga", "personal", "Yoga", at(day, 18, 0), at(day, 19, 0))
			e.Location = "Community Center"
			s.AddEvents(e)
		case time.Friday:
			s.AddEvents(ev("pizza", "family", "Pizza Night", at(day, 18, 30), at(day, 20, 0)))
		}
	}

	lunch := ev("lunch", "personal", "Lunch with Sam", at(base, 12, 0), at(base, 13, 0))
	lunch.Location = "Corner Cafe"
	oneOnOne := ev("one-on-one", "work", "1:1 with Manager", at(base, 14, 0), at(base, 14, 30))
	oneOnOne.Status = calendar.StatusTentative
	long := ev("long", "work", "Quarterly planning sync with the extended product, design, and engineering group 🚀",
		at(base.AddDate(0, 0, 1), 16, 0), at(base.AddDate(0, 0, 1), 16, 30))
	dentist := ev("dentist", "personal", "Dentist", at(base.AddDate(0, 0, 3), 8, 0), at(base.AddDate(0, 0, 3), 9, 0))
	dentist.Location = "123 Example St"
	offsite := ev("offsite", "work", "Team Offsite", base.AddDate(0, 0, 5), base.AddDate(0, 0, 7))
	offsite.AllDay = true
	holiday := ev("founders", "holidays", "Founders Day", base.AddDate(0, 0, 10), base.AddDate(0, 0, 11))
	holiday.AllDay = true
	movie := ev("movie", "personal", "Late Movie", at(base.AddDate(0, 0, -2), 22, 0), at(base.AddDate(0, 0, -1), 0, 30))
	s.AddEvents(lunch, oneOnOne, long, dentist, offsite, holiday, movie)

	for i := range s.events {
		if s.events[i].ID == "" {
			s.events[i].ID = fmt.Sprintf("demo-%d", i)
		}
	}
	return s
}
