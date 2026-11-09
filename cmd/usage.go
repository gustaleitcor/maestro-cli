package cmd

import (
	"fmt"
	"io"
	"os"
	"text/tabwriter"

	"github.com/spf13/cobra"

	"maestro-cli/internal/maestroapi"
)

var (
	usageAsJSON bool
	usageDays   int
	usageUser   int64
	usageDaily  bool
)

var usageCmd = &cobra.Command{
	Use:   "usage",
	Short: "Show how much your lines have used",
	Long: `Shows the core-hours and memory GB-hours your lines used over the last
days, counted as each line ends. What you used counts against you, less as
it gets older, when the scheduler weighs whose line goes next, if an
administrator set it up so.

Administrators see everyone's, or one user's with --user.`,
	Args:              cobra.NoArgs,
	ValidArgsFunction: cobra.NoFileCompletions,
	RunE: func(cmd *cobra.Command, args []string) error {
		usage, err := maestroapi.GetUsage(cmd.Context(), maestroKey, usageDays, usageUser)
		if err != nil {
			return err
		}
		if usageAsJSON {
			return writeJSON(os.Stdout, usage)
		}
		return writeUsage(os.Stdout, usage, usageDays, usageDaily)
	},
}

func init() {
	usageCmd.Flags().BoolVar(&usageAsJSON, "json", false, "print the usage as JSON, a day at a time")
	usageCmd.Flags().IntVar(&usageDays, "days", 30, "how many days back")
	usageCmd.Flags().Int64Var(&usageUser, "user", 0, "one user's usage, by their number (administrators)")
	usageCmd.Flags().BoolVar(&usageDaily, "daily", false, "a row for each day")
	rootCmd.AddCommand(usageCmd)
}

func writeUsage(out io.Writer, usage []maestroapi.Usage, days int, daily bool) error {
	if len(usage) == 0 {
		_, err := fmt.Fprintf(out, "Nothing used in the last %d day(s).\n", days)
		return err
	}
	w := tabwriter.NewWriter(out, 0, 0, 2, ' ', 0)
	if daily {
		fmt.Fprintln(w, "USER\tDAY\tCORE-HOURS\tGB-HOURS")
	} else {
		fmt.Fprintln(w, "USER\tCORE-HOURS\tGB-HOURS\tDAYS")
	}
	for _, u := range usage {
		name := u.Name
		if name == "" {
			name = u.Email
		}
		var cpu, mem float64
		for _, d := range u.Days {
			cpu, mem = cpu+d.CPUHours, mem+d.MemoryGBHours
			if daily {
				fmt.Fprintf(w, "%s\t%s\t%.1f\t%.1f\n", name, d.Day, d.CPUHours, d.MemoryGBHours)
			}
		}
		if !daily {
			fmt.Fprintf(w, "%s\t%.1f\t%.1f\t%d\n", name, cpu, mem, len(u.Days))
		}
	}
	if err := w.Flush(); err != nil {
		return err
	}
	_, err := fmt.Fprintf(out, "\nOver the last %d day(s), counted as lines end.\n", days)
	return err
}
