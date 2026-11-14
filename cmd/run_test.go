package cmd

import (
	"bytes"
	"encoding/json"
	"errors"
	"fmt"
	"net/http"
	"net/http/httptest"
	"strings"
	"testing"

	"github.com/charmbracelet/x/ansi"

	"maestro-cli/internal/maestroapi"
	"maestro-cli/tui"
)

// A script reads how maestro ended from its exit status.
func TestReportExitStatus(t *testing.T) {
	cases := []struct {
		err      error
		code     int
		printed  string
		silenced bool
	}{
		{errors.New("could not reach orq"), exitFailed, "Error: could not reach orq", false},
		{&exitError{exitUsage, errors.New("run needs a build")}, exitUsage, "Error: run needs a build", false},
		{fmt.Errorf("watching: %w", tui.ErrInterrupted), exitInterrupted, "", true},
	}
	for _, c := range cases {
		var w bytes.Buffer
		if code := report(&w, rootCmd, c.err); code != c.code {
			t.Errorf("%v: exit %d, want %d", c.err, code, c.code)
		}
		got := strings.TrimSpace(ansi.Strip(w.String()))
		if c.silenced && got != "" || !c.silenced && !strings.HasPrefix(got, c.printed) {
			t.Errorf("%v: printed %q, want %q", c.err, got, c.printed)
		}
	}
}

// fakeRunOrq answers a run started and asked for, as status says it ended.
func fakeRunOrq(t *testing.T, status string) {
	t.Helper()
	srv := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		run := map[string]any{"id": 31, "build_id": 12, "repo": "o/r", "ref": "main", "status": status,
			"lines":       map[string]int{"failed": 1, "succeeded": 2},
			"line_detail": []map[string]any{{"line": 1, "status": "failed"}, {"line": 2, "status": "succeeded"}, {"line": 3, "status": "succeeded"}}}
		switch {
		case r.Method == http.MethodPost && r.URL.Path == "/api/runs":
			run["status"] = "queued"
			w.WriteHeader(http.StatusCreated)
			json.NewEncoder(w).Encode(run)
		case r.URL.Path == "/api/runs/31":
			json.NewEncoder(w).Encode(run)
		default:
			w.WriteHeader(http.StatusNotFound)
		}
	}))
	t.Cleanup(srv.Close)
	t.Setenv("MAESTRO_ORQ_URL", srv.URL)
}

// A run watched to its end that failed fails the command; one that
// succeeded doesn't, and showing a run without watching it never does.
func TestWatchedRunExitsAsItEnded(t *testing.T) {
	runsFlags(t, false, false, false)
	fakeRunOrq(t, "failed")
	_, err := stdout(t, func() error { return showRun(background().Context(), 31, true) })
	var exit *exitError
	if !errors.As(err, &exit) || exit.code != exitFailed || !strings.Contains(err.Error(), "run 31 failed: 2 succeeded, 1 failed") {
		t.Errorf("watched failed run: %v", err)
	}
	if _, err := stdout(t, func() error { return showRun(background().Context(), 31, false) }); err != nil {
		t.Errorf("shown, not watched: %v", err)
	}

	fakeRunOrq(t, "succeeded")
	if _, err := stdout(t, func() error { return showRun(background().Context(), 31, true) }); err != nil {
		t.Errorf("watched run that succeeded: %v", err)
	}
}

// A file sent from here is named as the flag that named it.
func TestLocalFileErrorsNameTheFlag(t *testing.T) {
	dir := t.TempDir()
	cases := []struct {
		flags localFlags
		want  string
	}{
		{localFlags{config: dir + "/nope.toml"}, "-c " + dir + "/nope.toml: no such file"},
		{localFlags{generator: dir}, "-g " + dir + ": not a file"},
	}
	for _, c := range cases {
		if _, err := readLocalFiles(c.flags); err == nil || err.Error() != c.want {
			t.Errorf("%+v: %v; want %q", c.flags, err, c.want)
		}
	}
}

// With -q or --json, stdout has the answer and nothing else, for a script.
func TestRunPrintsOnlyTheAnswerForScripts(t *testing.T) {
	fakeRunOrq(t, "queued")
	runsFlags(t, false, false, false)
	t.Cleanup(func() { runsQuiet = false })

	runsQuiet = true
	got, err := stdout(t, func() error { return runRun(background(), []string{"12"}) })
	if err != nil || got != "31\n" {
		t.Errorf("-q printed %q, %v; want only 31", got, err)
	}

	runsQuiet, runsAsJSON = false, true
	got, err = stdout(t, func() error { return runRun(background(), []string{"#12"}) })
	var run maestroapi.Run
	if err != nil || json.Unmarshal([]byte(got), &run) != nil || run.ID != 31 || run.BuildID != 12 {
		t.Errorf("--json printed %q, %v", got, err)
	}

	runsAsJSON = false
	got, err = stdout(t, func() error { return runRun(background(), []string{"12"}) })
	if err != nil || !strings.Contains(got, "Queued run 31 of build #12") {
		t.Errorf("without either, printed %q, %v", got, err)
	}

	runsQuiet, runsWatch = true, true
	if err := runRun(background(), []string{"12"}); err == nil || !strings.Contains(err.Error(), "-q") {
		t.Errorf("-q with --watch: %v", err)
	}
}
