package main

import (
	"context"
	"os"

	"charm.land/fang/v2"

	"github.com/JacobAtchley/sundial/internal/cli"
	"github.com/JacobAtchley/sundial/internal/tui"
)

// version is set at build time via -ldflags "-X main.version=...".
var version = "dev"

func main() {
	opts := cli.DefaultOptions()
	opts.LaunchTUI = func(env cli.Env) error {
		return tui.Run(tui.Options{
			Config:     env.Config,
			ConfigPath: env.ConfigPath,
			Store:      env.Store,
			Source:     env.Source,
			Now:        env.Now,
			Log:        env.Log,
			Denied:     env.Denied,
			Getenv:     os.Getenv,
		})
	}
	root := cli.NewRootCommand(opts)
	if err := fang.Execute(context.Background(), root, fang.WithVersion(version)); err != nil {
		os.Exit(cli.ExitCode(err))
	}
}
