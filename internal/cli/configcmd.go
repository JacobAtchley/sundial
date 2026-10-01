package cli

import (
	"fmt"
	"os"

	"github.com/spf13/cobra"

	"github.com/JacobAtchley/sundial/internal/config"
)

func newConfigCommand(opts Options, f *flags) *cobra.Command {
	cmd := &cobra.Command{Use: "config", Short: "Manage the config file"}
	cmd.AddCommand(
		&cobra.Command{
			Use:   "path",
			Short: "Print the config file path",
			Args:  cobra.NoArgs,
			RunE: func(cmd *cobra.Command, _ []string) error {
				p, err := f.resolveConfigPath()
				if err != nil {
					return err
				}
				_, err = fmt.Fprintln(cmd.OutOrStdout(), p)
				return err
			},
		},
		&cobra.Command{
			Use:   "init",
			Short: "Write a commented starter config (never overwrites)",
			Args:  cobra.NoArgs,
			RunE: func(cmd *cobra.Command, _ []string) error {
				p, err := f.resolveConfigPath()
				if err != nil {
					return err
				}
				if err := config.Init(p); err != nil {
					return err
				}
				_, err = fmt.Fprintln(cmd.OutOrStdout(), "Wrote", p)
				return err
			},
		},
		&cobra.Command{
			Use:   "edit",
			Short: "Open the config file in $EDITOR (creating it if needed)",
			Args:  cobra.NoArgs,
			RunE: func(cmd *cobra.Command, _ []string) error {
				p, err := f.resolveConfigPath()
				if err != nil {
					return err
				}
				if _, err := os.Stat(p); os.IsNotExist(err) {
					if err := config.Init(p); err != nil {
						return err
					}
				}
				c := config.EditorCommand(opts.Getenv, p)
				c.Stdin, c.Stdout, c.Stderr = os.Stdin, os.Stdout, os.Stderr
				return c.Run()
			},
		},
	)
	return cmd
}
