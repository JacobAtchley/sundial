// Package cli wires sundial's cobra commands.
package cli

import "github.com/spf13/cobra"

// NewRootCommand builds the sundial command tree.
func NewRootCommand(opts Options) *cobra.Command {
	f := &flags{}
	root := &cobra.Command{
		Use:           "sundial",
		Short:         "A terminal calendar for macOS",
		Args:          cobra.NoArgs,
		SilenceUsage:  true,
		SilenceErrors: true,
		RunE: func(cmd *cobra.Command, _ []string) error {
			if opts.LaunchTUI == nil {
				return cmd.Help()
			}
			env, cleanup, err := loadEnv(cmd.Context(), opts, f, true)
			if err != nil {
				return err
			}
			defer cleanup()
			return opts.LaunchTUI(env)
		},
	}
	f.register(root)
	root.AddCommand(
		newTodayCommand(opts, f),
		newAgendaCommand(opts, f),
		newCalendarsCommand(opts, f),
		newConfigCommand(opts, f),
	)
	return root
}
