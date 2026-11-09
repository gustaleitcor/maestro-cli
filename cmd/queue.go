package cmd

import (
	"fmt"
	"io"
	"os"
	"strings"
	"text/tabwriter"

	"github.com/spf13/cobra"

	"maestro-cli/internal/maestroapi"
)

var queueAsJSON bool

var queueCmd = &cobra.Command{
	Use:   "queue",
	Short: "Show where lines wait, and your share of the machines",
	Long: `Shows how many lines wait at each level, everyone's together and yours:

  queued    among their user's lines
  staged    chosen from them, waiting for a machine
  assigned  given to a machine, waiting for room on it
  starting, running

and your share of the machines, as the scheduler weighs it when it chooses
whose line goes next. Administrators see every user, with the number an
administrator gives a user a weight by. It ends with how the
scheduler is set up; ` + "`maestro settings`" + ` says it all.`,
	Args:              cobra.NoArgs,
	ValidArgsFunction: cobra.NoFileCompletions,
	RunE: func(cmd *cobra.Command, args []string) error {
		q, err := maestroapi.GetQueue(cmd.Context(), maestroKey)
		if err != nil {
			return err
		}
		if queueAsJSON {
			return writeJSON(os.Stdout, q)
		}
		return writeQueue(os.Stdout, q)
	},
}

func init() {
	queueCmd.Flags().BoolVar(&queueAsJSON, "json", false, "print the queue as JSON")
	rootCmd.AddCommand(queueCmd)
}

var queueLevels = []string{"queued", "staged", "assigned", "starting", "running"}

func writeQueue(out io.Writer, q *maestroapi.Queue) error {
	w := tabwriter.NewWriter(out, 0, 0, 2, ' ', tabwriter.AlignRight)
	fmt.Fprintln(w, "USER\t"+strings.ToUpper(strings.Join(queueLevels, "\t"))+"\tSHARE\t")
	row := func(name string, lines map[string]int64, share string) {
		fmt.Fprint(w, name+"\t")
		for _, level := range queueLevels {
			fmt.Fprintf(w, "%d\t", lines[level])
		}
		fmt.Fprintln(w, share+"\t")
	}
	for _, u := range q.Users {
		name := u.Name
		if name == "" {
			name = u.Email
		}
		row(fmt.Sprintf("%s #%d", name, u.ID), u.Lines, fmt.Sprintf("%.0f%%", 100*u.Share))
	}
	row("everyone", q.Totals, "")
	if err := w.Flush(); err != nil {
		return err
	}
	_, err := fmt.Fprintf(out, "\n%s\n", schedulerSummary(q.Scheduler))
	return err
}

// schedulerSummary says in a line how lines are chosen.
func schedulerSummary(s maestroapi.Scheduler) string {
	order := fmt.Sprintf("Lines are chosen by %s among users (%s within each), placed by %s, and start by %s on each machine",
		s.Select, s.User, s.Place, s.Machine)
	if s.Select == "pq" || s.Allocate == "pq" || s.User == "pq" || s.Machine == "pq" {
		order += fmt.Sprintf("; the score weighs share (%s) by %g and waiting by %g", s.Share, s.ShareWeight, s.AgingWeight)
	}
	return order + "."
}
