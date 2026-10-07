package cmd

import (
	"encoding/json"
	"net/http"
	"net/http/httptest"
	"os"
	"path/filepath"
	"reflect"
	"testing"

	"github.com/spf13/cobra"
)

// orq answers what completions ask of it, for a user whose key is "k".
func fakeOrq(t *testing.T) {
	t.Helper()
	srv := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		if r.Header.Get("Authorization") != "Bearer k" {
			w.WriteHeader(http.StatusUnauthorized)
			return
		}
		switch r.URL.Path {
		case "/api/images":
			json.NewEncoder(w).Encode([]map[string]any{{"build_id": 12, "repo": "o/r", "ref": "main"}, {"build_id": 9, "repo": "o/s", "ref": "dev"}})
		case "/api/runs":
			json.NewEncoder(w).Encode([]map[string]any{{"id": 31, "status": "running", "repo": "o/r", "ref": "main"}, {"id": 4, "status": "succeeded", "repo": "o/s", "ref": "dev"}})
		case "/api/runs/31":
			json.NewEncoder(w).Encode(map[string]any{"id": 31, "line_detail": []map[string]any{
				{"line": 1, "status": "running", "args": []string{"-t", "10"}},
				{"line": 2, "status": "queued", "args": []string{"-t", "20"}},
			}})
		default:
			w.WriteHeader(http.StatusNotFound)
		}
	}))
	t.Cleanup(srv.Close)
	t.Setenv("MAESTRO_ORQ_URL", srv.URL)
}

// signedIn gives the CLI a stored Maestro key; without it, none.
func signedIn(t *testing.T, key string) {
	t.Helper()
	dir := t.TempDir()
	t.Setenv("XDG_CONFIG_HOME", dir)
	t.Setenv("MAESTRO_FORCE_FILE_STORE", "1")
	if key != "" {
		os.MkdirAll(filepath.Join(dir, "maestro"), 0o755)
		os.WriteFile(filepath.Join(dir, "maestro", "config.json"), []byte(`{"maestro_key":"`+key+`"}`), 0o600)
	}
}

func TestCompletions(t *testing.T) {
	fakeOrq(t)
	signedIn(t, "k")
	const keep = cobra.ShellCompDirectiveNoFileComp | cobra.ShellCompDirectiveKeepOrder

	cases := []struct {
		name       string
		complete   func(*cobra.Command, []string, string) ([]string, cobra.ShellCompDirective)
		args       []string
		toComplete string
		want       []string
		directive  cobra.ShellCompDirective
	}{
		{"builds", completeBuilds, nil, "", []string{"12\to/r@main", "9\to/s@dev"}, keep},
		{"builds by prefix, with #", completeBuilds, nil, "#1", []string{"12\to/r@main"}, keep},
		{"builds, already given", completeBuilds, []string{"12"}, "", nil, cobra.ShellCompDirectiveNoFileComp},
		{"runs", completeRuns, nil, "", []string{"31\trunning: o/r@main", "4\tsucceeded: o/s@dev"}, keep},
		{"run lines: the run first", completeRunLines, nil, "3", []string{"31\trunning: o/r@main"}, keep},
		{"run lines: then its lines", completeRunLines, []string{"31"}, "", []string{"1\trunning: -t 10", "2\tqueued: -t 20"}, keep},
		{"run lines: no third", completeRunLines, []string{"31", "1"}, "", nil, cobra.ShellCompDirectiveNoFileComp},
		{"run, then a directory", completeRunThenDir, []string{"31"}, "", nil, cobra.ShellCompDirectiveFilterDirs},
	}
	for _, c := range cases {
		t.Run(c.name, func(t *testing.T) {
			got, directive := c.complete(nil, c.args, c.toComplete)
			if !reflect.DeepEqual(got, c.want) || directive != c.directive {
				t.Errorf("got %q (%v), want %q (%v)", got, directive, c.want, c.directive)
			}
		})
	}
}

func TestCompletionsWithoutAKey(t *testing.T) {
	fakeOrq(t)
	signedIn(t, "")
	if got, _ := completeBuilds(nil, nil, ""); got != nil {
		t.Errorf("offered %q without a key", got)
	}
}

// Completing must not demand a key: `maestro forges rm <TAB>` works
// without one, and a completion that needs the key finds out by itself.
func TestCompletionRequestsSkipTheKeyCheck(t *testing.T) {
	signedIn(t, "")
	for _, name := range []string{cobra.ShellCompRequestCmd, cobra.ShellCompNoDescRequestCmd} {
		if err := rootCmd.PersistentPreRunE(&cobra.Command{Use: name}, nil); err != nil {
			t.Errorf("%s: %v", name, err)
		}
	}
}
