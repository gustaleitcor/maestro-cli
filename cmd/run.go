package cmd

import (
	"context"
	"fmt"
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

<build> is the build number shown by ` + "`maestro builds list`" + `, with or without #.
Containers start as machines have room; --watch follows them until the run
is over, as ` + "`maestro runs show <run> --watch`" + ` does later.

  maestro run 12 --watch
  maestro runs show <run> --watch
  maestro runs get <run>`,
	Args:              cobra.ExactArgs(1),
	ValidArgsFunction: completeBuilds,
	RunE:              runRun,
}

func init() {
	addWatchFlags(runCmd, "follow the run until it is over")
	rootCmd.AddCommand(runCmd)
}

func runRun(cmd *cobra.Command, args []string) error {
	buildID, err := parseID(args[0], "build")
	if err != nil {
		return fmt.Errorf("%w such as 12 (see `maestro builds list`)", err)
	}
	if err := checkWatch(); err != nil {
		return err
	}
	run, err := startRun(cmd.Context(), buildID)
	if err != nil || !runsWatch {
		return err
	}
	return showRun(cmd.Context(), run.ID, true)
}

// startRun queues a run of a build and says what was queued.
func startRun(ctx context.Context, buildID int64) (*maestroapi.Run, error) {
	run, err := maestroapi.StartRun(ctx, maestroKey, buildID)
	if err != nil {
		return nil, err
	}

	where := "any machine"
	if len(run.Machines) > 0 {
		where = strings.Join(run.Machines, ", ")
	}
	fmt.Printf("%s run %d: %d line(s) on %s.\n", successStyle.Render("Queued"), run.ID, len(run.Detail), where)
	if len(run.Outputs) > 0 {
		fmt.Printf("Keeping %s from each container.\n", strings.Join(run.Outputs, ", "))
	}
	fmt.Println(dimStyle.Render(fmt.Sprintf("Follow it with `maestro runs show %d --watch`.", run.ID)))
	return run, nil
}
