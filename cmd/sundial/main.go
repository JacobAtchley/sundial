package main

import (
	"context"
	"os"

	"charm.land/fang/v2"

	"github.com/JacobAtchley/sundial/internal/cli"
)

// version is set at build time via -ldflags "-X main.version=...".
var version = "dev"

func main() {
	root := cli.NewRootCommand(cli.Options{})
	if err := fang.Execute(context.Background(), root, fang.WithVersion(version)); err != nil {
		os.Exit(1)
	}
}
