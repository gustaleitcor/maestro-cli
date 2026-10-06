package cmd

import (
	"fmt"
	"strconv"
	"strings"

	"github.com/spf13/cobra"

	"maestro-cli/internal/maestroapi"
)

var runCmd = &cobra.Command{
	Use:   "run <build>",
	Short: "Run the image of one of your builds on the machines",
	Long: `Queues a run of the image a build produced: one container per line of
parameters in the repo's maestro.toml, spread over the machines it names.
Without a maestro.toml, a single container runs the image's own command.

<build> is the build number shown by ` + "`maestro image list`" + `, with or without #.
Containers start as machines have room; follow them with ` + "`maestro runs show`" + `.

  maestro run 12
  maestro runs show <run>
  maestro runs get <run>`,
	Args: cobra.ExactArgs(1),
	RunE: runRun,
}

func init() {
	rootCmd.AddCommand(runCmd)
}

func runRun(cmd *cobra.Command, args []string) error {
	buildID, err := strconv.ParseInt(strings.TrimPrefix(args[0], "#"), 10, 64)
	if err != nil || buildID <= 0 {
		return fmt.Errorf("invalid build %q: expected a build number such as 12 (see `maestro image list`)", args[0])
	}

	run, err := maestroapi.StartRun(cmd.Context(), maestroKey, buildID)
	if err != nil {
		return err
	}

	where := "any machine"
	if len(run.Machines) > 0 {
		where = strings.Join(run.Machines, ", ")
	}
	fmt.Printf("%s run %d: %d line(s) on %s.\n", successStyle.Render("Queued"), run.ID, len(run.Detail), where)
	if len(run.Outputs) > 0 {
		fmt.Printf("Keeping %s from each container.\n", strings.Join(run.Outputs, ", "))
	}
	fmt.Println(dimStyle.Render(fmt.Sprintf("Follow it with `maestro runs show %d`.", run.ID)))
	return nil
}
