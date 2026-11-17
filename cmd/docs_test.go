package cmd

import (
	"bytes"
	"context"
	"errors"
	"io"
	"net/http"
	"os"
	"regexp"
	"strings"
	"testing"

	"github.com/spf13/cobra"

	"maestro-cli/internal/book"
)

func TestTopicURL(t *testing.T) {
	base := "https://maestro.example.com/docs"
	for _, c := range []struct {
		args []string
		want string
	}{
		{nil, base + "/index.html"},
		{[]string{"queue"}, base + "/how/queue.html"},
		{[]string{"TOML"}, base + "/repo/maestro-toml.html"},
		{[]string{"quickstart"}, base + "/start/quickstart.html"},
		// A name a page had before the book was reordered still opens it.
		{[]string{"concepts"}, base + "/reference/glossary.html"},
		// A command a topic explains opens the topic, not the reference.
		{[]string{"run"}, base + "/use/runs.html"},
		{[]string{"settings"}, base + "/how/score.html"},
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

// carriedBook makes the book maestro carries be served at
// http://127.0.0.1:1 at once and no longer.
func carriedBook(t *testing.T) {
	t.Helper()
	old := serveBook
	t.Cleanup(func() { serveBook = old })
	serveBook = func(_ context.Context, ready func(string) error) error {
		return ready("http://127.0.0.1:1")
	}
}

func TestDocsBase(t *testing.T) {
	t.Setenv("MAESTRO_ORQ_URL", "http://localhost:8081/")
	t.Cleanup(func() { docsWeb = false })

	t.Setenv("MAESTRO_DOCS_URL", "http://localhost:3000/")
	if base, err := docsBase(); err != nil || base != "http://localhost:3000" {
		t.Errorf("MAESTRO_DOCS_URL: %s (%v)", base, err)
	}
	t.Setenv("MAESTRO_DOCS_URL", "")
	if base, err := docsBase(); err != nil || base != "" {
		t.Errorf("the carried book: %s (%v)", base, err)
	}
	docsWeb = true
	if base, err := docsBase(); err != nil || base != "http://localhost:8081/docs" {
		t.Errorf("--web: %s (%v)", base, err)
	}
}

// The book maestro carries was built from the pages, book.toml and the
// commands' help as they are now.
func TestBookIsBuilt(t *testing.T) {
	reference, err := cliReference()
	if err != nil {
		t.Fatal(err)
	}
	want, err := bookSources("../book", reference)
	if err != nil {
		t.Fatal(err)
	}
	got, err := os.ReadFile("../internal/book/sources.sha256")
	if err != nil {
		t.Fatal(err)
	}
	if strings.TrimSpace(string(got)) != want {
		t.Error("internal/book/site is older than the book's pages or a command's help: run book/build.sh and commit what it writes")
	}
}

// The carried book is served whole, styles and scripts included, until
// ctrl-c, and only to this computer.
func TestServeSite(t *testing.T) {
	ctx, cancel := context.WithCancel(context.Background())
	done := make(chan error, 1)
	var base string
	go func() {
		done <- serveSite(ctx, book.Site(), func(b string) error {
			base = b
			for _, path := range []string{"/", "/start/quickstart.html", "/css/general.css"} {
				resp, err := http.Get(b + path)
				if err != nil {
					return err
				}
				resp.Body.Close()
				if resp.StatusCode != http.StatusOK {
					t.Errorf("%s: %s", path, resp.Status)
				}
			}
			cancel()
			return nil
		})
	}()
	if err := <-done; err != nil {
		t.Fatal(err)
	}
	if !strings.HasPrefix(base, "http://127.0.0.1:") {
		t.Errorf("served at %s, not to this computer only", base)
	}
	if _, err := http.Get(base + "/index.html"); err == nil {
		t.Error("still served after ctrl-c")
	}
}

// Every topic opens a page the book has: one SUMMARY.md lists.
func TestTopicsAreInTheBook(t *testing.T) {
	summary, err := os.ReadFile("../book/src/SUMMARY.md")
	if err != nil {
		t.Fatal(err)
	}
	pages := map[string]bool{}
	for _, m := range regexp.MustCompile(`\]\(([^)]+)\.md\)`).FindAllStringSubmatch(string(summary), -1) {
		pages[m[1]+".html"] = true
	}
	for _, topic := range topics {
		page, _, _ := strings.Cut(topic.Page, "#")
		if !pages[page] {
			t.Errorf("maestro docs %s opens %s, which book/src/SUMMARY.md doesn't list", topic.Names[0], page)
		}
	}
}

// The book needs no key, nor a server when MAESTRO_DOCS_URL says where it
// is; a topic that isn't one is a usage mistake, found before anything else.
func TestDocsWithoutAServer(t *testing.T) {
	signedIn(t, "")
	t.Setenv("MAESTRO_ORQ_URL", "")
	t.Setenv("MAESTRO_DOCS_URL", "http://docs.test")
	carriedBook(t)
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
	if code, said := maestro(t, "docs", "queue", "--url"); code != 0 {
		t.Errorf("the carried book, without a server: exit %d, %q", code, said)
	}
	if code, said := maestro(t, "docs", "--web", "--url"); code != exitFailed || !strings.Contains(said, "no Maestro server is set") {
		t.Errorf("--web without a server: exit %d, %q", code, said)
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
	if !strings.Contains(out, "http://docs.test/how/queue.html") {
		t.Errorf("the link isn't printed: %q", out)
	}
	if len(opened) != 1 || opened[0] != "http://docs.test/how/queue.html" {
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
	if !strings.Contains(out, "http://docs.test/") || !strings.Contains(out, "Open the link") {
		t.Errorf("got %q", out)
	}
}

func TestCompleteTopics(t *testing.T) {
	got, _ := completeTopics(docsCmd, nil, "")
	if len(got) != len(topics) {
		t.Errorf("offered %d topics, want %d", len(got), len(topics))
	}
	got, _ = completeTopics(docsCmd, nil, "que")
	if len(got) != 1 || !strings.HasPrefix(got[0], "queue\t") {
		t.Errorf("que: %q", got)
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
