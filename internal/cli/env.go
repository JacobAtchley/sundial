package cli

import (
	"context"
	"errors"
	"time"

	"github.com/spf13/cobra"

	"github.com/JacobAtchley/sundial/internal/calendar"
	"github.com/JacobAtchley/sundial/internal/config"
	"github.com/JacobAtchley/sundial/internal/debuglog"
	"github.com/JacobAtchley/sundial/internal/store"
)

// Env is the loaded runtime state shared by commands and the TUI.
type Env struct {
	Config     config.Config
	ConfigPath string
	Source     calendar.Source
	Store      *store.Store
	Log        debuglog.Logger
	Now        func() time.Time
	// Denied is true when calendar access was refused. Source and Store are
	// nil in that case; only the TUI uses this (to show the help screen).
	Denied bool
}

type flags struct {
	configPath   string
	debug        bool
	debugVerbose bool
	source       string
}

func (f *flags) register(cmd *cobra.Command) {
	pf := cmd.PersistentFlags()
	pf.StringVar(&f.configPath, "config", "", "config file path (default $XDG_CONFIG_HOME/sundial/config.toml)")
	pf.BoolVar(&f.debug, "debug", false, "write a debug log (event content redacted)")
	pf.BoolVar(&f.debugVerbose, "debug-verbose", false, "write a debug log including event content")
	pf.StringVar(&f.source, "source", "", "calendar source: eventkit or fake")
	_ = pf.MarkHidden("source")
}

func (f *flags) resolveConfigPath() (string, error) {
	if f.configPath != "" {
		return f.configPath, nil
	}
	return config.Path()
}

// loadEnv loads config, opens the debug log and the source. With
// allowDenied, access denial yields Env{Denied: true} instead of an error.
func loadEnv(ctx context.Context, opts Options, f *flags, allowDenied bool) (Env, func(), error) {
	noop := func() {}
	path, err := f.resolveConfigPath()
	if err != nil {
		return Env{}, noop, err
	}
	cfg, err := config.Load(path)
	if err != nil {
		return Env{}, noop, err
	}
	env := Env{Config: cfg, ConfigPath: path, Now: opts.Now, Log: debuglog.Discard()}
	cleanup := noop
	if f.debug || f.debugVerbose {
		l, closeFn, err := debuglog.Open(f.debugVerbose)
		if err != nil {
			return Env{}, noop, err
		}
		env.Log, cleanup = l, func() { _ = closeFn() }
	}
	name := f.source
	if name == "" {
		name = opts.Getenv("SUNDIAL_SOURCE")
	}
	src, err := opts.OpenSource(name)
	if errors.Is(err, calendar.ErrAccessDenied) {
		env.Log.Info("calendar access denied")
		if allowDenied {
			env.Denied = true
			return env, cleanup, nil
		}
		cleanup()
		return Env{}, noop, &ExitError{Code: 2, Err: errDeniedHelp}
	}
	if err != nil {
		cleanup()
		return Env{}, noop, err
	}
	env.Source = src
	env.Store = store.New(src)
	env.Store.SetHidden(cfg.Calendars.Hidden)
	if err := env.Store.LoadCalendars(ctx); err != nil {
		cleanup()
		return Env{}, noop, err
	}
	env.Log.Info("calendars loaded", "count", len(env.Store.Calendars()))
	return env, cleanup, nil
}
