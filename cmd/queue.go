package cmd

import (
	"fmt"
	"io"
	"os"
	"slices"
	"strings"
	"text/tabwriter"

	"github.com/spf13/cobra"

	"maestro-cli/internal/maestroapi"
)

var queueAsJSON bool

var queueCmd = &cobra.Command{
	Use:   "queue",
	Short: "Show where lines wait, and your share of the machines",
	Long: `Shows how many lines wait and run, everyone's together and yours:

  queued    in their user's queue
  heads     how many of those are their user's head, in the dispatcher queue
  assigned  given to a machine, waiting for room on it
  starting, running

It also shows your share of the machines, as the scheduler weighs it when
it chooses whose line goes next.

A user's head is the first of their queued lines. Users are in groups, and
the heads of a group's users make its dispatcher queue. Groups take turns
at the machines, and in a group's turn one of its heads goes to a machine.
You see your group. Administrators see every user and every group, with
the number an administrator weighs a user by.

When an administrator has set the most cores a user's lines may hold, it
says so when your next line waits for that.

It ends with how the scheduler is set up. ` + "`maestro settings`" + ` shows all of it.`,
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

func writeQueue(out io.Writer, q *maestroapi.Queue) error {
	// A server from before the three queues counts staged lines, between
	// queued and assigned, and has no heads. It is shown as it answers.
	_, staged := q.Totals["staged"]
	columns := []string{"queued", "heads", "assigned", "starting", "running"}
	if staged {
		columns = []string{"queued", "staged", "assigned", "starting", "running"}
	}
	w := tabwriter.NewWriter(out, 0, 0, 2, ' ', tabwriter.AlignRight)
	groupName := map[int64]string{}
	for _, g := range q.Groups {
		groupName[g.ID] = g.Name
	}
	fmt.Fprintln(w, "USER\tGROUP\t"+strings.ToUpper(strings.Join(columns, "\t"))+"\tSHARE\t")
	row := func(name, group string, lines map[string]int64, heads, share string) {
		fmt.Fprint(w, name+"\t"+group+"\t")
		for _, column := range columns {
			if column == "heads" {
				fmt.Fprint(w, heads+"\t")
			} else {
				fmt.Fprintf(w, "%d\t", lines[column])
			}
		}
		fmt.Fprintln(w, share+"\t")
	}
	for _, u := range q.Users {
		name := u.Name
		if name == "" {
			name = u.Email
		}
		row(fmt.Sprintf("%s #%d", name, u.ID), groupName[u.Group], u.Lines, fmt.Sprint(u.Heads), fmt.Sprintf("%.0f%%", 100*u.Share))
	}
	row("everyone", "", q.Totals, "", "")
	if err := w.Flush(); err != nil {
		return err
	}
	if len(q.Groups) > 0 {
		fmt.Fprintln(out)
		w = tabwriter.NewWriter(out, 0, 0, 2, ' ', tabwriter.AlignRight)
		if staged {
			fmt.Fprintln(w, "GROUP\tSTAGED\t")
		} else {
			fmt.Fprintln(w, "GROUP\tQUEUED\tHEADS\tASSIGNED\t")
		}
		for _, g := range q.Groups {
			if staged {
				fmt.Fprintf(w, "%s\t%d/%d\t\n", g.Name, g.Staged, g.Capacity)
			} else {
				fmt.Fprintf(w, "%s\t%d\t%d\t%d\t\n", g.Name, g.Queued, g.Heads, g.Assigned)
			}
		}
		if err := w.Flush(); err != nil {
			return err
		}
	}
	if q.AtCeiling {
		fmt.Fprintf(out, "\nYour next line waits until some of yours end. With it, your lines assigned, starting and running would take more than %d cores, the most a user may hold.\n", q.UserMaxCPUs)
	}
	_, err := fmt.Fprintf(out, "\n%s\n", schedulerSummary(q.Scheduler))
	return err
}

// schedulerSummary says in a few sentences how lines are chosen: with the
// three queues' policies, or with the four of a server from before them.
func schedulerSummary(s maestroapi.Scheduler) string {
	order := fmt.Sprintf("Each user's head is chosen by %s. In a group's turn, %s picks one of its heads and gives it to a machine drawn among those with room. Lines start by %s on each machine.",
		s.User, s.Dispatcher, s.Machine)
	policies := []string{s.User, s.Dispatcher, s.Machine}
	if s.Dispatcher == "" && s.Select != "" {
		order = fmt.Sprintf("Lines are chosen by %s among each group's users (%s within each), given in turn between groups by %s to a machine with room, drawn, and start by %s on each machine.",
			s.Select, s.User, s.Allocate, s.Machine)
		policies = []string{s.User, s.Select, s.Allocate, s.Machine}
	}
	if slices.Contains(policies, "pq") {
		order += fmt.Sprintf(" The score weighs share (%s) by %g and waiting by %g.", s.Share, s.ShareWeight, s.AgingWeight)
	}
	return order
}
