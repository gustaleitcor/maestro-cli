package cmd

import (
	"bufio"
	"io"
	"net/http"
	"net/http/httptest"
	"strings"
	"testing"

	"github.com/charmbracelet/x/ansi"
	"github.com/spf13/cobra"
	"github.com/spf13/pflag"

	"maestro-cli/internal/config"
	"maestro-cli/internal/maestroapi"
)

// maestro runs the way it would from a shell: args, then what it exits
// with and says on stderr. Every flag goes back to its default after.
func maestro(t *testing.T, args ...string) (int, string) {
	t.Helper()
	resetFlags(rootCmd)
	t.Cleanup(func() { resetFlags(rootCmd) })
	rootCmd.SetArgs(args)
	cmd, err := execute()
	if err == nil {
		return 0, ""
	}
	var w strings.Builder
	return report(&w, cmd, err), ansi.Strip(w.String())
}

func resetFlags(c *cobra.Command) {
	for _, set := range []*pflag.FlagSet{c.Flags(), c.PersistentFlags()} {
		set.VisitAll(func(f *pflag.Flag) {
			f.Value.Set(f.DefValue)
			f.Changed = false
		})
	}
	for _, sub := range c.Commands() {
		resetFlags(sub)
	}
}

// With no server named anywhere, it is asked for until one is given: blank
// lines are asked again, and what isn't a URL is said and asked again.
func TestAskServer(t *testing.T) {
	var out strings.Builder
	got, err := askServer(bufio.NewReader(strings.NewReader("\n  \nnot a url\nmaestro.example.org/\n")), &out)
	if err != nil || got != "https://maestro.example.org" {
		t.Errorf("askServer = %q, %v", got, err)
	}
	said := ansi.Strip(out.String())
	if !strings.HasPrefix(said, "No Maestro server is set yet.\n") || strings.Count(said, "Maestro server URL") != 4 || !strings.Contains(said, `"not a url" isn't a server's URL`) {
		t.Errorf("said %q", said)
	}

	out.Reset()
	if _, err := askServer(bufio.NewReader(strings.NewReader("")), &out); err == nil || !strings.Contains(err.Error(), "no Maestro server given") {
		t.Errorf("nothing typed before the end: %v", err)
	}
}

func TestCheckServer(t *testing.T) {
	for raw, want := range map[string]string{
		"maestro.example.org":           "https://maestro.example.org",
		"http://localhost:8080/":        "http://localhost:8080",
		" https://maestro.example.org ": "https://maestro.example.org",
		"ftp://maestro.example.org":     "",
		"https://":                      "",
		"not a url":                     "",
	} {
		got, err := checkServer(raw)
		if got != want || (want == "") != (err != nil) {
			t.Errorf("checkServer(%q) = %q, %v; want %q", raw, got, err, want)
		}
	}
}

// What talks to Maestro needs a server first; with no terminal to ask at,
// it says how to name one. What doesn't talk to it works without.
func TestNoServerSet(t *testing.T) {
	signedIn(t, "")
	t.Setenv("MAESTRO_ORQ_URL", "")
	rootCmd.SetOut(io.Discard)
	t.Cleanup(func() { rootCmd.SetOut(nil) })

	for _, args := range [][]string{{}, {"runs"}, {"login"}} {
		code, said := maestro(t, args...)
		if code != exitFailed || !strings.Contains(said, "no Maestro server is set") || !strings.Contains(said, "--server <url>") {
			t.Errorf("maestro %s: exit %d, %q", strings.Join(args, " "), code, said)
		}
	}
	// completion bash is how a package gets its completions, built where no
	// one ever signed in.
	for _, args := range [][]string{{"forges"}, {"forges", "list"}, {"--help"}, {"help", "concepts"}, {"logout", "--yes"},
		{"completion", "bash"}, {"completion", "fish"}, {"completion", "zsh"}} {
		if code, said := maestro(t, args...); code != 0 {
			t.Errorf("maestro %s without a server: exit %d, %q", strings.Join(args, " "), code, said)
		}
	}
}

// Which server is talked to: --server, then MAESTRO_ORQ_URL, then the one
// signed in to, then the default.
func TestServerPrecedence(t *testing.T) {
	signedIn(t, "")
	t.Setenv("MAESTRO_ORQ_URL", "")
	t.Cleanup(func() { serverFlag = "" })

	if url, from := server(); url != "" || from != "" {
		t.Errorf("nothing set: %q from %q; want none, there is no default", url, from)
	}
	if err := config.SaveMaestroKey("https://stored.example.org", "k"); err != nil {
		t.Fatal(err)
	}
	if url, from := server(); url != "https://stored.example.org" || from != "config" {
		t.Errorf("signed in: %q from %q", url, from)
	}
	t.Setenv("MAESTRO_ORQ_URL", "http://localhost:8080/")
	if url, from := server(); url != "http://localhost:8080" || from != "MAESTRO_ORQ_URL" {
		t.Errorf("env: %q from %q", url, from)
	}
	serverFlag = "flag.example.org/"
	if url, from := server(); url != "https://flag.example.org" || from != "--server" {
		t.Errorf("flag: %q from %q", url, from)
	}
	if maestroapi.BaseURL() != "https://flag.example.org" {
		t.Errorf("requests go to %q, not what --server said", maestroapi.BaseURL())
	}
}

// A key minted by one server is never sent to another: --server naming
// another one is told to sign in there, and that server hears nothing.
func TestKeyIsNotSentToAnotherServer(t *testing.T) {
	signedIn(t, "")
	t.Setenv("MAESTRO_ORQ_URL", "")
	asked := 0
	other := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) { asked++ }))
	defer other.Close()
	if err := config.SaveMaestroKey("https://maestro.example.org", "k"); err != nil {
		t.Fatal(err)
	}

	code, said := maestro(t, "runs", "--server", other.URL)
	if code != exitFailed || !strings.Contains(said, "not signed in to "+other.URL) || !strings.Contains(said, "maestro login --server "+other.URL) {
		t.Errorf("exit %d, said %q", code, said)
	}
	if asked != 0 {
		t.Errorf("the other server was asked %d time(s)", asked)
	}
}

// Deleting asks first; with no terminal to ask at, as in a script, only
// --yes deletes, and nothing is asked of the server before.
func TestDeletingWithoutATerminalTakesYes(t *testing.T) {
	var deleted []string
	srv := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		if r.Method == http.MethodDelete {
			deleted = append(deleted, r.URL.Path)
		}
		w.WriteHeader(http.StatusNoContent)
	}))
	defer srv.Close()
	signedIn(t, "k")
	t.Setenv("MAESTRO_ORQ_URL", srv.URL)

	for _, args := range [][]string{{"runs", "rm", "31"}, {"builds", "rm", "12"}} {
		code, said := maestro(t, args...)
		if code != exitUsage || !strings.Contains(said, "add --yes") {
			t.Errorf("maestro %s without --yes: exit %d, %q", strings.Join(args, " "), code, said)
		}
	}
	if len(deleted) != 0 {
		t.Fatalf("deleted %q without --yes", deleted)
	}
	if code, said := maestro(t, "runs", "rm", "31", "--yes"); code != 0 || len(deleted) != 1 || deleted[0] != "/api/runs/31" {
		t.Errorf("with --yes: exit %d, %q, deleted %q", code, said, deleted)
	}
}

// --user takes a number, or an email that picks one user out of everyone's
// usage; anything else is a mistake said before the server is asked.
func TestUsageOfOneUser(t *testing.T) {
	var asked []string
	srv := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		asked = append(asked, r.URL.RequestURI())
		w.Write([]byte(`[{"id": 3, "name": "Ana", "email": "ana@example.org", "days": [{"day": "2026-11-13", "cpu_hours": 2}]},
			{"id": 4, "name": "Bob", "email": "bob@example.org", "days": [{"day": "2026-11-13", "cpu_hours": 5}]}]`))
	}))
	defer srv.Close()
	signedIn(t, "k")
	t.Setenv("MAESTRO_ORQ_URL", srv.URL)

	got, err := stdout(t, func() error {
		_, err := maestroOK(t, "usage", "--user", "ANA@example.org")
		return err
	})
	if err != nil || !strings.Contains(got, "Ana") || strings.Contains(got, "Bob") {
		t.Errorf("--user by email printed %q, %v", got, err)
	}
	if code, said := maestro(t, "usage", "--user", "ana"); code != exitUsage || !strings.Contains(said, "number or email") {
		t.Errorf("--user ana: exit %d, %q", code, said)
	}
	if code, said := maestro(t, "usage", "--user", "zoe@example.org"); code != exitFailed || !strings.Contains(said, "nothing used by zoe@example.org") {
		t.Errorf("--user of no one: exit %d, %q", code, said)
	}
	if len(asked) != 2 || asked[0] != "/api/usage?days=30" {
		t.Errorf("asked %q", asked)
	}
}

// maestroOK is maestro for a command expected to work, with its error.
func maestroOK(t *testing.T, args ...string) (int, error) {
	t.Helper()
	resetFlags(rootCmd)
	t.Cleanup(func() { resetFlags(rootCmd) })
	rootCmd.SetArgs(args)
	_, err := execute()
	return 0, err
}

// maestro --help lists every command under a group: one added without a
// place in commandGroups would fall into a list of its own.
func TestEveryCommandHasAGroup(t *testing.T) {
	rootCmd.SetOut(io.Discard)
	t.Cleanup(func() { rootCmd.SetOut(nil) })
	maestro(t, "--help")
	for _, c := range rootCmd.Commands() {
		if c.IsAvailableCommand() && c.GroupID == "" {
			t.Errorf("maestro %s is in no group of commandGroups", c.Name())
		}
	}
	if c, _, _ := rootCmd.Find([]string{"builds", "run"}); c.Name() != "run" || c.Parent().Name() != "builds" {
		t.Error("builds run isn't there")
	}
}

// What a command is given is checked first: a mistake is said as itself,
// with the usage, and exits 2, signed in or not and before the server is
// asked anything.
func TestUsageMistakesComeFirst(t *testing.T) {
	signedIn(t, "") // no key: the mistake must still be what is said
	t.Setenv("MAESTRO_ORQ_URL", "http://127.0.0.1:9")

	cases := []struct {
		args []string
		want []string
	}{
		{[]string{"run"}, []string{"maestro run needs a build number (see `maestro builds`)", "Usage: maestro run <build>", "Run 'maestro run --help'"}},
		{[]string{"run", "abc"}, []string{`invalid build "abc": expected a number such as 12`}},
		{[]string{"run", "12", "13"}, []string{"maestro run takes at most 1 argument(s), not 2"}},
		{[]string{"runs", "logs"}, []string{"maestro runs logs needs a run number (see `maestro runs`)"}},
		{[]string{"runs", "shw"}, []string{`unknown command "shw" for "maestro runs"`, "Did you mean this?\n\tshow", "Run 'maestro runs --help'"}},
		{[]string{"runs", "--watch", "--json"}, []string{"--json prints once"}},
		{[]string{"top", "-n", "100ms"}, []string{"--interval must be at least 1s"}},
		{[]string{"build"}, []string{"maestro build needs a repo"}},
		{[]string{"run", "12", "--bogus"}, []string{"unknown flag: --bogus", "Usage: maestro run <build>"}},
		{[]string{"runz"}, []string{`unknown command "runz" for "maestro"`, "Run 'maestro --help'"}},
	}
	for _, c := range cases {
		code, said := maestro(t, c.args...)
		if code != exitUsage {
			t.Errorf("maestro %s: exit %d, want %d; said %q", strings.Join(c.args, " "), code, exitUsage, said)
		}
		for _, want := range c.want {
			if !strings.Contains(said, want) {
				t.Errorf("maestro %s said %q; want %q in it", strings.Join(c.args, " "), said, want)
			}
		}
		if strings.Contains(said, "not signed in") {
			t.Errorf("maestro %s spoke of signing in before the mistake: %q", strings.Join(c.args, " "), said)
		}
	}

	// Asked rightly, the key is what's missing, said once.
	code, said := maestro(t, "run", "12")
	if code != exitFailed || strings.Count(said, "maestro login") != 1 || !strings.Contains(said, "not signed in to Maestro") {
		t.Errorf("maestro run 12 signed out: exit %d, %q", code, said)
	}
}

// The names commands had before still reach them.
func TestOldNamesStillWork(t *testing.T) {
	for old, now := range map[string]string{
		"image":         "builds",
		"image list":    "builds list",
		"builds remove": "builds rm",
		"machine":       "machines",
		"machine list":  "machines list",
		"machine stats": "machines stats",
		"repo":          "repos",
		"repo list":     "repos list",
		"forge":         "forges",
		"forge add":     "forges add",
		"forge remove":  "forges rm",
		"runs remove":   "runs rm",
	} {
		was, _, err := rootCmd.Find(strings.Fields(old))
		if err != nil {
			t.Errorf("maestro %s: %v", old, err)
			continue
		}
		is, _, err := rootCmd.Find(strings.Fields(now))
		if err != nil {
			t.Errorf("maestro %s: %v", now, err)
			continue
		}
		if was != is || is.CommandPath() != "maestro "+now {
			t.Errorf("maestro %s is %q, want %q", old, was.CommandPath(), "maestro "+now)
		}
	}
}
