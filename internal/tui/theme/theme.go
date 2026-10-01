// Package theme builds sundial's pastel Lip Gloss styles.
package theme

import (
	"fmt"
	"image/color"
	"math"
	"regexp"
	"strconv"

	"charm.land/lipgloss/v2"

	"github.com/JacobAtchley/sundial/internal/config"
)

type Palette struct {
	Header    color.Color
	Day       color.Color
	Today     color.Color
	Selection color.Color
	Accent    color.Color
	Muted     color.Color
	Warning   color.Color
	Ink       color.Color // text drawn on pastel backgrounds
}

func DefaultPalette(isDark bool) Palette {
	ld := lipgloss.LightDark(isDark)
	c := lipgloss.Color
	return Palette{
		Header:    ld(c("#8839EF"), c("#C6A0F6")),
		Day:       ld(c("#04A5E5"), c("#91D7E3")),
		Today:     ld(c("#FE640B"), c("#F5A97F")),
		Selection: ld(c("#EA76CB"), c("#F5BDE6")),
		Accent:    ld(c("#40A02B"), c("#A6DA95")),
		Muted:     ld(c("#7C7F93"), c("#8087A2")),
		Warning:   ld(c("#D20F39"), c("#ED8796")),
		Ink:       c("#1E1E2E"),
	}
}

type Theme struct {
	P         Palette
	Pastelize bool

	Title     lipgloss.Style
	DayHeader lipgloss.Style
	Today     lipgloss.Style
	Selected  lipgloss.Style
	Accent    lipgloss.Style
	Muted     lipgloss.Style
	Warning   lipgloss.Style
	Bold      lipgloss.Style
	Border    lipgloss.Style
}

func New(cfg config.Theme, isDark bool) Theme {
	p := DefaultPalette(isDark)
	targets := map[string]*color.Color{
		"header": &p.Header, "day": &p.Day, "today": &p.Today, "selection": &p.Selection,
		"accent": &p.Accent, "muted": &p.Muted, "warning": &p.Warning,
	}
	for _, o := range cfg.Overrides() {
		if c, ok := ParseHex(o.Hex); ok {
			*targets[o.Name] = c
		}
	}
	s := lipgloss.NewStyle
	return Theme{
		P:         p,
		Pastelize: cfg.Pastelize,
		Title:     s().Foreground(p.Header).Bold(true),
		DayHeader: s().Foreground(p.Day).Bold(true),
		Today:     s().Foreground(p.Today).Bold(true),
		Selected:  s().Background(p.Selection).Foreground(p.Ink).Bold(true),
		Accent:    s().Foreground(p.Accent),
		Muted:     s().Foreground(p.Muted),
		Warning:   s().Foreground(p.Warning).Bold(true),
		Bold:      s().Bold(true),
		Border:    s().Foreground(p.Muted),
	}
}

// CalendarColor parses a calendar's hex color, softening it when pastelize
// is on. Invalid colors fall back to the muted color.
func (t Theme) CalendarColor(hex string) color.Color {
	c, ok := ParseHex(hex)
	if !ok {
		return t.P.Muted
	}
	if t.Pastelize {
		return Pastelize(c, 0.35)
	}
	return c
}

// Chip styles an event block: calendar color background, dark ink text.
func (t Theme) Chip(hex string) lipgloss.Style {
	return lipgloss.NewStyle().Background(t.CalendarColor(hex)).Foreground(t.P.Ink)
}

// Dot renders a colored bullet for a calendar.
func (t Theme) Dot(hex string) string {
	return lipgloss.NewStyle().Foreground(t.CalendarColor(hex)).Render("●")
}

// Pastelize blends c toward white by amount (0..1).
func Pastelize(c color.Color, amount float64) color.Color {
	r, g, b, _ := c.RGBA()
	mix := func(v uint32) uint8 {
		f := float64(v >> 8)
		return uint8(math.Round(f + (255-f)*amount))
	}
	return color.RGBA{R: mix(r), G: mix(g), B: mix(b), A: 255}
}

var hexRe = regexp.MustCompile(`^#[0-9A-Fa-f]{6}$`)

func ParseHex(s string) (color.Color, bool) {
	if !hexRe.MatchString(s) {
		return nil, false
	}
	v, err := strconv.ParseUint(s[1:], 16, 32)
	if err != nil {
		return nil, false
	}
	return color.RGBA{R: uint8(v >> 16), G: uint8(v >> 8), B: uint8(v), A: 255}, true
}

func Hex(c color.Color) string {
	r, g, b, _ := c.RGBA()
	return fmt.Sprintf("#%02X%02X%02X", r>>8, g>>8, b>>8)
}
