package cmd

import "github.com/spf13/cobra"

// conceptsCmd is a help topic, not a command: it has no Run, so cobra lists
// it under "Additional help topics" and `maestro help concepts` shows it.
var conceptsCmd = &cobra.Command{
	Use:   "concepts",
	Short: "What builds, runs, lines and containers are",
	Long: `How Maestro's words fit together.

BUILD
  ` + "`maestro build <repo>`" + ` clones a repo from one of your forges and builds a
  container image from its Dockerfile. Each build has a number, shown as #12;
  ` + "`maestro builds`" + ` lists them. You may keep a few images at a time; the one
  run least recently goes when a new build needs room.

RUN
  ` + "`maestro run <build>`" + ` runs a build's image on the machines. Each run has a
  number of its own, shown as "run 31": ` + "`maestro runs`" + ` lists them. A build
  number and a run number are never the same thing: run takes a build,
  runs show takes a run.

LINE
  The repo's maestro.toml lists lines of parameters (and may name a
  generator that yields more). A run starts one container per line, with
  that line's arguments. Without a maestro.toml, a run is a single line that
  runs the image's own command. ` + "`maestro runs show <run>`" + ` shows each line,
  and ` + "`maestro runs logs <run> <line>`" + ` what it printed.

WAITING
  A line waits at three levels before it runs:
    queued    among your own lines
    staged    chosen from them, waiting for a machine
    assigned  given to a machine, waiting for room on it
  then starting, running, and one of succeeded, failed (it exited non-zero),
  error (it couldn't run) or cancelled. ` + "`maestro queue`" + ` shows how many wait
  where, and your share of the machines.

OUTPUTS
  maestro.toml names the files to keep from each container. ` + "`maestro runs get`" + `
  downloads them, with what each line printed, as stdout.log and stderr.log.

EXIT STATUS
  0 when all went well; 1 when something failed, including a run watched to
  its end (--watch) that failed or was cancelled; 2 when a command was asked
  wrongly; 130 when ctrl-c stopped it before the end.`,
}

func init() {
	rootCmd.AddCommand(conceptsCmd)
}
