package cmd

import (
	"fmt"
	"io"
	"os"
	"slices"
	"strconv"
	"strings"
	"text/tabwriter"

	"github.com/spf13/cobra"

	"maestro-cli/internal/maestroapi"
)

var (
	usageAsJSON bool
	usageDays   int
	usageUser   string
	usageDaily  bool
)

var usageCmd = &cobra.Command{
	Use:   "usage",
	Short: "Show how much your lines have used",
	Long: `Shows the core-hours and memory GB-hours your lines used over the last
days, counted as each line ends. What you used counts against you, less as
it gets older, when the scheduler weighs whose line goes next, if an
administrator set it up so.

Administrators see everyone's, or one user's with --user, by their number
or email.`,
	Example: `  maestro usage --daily --days 7
  maestro usage --user ana@example.org`,
	Args:              cobra.MatchAll(cobra.NoArgs, flagCheck(checkUsageUser)),
	ValidArgsFunction: cobra.NoFileCompletions,
	RunE: func(cmd *cobra.Command, args []string) error {
		id, email, _ := parseUser(usageUser)
		usage, err := maestroapi.GetUsage(cmd.Context(), maestroKey, usageDays, id)
		if err != nil {
			return err
		}
		if email != "" {
			// Everyone's usage says whose each is: the one asked for is kept.
			usage = slices.DeleteFunc(usage, func(u maestroapi.Usage) bool { return !strings.EqualFold(u.Email, email) })
			if len(usage) == 0 {
				return fmt.Errorf("nothing used by %s in the last %d day(s), or no such user", email, usageDays)
			}
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
	usageCmd.Flags().StringVar(&usageUser, "user", "", "one user's usage, by their number or email (administrators)")
	usageCmd.Flags().BoolVar(&usageDaily, "daily", false, "a row for each day")
	rootCmd.AddCommand(usageCmd)
}

// parseUser reads --user: a user's number, or their email.
func parseUser(arg string) (id int64, email string, err error) {
	switch {
	case arg == "":
		return 0, "", nil
	case strings.Contains(arg, "@"):
		return 0, arg, nil
	}
	id, err = strconv.ParseInt(arg, 10, 64)
	if err != nil || id <= 0 {
		return 0, "", fmt.Errorf("--user takes a user's number or email, not %q", arg)
	}
	return id, "", nil
}

func checkUsageUser() error {
	_, _, err := parseUser(usageUser)
	return err
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
