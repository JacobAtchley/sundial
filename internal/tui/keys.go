package tui

import (
	"strings"

	bkey "charm.land/bubbles/v2/key"
	tea "charm.land/bubbletea/v2"

	"github.com/JacobAtchley/sundial/internal/config"
	"github.com/JacobAtchley/sundial/internal/tui/shared"
)

type keyMap map[string]bkey.Binding

func newKeyMap(cfg config.Config) keyMap {
	km := keyMap{}
	for action := range config.DefaultKeys {
		km[action] = bkey.NewBinding(bkey.WithKeys(cfg.KeysFor(action)...))
	}
	return km
}

func (k keyMap) is(msg tea.KeyPressMsg, action string) bool { return bkey.Matches(msg, k[action]) }

func (k keyMap) label(action string) string { return strings.Join(k[action].Keys(), "/") }

// navActions maps bindable actions to navigation intents, in a fixed order.
var navActions = []struct {
	action string
	nav    shared.Nav
}{
	{"left", shared.NavLeft}, {"right", shared.NavRight},
	{"up", shared.NavUp}, {"down", shared.NavDown},
	{"prev", shared.NavPrev}, {"next", shared.NavNext},
}
