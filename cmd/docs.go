package cmd

import (
	"context"
	"fmt"
	"io"
	"io/fs"
	"net"
	"net/http"
	"os"
	"os/exec"
	"os/signal"
	"runtime"
	"strings"
	"time"

	"github.com/spf13/cobra"
	"golang.org/x/term"

	"maestro-cli/internal/book"
	"maestro-cli/internal/maestroapi"
)

// docTopic is a page of the Maestro book. Names are what `maestro docs`
// takes: the first is the topic's own, the others are commands it explains
// and names it had before the book was reordered.
type docTopic struct {
	Names []string
	Page  string
	Title string
}

// topics are the book's pages, in the book's order. A test checks that every
// page here is one book/src/SUMMARY.md lists, so a page moved there must move
// here too.
var topics = []docTopic{
	{[]string{"intro"}, "introduction.html", "What Maestro is"},
	{[]string{"quickstart", "start"}, "start/quickstart.html", "Quickstart: run the template"},
	{[]string{"template"}, "start/your-copy.html", "Your own copy of the template"},
	{[]string{"dockerfile"}, "repo/dockerfile.html", "The minimum: a Dockerfile"},
	{[]string{"toml"}, "repo/maestro-toml.html", "maestro.toml: how the image is run"},
	{[]string{"generator"}, "repo/generator.html", "Generating lines with Python"},
	{[]string{"env"}, "repo/container.html", "Inside the container"},
	{[]string{"local"}, "repo/local-files.html", "Trying a change without pushing it"},
	{[]string{"examples", "example"}, "repo/examples.html", "Fuller examples"},
	{[]string{"install", "completion"}, "use/install.html", "Installing the CLI"},
	{[]string{"login", "logout", "keys"}, "use/login.html", "Signing in"},
	{[]string{"forges", "repos"}, "use/forges.html", "Git forges"},
	{[]string{"conventions"}, "use/cli.html", "How the CLI behaves"},
	{[]string{"builds", "build"}, "use/builds.html", "Builds and images"},
	{[]string{"runs", "run"}, "use/runs.html", "Runs and lines"},
	{[]string{"outputs"}, "use/results.html", "What a line leaves behind"},
	{[]string{"view"}, "use/page.html", "The Maestro page"},
	{[]string{"top", "machines"}, "use/machines.html", "Watching the machines"},
	{[]string{"usage", "export"}, "use/usage.html", "Usage and export"},
	{[]string{"parts", "architecture"}, "how/parts.html", "The parts"},
	{[]string{"queue"}, "how/queue.html", "The queue"},
	{[]string{"score", "settings"}, "how/score.html", "Policies and the score"},
	{[]string{"deploy"}, "admin/deploy.html", "Deploying Maestro"},
	{[]string{"identity"}, "admin/identity.html", "The identity provider"},
	{[]string{"add-machine"}, "admin/machines.html", "Adding machines"},
	{[]string{"proxy"}, "admin/proxy.html", "Machines behind a proxy"},
	{[]string{"policies"}, "admin/policies.html", "Choosing the policies"},
	{[]string{"groups"}, "admin/groups.html", "Groups"},
	{[]string{"sim"}, "admin/sim.html", "Comparing policies offline"},
	{[]string{"retention"}, "admin/retention.html", "What orq keeps, and for how long"},
	{[]string{"cli"}, "reference/cli.html", "Every command"},
	{[]string{"toml-keys"}, "reference/maestro-toml.html", "maestro.toml keys"},
	{[]string{"variables"}, "reference/env.html", "Environment variables"},
	{[]string{"troubleshooting"}, "reference/troubleshooting.html", "When something goes wrong"},
	{[]string{"glossary", "concepts"}, "reference/glossary.html", "Glossary"},
	{[]string{"contributing"}, "contributing.html", "Contributing"},
}

var (
	docsURLOnly   bool
	docsWeb       bool
	docsTopicList bool

	// serveBook serves the book maestro carries until ctx ends; tests
	// replace it.
	serveBook = func(ctx context.Context, ready func(base string) error) error {
		return serveSite(ctx, book.Site(), ready)
	}

	// openBrowser opens url in the user's browser; tests replace it.
	openBrowser = startBrowser
	// stdoutIsTerminal says whether someone is there to read a browser
	// opening; tests replace it.
	stdoutIsTerminal = func() bool { return term.IsTerminal(int(os.Stdout.Fd())) }
)

var docsCmd = &cobra.Command{
	Use:     "docs [topic | command…]",
	Aliases: []string{"doc"},
	Short:   "Open the Maestro book, on a topic or a command",
	Long: `Opens the Maestro book in a browser, and prints its link. The book
explains how Maestro builds, queues and runs your containers, and how to
set it up.

Name a topic (` + "`maestro docs queue`" + `) to open its page, or a command
(` + "`maestro docs runs logs`" + `) to see what it takes.

maestro carries the book, so it opens offline and without signing in. The
CLI serves it from your computer only (127.0.0.1) until you press Ctrl-C,
so any browser opens it whole, a sandboxed one too. --web opens the copy
your Maestro server serves instead, which you can open on another
computer, as when you reach this one over SSH. MAESTRO_DOCS_URL points at
another copy still.`,
	Example: "  maestro docs\n  maestro docs toml\n  maestro docs runs logs\n  maestro docs queue --web --url",
	// What args name is checked before a server is asked for.
	Args: func(cmd *cobra.Command, args []string) error {
		_, err := topicPath(args)
		return err
	},
	ValidArgsFunction: completeTopics,
	RunE:              runDocs,
}

func init() {
	docsCmd.Flags().BoolVar(&docsURLOnly, "url", false, "only print the link, without opening a browser")
	docsCmd.Flags().BoolVar(&docsWeb, "web", false, "open the book your Maestro server serves, instead of the one maestro carries")
	docsCmd.Flags().BoolVar(&docsTopicList, "topics", false, "print every topic and its page, for scripts")
	_ = docsCmd.Flags().MarkHidden("topics")
	rootCmd.AddCommand(docsCmd)

	// Every command's help ends with where the book explains it.
	cobra.AddTemplateFunc("docsFooter", docsFooter)
	rootCmd.SetUsageTemplate(rootCmd.UsageTemplate() + "{{with docsFooter .}}\nDocs: {{.}}\n{{end}}")
}

func runDocs(cmd *cobra.Command, args []string) error {
	out := cmd.OutOrStdout()
	if docsTopicList {
		return writeTopics(out)
	}
	base, err := docsBase()
	if err != nil {
		return err
	}
	if base != "" {
		return showDocs(out, base, args, "No browser opened here. Open the link in one.")
	}

	ctx := cmd.Context()
	if ctx == nil {
		ctx = context.Background()
	}
	// Ctrl-C is how serving ends, so it isn't an interruption.
	ctx, stop := signal.NotifyContext(ctx, os.Interrupt)
	defer stop()
	return serveBook(ctx, func(base string) error {
		if err := showDocs(out, base, args, "No browser opened here. `maestro docs --web --url` gives the server's copy, to open elsewhere."); err != nil {
			return err
		}
		fmt.Fprintln(out, dimStyle.Render("Serving the book from your computer. Ctrl-C stops it."))
		return nil
	})
}

// showDocs prints the link to what args name in the book at base, and opens
// it unless --url says not to or no one is there to see it. The link is
// printed already, so a browser that won't open is no error: noBrowser says
// what to do instead.
func showDocs(out io.Writer, base string, args []string, noBrowser string) error {
	link, err := topicURL(base, args)
	if err != nil {
		return err
	}
	fmt.Fprintln(out, accentStyle.Render(link))
	if docsURLOnly || !stdoutIsTerminal() {
		return nil
	}
	if err := openBrowser(link); err != nil {
		fmt.Fprintln(out, dimStyle.Render(noBrowser))
	}
	return nil
}

// docsBase is where the book is when it isn't the one maestro carries:
// MAESTRO_DOCS_URL, or with --web the server's /docs, beside its page. It is
// empty for the carried one, which serveBook serves.
func docsBase() (string, error) {
	if u := os.Getenv("MAESTRO_DOCS_URL"); u != "" {
		return strings.TrimRight(u, "/"), nil
	}
	if !docsWeb {
		return "", nil
	}
	if err := ensureServer(os.Stdin, os.Stderr); err != nil {
		return "", err
	}
	return maestroapi.BaseURL() + "/docs", nil
}

// serveSite serves site on 127.0.0.1, on a port the system picks, until ctx
// ends, calling ready with its address once it can be opened. The book is
// served, and not written out for a file:// link, because a sandboxed
// browser, as from Flatpak or Snap, is handed that one file and none of the
// styles and scripts beside it. Only this computer can reach it.
func serveSite(ctx context.Context, site fs.FS, ready func(base string) error) error {
	ln, err := net.Listen("tcp", "127.0.0.1:0")
	if err != nil {
		return fmt.Errorf("couldn't serve the book: %w", err)
	}
	srv := &http.Server{Handler: http.FileServer(http.FS(site)), ReadHeaderTimeout: 10 * time.Second}
	served := make(chan error, 1)
	go func() { served <- srv.Serve(ln) }()
	if err := ready("http://" + ln.Addr().String()); err != nil {
		srv.Close()
		return err
	}
	select {
	case <-ctx.Done():
		srv.Close()
		return nil
	case err := <-served:
		return fmt.Errorf("couldn't serve the book: %w", err)
	}
}

// topicURL is the link to what args name, in the book at base.
func topicURL(base string, args []string) (string, error) {
	page, err := topicPath(args)
	if err != nil {
		return "", err
	}
	return base + "/" + page, nil
}

// topicPath is the page args name: nothing is the book's first page, one
// word a topic, and otherwise a command's entry in the CLI reference.
func topicPath(args []string) (string, error) {
	// By name: a folder opened from disk is a listing, not its index.
	if len(args) == 0 {
		return "index.html", nil
	}
	if len(args) == 1 {
		if t, ok := findTopic(args[0]); ok {
			return t.Page, nil
		}
	}
	if c, ok := findCommand(args); ok {
		return "reference/cli.html#" + commandAnchor(c), nil
	}
	return "", fmt.Errorf("there is no topic or command %q\n\nTopics: %s", strings.Join(args, " "), strings.Join(topicNames(), ", "))
}

func findTopic(name string) (docTopic, bool) {
	name = strings.ToLower(name)
	for _, t := range topics {
		for _, n := range t.Names {
			if n == name {
				return t, true
			}
		}
	}
	return docTopic{}, false
}

// findCommand finds the command args spell out exactly, without its flags.
func findCommand(args []string) (*cobra.Command, bool) {
	c, rest, err := rootCmd.Find(args)
	if err != nil || len(rest) > 0 || c == rootCmd || !c.IsAvailableCommand() {
		return nil, false
	}
	return c, true
}

// commandAnchor is the id mdBook gives the heading gen-docs writes for c,
// "maestro runs logs" becoming maestro-runs-logs.
func commandAnchor(c *cobra.Command) string {
	return strings.ReplaceAll(c.CommandPath(), " ", "-")
}

// topicNames are the topics' own names, without the commands they explain.
func topicNames() []string {
	names := make([]string, len(topics))
	for i, t := range topics {
		names[i] = t.Names[0]
	}
	return names
}

func writeTopics(w io.Writer) error {
	for _, t := range topics {
		for _, n := range t.Names {
			if _, err := fmt.Fprintf(w, "%s\t%s\n", n, t.Page); err != nil {
				return err
			}
		}
	}
	return nil
}

func completeTopics(cmd *cobra.Command, args []string, toComplete string) ([]string, cobra.ShellCompDirective) {
	if len(args) > 0 {
		return nil, cobra.ShellCompDirectiveNoFileComp
	}
	var offers []string
	for _, t := range topics {
		if strings.HasPrefix(t.Names[0], toComplete) {
			offers = append(offers, completion(t.Names[0], t.Title))
		}
	}
	return offers, cobra.ShellCompDirectiveNoFileComp | cobra.ShellCompDirectiveKeepOrder
}

// startBrowser asks the system to open url. On Linux that needs a desktop
// to open it on: without one, as over SSH, xdg-open would start a text
// browser in this terminal, or nothing at all.
func startBrowser(url string) error {
	var c *exec.Cmd
	switch runtime.GOOS {
	case "darwin":
		c = exec.Command("open", url)
	case "windows":
		c = exec.Command("rundll32", "url.dll,FileProtocolHandler", url)
	default:
		if os.Getenv("DISPLAY") == "" && os.Getenv("WAYLAND_DISPLAY") == "" {
			return fmt.Errorf("there is no desktop to open a browser on")
		}
		c = exec.Command("xdg-open", url)
	}
	return c.Start()
}

// docsFooter is the `maestro docs` that explains c, which every command's
// help ends with: its topic, or else its parent's, or else its entry in the
// CLI reference.
func docsFooter(c *cobra.Command) string {
	if c == rootCmd {
		return "maestro docs"
	}
	if c.Hidden || c == docsCmd || !c.IsAvailableCommand() {
		return ""
	}
	for p := c; p != nil && p != rootCmd; p = p.Parent() {
		if _, ok := findTopic(p.Name()); ok {
			return "maestro docs " + p.Name()
		}
	}
	return "maestro docs " + strings.TrimPrefix(c.CommandPath(), "maestro ")
}
