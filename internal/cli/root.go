// Package cli wires sundial's cobra commands.
package cli

import "github.com/spf13/cobra"

// Options carries dependencies the commands need. Tests replace them.
type Options struct{}

// NewRootCommand builds the sundial command tree.
func NewRootCommand(_ Options) *cobra.Command {
	return &cobra.Command{
		Use:           "sundial",
		Short:         "A terminal calendar for macOS",
		SilenceUsage:  true,
		SilenceErrors: true,
	}
}
