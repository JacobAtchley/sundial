package datemath

import (
	"testing"
	"time"
)

func TestParseDateExpr(t *testing.T) {
	now := time.Date(2026, 10, 1, 8, 0, 0, 0, time.UTC) // Thursday
	cases := []struct {
		in   string
		want time.Time
		ok   bool
	}{
		{"today", d(2026, 10, 1), true},
		{"  Tomorrow ", d(2026, 10, 2), true},
		{"yesterday", d(2026, 9, 30), true},
		{"+3d", d(2026, 10, 4), true},
		{"-2d", d(2026, 9, 29), true},
		{"+3w", d(2026, 10, 22), true},
		{"+1m", d(2026, 11, 1), true},
		{"2026-10-15", d(2026, 10, 15), true},
		{"10/15", d(2026, 10, 15), true},
		{"2/30", time.Time{}, false},
		{"thu", d(2026, 10, 1), true},
		{"next thu", d(2026, 10, 8), true},
		{"fri", d(2026, 10, 2), true},
		{"next friday", d(2026, 10, 2), true},
		{"mon", d(2026, 10, 5), true},
		{"standup", time.Time{}, false},
		{"", time.Time{}, false},
		{"2026-13-01", time.Time{}, false},
	}
	for _, tc := range cases {
		got, ok := ParseDateExpr(tc.in, now)
		if ok != tc.ok || (ok && !got.Equal(tc.want)) {
			t.Errorf("ParseDateExpr(%q) = %v, %v; want %v, %v", tc.in, got, ok, tc.want, tc.ok)
		}
	}
}
