package cmd

import (
	"bytes"
	"errors"
	"io"
	"strings"
	"testing"

	"github.com/spf13/cobra"
)

func TestTopicURL(t *testing.T) {
	base := "https://maestro.example.com/docs"
	for _, c := range []struct {
		args []string
		want string
	}{
		{nil, base + "/"},
		{[]string{"queue"}, base + "/watch/queue.html"},
		{[]string{"TOML"}, base + "/run/maestro-toml.html"},
		// A command a topic explains opens the topic, not the reference.
		{[]string{"run"}, base + "/watch/runs.html"},
		{[]string{"runs", "logs"}, base + "/reference/cli.html#maestro-runs-logs"},
		{[]string{"builds", "rm"}, base + "/reference/cli.html#maestro-builds-rm"},
	} {
		got, err := topicURL(base, c.args)
		if err != nil {
			t.Errorf("%q: %v", c.args, err)
			continue
		}
		if got != c.want {
			t.Errorf("%q: %s, want %s", c.args, got, c.want)
		}
	}

	_, err := topicURL(base, []string{"nope"})
	if err == nil || !strings.Contains(err.Error(), "queue") || !strings.Contains(err.Error(), "toml") {
		t.Errorf("an unknown topic should list the topics, got %v", err)
	}
	if _, err := topicURL(base, []string{"runs", "nope"}); err == nil {
		t.Error("a command that doesn't exist got a link")
	}
	if _, err := topicURL(base, []string{"gen-docs"}); err == nil {
		t.Error("a hidden command got a link")
	}
}

// Every name leads to one topic, so none hides another.
func TestTopicNamesAreUnique(t *testing.T) {
	seen := map[string]string{}
	for _, topic := range topics {
		for _, n := range topic.Names {
			if was, ok := seen[n]; ok {
				t.Errorf("%q is both %s and %s", n, was, topic.Page)
			}
			seen[n] = topic.Page
		}
	}
}

func TestDocsBaseURL(t *testing.T) {
	t.Setenv("MAESTRO_ORQ_URL", "http://localhost:8081/")
	t.Setenv("MAESTRO_DOCS_URL", "")
	if got, err := docsBaseURL(); err != nil || got != "http://localhost:8081/docs" {
		t.Errorf("got %s (%v)", got, err)
	}
	t.Setenv("MAESTRO_DOCS_URL", "http://localhost:3000/")
	if got, err := docsBaseURL(); err != nil || got != "http://localhost:3000" {
		t.Errorf("MAESTRO_DOCS_URL: got %s (%v)", got, err)
	}
}

// The book needs no key, nor a server when MAESTRO_DOCS_URL says where it
// is; a topic that isn't one is a usage mistake, found before anything else.
func TestDocsWithoutAServer(t *testing.T) {
	signedIn(t, "")
	t.Setenv("MAESTRO_ORQ_URL", "")
	t.Setenv("MAESTRO_DOCS_URL", "http://docs.test")
	rootCmd.SetOut(io.Discard)
	t.Cleanup(func() { rootCmd.SetOut(nil) })

	for _, args := range [][]string{{"docs", "--url"}, {"doc", "queue", "--url"}, {"docs", "runs", "logs", "--url"}, {"docs", "--topics"}} {
		if code, said := maestro(t, args...); code != 0 {
			t.Errorf("maestro %s: exit %d, %q", strings.Join(args, " "), code, said)
		}
	}
	if code, said := maestro(t, "docs", "nope"); code != exitUsage || !strings.Contains(said, "Topics: intro") {
		t.Errorf("an unknown topic: exit %d, %q", code, said)
	}
	t.Setenv("MAESTRO_DOCS_URL", "")
	if code, said := maestro(t, "docs", "--url"); code != exitFailed || !strings.Contains(said, "no Maestro server is set") {
		t.Errorf("no server and no MAESTRO_DOCS_URL: exit %d, %q", code, said)
	}
}

func runDocsWith(t *testing.T, terminal, urlOnly bool, open func(string) error, args ...string) (string, error) {
	t.Helper()
	t.Setenv("MAESTRO_DOCS_URL", "http://docs.test")
	oldOpen, oldTerm, oldURL := openBrowser, stdoutIsTerminal, docsURLOnly
	t.Cleanup(func() { openBrowser, stdoutIsTerminal, docsURLOnly = oldOpen, oldTerm, oldURL })
	openBrowser = open
	stdoutIsTerminal = func() bool { return terminal }
	docsURLOnly = urlOnly
	var out bytes.Buffer
	c := &cobra.Command{}
	c.SetOut(&out)
	err := runDocs(c, args)
	return out.String(), err
}

func TestDocsOpensTheBrowser(t *testing.T) {
	var opened []string
	open := func(u string) error { opened = append(opened, u); return nil }

	out, err := runDocsWith(t, true, false, open, "queue")
	if err != nil {
		t.Fatal(err)
	}
	if !strings.Contains(out, "http://docs.test/watch/queue.html") {
		t.Errorf("the link isn't printed: %q", out)
	}
	if len(opened) != 1 || opened[0] != "http://docs.test/watch/queue.html" {
		t.Errorf("opened %q", opened)
	}

	opened = nil
	if _, err := runDocsWith(t, true, true, open, "queue"); err != nil || opened != nil {
		t.Errorf("--url opened %q (%v)", opened, err)
	}
	if _, err := runDocsWith(t, false, false, open); err != nil || opened != nil {
		t.Errorf("piped output opened %q (%v)", opened, err)
	}
}

func TestDocsWithoutABrowserIsNoError(t *testing.T) {
	out, err := runDocsWith(t, true, false, func(string) error { return errors.New("no desktop") })
	if err != nil {
		t.Fatal(err)
	}
	if !strings.Contains(out, "http://docs.test/") || !strings.Contains(out, "open the link") {
		t.Errorf("got %q", out)
	}
}

func TestCompleteTopics(t *testing.T) {
	got, _ := completeTopics(docsCmd, nil, "")
	if len(got) != len(topics) {
		t.Errorf("offered %d topics, want %d", len(got), len(topics))
	}
	got, _ = completeTopics(docsCmd, nil, "qu")
	if len(got) != 1 || !strings.HasPrefix(got[0], "queue\t") {
		t.Errorf("qu: %q", got)
	}
}

// Help ends with the footer, after the flags.
func TestHelpEndsWithDocs(t *testing.T) {
	c, _, _ := rootCmd.Find([]string{"runs", "logs"})
	var out bytes.Buffer
	c.SetOut(&out)
	t.Cleanup(func() { c.SetOut(nil) })
	if err := c.Help(); err != nil {
		t.Fatal(err)
	}
	if got := strings.TrimSpace(out.String()); !strings.HasSuffix(got, "Docs: maestro docs runs") {
		t.Errorf("help ends with %q", got[max(0, len(got)-80):])
	}
}

func TestDocsFooter(t *testing.T) {
	for path, want := range map[string]string{
		"runs logs":  "maestro docs runs",
		"build":      "maestro docs build",
		"forges add": "maestro docs forges",
		"export":     "maestro docs export",
	} {
		c, _, err := rootCmd.Find(strings.Fields(path))
		if err != nil {
			t.Fatal(err)
		}
		if got := docsFooter(c); got != want {
			t.Errorf("%s: %s, want %s", path, got, want)
		}
	}
}

// Every footer names something `maestro docs` opens.
func TestDocsFootersResolve(t *testing.T) {
	var walk func(c *cobra.Command)
	walk = func(c *cobra.Command) {
		for _, sub := range c.Commands() {
			// Help topics show no usage, so no footer either.
			if !sub.IsAvailableCommand() || sub == docsCmd {
				continue
			}
			footer := docsFooter(sub)
			if footer == "" {
				t.Errorf("%s has no footer", sub.CommandPath())
			}
			args := strings.Fields(strings.TrimPrefix(footer, "maestro docs "))
			if _, err := topicURL("x", args); err != nil {
				t.Errorf("%s: %v", sub.CommandPath(), err)
			}
			walk(sub)
		}
	}
	walk(rootCmd)
}

func TestCLIReference(t *testing.T) {
	var out bytes.Buffer
	if err := writeCLIReference(&out, rootCmd); err != nil {
		t.Fatal(err)
	}
	got := out.String()
	for _, want := range []string{"## maestro runs logs\n", "## maestro docs\n", "[maestro runs logs](#maestro-runs-logs)", "--json"} {
		if !strings.Contains(got, want) {
			t.Errorf("no %q", want)
		}
	}
	if strings.Contains(got, "gen-docs") {
		t.Error("the hidden command is in the reference")
	}
	if !strings.Contains(got, "\n&lt;repo&gt; is a name") || !strings.Contains(got, "`maestro run <build>`") {
		t.Error("<repo> isn't escaped, or <build> is escaped inside backticks")
	}
	if !strings.Contains(got, "```sh\nmaestro build my-app\n") {
		t.Error("examples aren't a code block, or keep their indent")
	}
}
