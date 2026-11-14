package cmd

import (
	"context"
	"errors"
	"fmt"
	"io"
	"os"
	"os/signal"
	"path/filepath"
	"slices"
	"strconv"
	"strings"
	"sync"
	"text/tabwriter"
	"time"

	"github.com/spf13/cobra"
	"golang.org/x/term"

	"maestro-cli/internal/humanize"
	"maestro-cli/internal/maestroapi"
	"maestro-cli/tui"
)

var (
	runsLogsTail int
	runsWatch    bool
	runsInterval time.Duration
	runsAsJSON   bool
	runsAll      bool
	// runsQuiet is run's and build's -q: only the number of what they made.
	runsQuiet bool

	runsLogsFollow bool
	runsLimit      int
	runsStatus     string
	runsRepo       string
	runsFailed     bool
	runsGetLines   []int64
	runsGetForce   bool
)

// followPoll is how often runs logs -f looks whether a line has started.
var followPoll = 2 * time.Second

const noRuns = "No runs yet. Start one with: maestro run <build>"

// On its own it lists.
var runsCmd = &cobra.Command{
	Use:   "runs",
	Short: "Commands for the runs started with `maestro run`",
	Args:  cobra.MatchAll(nounArgs, flagCheck(checkWatch)),
	RunE:  runRunsList,
}

var runsListCmd = &cobra.Command{
	Use:   "list",
	Short: "List your runs",
	Long: `Lists your runs, newest first: the last --limit of them (20; 0 for all),
of a --status (queued, running, succeeded, failed, cancelled) or a --repo
if asked. --json prints the same runs. With --watch, the list is drawn again
every --interval until you press ctrl-c. With --all, an administrator is
given everyone's runs, and whose each is.`,
	Args:              cobra.MatchAll(cobra.NoArgs, flagCheck(checkWatch)),
	ValidArgsFunction: cobra.NoFileCompletions,
	RunE:              runRunsList,
}

var runsShowCmd = &cobra.Command{
	Use:   "show [run]",
	Short: "Show each line of a run: where it ran, how it ended, what it kept",
	Long: `Shows each line of a run: where it ran, how it ended, what it kept. With
--watch, it is drawn again every --interval until the run is over, or you
press ctrl-c, and exits 1 if the run failed. Without a run, it shows your
newest. --failed shows only the lines that failed, errored or were
cancelled.`,
	Args:              cobra.MatchAll(numberArgs(0, 1, "run"), flagCheck(checkWatch)),
	ValidArgsFunction: completeRuns,
	RunE:              runRunsShow,
}

var runsLogsCmd = &cobra.Command{
	Use:   "logs <run> [line]",
	Short: "Print what a line's container wrote to stdout and stderr",
	Long: `Prints what a line's container wrote: its stdout to stdout and its stderr
to stderr, so the two can be told apart or sent to different places. The
line may be left out of a run that has only one.

With -f, a line that hasn't started yet is waited for, and what a running
line prints is followed until it ends, as tail -f does.

Once a line is over, all of its stdout comes before all of its stderr; how
they were mixed as the container ran isn't kept.`,
	Example: `  maestro runs logs 31 2 -f
  maestro runs logs 31 2 2>/dev/null   # only what it printed to stdout
  maestro runs logs 31 2 >/dev/null    # only what it printed to stderr`,
	Args:              numberArgs(1, 2, "run", "line"),
	ValidArgsFunction: completeRunLines,
	RunE:              runRunsLogs,
}

var runsGetCmd = &cobra.Command{
	Use:   "get <run> [dir]",
	Short: "Download the files kept from a run",
	Long: `Downloads every file kept from a run's containers into <dir>/<run>/<line>/,
and what each printed among them, as stdout.log and stderr.log (dir defaults
to the current directory). --line picks some lines only. A run already
downloaded there is only downloaded over with --force.`,
	Example: `  maestro runs get 31
  maestro runs get 31 results/ --line 2 --line 5`,
	Args:              numberArgs(1, 2, "run"),
	ValidArgsFunction: completeRunThenDir,
	RunE:              runRunsGet,
}

var runsCancelCmd = &cobra.Command{
	Use:               "cancel <run>",
	Short:             "Cancel a run: drop what hasn't started and stop what has",
	Args:              numberArgs(1, 1, "run"),
	ValidArgsFunction: completeRuns,
	RunE:              runRunsCancel,
}

var runsRmCmd = &cobra.Command{
	Use:               "rm <run>",
	Aliases:           []string{"remove"},
	Short:             "Delete a finished run and the files kept from it",
	Args:              numberArgs(1, 1, "run"),
	ValidArgsFunction: completeRuns,
	RunE:              runRunsRm,
}

func init() {
	runsLogsCmd.Flags().IntVar(&runsLogsTail, "tail", 0, "only the last N lines of a line still running (all when left out)")
	runsLogsCmd.Flags().BoolVarP(&runsLogsFollow, "follow", "f", false, "wait for the line to start, and follow what it prints until it ends")
	for _, c := range []*cobra.Command{runsCmd, runsListCmd} {
		addWatchFlags(c, "keep showing it, refreshed as it changes")
		c.Flags().BoolVar(&runsAsJSON, "json", false, "print the runs as JSON")
		c.Flags().BoolVar(&runsAll, "all", false, "everyone's runs (administrators only)")
		c.Flags().IntVar(&runsLimit, "limit", 20, "how many runs, newest first; 0 for all")
		c.Flags().StringVar(&runsStatus, "status", "", "only the runs with this status")
		c.Flags().StringVar(&runsRepo, "repo", "", "only the runs of this repo (owner/name, or name)")
	}
	addWatchFlags(runsShowCmd, "keep showing it, refreshed as it changes; exit 1 if the run failed")
	runsShowCmd.Flags().BoolVar(&runsAsJSON, "json", false, "print the run as JSON")
	runsShowCmd.Flags().BoolVar(&runsFailed, "failed", false, "only the lines that failed, errored or were cancelled")
	runsGetCmd.Flags().Int64SliceVar(&runsGetLines, "line", nil, "only this line (again for more)")
	runsGetCmd.Flags().BoolVar(&runsGetForce, "force", false, "download over a run already downloaded there")
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
	for _, status := range []string{"queued", "staged", "assigned", "starting", "running", "succeeded", "failed", "error", "cancelled"} {
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
	if runsWatch && runsQuiet {
		return fmt.Errorf("-q prints only the number; it can't be used with --watch")
	}
	if runsAsJSON && runsQuiet {
		return fmt.Errorf("--json and -q each print the answer their own way; pick one")
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
		shown, _ := pickRuns(runs)
		return writeJSON(os.Stdout, shown)
	}
	show := func(ctx context.Context, w io.Writer) (bool, error) {
		runs, err := maestroapi.ListRuns(ctx, maestroKey, runsAll)
		if err != nil {
			return false, err
		}
		shown, matching := pickRuns(runs)
		return false, writeRunsList(w, shown, runsAll, matching)
	}
	if runsWatch {
		// A list is never over: ctrl-c is the way out of it.
		if err := tui.Watch(cmd.Context(), runsInterval, show); !errors.Is(err, tui.ErrInterrupted) {
			return err
		}
		return nil
	}
	_, err := show(cmd.Context(), os.Stdout)
	return err
}

// pickRuns is the runs --status and --repo ask for, the newest --limit of
// them, and how many matched before the limit.
func pickRuns(runs []maestroapi.Run) ([]maestroapi.Run, int) {
	var matching []maestroapi.Run
	for _, r := range runs {
		if runsStatus != "" && !strings.EqualFold(r.Status, runsStatus) {
			continue
		}
		if runsRepo != "" && !strings.EqualFold(r.Repo, runsRepo) && !strings.HasSuffix(strings.ToLower(r.Repo), "/"+strings.ToLower(runsRepo)) {
			continue
		}
		matching = append(matching, r)
	}
	if runsLimit > 0 && len(matching) > runsLimit {
		return matching[:runsLimit], len(matching)
	}
	return matching, len(matching)
}

// writeRunsList says whose each run is when all, which is when they aren't
// all the caller's. matching is how many runs there were to show, of which
// runs are the first.
func writeRunsList(out io.Writer, runs []maestroapi.Run, all bool, matching int) error {
	if len(runs) == 0 {
		msg := noRuns
		if runsStatus != "" || runsRepo != "" {
			msg = "No runs match."
		}
		_, err := fmt.Fprintln(out, msg)
		return err
	}
	now := time.Now()
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
		fmt.Fprintf(w, "%d\t%s%s\t#%d\t%s\t%s\t%s\n", r.ID, user, r.Repo+"@"+r.Ref, r.BuildID, r.Status, lineCounts(r.Lines), humanize.Ago(r.CreatedAt, now))
	}
	if err := w.Flush(); err != nil {
		return err
	}
	if matching > len(runs) {
		_, err := fmt.Fprintln(out, dimStyle.Render(fmt.Sprintf("\nThe newest %d of %d runs; --limit 0 for all.", len(runs), matching)))
		return err
	}
	return nil
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
// A watched run that ended failed or cancelled is a failure of the command,
// so that a script can tell; one left before its end says it goes on.
func showRun(ctx context.Context, id int64, watch bool) error {
	var last *maestroapi.Run
	// A run that is neither queued nor running won't change again.
	show := func(ctx context.Context, w io.Writer) (bool, error) {
		run, err := maestroapi.GetRun(ctx, maestroKey, id)
		if err != nil {
			return false, err
		}
		last = run
		return !runActive(run), writeRunShow(w, run, runsFailed)
	}
	if !watch {
		_, err := show(ctx, os.Stdout)
		return err
	}
	err := tui.Watch(ctx, runsInterval, show)
	if errors.Is(err, tui.ErrInterrupted) {
		fmt.Fprintln(os.Stderr, dimStyle.Render(fmt.Sprintf("Stopped watching; run %d goes on. Cancel it with `maestro runs cancel %d`.", id, id)))
	}
	if err != nil {
		return err
	}
	return runOutcome(last)
}

func runActive(run *maestroapi.Run) bool {
	return run.Status == "queued" || run.Status == "running"
}

// runOutcome is nil for a run that succeeded, and the failure, with how its
// lines ended, for one that didn't.
func runOutcome(run *maestroapi.Run) error {
	switch run.Status {
	case "failed":
		return &exitError{exitFailed, fmt.Errorf("run %d failed: %s", run.ID, lineCounts(runLineCounts(run)))}
	case "cancelled":
		return &exitError{exitFailed, fmt.Errorf("run %d was cancelled: %s", run.ID, lineCounts(runLineCounts(run)))}
	}
	return nil
}

// runLineCounts is how many of a run's lines are in each status, from the
// counts the server gave or, failing those, its lines.
func runLineCounts(run *maestroapi.Run) map[string]int {
	if len(run.Lines) > 0 {
		return run.Lines
	}
	counts := map[string]int{}
	for _, l := range run.Detail {
		counts[l.Status]++
	}
	return counts
}

// failedStatus is how a line ended that didn't succeed.
func failedStatus(status string) bool {
	return status == "failed" || status == "error" || status == "cancelled"
}

// writeRunShow shows a run's lines, or with onlyFailed those that didn't
// succeed.
func writeRunShow(out io.Writer, run *maestroapi.Run, onlyFailed bool) error {
	where := "any machine"
	if len(run.Machines) > 0 {
		where = strings.Join(run.Machines, ", ")
	}
	fmt.Fprintf(out, "Run %d of %s@%s (build #%d): %s, on %s\n", run.ID, run.Repo, run.Ref, run.BuildID, run.Status, where)
	if len(run.LocalFiles) > 0 {
		fmt.Fprintln(out, dimStyle.Render(fmt.Sprintf("Lines from your %s, not the build's.", strings.Join(run.LocalFiles, ", "))))
	}
	if run.OutputsLayout == "shared" {
		fmt.Fprintln(out, dimStyle.Render("Outputs in one tree shared by every line: a later line overwrites a path an earlier one wrote."))
	}
	if counts := lineCounts(runLineCounts(run)); counts != "" {
		fmt.Fprintf(out, "%d line(s): %s\n", len(run.Detail), counts)
	}
	fmt.Fprintln(out)

	lines := run.Detail
	if onlyFailed {
		lines = nil
		for _, l := range run.Detail {
			if failedStatus(l.Status) {
				lines = append(lines, l)
			}
		}
		if len(lines) == 0 {
			_, err := fmt.Fprintln(out, "No line failed.")
			return err
		}
	}

	w := tabwriter.NewWriter(out, 0, 0, 2, ' ', 0)
	fmt.Fprintln(w, "LINE\tPARAMETERS\tMACHINE\tSTATUS\tFILES")
	for _, l := range lines {
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
	for _, l := range lines {
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
	run, err := maestroapi.GetRun(cmd.Context(), maestroKey, id)
	if err != nil {
		return err
	}
	l, err := pickLine(run, args[1:])
	if err != nil {
		return err
	}
	if runsLogsFollow {
		return followLine(cmd.Context(), id, l, os.Stdout, os.Stderr)
	}
	return writeLineLogs(cmd.Context(), id, l, runsLogsTail, false, os.Stdout, os.Stderr)
}

// pickLine is the line args name, or the only line of a run that has one.
func pickLine(run *maestroapi.Run, args []string) (maestroapi.RunLine, error) {
	if len(args) == 0 {
		switch len(run.Detail) {
		case 0:
			return maestroapi.RunLine{}, fmt.Errorf("run %d has no lines", run.ID)
		case 1:
			return run.Detail[0], nil
		}
		first, last := run.Detail[0].Line, run.Detail[len(run.Detail)-1].Line
		return maestroapi.RunLine{}, usageError(fmt.Errorf("run %d has %d lines (%d–%d): name one, as `maestro runs logs %d <line>`", run.ID, len(run.Detail), first, last, run.ID))
	}
	line, err := parseID(args[0], "line")
	if err != nil {
		return maestroapi.RunLine{}, err
	}
	for _, l := range run.Detail {
		if l.Line == line {
			return l, nil
		}
	}
	return maestroapi.RunLine{}, fmt.Errorf("run %d has no line %d", run.ID, line)
}

// lineWaiting is whether a line hasn't started printing yet.
func lineWaiting(status string) bool {
	switch status {
	case "queued", "staged", "assigned", "starting":
		return true
	}
	return false
}

// followLine waits for a line to start, then follows what it prints until
// it ends, as tail -f does: ctrl-c is the way out (exit 130). A line already
// over prints what was kept.
func followLine(ctx context.Context, runID int64, l maestroapi.RunLine, stdout, stderr io.Writer) error {
	ctx, stop := signal.NotifyContext(ctx, os.Interrupt)
	defer stop()

	said := ""
	for lineWaiting(l.Status) {
		if l.Status != said {
			fmt.Fprintln(stderr, dimStyle.Render(fmt.Sprintf("Line %d is %s; waiting for it to start...", l.Line, l.Status)))
			said = l.Status
		}
		select {
		case <-ctx.Done():
			return tui.ErrInterrupted
		case <-time.After(followPoll):
		}
		run, err := maestroapi.GetRun(ctx, maestroKey, runID)
		if ctx.Err() != nil {
			return tui.ErrInterrupted
		}
		if err != nil {
			return err
		}
		if l, err = pickLine(run, []string{strconv.FormatInt(l.Line, 10)}); err != nil {
			return err
		}
	}
	if l.Status != "running" {
		return writeLineLogs(ctx, runID, l, 0, false, stdout, stderr)
	}

	err := writeLineLogs(ctx, runID, l, runsLogsTail, true, stdout, stderr)
	if ctx.Err() != nil {
		return tui.ErrInterrupted
	}
	if err != nil {
		return err
	}
	// The stream ends when the line does. If it still runs, the server
	// answered what it had and no more: it is from before follow.
	if run, err := maestroapi.GetRun(ctx, maestroKey, runID); err == nil {
		if now, err := pickLine(run, []string{strconv.FormatInt(l.Line, 10)}); err == nil && now.Status == "running" {
			fmt.Fprintln(stderr, warnStyle.Render("This Maestro server can't follow logs: that is what the line had printed so far."))
		}
	}
	return nil
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
// stdout. Followed, the streams are read at once, each as it comes.
func writeLineLogs(ctx context.Context, runID int64, l maestroapi.RunLine, tail int, follow bool, stdout, stderr io.Writer) error {
	streams := logStreams(l)
	errs := make([]error, len(streams))
	var wg sync.WaitGroup
	for i, stream := range streams {
		w := stdout
		if stream == maestroapi.LogStderr {
			w = stderr
		}
		read := func() {
			errs[i] = maestroapi.RunLogs(ctx, maestroKey, runID, l.Line, maestroapi.LogOptions{Stream: stream, Tail: tail, Follow: follow}, w)
		}
		if !follow {
			// One after the other, so that stdout comes first.
			if read(); errs[i] != nil {
				return errs[i]
			}
			continue
		}
		wg.Go(read)
	}
	wg.Wait()
	return errors.Join(errs...)
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
	if _, err := os.Stat(runDir); err == nil && !runsGetForce {
		return fmt.Errorf("%s is there already: pass --force to download over it", runDir)
	}
	lines, err := pickLines(run, runsGetLines)
	if err != nil {
		return err
	}
	progress := newProgress(os.Stderr, countFiles(lines))
	defer progress.done()
	// A run whose outputs are shared has one tree of them for every line:
	// each file is downloaded once, into shared/.
	shared := run.OutputsLayout == "shared"
	seen := map[string]bool{}
	count := 0
	for _, l := range lines {
		lineDir := filepath.Join(runDir, strconv.FormatInt(l.Line, 10))
		filesDir := lineDir
		if shared {
			filesDir = filepath.Join(runDir, "shared")
		}
		for _, file := range l.Files {
			if file.Missing || (shared && seen[file.Path]) {
				continue
			}
			seen[file.Path] = true
			target := filepath.Join(filesDir, filepath.FromSlash(file.Path))
			// The server only lists paths inside the line, but don't trust that blindly.
			if rel, err := filepath.Rel(filesDir, target); err != nil || strings.HasPrefix(rel, "..") {
				return fmt.Errorf("refusing to write %q outside %s", file.Path, filesDir)
			}
			if err := save(target, func(f *os.File) error {
				return maestroapi.DownloadRunFile(cmd.Context(), maestroKey, run.ID, l.Line, file.Path, f)
			}); err != nil {
				return err
			}
			count++
			progress.step()
		}
		// After the files, so that a file named as a log never takes its
		// place. A line that never started printed nothing, and has no streams.
		for _, stream := range logStreams(l) {
			if err := save(filepath.Join(runDir, filepath.FromSlash(logFileName(l.Line, stream))), func(f *os.File) error {
				return maestroapi.RunLogs(cmd.Context(), maestroKey, run.ID, l.Line, maestroapi.LogOptions{Stream: stream}, f)
			}); err != nil {
				return err
			}
		}
	}
	progress.done()
	fmt.Printf("%s %d file(s) and the logs into %s\n", successStyle.Render("Downloaded"), count, runDir)
	if run.Status == "queued" || run.Status == "running" {
		fmt.Println(dimStyle.Render("The run isn't over yet: lines still going have nothing to download."))
	}
	return nil
}

// pickLines is the lines of a run that wanted names, or all of them.
func pickLines(run *maestroapi.Run, wanted []int64) ([]maestroapi.RunLine, error) {
	if len(wanted) == 0 {
		return run.Detail, nil
	}
	var lines []maestroapi.RunLine
	for _, n := range wanted {
		at := slices.IndexFunc(run.Detail, func(l maestroapi.RunLine) bool { return l.Line == n })
		if at < 0 {
			return nil, fmt.Errorf("run %d has no line %d", run.ID, n)
		}
		lines = append(lines, run.Detail[at])
	}
	return lines, nil
}

func countFiles(lines []maestroapi.RunLine) int {
	n := 0
	for _, l := range lines {
		for _, f := range l.Files {
			if !f.Missing {
				n++
			}
		}
	}
	return n
}

// progress counts files downloaded on a terminal's line, which it rewrites;
// anywhere else it says nothing.
type progress struct {
	out          io.Writer
	shown, total int
	on           bool
}

func newProgress(out *os.File, total int) *progress {
	return &progress{out: out, total: total, on: total > 1 && term.IsTerminal(int(out.Fd()))}
}

func (p *progress) step() {
	p.shown++
	if p.on {
		fmt.Fprintf(p.out, "\r%d/%d files", p.shown, p.total)
	}
}

// done clears the count, once.
func (p *progress) done() {
	if p.on {
		fmt.Fprint(p.out, "\r\x1b[K")
		p.on = false
	}
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
