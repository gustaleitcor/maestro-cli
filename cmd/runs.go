package cmd

import (
	"fmt"
	"os"
	"path/filepath"
	"strconv"
	"strings"
	"text/tabwriter"

	"github.com/spf13/cobra"

	"maestro-cli/internal/maestroapi"
)

var runsLogsTail int

var runsCmd = &cobra.Command{
	Use:   "runs",
	Short: "Commands for the runs started with `maestro run`",
}

var runsListCmd = &cobra.Command{
	Use:               "list",
	Short:             "List your runs",
	Args:              cobra.NoArgs,
	ValidArgsFunction: cobra.NoFileCompletions,
	RunE:              runRunsList,
}

var runsShowCmd = &cobra.Command{
	Use:               "show <run>",
	Short:             "Show each line of a run: where it ran, how it ended, what it kept",
	Args:              cobra.ExactArgs(1),
	ValidArgsFunction: completeRuns,
	RunE:              runRunsShow,
}

var runsLogsCmd = &cobra.Command{
	Use:               "logs <run> <line>",
	Short:             "Print what a line's container wrote to stdout and stderr",
	Args:              cobra.ExactArgs(2),
	ValidArgsFunction: completeRunLines,
	RunE:              runRunsLogs,
}

var runsGetCmd = &cobra.Command{
	Use:   "get <run> [dir]",
	Short: "Download the files kept from a run",
	Long: `Downloads every file kept from a run's containers, and their logs, into
<dir>/<run>/<line>/ (dir defaults to the current directory).`,
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
	Short:             "Delete a finished run and the files kept from it",
	Args:              cobra.ExactArgs(1),
	ValidArgsFunction: completeRuns,
	RunE:              runRunsRm,
}

func init() {
	runsLogsCmd.Flags().IntVar(&runsLogsTail, "tail", -1, "only the last N lines, while the container runs")
	rootCmd.AddCommand(runsCmd)
	runsCmd.AddCommand(runsListCmd, runsShowCmd, runsLogsCmd, runsGetCmd, runsCancelCmd, runsRmCmd)
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

func runRunsList(cmd *cobra.Command, args []string) error {
	runs, err := maestroapi.ListRuns(cmd.Context(), maestroKey)
	if err != nil {
		return err
	}
	if len(runs) == 0 {
		fmt.Println("No runs yet. Start one with: maestro run <build>")
		return nil
	}
	w := tabwriter.NewWriter(os.Stdout, 0, 0, 2, ' ', 0)
	fmt.Fprintln(w, "RUN\tREPO\tBUILD\tSTATUS\tLINES\tCREATED")
	for _, r := range runs {
		fmt.Fprintf(w, "%d\t%s\t#%d\t%s\t%s\t%s\n", r.ID, r.Repo+"@"+r.Ref, r.BuildID, r.Status, lineCounts(r.Lines), r.CreatedAt.Local().Format("2006-01-02 15:04"))
	}
	return w.Flush()
}

func runRunsShow(cmd *cobra.Command, args []string) error {
	id, err := parseID(args[0], "run")
	if err != nil {
		return err
	}
	run, err := maestroapi.GetRun(cmd.Context(), maestroKey, id)
	if err != nil {
		return err
	}

	where := "any machine"
	if len(run.Machines) > 0 {
		where = strings.Join(run.Machines, ", ")
	}
	fmt.Printf("Run %d of %s@%s (build #%d): %s, on %s\n\n", run.ID, run.Repo, run.Ref, run.BuildID, run.Status, where)

	w := tabwriter.NewWriter(os.Stdout, 0, 0, 2, ' ', 0)
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
			fmt.Printf("\nline %d: %s", l.Line, l.Error)
		}
	}
	fmt.Println()
	return nil
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
	return maestroapi.RunLogs(cmd.Context(), maestroKey, id, line, runsLogsTail, os.Stdout)
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
		// A line that never started printed nothing.
		if !l.StartedAt.IsZero() {
			if err := save(filepath.Join(runDir, strconv.FormatInt(l.Line, 10)+".log"), func(f *os.File) error {
				return maestroapi.RunLogs(cmd.Context(), maestroKey, run.ID, l.Line, -1, f)
			}); err != nil {
				return err
			}
		}
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
