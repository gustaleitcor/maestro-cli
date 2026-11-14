package cmd

import (
	"strings"
	"testing"

	"github.com/charmbracelet/x/ansi"
	"github.com/spf13/cobra"
	"github.com/spf13/pflag"
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
		{[]string{"runs", "logs", "31"}, []string{"maestro runs logs needs a line number"}},
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
