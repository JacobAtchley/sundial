package theme

import (
	"testing"

	"github.com/JacobAtchley/sundial/internal/config"
)

func TestParseHex(t *testing.T) {
	if c, ok := ParseHex("#C6A0F6"); !ok || Hex(c) != "#C6A0F6" {
		t.Errorf("round trip failed: %v %v", c, ok)
	}
	for _, bad := range []string{"", "C6A0F6", "#C6A0F", "#GGGGGG", "lavender"} {
		if _, ok := ParseHex(bad); ok {
			t.Errorf("ParseHex(%q) should fail", bad)
		}
	}
}

func TestPastelize(t *testing.T) {
	cases := map[string]string{"#000000": "#595959", "#FF0000": "#FF5959", "#FFFFFF": "#FFFFFF"}
	for in, want := range cases {
		c, _ := ParseHex(in)
		if got := Hex(Pastelize(c, 0.35)); got != want {
			t.Errorf("Pastelize(%s) = %s, want %s", in, got, want)
		}
	}
}

func TestDefaultPaletteDarkAndLight(t *testing.T) {
	if got := Hex(DefaultPalette(true).Header); got != "#C6A0F6" {
		t.Errorf("dark header = %s", got)
	}
	if got := Hex(DefaultPalette(false).Header); got != "#8839EF" {
		t.Errorf("light header = %s", got)
	}
}

func TestOverridesApply(t *testing.T) {
	th := New(config.Theme{Pastelize: true, Header: "#112233", Warning: "#445566"}, true)
	if Hex(th.P.Header) != "#112233" || Hex(th.P.Warning) != "#445566" {
		t.Errorf("overrides not applied: %s %s", Hex(th.P.Header), Hex(th.P.Warning))
	}
	if Hex(th.P.Day) != "#91D7E3" {
		t.Error("non-overridden role changed")
	}
}

func TestCalendarColor(t *testing.T) {
	soft := New(config.Theme{Pastelize: true}, true)
	if got := Hex(soft.CalendarColor("#FF0000")); got != "#FF5959" {
		t.Errorf("pastelized = %s", got)
	}
	exact := New(config.Theme{Pastelize: false}, true)
	if got := Hex(exact.CalendarColor("#FF0000")); got != "#FF0000" {
		t.Errorf("exact = %s", got)
	}
	if got := Hex(exact.CalendarColor("not-a-color")); got != Hex(exact.P.Muted) {
		t.Errorf("invalid color should fall back to muted, got %s", got)
	}
}
