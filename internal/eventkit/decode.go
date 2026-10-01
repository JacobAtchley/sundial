package eventkit

import (
	"encoding/json"
	"fmt"
	"math"
	"time"

	"github.com/JacobAtchley/sundial/internal/calendar"
)

type rawEvent struct {
	ID         string   `json:"id"`
	CalendarID string   `json:"calendarId"`
	Title      string   `json:"title"`
	Start      float64  `json:"start"`
	End        float64  `json:"end"`
	AllDay     bool     `json:"allDay"`
	StartDay   string   `json:"startDay"`
	EndDay     string   `json:"endDay"`
	Location   string   `json:"location"`
	Notes      string   `json:"notes"`
	URL        string   `json:"url"`
	Attendees  []string `json:"attendees"`
	Status     int      `json:"status"`
}

func decodeCalendars(data []byte) ([]calendar.Calendar, error) {
	var cals []calendar.Calendar
	if err := json.Unmarshal(data, &cals); err != nil {
		return nil, fmt.Errorf("decode calendars: %w", err)
	}
	return cals, nil
}

func decodeEvents(data []byte, loc *time.Location) ([]calendar.Event, error) {
	var raws []rawEvent
	if err := json.Unmarshal(data, &raws); err != nil {
		return nil, fmt.Errorf("decode events: %w", err)
	}
	out := make([]calendar.Event, 0, len(raws))
	for _, r := range raws {
		e := calendar.Event{
			ID: r.ID, CalendarID: r.CalendarID, Title: r.Title, AllDay: r.AllDay,
			Start: unixToTime(r.Start, loc), End: unixToTime(r.End, loc),
			Location: r.Location, Notes: r.Notes, URL: r.URL, Attendees: r.Attendees,
		}
		if r.Status >= int(calendar.StatusNone) && r.Status <= int(calendar.StatusCanceled) {
			e.Status = calendar.Status(r.Status)
		}
		if r.AllDay && r.StartDay != "" && r.EndDay != "" {
			s, err1 := time.ParseInLocation("2006-01-02", r.StartDay, loc)
			en, err2 := time.ParseInLocation("2006-01-02", r.EndDay, loc)
			if err1 == nil && err2 == nil {
				e.Start, e.End = s, en
			}
		}
		out = append(out, e)
	}
	calendar.SortEvents(out)
	return out, nil
}

func unixToTime(f float64, loc *time.Location) time.Time {
	sec, frac := math.Modf(f)
	return time.Unix(int64(sec), int64(frac*1e9)).In(loc)
}
