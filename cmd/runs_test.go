package cmd

import (
	"bytes"
	"context"
	"encoding/json"
	"fmt"
	"io"
	"net/http"
	"net/http/httptest"
	"os"
	"path/filepath"
	"reflect"
	"strings"
	"testing"
	"time"

	"github.com/spf13/cobra"

	"maestro-cli/internal/maestroapi"
)

func TestWriteRunsList(t *testing.T) {
	var out bytes.Buffer
	if err := writeRunsList(&out, nil, false, 0); err != nil || !strings.Contains(out.String(), "No runs yet") {
		t.Errorf("no runs: %q, %v", out.String(), err)
	}

	out.Reset()
	created := time.Date(2026, 10, 7, 9, 30, 0, 0, time.Local)
	err := writeRunsList(&out, []maestroapi.Run{
		{ID: 31, Repo: "lab/sim", Ref: "main", BuildID: 12, Status: "running", Lines: map[string]int{"running": 2, "queued": 1, "succeeded": 4}, CreatedAt: created},
		{ID: 4, Repo: "lab/fit", Ref: "v2", BuildID: 9, Status: "failed", Lines: map[string]int{"failed": 1}, CreatedAt: created},
	}, false, 2)
	if err != nil {
		t.Fatal(err)
	}
	wantTable(t, out.String(), [][]string{
		{"RUN", "REPO", "BUILD", "STATUS", "LINES", "CREATED"},
		{"31", "lab/sim@main", "#12", "running", "1 queued, 2 running, 4 succeeded", "2026-10-07"},
		{"4", "lab/fit@v2", "#9", "failed", "1 failed"},
	})
	if strings.Contains(out.String(), "USER") {
		t.Errorf("says whose the caller's own runs are: %q", out.String())
	}
}

// With everyone's runs, each says whose it is: by name, or by email for
// someone the provider gave no name.
func TestWriteRunsListOfEveryone(t *testing.T) {
	ana := maestroapi.Run{ID: 31, Repo: "lab/sim", Ref: "main", BuildID: 12, Status: "running"}
	ana.User.Name, ana.User.Email = "Ana", "ana@example.com"
	bob := maestroapi.Run{ID: 4, Repo: "lab/fit", Ref: "v2", BuildID: 9, Status: "failed"}
	bob.User.Email = "bob@example.com"

	var out bytes.Buffer
	if err := writeRunsList(&out, []maestroapi.Run{ana, bob}, true, 2); err != nil {
		t.Fatal(err)
	}
	wantTable(t, out.String(), [][]string{
		{"RUN", "USER", "REPO", "BUILD", "STATUS", "LINES", "CREATED"},
		{"31", "Ana", "lab/sim@main", "#12", "running"},
		{"4", "bob@example.com", "lab/fit@v2", "#9", "failed"},
	})
}

// wantTable checks that got has exactly these rows, each with its fields in
// this order.
func wantTable(t *testing.T, got string, rows [][]string) {
	t.Helper()
	lines := strings.Split(strings.TrimSpace(got), "\n")
	if len(lines) != len(rows) {
		t.Fatalf("wrote %d lines, want %d: %q", len(lines), len(rows), got)
	}
	for i, want := range rows {
		at := 0
		for _, field := range want {
			found := strings.Index(lines[i][at:], field)
			if found < 0 {
				t.Errorf("line %d lacks %q, or has it out of place: %q", i, field, lines[i])
				break
			}
			at += found + len(field)
		}
	}
}

// stdout is what f printed.
func stdout(t *testing.T, f func() error) (string, error) {
	t.Helper()
	r, w, err := os.Pipe()
	if err != nil {
		t.Fatal(err)
	}
	real := os.Stdout
	os.Stdout = w
	printed := make(chan string)
	go func() {
		all, _ := io.ReadAll(r)
		printed <- string(all)
	}()
	err = f()
	os.Stdout = real
	w.Close()
	return <-printed, err
}

// runsFlags sets what the flags of runs list and runs show would, for one test.
func runsFlags(t *testing.T, watch, asJSON, all bool) {
	t.Helper()
	t.Cleanup(func() { runsWatch, runsAsJSON, runsAll, maestroKey = false, false, false, "" })
	runsWatch, runsAsJSON, runsAll, maestroKey = watch, asJSON, all, "k"
}

func background() *cobra.Command {
	cmd := &cobra.Command{}
	cmd.SetContext(context.Background())
	return cmd
}

func TestRunsListAll(t *testing.T) {
	var asked []string
	srv := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		asked = append(asked, r.URL.RequestURI())
		fmt.Fprint(w, `[{"id": 31, "user": {"name": "Ana"}, "repo": "o/r", "ref": "main", "status": "running"}]`)
	}))
	defer srv.Close()
	t.Setenv("MAESTRO_ORQ_URL", srv.URL)

	runsFlags(t, false, false, true)
	got, err := stdout(t, func() error { return runRunsList(background(), nil) })
	if err != nil || !strings.Contains(got, "USER") || !strings.Contains(got, "Ana") {
		t.Errorf("--all printed %q, %v", got, err)
	}

	runsFlags(t, false, true, false)
	got, err = stdout(t, func() error { return runRunsList(background(), nil) })
	var runs []maestroapi.Run
	if err != nil || json.Unmarshal([]byte(got), &runs) != nil || len(runs) != 1 || runs[0].ID != 31 || runs[0].User.Name != "Ana" {
		t.Errorf("--json printed %q, %v", got, err)
	}

	if !reflect.DeepEqual(asked, []string{"/api/runs?all=true", "/api/runs"}) {
		t.Errorf("asked %q", asked)
	}
}

// --json is for a script, which reads one answer; nothing is asked of the
// server before saying so.
func TestJSONWithWatchIsRefused(t *testing.T) {
	asked := 0
	srv := httptest.NewServer(http.HandlerFunc(func(http.ResponseWriter, *http.Request) { asked++ }))
	defer srv.Close()
	t.Setenv("MAESTRO_ORQ_URL", srv.URL)
	runsFlags(t, true, true, false)
	runsInterval = 2 * time.Second

	for name, run := range map[string]func(*cobra.Command, []string) error{"runs list": runRunsList, "runs show": runRunsShow} {
		if err := run(background(), []string{}); err == nil || !strings.Contains(err.Error(), "can't be used with --watch") {
			t.Errorf("%s --json --watch: %v", name, err)
		}
	}
	if asked != 0 {
		t.Errorf("asked the server %d time(s)", asked)
	}
}

func TestRunsShowWithoutARun(t *testing.T) {
	fakeOrq(t)
	runsFlags(t, false, false, false)
	got, err := stdout(t, func() error { return runRunsShow(background(), nil) })
	if err != nil || !strings.HasPrefix(got, "Run 31 ") {
		t.Errorf("showed %q, %v; want the newest, run 31", got, err)
	}

	runsFlags(t, false, true, false)
	got, err = stdout(t, func() error { return runRunsShow(background(), nil) })
	var run maestroapi.Run
	if err != nil || json.Unmarshal([]byte(got), &run) != nil || run.ID != 31 || len(run.Detail) != 2 {
		t.Errorf("--json printed %q, %v", got, err)
	}

	srv := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) { fmt.Fprint(w, "[]") }))
	defer srv.Close()
	t.Setenv("MAESTRO_ORQ_URL", srv.URL)
	if err := runRunsShow(background(), nil); err == nil || !strings.Contains(err.Error(), "No runs yet") {
		t.Errorf("with no runs: %v", err)
	}
}

func TestWriteRunShow(t *testing.T) {
	two := 2
	var out bytes.Buffer
	err := writeRunShow(&out, &maestroapi.Run{
		ID: 31, Repo: "lab/sim", Ref: "main", BuildID: 12, Status: "running", Machines: []string{"Q1", "Q2"},
		Detail: []maestroapi.RunLine{
			{Line: 1, Args: []string{"-t", "10", "long run"}, Status: "succeeded", Machine: "Q1", Files: []maestroapi.RunFile{{Path: "out.txt"}, {Path: "plot.png", Missing: true}}},
			{Line: 2, Args: []string{"-t", "20"}, Status: "failed", Machine: "Q2", ExitCode: &two, Error: "logs cut at the 10 MB limit"},
			{Line: 3, Status: "queued"},
		},
	}, false)
	if err != nil {
		t.Fatal(err)
	}
	got := out.String()
	for _, want := range []string{
		"Run 31 of lab/sim@main (build #12): running, on Q1, Q2\n3 line(s): 1 queued, 1 succeeded, 1 failed\n\n",
		"LINE  PARAMETERS", "-t 10 'long run'", "out.txt, plot.png (missing)", "failed (exit 2)", "(image's command)",
		"\nline 2: logs cut at the 10 MB limit\n",
	} {
		if !strings.Contains(got, want) {
			t.Errorf("lacks %q in:\n%s", want, got)
		}
	}
	if !strings.HasSuffix(got, "\n") || strings.HasSuffix(got, "\n\n\n") {
		t.Errorf("ends oddly: %q", got)
	}

	out.Reset()
	writeRunShow(&out, &maestroapi.Run{ID: 5, Status: "queued"}, false)
	if !strings.Contains(out.String(), "on any machine") {
		t.Errorf("a run that names no machine: %q", out.String())
	}
}

// A line's stdout goes to stdout and its stderr to stderr; one kept before
// the two were apart, or by a server that doesn't know them apart, goes to
// stdout whole.
func TestWriteLineLogs(t *testing.T) {
	var asked []string
	srv := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		asked = append(asked, r.URL.RequestURI())
		fmt.Fprintf(w, "<%s>", r.URL.Query().Get("stream"))
	}))
	defer srv.Close()
	t.Setenv("MAESTRO_ORQ_URL", srv.URL)
	started := time.Now()

	cases := []struct {
		name           string
		line           maestroapi.RunLine
		stdout, stderr string
	}{
		{"kept apart", maestroapi.RunLine{Line: 2, StartedAt: started, Logs: []string{"stdout", "stderr"}}, "<stdout>", "<stderr>"},
		{"kept together", maestroapi.RunLine{Line: 2, StartedAt: started, Logs: []string{"combined"}}, "<combined>", ""},
		{"an older server", maestroapi.RunLine{Line: 2, StartedAt: started}, "<combined>", ""},
		{"never started", maestroapi.RunLine{Line: 2, Logs: []string{}}, "", ""},
		{"never started, older server", maestroapi.RunLine{Line: 2}, "", ""},
	}
	for _, c := range cases {
		var stdout, stderr bytes.Buffer
		if err := writeLineLogs(context.Background(), 31, c.line, 5, false, &stdout, &stderr); err != nil {
			t.Fatalf("%s: %v", c.name, err)
		}
		if stdout.String() != c.stdout || stderr.String() != c.stderr {
			t.Errorf("%s: stdout %q, stderr %q; want %q, %q", c.name, stdout.String(), stderr.String(), c.stdout, c.stderr)
		}
	}
	if len(asked) == 0 || asked[0] != "/api/runs/31/lines/2/logs?stream=stdout&tail=5" {
		t.Errorf("asked %q", asked)
	}
	for stream, want := range map[string]string{"stdout": "2/stdout.log", "stderr": "2/stderr.log", "combined": "2.log"} {
		if got := logFileName(2, stream); got != want {
			t.Errorf("the %s of line 2 is kept as %q, want %q", stream, got, want)
		}
	}
}

func TestLineCountsSayEveryLevelALineWaitsAt(t *testing.T) {
	got := lineCounts(map[string]int{"queued": 4, "staged": 2, "assigned": 3, "running": 1})
	if want := "4 queued, 2 staged, 3 assigned, 1 running"; got != want {
		t.Errorf("lineCounts = %q, want %q", got, want)
	}
}

// A run whose outputs are shared downloads one tree of them, each file
// once however many lines wrote it, beside each line's logs.
func TestRunsGetShared(t *testing.T) {
	var asked []string
	srv := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		asked = append(asked, r.URL.Path)
		switch {
		case r.URL.Path == "/api/runs/7":
			fmt.Fprint(w, `{"id": 7, "status": "succeeded", "outputs_layout": "shared", "line_detail": [
				{"line": 1, "status": "succeeded", "started_at": "2026-11-14T10:00:00Z", "logs": ["stdout", "stderr"],
				 "files": [{"path": "results/1/x.txt"}, {"path": "results/summary.txt"}]},
				{"line": 2, "status": "succeeded", "started_at": "2026-11-14T10:00:00Z", "logs": ["stdout", "stderr"],
				 "files": [{"path": "results/2/x.txt"}, {"path": "results/summary.txt"}]}]}`)
		case strings.Contains(r.URL.Path, "/files/"):
			fmt.Fprint(w, "content of "+r.URL.Path)
		case strings.HasSuffix(r.URL.Path, "/logs"):
			fmt.Fprint(w, "log")
		default:
			http.NotFound(w, r)
		}
	}))
	defer srv.Close()
	t.Setenv("MAESTRO_ORQ_URL", srv.URL)
	maestroKey = "k"
	t.Cleanup(func() { maestroKey = "" })

	dir := t.TempDir()
	if err := runRunsGet(background(), []string{"7", dir}); err != nil {
		t.Fatal(err)
	}
	for _, rel := range []string{"7/shared/results/1/x.txt", "7/shared/results/2/x.txt", "7/shared/results/summary.txt", "7/1/stdout.log", "7/2/stderr.log"} {
		if _, err := os.Stat(filepath.Join(dir, rel)); err != nil {
			t.Errorf("%s: %v", rel, err)
		}
	}
	if _, err := os.Stat(filepath.Join(dir, "7/1/results")); err == nil {
		t.Error("a shared output was put under a line")
	}
	summaries := 0
	for _, p := range asked {
		if strings.HasSuffix(p, "/files/results/summary.txt") {
			summaries++
		}
	}
	if summaries != 1 {
		t.Errorf("summary.txt downloaded %d times, want once", summaries)
	}
}
