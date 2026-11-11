package cmd

import (
	"fmt"
	"os"
	"time"

	"github.com/spf13/cobra"

	"maestro-cli/internal/maestroapi"
)

var (
	exportSince string
	exportUntil string
	exportAll   bool
	exportOut   string
)

var exportCmd = &cobra.Command{
	Use:   "export",
	Short: "Write what happened to your lines over a period, as CSV",
	Long: `Writes a row for each line of the runs queued over a period: its run and
line, whose it is, its cores and memory, the machine it ran on, its status,
the scheduler settings it was chosen under, and when it was submitted,
staged, assigned, started and finished, and the group its user was in when
it was staged. It is what ` + "`sim evaluate`" + `, in maestro-orq, measures
waits, slowdowns, utilization, fairness and the objective Z from, the same
as for a simulation.

Administrators can export everyone's lines with --all.`,
	Example: `  maestro export --since 2026-11-01 --until 2026-11-08 > lines.csv
  maestro export --since 2026-11-01 --all -o lines.csv`,
	Args:              cobra.NoArgs,
	ValidArgsFunction: cobra.NoFileCompletions,
	RunE: func(cmd *cobra.Command, args []string) error {
		since := exportSince
		if since == "" {
			since = time.Now().AddDate(0, 0, -7).Format(time.DateOnly)
		}
		out := os.Stdout
		if exportOut != "" && exportOut != "-" {
			f, err := os.Create(exportOut)
			if err != nil {
				return err
			}
			defer f.Close()
			out = f
		}
		if err := maestroapi.ExportLines(cmd.Context(), maestroKey, since, exportUntil, exportAll, out); err != nil {
			return err
		}
		if out != os.Stdout {
			fmt.Fprintf(os.Stderr, "%s the lines into %s\n", successStyle.Render("Exported"), exportOut)
		}
		return nil
	},
}

func init() {
	exportCmd.Flags().StringVar(&exportSince, "since", "", "from this day (2026-11-01) or RFC 3339 time on; a week ago when left out")
	exportCmd.Flags().StringVar(&exportUntil, "until", "", "until before this day or time; now when left out")
	exportCmd.Flags().BoolVar(&exportAll, "all", false, "everyone's lines (administrators)")
	exportCmd.Flags().StringVarP(&exportOut, "output", "o", "", "the file to write; the standard output when left out")
	rootCmd.AddCommand(exportCmd)
}
