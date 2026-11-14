package cmd

import (
	"context"
	"fmt"
	"io"
	"os"
	"slices"
	"strings"

	"github.com/spf13/cobra"

	"maestro-cli/internal/maestroapi"
)

var runCmd = &cobra.Command{
	Use:   "run <build>",
	Short: "Run the image of one of your builds on the machines",
	Long: `Queues a run of the image a build produced: one container per line of
parameters in the repo's maestro.toml, and per line its generator yielded
when the image was built, spread over the machines it names.
Without a maestro.toml, a single container runs the image's own command.

<build> is the build number shown by ` + "`maestro builds list`" + `, with or without #.
Containers start as machines have room; --watch follows them until the run
is over, as ` + "`maestro runs show <run> --watch`" + ` does later.

  maestro run 12 --watch
  maestro runs show <run> --watch
  maestro runs get <run>

To try other lines without a new build, send your own maestro.toml (-c) or
the generator it names (-g) from here: the server reads them against the
repo as the build had it and runs the build's image on the lines they make.
Each may have any name here. A Dockerfile changes the image, which takes
` + "`maestro build -f`" + `.

  maestro run 12 -c small.toml --watch
  maestro run 12 -g genvrs.py`,
	Args:              cobra.MatchAll(numberArgs(1, 1, "build"), flagCheck(checkWatch)),
	ValidArgsFunction: completeBuilds,
	RunE:              runRun,
}

var runLocal localFlags

func init() {
	addWatchFlags(runCmd, "follow the run until it is over")
	runCmd.Flags().StringVarP(&runLocal.config, "config", "c", "", "a maestro.toml here to make the lines from in place of the build's")
	runCmd.Flags().StringVarP(&runLocal.generator, "generator", "g", "", "a generator script here, in place of the one maestro.toml names")
	// Only to say it takes a build: as on build, so it isn't taken for a typo.
	runCmd.Flags().StringVarP(&runLocal.dockerfile, "dockerfile", "f", "", "not on run: a Dockerfile takes a new image, from maestro build -f")
	addAnswerFlags(runCmd, "the queued run")
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
	if runLocal.dockerfile != "" {
		return fmt.Errorf("a Dockerfile changes the image, which a run can't: build it with `maestro build <repo> -f %s`", runLocal.dockerfile)
	}
	files, err := readLocalFiles(runLocal)
	if err != nil {
		return err
	}
	var run *maestroapi.Run
	if files.Empty() {
		run, err = startRun(cmd.Context(), buildID)
	} else {
		run, err = startRunWith(cmd.Context(), buildID, files)
	}
	if err != nil {
		return err
	}
	if runsWatch {
		return showRun(cmd.Context(), run.ID, true)
	}
	return printAnswer(run, run.ID)
}

// addAnswerFlags gives run and build --json and -q, for scripts: stdout
// then has the answer alone, and what they say along the way goes to stderr.
func addAnswerFlags(c *cobra.Command, what string) {
	c.Flags().BoolVar(&runsAsJSON, "json", false, "print "+what+" as JSON, and nothing else on stdout")
	c.Flags().BoolVarP(&runsQuiet, "quiet", "q", false, "print only the number of "+what+", and nothing else on stdout")
}

// chatter is where a command says what it does along the way: stdout,
// unless --json or -q keep stdout for the answer alone.
func chatter() io.Writer {
	if runsAsJSON || runsQuiet {
		return os.Stderr
	}
	return os.Stdout
}

// printAnswer prints, with --json, v, and with -q, id; otherwise chatter
// has said it all already.
func printAnswer(v any, id int64) error {
	switch {
	case runsAsJSON:
		return writeJSON(os.Stdout, v)
	case runsQuiet:
		_, err := fmt.Println(id)
		return err
	}
	return nil
}

// startRunWith queues a run of a build whose lines come from files: the
// server clones its repo again for them, with the token of the forge it
// came from, if one is set up, as a private repo needs.
func startRunWith(ctx context.Context, buildID int64, files maestroapi.LocalFiles) (*maestroapi.Run, error) {
	builds, err := maestroapi.ListBuilds(ctx, maestroKey)
	if err != nil {
		return nil, err
	}
	at := slices.IndexFunc(builds, func(b maestroapi.Build) bool { return b.ID == buildID })
	if at < 0 {
		return nil, fmt.Errorf("no build #%d (see `maestro builds list`)", buildID)
	}
	token := ""
	if active, err := selectForgeByHost(builds[at].Host, ""); err == nil {
		token = active.Token
	}
	sayLocalFiles(runLocal)
	fmt.Fprintln(chatter(), dimStyle.Render(fmt.Sprintf("Reading them against %s@%s, as build #%d had it...", builds[at].Repo, builds[at].Ref, buildID)))
	run, err := maestroapi.StartRun(ctx, maestroKey, buildID, files, token)
	if err != nil {
		return nil, err
	}
	if len(run.LocalFiles) == 0 {
		// Queued all the same, on the build's lines.
		return nil, fmt.Errorf("run %d was queued on build #%d's own lines, not yours: this Maestro server is too old to take them; cancel it with `maestro runs cancel %d` and update the server", run.ID, buildID, run.ID)
	}
	sayQueued(run)
	return run, nil
}

// startRun queues a run of a build and says what was queued.
func startRun(ctx context.Context, buildID int64) (*maestroapi.Run, error) {
	run, err := maestroapi.StartRun(ctx, maestroKey, buildID, maestroapi.LocalFiles{}, "")
	if err != nil {
		return nil, err
	}
	sayQueued(run)
	return run, nil
}

// sayQueued says what was queued.
func sayQueued(run *maestroapi.Run) {
	out := chatter()
	where := "any machine"
	if len(run.Machines) > 0 {
		where = strings.Join(run.Machines, ", ")
	}
	fmt.Fprintf(out, "%s run %d of build #%d: %d line(s) on %s.\n", successStyle.Render("Queued"), run.ID, run.BuildID, len(run.Detail), where)
	if len(run.Outputs) > 0 {
		fmt.Fprintf(out, "Keeping %s from each container.\n", strings.Join(run.Outputs, ", "))
	}
	if len(run.LocalFiles) > 0 {
		fmt.Fprintf(out, "Lines from your %s.\n", strings.Join(run.LocalFiles, ", "))
	}
	fmt.Fprintln(out, dimStyle.Render(fmt.Sprintf("Follow it with `maestro runs show %d --watch`.", run.ID)))
}
