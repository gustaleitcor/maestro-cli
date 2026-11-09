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

var settingsAsJSON bool

var settingsCmd = &cobra.Command{
	Use:   "settings",
	Short: "Show the server's settings: the image limit and how lines are scheduled",
	Long: `Shows how many images each user may keep, and how the scheduler chooses
whose line goes next and where: the policy of each level, the score's
weights and how a user's share of the machines is measured. An
administrator changes them on the Maestro page.`,
	Args:              cobra.NoArgs,
	ValidArgsFunction: cobra.NoFileCompletions,
	RunE: func(cmd *cobra.Command, args []string) error {
		settings, err := maestroapi.GetSettings(cmd.Context(), maestroKey)
		if err != nil {
			return err
		}
		if settingsAsJSON {
			return writeJSON(os.Stdout, settings)
		}
		return writeSettings(os.Stdout, settings)
	},
}

func init() {
	settingsCmd.Flags().BoolVar(&settingsAsJSON, "json", false, "print the settings as JSON")
	rootCmd.AddCommand(settingsCmd)
}

func writeSettings(out io.Writer, s *maestroapi.Settings) error {
	w := tabwriter.NewWriter(out, 0, 0, 2, ' ', 0)
	fmt.Fprintf(w, "images per user\t%d\n", s.ImagesPerUser)
	if c := s.Scheduler; c != nil {
		staged := fmt.Sprint(c.StagedCapacity)
		if c.StagedCapacity == 0 {
			staged = "twice the machines"
		}
		var weights []string
		for _, u := range c.Weights {
			weights = append(weights, fmt.Sprintf("user %d: %g cpu, %g memory", u.User, u.CPUs, u.Memory))
		}
		if len(weights) == 0 {
			weights = []string{"everyone 1"}
		}
		for _, row := range [][2]string{
			{"within a user", c.User},
			{"among users", c.Select},
			{"staged lines", c.Allocate},
			{"placement", c.Place},
			{"on a machine", c.Machine},
			{"staged at most", staged},
			{"waiting on a machine", fmt.Sprint(c.Depth)},
			{"score weights", fmt.Sprintf("cpu %g · memory %g · waiting %g · share %g", c.CPUWeight, c.MemoryWeight, c.AgingWeight, c.ShareWeight)},
			{"waiting horizon", fmt.Sprintf("%g min", c.HorizonMinutes)},
			{"share measure", c.Share},
			{"half-life", fmt.Sprintf("%g h", c.HalfLifeHours)},
			{"fairness (tetris)", fmt.Sprint(c.Fairness)},
			{"siro seed", fmt.Sprint(c.Seed)},
			{"user weights", strings.Join(weights, "; ")},
		} {
			fmt.Fprintf(w, "%s\t%s\n", row[0], row[1])
		}
	}
	return w.Flush()
}
