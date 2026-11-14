package cmd

import (
	"bytes"
	"context"
	"encoding/json"
	"errors"
	"fmt"
	"net/http"
	"net/http/httptest"
	"os"
	"path/filepath"
	"strings"
	"sync"
	"testing"
	"time"

	"github.com/charmbracelet/x/ansi"

	"maestro-cli/internal/maestroapi"
)

// fakeLineOrq serves run 31 of one line, whose status goes through
// statuses, one per time it is asked for, the last one staying. Its logs
// say whether they were asked to follow; a server that follows answers
// "followed", one from before follow answers "so far".
func fakeLineOrq(t *testing.T, follows bool, statuses ...string) *[]string {
	t.Helper()
	var mu sync.Mutex
	var asked []string
	at := 0
	srv := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		mu.Lock()
		defer mu.Unlock()
		asked = append(asked, r.URL.RequestURI())
		switch {
		case r.URL.Path == "/api/runs/31":
			status := statuses[min(at, len(statuses)-1)]
			at++
			json.NewEncoder(w).Encode(map[string]any{"id": 31, "status": "running", "line_detail": []map[string]any{
				{"line": 1, "status": status, "started_at": "2026-11-14T10:00:00Z", "logs": []string{"stdout", "stderr"}},
			}})
		case strings.HasSuffix(r.URL.Path, "/logs"):
			q := r.URL.Query()
			if q.Get("follow") == "true" && follows {
				fmt.Fprintf(w, "followed %s\n", q.Get("stream"))
				return
			}
			fmt.Fprintf(w, "so far %s\n", q.Get("stream"))
		default:
			http.NotFound(w, r)
		}
	}))
	t.Cleanup(srv.Close)
	t.Setenv("MAESTRO_ORQ_URL", srv.URL)
	maestroKey = "k"
	followPoll = time.Millisecond
	t.Cleanup(func() { maestroKey, followPoll = "", 2*time.Second })
	return &asked
}

// -f waits for a line to start, says so once per status, then follows
// both streams to where they belong.
func TestFollowWaitsThenFollows(t *testing.T) {
	asked := fakeLineOrq(t, true, "queued", "queued", "starting", "running", "succeeded")
	run, _ := maestroapi.GetRun(context.Background(), "k", 31)
	var stdout, stderr bytes.Buffer
	if err := followLine(context.Background(), 31, run.Detail[0], &stdout, &stderr); err != nil {
		t.Fatal(err)
	}
	if stdout.String() != "followed stdout\n" || !strings.Contains(stderr.String(), "followed stderr\n") {
		t.Errorf("stdout %q, stderr %q", stdout.String(), stderr.String())
	}
	said := ansi.Strip(stderr.String())
	if strings.Count(said, "Line 1 is queued; waiting") != 1 || strings.Count(said, "Line 1 is starting; waiting") != 1 {
		t.Errorf("said while waiting: %q", said)
	}
	if strings.Contains(said, "can't follow") {
		t.Errorf("took a server that follows for one that doesn't: %q", said)
	}
	followed := 0
	for _, uri := range *asked {
		if strings.Contains(uri, "follow=true") {
			followed++
		}
	}
	if followed != 2 {
		t.Errorf("asked to follow %d stream(s), want 2: %q", followed, *asked)
	}
}

// A server from before follow answers what there is and ends while the line
// still runs: that is said, not taken for the end of the line.
func TestFollowOnAServerThatCannot(t *testing.T) {
	fakeLineOrq(t, false, "running")
	run, _ := maestroapi.GetRun(context.Background(), "k", 31)
	var stdout, stderr bytes.Buffer
	if err := followLine(context.Background(), 31, run.Detail[0], &stdout, &stderr); err != nil {
		t.Fatal(err)
	}
	if stdout.String() != "so far stdout\n" || !strings.Contains(ansi.Strip(stderr.String()), "This Maestro server can't follow logs") {
		t.Errorf("stdout %q, stderr %q", stdout.String(), stderr.String())
	}
}

// A line over already prints what was kept, without following.
func TestFollowALineThatIsOver(t *testing.T) {
	asked := fakeLineOrq(t, true, "failed")
	run, _ := maestroapi.GetRun(context.Background(), "k", 31)
	var stdout, stderr bytes.Buffer
	if err := followLine(context.Background(), 31, run.Detail[0], &stdout, &stderr); err != nil || stdout.String() != "so far stdout\n" {
		t.Errorf("stdout %q, %v", stdout.String(), err)
	}
	for _, uri := range *asked {
		if strings.Contains(uri, "follow=true") {
			t.Errorf("asked to follow a line that is over: %s", uri)
		}
	}
}

// A run of one line needs no line named; one of several says which there are.
func TestPickLine(t *testing.T) {
	one := &maestroapi.Run{ID: 31, Detail: []maestroapi.RunLine{{Line: 1}}}
	if l, err := pickLine(one, nil); err != nil || l.Line != 1 {
		t.Errorf("the only line: %+v, %v", l, err)
	}
	several := &maestroapi.Run{ID: 31, Detail: []maestroapi.RunLine{{Line: 1}, {Line: 2}, {Line: 8}}}
	_, err := pickLine(several, nil)
	var exit *exitError
	if !errors.As(err, &exit) || exit.code != exitUsage || !strings.Contains(err.Error(), "run 31 has 3 lines (1–8)") {
		t.Errorf("several lines: %v", err)
	}
	if l, err := pickLine(several, []string{"2"}); err != nil || l.Line != 2 {
		t.Errorf("line 2: %+v, %v", l, err)
	}
	if _, err := pickLine(several, []string{"5"}); err == nil || !strings.Contains(err.Error(), "no line 5") {
		t.Errorf("line 5: %v", err)
	}
}

// --status, --repo and --limit pick from the list, and the list says when
// there are more than it shows.
func TestRunsListPicks(t *testing.T) {
	t.Cleanup(func() { runsLimit, runsStatus, runsRepo = 20, "", "" })
	var runs []maestroapi.Run
	for i := 10; i > 0; i-- {
		status, repo := "succeeded", "lab/sim"
		if i%3 == 0 {
			status = "failed"
		}
		if i%2 == 0 {
			repo = "lab/fit"
		}
		runs = append(runs, maestroapi.Run{ID: int64(i), Status: status, Repo: repo, CreatedAt: time.Now().Add(-time.Duration(i) * time.Hour)})
	}

	runsLimit, runsStatus, runsRepo = 20, "failed", ""
	if shown, matching := pickRuns(runs); matching != 3 || len(shown) != 3 || shown[0].ID != 9 {
		t.Errorf("--status failed: %d of %d, first %d", len(shown), matching, shown[0].ID)
	}
	runsStatus, runsRepo = "", "fit"
	if shown, matching := pickRuns(runs); matching != 5 || shown[0].Repo != "lab/fit" {
		t.Errorf("--repo fit: %d matching, %+v", matching, shown)
	}
	runsRepo, runsLimit = "", 4
	shown, matching := pickRuns(runs)
	if len(shown) != 4 || matching != 10 || shown[0].ID != 10 {
		t.Errorf("--limit 4: %d of %d", len(shown), matching)
	}
	var out bytes.Buffer
	if err := writeRunsList(&out, shown, false, matching); err != nil {
		t.Fatal(err)
	}
	if got := ansi.Strip(out.String()); !strings.Contains(got, "The newest 4 of 10 runs; --limit 0 for all.") || !strings.Contains(got, "10h ago") {
		t.Errorf("printed %q", got)
	}

	runsStatus = "cancelled"
	out.Reset()
	shown, matching = pickRuns(runs)
	writeRunsList(&out, shown, false, matching)
	if out.String() != "No runs match.\n" {
		t.Errorf("nothing matching: %q", out.String())
	}
}

func TestRunsShowFailed(t *testing.T) {
	run := &maestroapi.Run{ID: 31, Status: "failed", Detail: []maestroapi.RunLine{
		{Line: 1, Status: "succeeded"}, {Line: 2, Status: "failed", Args: []string{"-t", "2"}}, {Line: 3, Status: "error", Error: "no room"},
	}}
	var out bytes.Buffer
	if err := writeRunShow(&out, run, true); err != nil {
		t.Fatal(err)
	}
	got := out.String()
	if strings.Contains(got, "\n1  ") || !strings.Contains(got, "-t 2") || !strings.Contains(got, "line 3: no room") || !strings.Contains(got, "3 line(s): 1 succeeded, 1 failed, 1 error") {
		t.Errorf("--failed showed:\n%s", got)
	}

	run.Detail = run.Detail[:1]
	out.Reset()
	writeRunShow(&out, run, true)
	if !strings.HasSuffix(out.String(), "No line failed.\n") {
		t.Errorf("none failed: %q", out.String())
	}
}

// A run is downloaded over one already there only with --force, and
// --line downloads only the lines it names.
func TestRunsGetLinesAndForce(t *testing.T) {
	srv := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		switch {
		case r.URL.Path == "/api/runs/7":
			fmt.Fprint(w, `{"id": 7, "status": "succeeded", "line_detail": [
				{"line": 1, "status": "succeeded", "logs": [], "files": [{"path": "a.txt"}]},
				{"line": 2, "status": "succeeded", "logs": [], "files": [{"path": "b.txt"}]}]}`)
		case strings.Contains(r.URL.Path, "/files/"):
			fmt.Fprint(w, "content")
		default:
			http.NotFound(w, r)
		}
	}))
	defer srv.Close()
	t.Setenv("MAESTRO_ORQ_URL", srv.URL)
	maestroKey = "k"
	t.Cleanup(func() { maestroKey, runsGetLines, runsGetForce = "", nil, false })

	dir := t.TempDir()
	runsGetLines = []int64{2}
	if _, err := stdout(t, func() error { return runRunsGet(background(), []string{"7", dir}) }); err != nil {
		t.Fatal(err)
	}
	if _, err := os.Stat(filepath.Join(dir, "7/2/b.txt")); err != nil {
		t.Error(err)
	}
	if _, err := os.Stat(filepath.Join(dir, "7/1")); err == nil {
		t.Error("downloaded line 1, which --line left out")
	}

	if err := runRunsGet(background(), []string{"7", dir}); err == nil || !strings.Contains(err.Error(), "--force") {
		t.Errorf("again without --force: %v", err)
	}
	runsGetForce, runsGetLines = true, nil
	if _, err := stdout(t, func() error { return runRunsGet(background(), []string{"7", dir}) }); err != nil {
		t.Errorf("again with --force: %v", err)
	}
	runsGetLines = []int64{9}
	if err := runRunsGet(background(), []string{"7", dir}); err == nil || !strings.Contains(err.Error(), "no line 9") {
		t.Errorf("a line that isn't there: %v", err)
	}
}
