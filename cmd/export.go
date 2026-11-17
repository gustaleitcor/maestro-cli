package cmd

import (
	"fmt"
	"os"
	"strconv"
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
the scheduler settings it was assigned under, when it was submitted,
assigned, started and finished, and the group its user was in when it was
assigned. An administrator can measure it with orq's ` + "`sim evaluate`" + `,
as for a simulation: waits, slowdowns, utilization, fairness and the
objective Z.

Administrators can export everyone's lines with --all.`,
	Example: `  maestro export --since 2026-11-01 --until 2026-11-08 > lines.csv
  maestro export --since 2026-11-01 --all -o lines.csv
  maestro export --since 30d -o month.csv`,
	Args:              cobra.NoArgs,
	ValidArgsFunction: cobra.NoFileCompletions,
	RunE: func(cmd *cobra.Command, args []string) error {
		now := time.Now()
		since, until := relativeTime(exportSince, now), relativeTime(exportUntil, now)
		if since == "" {
			since = now.AddDate(0, 0, -7).Format(time.DateOnly)
		}
		export := func(f *os.File) error {
			return maestroapi.ExportLines(cmd.Context(), maestroKey, since, until, exportAll, f)
		}
		if exportOut == "" || exportOut == "-" {
			return export(os.Stdout)
		}
		// A file is only left when the export worked.
		if err := save(exportOut, export); err != nil {
			return err
		}
		fmt.Fprintf(os.Stderr, "%s the lines into %s\n", successStyle.Render("Exported"), exportOut)
		return nil
	},
}

func init() {
	exportCmd.Flags().StringVar(&exportSince, "since", "", "start at this `time`: a day (2026-11-01), an RFC 3339 time, or an age such as 12h, 7d or 2w (default: a week ago)")
	exportCmd.Flags().StringVar(&exportUntil, "until", "", "end before this `time`, given the same ways (default: now)")
	exportCmd.Flags().BoolVar(&exportAll, "all", false, "everyone's lines (administrators only)")
	exportCmd.Flags().StringVarP(&exportOut, "output", "o", "", "the `file` to write (default: standard output)")
	rootCmd.AddCommand(exportCmd)
}

// relativeTime turns 12h, 7d or 2w into the time that long before now, for
// --since and --until; anything else goes to the server as it is.
func relativeTime(s string, now time.Time) string {
	if len(s) < 2 {
		return s
	}
	n, err := strconv.Atoi(s[:len(s)-1])
	if err != nil || n < 0 {
		return s
	}
	var unit time.Duration
	switch s[len(s)-1] {
	case 'h':
		unit = time.Hour
	case 'd':
		unit = 24 * time.Hour
	case 'w':
		unit = 7 * 24 * time.Hour
	default:
		return s
	}
	return now.Add(-time.Duration(n) * unit).UTC().Format(time.RFC3339)
}
