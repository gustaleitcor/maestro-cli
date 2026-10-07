package cmd

import (
	"context"
	"errors"
	"fmt"
	"io"
	"os"
	"path/filepath"
	"strconv"
	"strings"
	"text/tabwriter"
	"time"

	"github.com/spf13/cobra"

	"maestro-cli/internal/maestroapi"
	"maestro-cli/tui"
)

var (
	runsLogsTail int
	runsWatch    bool
	runsInterval time.Duration
	runsAsJSON   bool
	runsAll      bool
)

const noRuns = "No runs yet. Start one with: maestro run <build>"

// On its own it lists.
var runsCmd = &cobra.Command{
	Use:   "runs",
	Short: "Commands for the runs started with `maestro run`",
	Args:  cobra.NoArgs,
	RunE:  runRunsList,
}

var runsListCmd = &cobra.Command{
	Use:   "list",
	Short: "List your runs",
	Long: `Lists your runs, newest first. With --watch, the list is drawn again every
--interval until you press ctrl-c. With --all, an administrator is given
everyone's runs, and whose each is.`,
	Args:              cobra.NoArgs,
	ValidArgsFunction: cobra.NoFileCompletions,
	RunE:              runRunsList,
}

var runsShowCmd = &cobra.Command{
	Use:   "show [run]",
	Short: "Show each line of a run: where it ran, how it ended, what it kept",
	Long: `Shows each line of a run: where it ran, how it ended, what it kept. With
--watch, it is drawn again every --interval until the run is over, or you
press ctrl-c. Without a run, it shows your newest.`,
	Args:              cobra.MaximumNArgs(1),
	ValidArgsFunction: completeRuns,
	RunE:              runRunsShow,
}

var runsLogsCmd = &cobra.Command{
	Use:   "logs <run> <line>",
	Short: "Print what a line's container wrote to stdout and stderr",
	Long: `Prints what a line's container wrote: its stdout to stdout and its stderr
to stderr, so the two can be told apart or sent to different places.

  maestro runs logs 31 2 2>/dev/null   # only what it printed to stdout
  maestro runs logs 31 2 >/dev/null    # only what it printed to stderr

All of stdout comes before all of stderr; how they were mixed as the
container ran isn't kept.`,
	Args:              cobra.ExactArgs(2),
	ValidArgsFunction: completeRunLines,
	RunE:              runRunsLogs,
}

var runsGetCmd = &cobra.Command{
	Use:   "get <run> [dir]",
	Short: "Download the files kept from a run",
	Long: `Downloads every file kept from a run's containers into <dir>/<run>/<line>/,
and what each printed among them, as stdout.log and stderr.log (dir defaults
to the current directory).`,
	Args:              cobra.RangeArgs(1, 2),
	ValidArgsFunction: completeRunThenDir,
	RunE:              runRunsGet,
}

var runsCancelCmd = &cobra.Command{
	Use:               "cancel <run>",
	Short:             "Cancel a run: drop what hasn't started and stop what has",
	Args:              cobra.ExactArgs(1),
	ValidArgsFunction: completeRuns,
	RunE:              runRunsCancel,
}

var runsRmCmd = &cobra.Command{
	Use:               "rm <run>",
	Aliases:           []string{"remove"},
	Short:             "Delete a finished run and the files kept from it",
	Args:              cobra.ExactArgs(1),
	ValidArgsFunction: completeRuns,
	RunE:              runRunsRm,
}

func init() {
	runsLogsCmd.Flags().IntVar(&runsLogsTail, "tail", -1, "only the last N lines, while the container runs")
	for _, c := range []*cobra.Command{runsCmd, runsListCmd} {
		addWatchFlags(c, "keep showing it, refreshed as it changes")
		c.Flags().BoolVar(&runsAsJSON, "json", false, "print the runs as JSON")
		c.Flags().BoolVar(&runsAll, "all", false, "everyone's runs (administrators only)")
	}
	addWatchFlags(runsShowCmd, "keep showing it, refreshed as it changes")
	runsShowCmd.Flags().BoolVar(&runsAsJSON, "json", false, "print the run as JSON")
	rootCmd.AddCommand(runsCmd)
	runsCmd.AddCommand(runsListCmd, runsShowCmd, runsLogsCmd, runsGetCmd, runsCancelCmd, runsRmCmd)
}

// addWatchFlags gives a command --watch and --interval, for whatever ends
// in the watch of runs list or runs show.
func addWatchFlags(c *cobra.Command, usage string) {
	c.Flags().BoolVarP(&runsWatch, "watch", "w", false, usage)
	c.Flags().DurationVarP(&runsInterval, "interval", "n", 2*time.Second, "how often to refresh with --watch (at least 1s)")
}

func parseID(arg, what string) (int64, error) {
	id, err := strconv.ParseInt(strings.TrimPrefix(arg, "#"), 10, 64)
	if err != nil || id <= 0 {
		return 0, fmt.Errorf("invalid %s %q: expected a number", what, arg)
	}
	return id, nil
}

func lineCounts(counts map[string]int) string {
	var parts []string
	for _, status := range []string{"queued", "starting", "running", "succeeded", "failed", "error", "cancelled"} {
		if n := counts[status]; n > 0 {
			parts = append(parts, fmt.Sprintf("%d %s", n, status))
		}
	}
	return strings.Join(parts, ", ")
}

func checkWatch() error {
	if runsWatch && runsAsJSON {
		return fmt.Errorf("--json prints once; it can't be used with --watch")
	}
	if runsWatch && runsInterval < minTopPeriod {
		return fmt.Errorf("--interval must be at least %s", minTopPeriod)
	}
	return nil
}

func runRunsList(cmd *cobra.Command, args []string) error {
	if err := checkWatch(); err != nil {
		return err
	}
	if runsAsJSON {
		runs, err := maestroapi.ListRuns(cmd.Context(), maestroKey, runsAll)
		if err != nil {
			return err
		}
		return writeJSON(os.Stdout, runs)
	}
	show := func(ctx context.Context, w io.Writer) (bool, error) {
		runs, err := maestroapi.ListRuns(ctx, maestroKey, runsAll)
		if err != nil {
			return false, err
		}
		return false, writeRunsList(w, runs, runsAll)
	}
	if runsWatch {
		return tui.Watch(cmd.Context(), runsInterval, show)
	}
	_, err := show(cmd.Context(), os.Stdout)
	return err
}

// writeRunsList says whose each run is when all, which is when they aren't
// all the caller's.
func writeRunsList(out io.Writer, runs []maestroapi.Run, all bool) error {
	if len(runs) == 0 {
		_, err := fmt.Fprintln(out, noRuns)
		return err
	}
	w := tabwriter.NewWriter(out, 0, 0, 2, ' ', 0)
	user := ""
	if all {
		user = "USER\t"
	}
	fmt.Fprintln(w, "RUN\t"+user+"REPO\tBUILD\tSTATUS\tLINES\tCREATED")
	for _, r := range runs {
		if all {
			user = r.User.Name
			if user == "" {
				user = r.User.Email
			}
			user += "\t"
		}
		fmt.Fprintf(w, "%d\t%s%s\t#%d\t%s\t%s\t%s\n", r.ID, user, r.Repo+"@"+r.Ref, r.BuildID, r.Status, lineCounts(r.Lines), r.CreatedAt.Local().Format("2006-01-02 15:04"))
	}
	return w.Flush()
}

func runRunsShow(cmd *cobra.Command, args []string) error {
	if err := checkWatch(); err != nil {
		return err
	}
	var id int64
	if len(args) == 1 {
		var err error
		if id, err = parseID(args[0], "run"); err != nil {
			return err
		}
	} else {
		// Newest first: the one just started, most of the time.
		runs, err := maestroapi.ListRuns(cmd.Context(), maestroKey, false)
		if err != nil {
			return err
		}
		if len(runs) == 0 {
			return errors.New(noRuns)
		}
		id = runs[0].ID
	}
	if runsAsJSON {
		run, err := maestroapi.GetRun(cmd.Context(), maestroKey, id)
		if err != nil {
			return err
		}
		return writeJSON(os.Stdout, run)
	}
	return showRun(cmd.Context(), id, runsWatch)
}

// showRun prints a run, and with watch keeps it on screen until it is over.
func showRun(ctx context.Context, id int64, watch bool) error {
	// A run that is neither queued nor running won't change again.
	show := func(ctx context.Context, w io.Writer) (bool, error) {
		run, err := maestroapi.GetRun(ctx, maestroKey, id)
		if err != nil {
			return false, err
		}
		return run.Status != "queued" && run.Status != "running", writeRunShow(w, run)
	}
	if watch {
		return tui.Watch(ctx, runsInterval, show)
	}
	_, err := show(ctx, os.Stdout)
	return err
}

func writeRunShow(out io.Writer, run *maestroapi.Run) error {
	where := "any machine"
	if len(run.Machines) > 0 {
		where = strings.Join(run.Machines, ", ")
	}
	fmt.Fprintf(out, "Run %d of %s@%s (build #%d): %s, on %s\n\n", run.ID, run.Repo, run.Ref, run.BuildID, run.Status, where)

	w := tabwriter.NewWriter(out, 0, 0, 2, ' ', 0)
	fmt.Fprintln(w, "LINE\tPARAMETERS\tMACHINE\tSTATUS\tFILES")
	for _, l := range run.Detail {
		status := l.Status
		if l.ExitCode != nil && l.Status != "succeeded" {
			status = fmt.Sprintf("%s (exit %d)", l.Status, *l.ExitCode)
		}
		machine := l.Machine
		if machine == "" {
			machine = "-"
		}
		var files []string
		for _, f := range l.Files {
			if f.Missing {
				files = append(files, f.Path+" (missing)")
			} else {
				files = append(files, f.Path)
			}
		}
		if len(files) == 0 {
			files = []string{"-"}
		}
		fmt.Fprintf(w, "%d\t%s\t%s\t%s\t%s\n", l.Line, showArgs(l.Args), machine, status, strings.Join(files, ", "))
	}
	if err := w.Flush(); err != nil {
		return err
	}
	for _, l := range run.Detail {
		if l.Error != "" {
			fmt.Fprintf(out, "\nline %d: %s", l.Line, l.Error)
		}
	}
	_, err := fmt.Fprintln(out)
	return err
}

// showArgs quotes the arguments that need it, so a line reads back as it
// was written in maestro.toml.
func showArgs(args []string) string {
	if len(args) == 0 {
		return "(image's command)"
	}
	quoted := make([]string, len(args))
	for i, a := range args {
		if a == "" || strings.ContainsAny(a, " \t'\"\\") {
			a = "'" + strings.ReplaceAll(a, "'", `'\''`) + "'"
		}
		quoted[i] = a
	}
	return strings.Join(quoted, " ")
}

func runRunsLogs(cmd *cobra.Command, args []string) error {
	id, err := parseID(args[0], "run")
	if err != nil {
		return err
	}
	line, err := parseID(args[1], "line")
	if err != nil {
		return err
	}
	run, err := maestroapi.GetRun(cmd.Context(), maestroKey, id)
	if err != nil {
		return err
	}
	for _, l := range run.Detail {
		if l.Line == line {
			return writeLineLogs(cmd.Context(), id, l, runsLogsTail, os.Stdout, os.Stderr)
		}
	}
	return fmt.Errorf("run %d has no line %d", id, line)
}

// logStreams is the streams to ask a line's logs by. A server that only
// knows the two together doesn't name any.
func logStreams(l maestroapi.RunLine) []string {
	if l.Logs == nil && !l.StartedAt.IsZero() {
		return []string{maestroapi.LogCombined}
	}
	return l.Logs
}

// logFileName is where a stream of a line's logs goes in a run's folder:
// <line>/stdout.log and <line>/stderr.log, among the line's files, or
// <line>.log beside them for both in one.
func logFileName(line int64, stream string) string {
	if stream == maestroapi.LogCombined {
		return fmt.Sprintf("%d.log", line)
	}
	return fmt.Sprintf("%d/%s.log", line, stream)
}

// writeLineLogs sends each stream of a line's logs where it belongs. A line
// kept with both in one file has no way to tell them apart: it all goes to
// stdout.
func writeLineLogs(ctx context.Context, runID int64, l maestroapi.RunLine, tail int, stdout, stderr io.Writer) error {
	for _, stream := range logStreams(l) {
		w := stdout
		if stream == maestroapi.LogStderr {
			w = stderr
		}
		if err := maestroapi.RunLogs(ctx, maestroKey, runID, l.Line, stream, tail, w); err != nil {
			return err
		}
	}
	return nil
}

func runRunsGet(cmd *cobra.Command, args []string) error {
	id, err := parseID(args[0], "run")
	if err != nil {
		return err
	}
	dir := "."
	if len(args) == 2 {
		dir = args[1]
	}
	run, err := maestroapi.GetRun(cmd.Context(), maestroKey, id)
	if err != nil {
		return err
	}

	runDir := filepath.Join(dir, strconv.FormatInt(run.ID, 10))
	count := 0
	for _, l := range run.Detail {
		lineDir := filepath.Join(runDir, strconv.FormatInt(l.Line, 10))
		for _, file := range l.Files {
			if file.Missing {
				continue
			}
			target := filepath.Join(lineDir, filepath.FromSlash(file.Path))
			// The server only lists paths inside the line, but don't trust that blindly.
			if rel, err := filepath.Rel(lineDir, target); err != nil || strings.HasPrefix(rel, "..") {
				return fmt.Errorf("refusing to write %q outside %s", file.Path, lineDir)
			}
			if err := save(target, func(f *os.File) error {
				return maestroapi.DownloadRunFile(cmd.Context(), maestroKey, run.ID, l.Line, file.Path, f)
			}); err != nil {
				return err
			}
			count++
		}
		// After the files, so that a file named as a log never takes its
		// place. A line that never started printed nothing, and has no streams.
		for _, stream := range logStreams(l) {
			if err := save(filepath.Join(runDir, filepath.FromSlash(logFileName(l.Line, stream))), func(f *os.File) error {
				return maestroapi.RunLogs(cmd.Context(), maestroKey, run.ID, l.Line, stream, -1, f)
			}); err != nil {
				return err
			}
		}
	}
	fmt.Printf("%s %d file(s) and the logs into %s\n", successStyle.Render("Downloaded"), count, runDir)
	if run.Status == "queued" || run.Status == "running" {
		fmt.Println(dimStyle.Render("The run isn't over yet: lines still going have nothing to download."))
	}
	return nil
}

func save(path string, fill func(*os.File) error) error {
	if err := os.MkdirAll(filepath.Dir(path), 0o755); err != nil {
		return err
	}
	f, err := os.Create(path)
	if err != nil {
		return err
	}
	err = fill(f)
	if closeErr := f.Close(); err == nil {
		err = closeErr
	}
	if err != nil {
		os.Remove(path)
		return fmt.Errorf("%s: %w", path, err)
	}
	return nil
}

func runRunsCancel(cmd *cobra.Command, args []string) error {
	id, err := parseID(args[0], "run")
	if err != nil {
		return err
	}
	if err := maestroapi.CancelRun(cmd.Context(), maestroKey, id); err != nil {
		return err
	}
	fmt.Println(successStyle.Render(fmt.Sprintf("Cancelled run %d.", id)) + " Running containers stop shortly; their files are still kept.")
	return nil
}

func runRunsRm(cmd *cobra.Command, args []string) error {
	id, err := parseID(args[0], "run")
	if err != nil {
		return err
	}
	if err := maestroapi.DeleteRun(cmd.Context(), maestroKey, id); err != nil {
		return err
	}
	fmt.Println(successStyle.Render(fmt.Sprintf("Deleted run %d and its files.", id)))
	return nil
}
