// Package config loads sundial's TOML settings.
package config

import "time"

// Config is the full settings file. Zero values never reach callers: Load
// starts from Default and the file only overrides keys it sets.
type Config struct {
	DefaultView string              `toml:"default_view"`
	WeekStart   string              `toml:"week_start"`
	TimeFormat  string              `toml:"time_format"`
	DayStart    int                 `toml:"day_start"`
	Calendars   Calendars           `toml:"calendars"`
	Agenda      Agenda              `toml:"agenda"`
	Theme       Theme               `toml:"theme"`
	Keys        map[string][]string `toml:"keys"`
}

type Calendars struct {
	Hidden []string `toml:"hidden"`
}

type Agenda struct {
	Days int `toml:"days"`
}

// Theme holds optional hex color overrides. Empty means "use default".
type Theme struct {
	Pastelize bool   `toml:"pastelize"`
	Header    string `toml:"header"`
	Day       string `toml:"day"`
	Today     string `toml:"today"`
	Selection string `toml:"selection"`
	Accent    string `toml:"accent"`
	Muted     string `toml:"muted"`
	Warning   string `toml:"warning"`
}

type ColorOverride struct {
	Name string
	Hex  string
}

// Overrides lists every palette role in a fixed order.
func (t Theme) Overrides() []ColorOverride {
	return []ColorOverride{
		{"header", t.Header}, {"day", t.Day}, {"today", t.Today}, {"selection", t.Selection},
		{"accent", t.Accent}, {"muted", t.Muted}, {"warning", t.Warning},
	}
}

// DefaultKeys maps every bindable action to its default keys. Its key set
// is also the list of valid [keys] actions.
var DefaultKeys = map[string][]string{
	"month":    {"m"},
	"week":     {"w"},
	"agenda":   {"a"},
	"today":    {"t"},
	"prev":     {"[", "p"},
	"next":     {"]", "n"},
	"left":     {"h", "left"},
	"right":    {"l", "right"},
	"up":       {"k", "up"},
	"down":     {"j", "down"},
	"open":     {"enter"},
	"close":    {"esc"},
	"palette":  {"ctrl+k", ":"},
	"help":     {"?"},
	"quit":     {"q", "ctrl+c"},
	"filter":   {"/"},
	"open_url": {"o"},
}

func Default() Config {
	return Config{
		DefaultView: "agenda",
		WeekStart:   "sunday",
		TimeFormat:  "12h",
		DayStart:    8,
		Calendars:   Calendars{Hidden: []string{}},
		Agenda:      Agenda{Days: 14},
		Theme:       Theme{Pastelize: true},
		Keys:        map[string][]string{},
	}
}

func (c Config) WeekStartDay() time.Weekday {
	if c.WeekStart == "monday" {
		return time.Monday
	}
	return time.Sunday
}

func (c Config) Use24h() bool { return c.TimeFormat == "24h" }

// KeysFor returns the configured keys for action, or its defaults.
func (c Config) KeysFor(action string) []string {
	if k, ok := c.Keys[action]; ok {
		return k
	}
	return DefaultKeys[action]
}
