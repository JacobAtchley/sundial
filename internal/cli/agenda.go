package cli

import (
	"encoding/json"
	"fmt"
	"io"
	"strings"
	"time"

	"charm.land/lipgloss/v2"
	"github.com/spf13/cobra"

	"github.com/JacobAtchley/sundial/internal/calendar"
	"github.com/JacobAtchley/sundial/internal/datemath"
	"github.com/JacobAtchley/sundial/internal/tui/shared"
	"github.com/JacobAtchley/sundial/internal/tui/theme"
)

func newTodayCommand(opts Options, f *flags) *cobra.Command {
	return newDayCommand(opts, f, "today", "Print today's agenda", 0)
}

func newTomorrowCommand(opts Options, f *flags) *cobra.Command {
	return newDayCommand(opts, f, "tomorrow", "Print tomorrow's agenda", 1)
}

// newDayCommand prints a single day's agenda, offset days from today.
func newDayCommand(opts Options, f *flags, use, short string, offset int) *cobra.Command {
	var asJSON bool
	cmd := &cobra.Command{
		Use:   use,
		Short: short,
		Args:  cobra.NoArgs,
		RunE: func(cmd *cobra.Command, _ []string) error {
			start := calendar.StartOfDay(opts.Now()).AddDate(0, 0, offset)
			return printAgenda(cmd, opts, f, start, 1, asJSON)
		},
	}
	cmd.Flags().BoolVar(&asJSON, "json", false, "print JSON")
	return cmd
}

func newAgendaCommand(opts Options, f *flags) *cobra.Command {
	var (
		days   int
		from   string
		asJSON bool
	)
	cmd := &cobra.Command{
		Use:   "agenda",
		Short: "Print upcoming events",
		Args:  cobra.NoArgs,
		RunE: func(cmd *cobra.Command, _ []string) error {
			start := calendar.StartOfDay(opts.Now())
			if from != "" {
				t, ok := datemath.ParseDateExpr(from, opts.Now())
				if !ok {
					return fmt.Errorf("can't parse --from %q (try 2026-10-15, tomorrow, +3d, or fri)", from)
				}
				start = t
			}
			if days < 0 || days > 365 {
				return fmt.Errorf("--days must be between 1 and 365")
			}
			return printAgenda(cmd, opts, f, start, days, asJSON)
		},
	}
	cmd.Flags().IntVar(&days, "days", 0, "number of days (default agenda.days from config)")
	cmd.Flags().StringVar(&from, "from", "", "start date (default today)")
	cmd.Flags().BoolVar(&asJSON, "json", false, "print JSON")
	return cmd
}

type jsonEvent struct {
	calendar.Event
	Calendar string `json:"calendar"`
}

func printAgenda(cmd *cobra.Command, opts Options, f *flags, start time.Time, days int, asJSON bool) error {
	env, cleanup, err := loadEnv(cmd.Context(), opts, f, false)
	if err != nil {
		return err
	}
	defer cleanup()
	if days == 0 {
		days = env.Config.Agenda.Days
	}
	r := calendar.Range{Start: start, End: start.AddDate(0, 0, days)}
	if err := env.Store.Load(cmd.Context(), r); err != nil {
		return err
	}
	evs := env.Store.EventsIn(r)
	env.Log.Info("agenda loaded", "days", days, "events", len(evs))
	out := cmd.OutOrStdout()
	if asJSON {
		rows := make([]jsonEvent, len(evs))
		for i, e := range evs {
			c, _ := env.Store.Calendar(e.CalendarID)
			rows[i] = jsonEvent{Event: e, Calendar: c.Title}
		}
		enc := json.NewEncoder(out)
		enc.SetIndent("", "  ")
		return enc.Encode(rows)
	}
	if len(evs) == 0 {
		noun := "days"
		if days == 1 {
			noun = "day"
		}
		_, err := fmt.Fprintf(out, "No events in the next %d %s.\n", days, noun)
		return err
	}
	th := theme.New(env.Config.Theme, true)
	return writeAgendaText(out, env, th, r)
}

func writeAgendaText(w io.Writer, env Env, th theme.Theme, r calendar.Range) error {
	var b strings.Builder
	for _, day := range r.Days() {
		evs := env.Store.EventsOn(day)
		if len(evs) == 0 {
			continue
		}
		b.WriteString(th.DayHeader.Render(day.Format("Mon Jan 2")) + "\n")
		for _, e := range evs {
			c, _ := env.Store.Calendar(e.CalendarID)
			span := shared.PadRight(shared.FormatSpan(e, env.Config.Use24h()), 16)
			line := "  " + span + " " + th.Dot(c.Color) + " " + shared.OneLine(e.Title)
			if e.Location != "" {
				line += th.Muted.Render(" · " + shared.OneLine(e.Location))
			}
			b.WriteString(line + "\n")
		}
	}
	// lipgloss.Fprint strips ANSI when w is not a terminal.
	_, err := lipgloss.Fprint(w, b.String())
	return err
}
