package datemath

import (
	"testing"
	"time"
)

func d(y int, m time.Month, day int) time.Time { return time.Date(y, m, day, 0, 0, 0, 0, time.UTC) }

func TestStartOfWeek(t *testing.T) {
	thu := time.Date(2026, 10, 1, 15, 30, 0, 0, time.UTC)
	if got := StartOfWeek(thu, time.Sunday); !got.Equal(d(2026, 9, 27)) {
		t.Errorf("sunday start = %v", got)
	}
	if got := StartOfWeek(thu, time.Monday); !got.Equal(d(2026, 9, 28)) {
		t.Errorf("monday start = %v", got)
	}
	sun := d(2026, 10, 4)
	if got := StartOfWeek(sun, time.Sunday); !got.Equal(sun) {
		t.Errorf("week start on its own start day = %v", got)
	}
	if got := StartOfWeek(sun, time.Monday); !got.Equal(d(2026, 9, 28)) {
		t.Errorf("sunday with monday start = %v", got)
	}
}

func TestDaysAcrossFallBack(t *testing.T) {
	ny, err := time.LoadLocation("America/New_York")
	if err != nil {
		t.Skip("tzdata unavailable")
	}
	start := StartOfWeek(time.Date(2026, 11, 3, 12, 0, 0, 0, ny), time.Sunday)
	days := Days(start, 7)
	if days[0].Day() != 1 || days[0].Month() != time.November {
		t.Fatalf("week starts %v, want Nov 1", days[0])
	}
	for _, day := range days {
		if day.Hour() != 0 {
			t.Errorf("%v is not local midnight", day)
		}
	}
}

func TestMonthGridRows(t *testing.T) {
	cases := []struct {
		name  string
		year  int
		month time.Month
		ws    time.Weekday
		rows  int
		first time.Time
		last  time.Time
	}{
		{"oct sunday", 2026, time.October, time.Sunday, 5, d(2026, 9, 27), d(2026, 10, 31)},
		{"oct monday", 2026, time.October, time.Monday, 5, d(2026, 9, 28), d(2026, 11, 1)},
		{"feb exact fit", 2026, time.February, time.Sunday, 4, d(2026, 2, 1), d(2026, 2, 28)},
		{"aug six rows", 2026, time.August, time.Sunday, 6, d(2026, 7, 26), d(2026, 9, 5)},
	}
	for _, tc := range cases {
		t.Run(tc.name, func(t *testing.T) {
			g := MonthGrid(tc.year, tc.month, tc.ws, time.UTC)
			if len(g) != tc.rows {
				t.Fatalf("rows = %d, want %d", len(g), tc.rows)
			}
			for _, row := range g {
				if len(row) != 7 {
					t.Fatalf("row has %d days", len(row))
				}
			}
			if !g[0][0].Equal(tc.first) || !g[len(g)-1][6].Equal(tc.last) {
				t.Errorf("grid spans %v..%v, want %v..%v", g[0][0], g[len(g)-1][6], tc.first, tc.last)
			}
		})
	}
}

func TestAddMonthsClamped(t *testing.T) {
	cases := []struct {
		in   time.Time
		n    int
		want time.Time
	}{
		{d(2026, 1, 31), 1, d(2026, 2, 28)},
		{d(2028, 1, 31), 1, d(2028, 2, 29)},
		{d(2026, 3, 31), -1, d(2026, 2, 28)},
		{d(2026, 12, 15), 1, d(2027, 1, 15)},
		{d(2026, 10, 1), -12, d(2025, 10, 1)},
	}
	for _, tc := range cases {
		if got := AddMonthsClamped(tc.in, tc.n); !got.Equal(tc.want) {
			t.Errorf("AddMonthsClamped(%v, %d) = %v, want %v", tc.in, tc.n, got, tc.want)
		}
	}
}
