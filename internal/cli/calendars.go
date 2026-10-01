package cli

import (
	"encoding/json"
	"fmt"
	"strings"

	"charm.land/lipgloss/v2"
	"github.com/spf13/cobra"

	"github.com/JacobAtchley/sundial/internal/tui/theme"
)

func newCalendarsCommand(opts Options, f *flags) *cobra.Command {
	var asJSON bool
	cmd := &cobra.Command{
		Use:   "calendars",
		Short: "List calendars (titles and IDs for [calendars] hidden)",
		Args:  cobra.NoArgs,
		RunE: func(cmd *cobra.Command, _ []string) error {
			env, cleanup, err := loadEnv(cmd.Context(), opts, f, false)
			if err != nil {
				return err
			}
			defer cleanup()
			cals := env.Store.Calendars()
			out := cmd.OutOrStdout()
			if asJSON {
				enc := json.NewEncoder(out)
				enc.SetIndent("", "  ")
				return enc.Encode(cals)
			}
			th := theme.New(env.Config.Theme, true)
			var b strings.Builder
			for _, c := range cals {
				hidden := ""
				if env.Store.IsHidden(c.ID) {
					hidden = th.Muted.Render(" (hidden)")
				}
				fmt.Fprintf(&b, "%s %s%s %s\n", th.Dot(c.Color), c.Title, hidden, th.Muted.Render(c.Source+" · "+c.ID))
			}
			_, err = lipgloss.Fprint(out, b.String())
			return err
		},
	}
	cmd.Flags().BoolVar(&asJSON, "json", false, "print JSON")
	return cmd
}
