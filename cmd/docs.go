package cmd

import (
	"context"
	"errors"
	"fmt"
	"io"
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
// takes: the first is the topic's own, the others are commands it explains.
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
	{[]string{"concepts"}, "concepts.html", "The words Maestro uses"},
	{[]string{"architecture"}, "architecture.html", "How the parts fit"},
	{[]string{"install", "completion"}, "start/install.html", "Installing the CLI"},
	{[]string{"login", "logout"}, "start/login.html", "Signing in"},
	{[]string{"forges", "repos"}, "start/forges.html", "Git forges"},
	{[]string{"start"}, "start/first-run.html", "Your first run"},
	{[]string{"toml"}, "run/maestro-toml.html", "maestro.toml, key by key"},
	{[]string{"generator"}, "run/generators.html", "Computing lines with a generator"},
	{[]string{"env"}, "run/container.html", "Inside the container"},
	{[]string{"local"}, "run/local-files.html", "Trying a change without pushing it"},
	{[]string{"example"}, "run/example.html", "A worked example: the stress test"},
	{[]string{"builds", "build"}, "watch/builds.html", "Builds and images"},
	{[]string{"runs", "run"}, "watch/runs.html", "Runs and lines"},
	{[]string{"outputs"}, "watch/results.html", "What a line leaves behind"},
	{[]string{"queue"}, "watch/queue.html", "The queue, explained"},
	{[]string{"top", "machines"}, "watch/top.html", "Watching the machines"},
	{[]string{"usage", "export"}, "watch/usage.html", "Usage and export"},
	{[]string{"view"}, "view/tour.html", "The Maestro page"},
	{[]string{"deploy"}, "admin/deploy.html", "Deploying Maestro"},
	{[]string{"identity"}, "admin/identity.html", "The identity provider"},
	{[]string{"add-machine"}, "admin/machines.html", "Adding machines"},
	{[]string{"proxy"}, "admin/proxy.html", "Machines behind a proxy"},
	{[]string{"policies", "settings"}, "admin/policies.html", "Scheduler policies"},
	{[]string{"groups"}, "admin/groups.html", "Groups"},
	{[]string{"sim"}, "admin/sim.html", "Comparing policies offline"},
	{[]string{"retention"}, "admin/retention.html", "What orq keeps, and for how long"},
	{[]string{"cli"}, "reference/cli.html", "Every command and flag"},
	{[]string{"variables"}, "reference/env.html", "Environment variables"},
	{[]string{"troubleshooting"}, "reference/troubleshooting.html", "When something goes wrong"},
}

var (
	docsURLOnly   bool
	docsWeb       bool
	docsTopicList bool

	// bookDir is where the book maestro carries is on disk; tests replace it.
	bookDir = book.Dir
	// serveBook serves the book at dir until ctx ends; tests replace it.
	serveBook = serveDir

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
	Long: `Opens the Maestro book in a browser, and prints its link: the book explains
how Maestro builds, queues and runs your containers, and how to set it up.

Name a topic (maestro docs queue) to open its page, or a command
(maestro docs runs logs) for what it takes.

maestro carries the book, so it opens offline, without signing in: it is
unpacked into your cache the first time, and served from this computer only
(127.0.0.1), until ctrl-c, so that any browser opens it whole, a sandboxed
one too. --web opens the copy your Maestro server serves instead, which can
be opened on another machine, as when this one is reached over SSH.
MAESTRO_DOCS_URL points at another copy still.`,
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
	docsCmd.Flags().BoolVar(&docsWeb, "web", false, "the book your Maestro server serves, instead of the one maestro carries")
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
	base, dir, err := docsBase(cmd.ErrOrStderr())
	if err != nil {
		return err
	}
	if dir == "" {
		return showDocs(out, base, args, "No browser opened here; open the link in one.")
	}

	ctx := cmd.Context()
	if ctx == nil {
		ctx = context.Background()
	}
	// ctrl-c is how serving ends, so it isn't an interruption.
	ctx, stop := signal.NotifyContext(ctx, os.Interrupt)
	defer stop()
	return serveBook(ctx, dir, func(base string) error {
		if err := showDocs(out, base, args, "No browser opened here. `maestro docs --web --url` gives the server's copy, to open elsewhere."); err != nil {
			return err
		}
		fmt.Fprintln(out, dimStyle.Render("Serving the book from this computer; ctrl-c stops it."))
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

// docsBase is where the book is: MAESTRO_DOCS_URL, then with --web the
// server's /docs, beside its page, and else dir, the one maestro carries,
// unpacked, for serveBook to serve. A maestro built without its book says
// so and goes to the server's.
func docsBase(notes io.Writer) (base, dir string, err error) {
	if u := os.Getenv("MAESTRO_DOCS_URL"); u != "" {
		return strings.TrimRight(u, "/"), "", nil
	}
	if !docsWeb {
		dir, err := bookDir()
		if err == nil {
			return "", dir, nil
		}
		if !errors.Is(err, book.ErrMissing) {
			return "", "", err
		}
		fmt.Fprintln(notes, dimStyle.Render(err.Error()+" (book/build.sh adds it); opening the server's."))
	}
	if err := ensureServer(os.Stdin, os.Stderr); err != nil {
		return "", "", err
	}
	return maestroapi.BaseURL() + "/docs", "", nil
}

// serveDir serves dir on 127.0.0.1, on a port the system picks, until ctx
// ends, calling ready with its address once it can be opened. A browser
// opens a file:// page alone: a sandboxed one, as from Flatpak or Snap,
// gets that one file and none of the styles and scripts beside it, so the
// book is served instead, to this computer only.
func serveDir(ctx context.Context, dir string, ready func(base string) error) error {
	ln, err := net.Listen("tcp", "127.0.0.1:0")
	if err != nil {
		return fmt.Errorf("serving the book: %w", err)
	}
	srv := &http.Server{Handler: http.FileServer(http.Dir(dir)), ReadHeaderTimeout: 10 * time.Second}
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
		return fmt.Errorf("serving the book: %w", err)
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
	return "", fmt.Errorf("no topic or command %q\n\nTopics: %s", strings.Join(args, " "), strings.Join(topicNames(), ", "))
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
			return fmt.Errorf("no desktop to open a browser on")
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
