package cmd

import (
	"bufio"
	"context"
	"encoding/json"
	"errors"
	"fmt"
	"io"
	"net/url"
	"os"
	"strings"
	"sync"
	"time"

	"github.com/spf13/cobra"
	"golang.org/x/term"

	"maestro-cli/internal/config"
	"maestro-cli/internal/maestroapi"
	"maestro-cli/tui"
)

var maestroKey string

var Version = "dev"

var rootCmd = &cobra.Command{
	Use:   "maestro",
	Short: "Build images from your repos and run them across Maestro's machines",
	Long: `Builds container images from your git repos and runs them across Maestro's
machines, one container per line of parameters.

  maestro login            sign in
  maestro build my-app     build an image; the build gets a number
  maestro run 12 --watch   run build #12's image, and follow the run
  maestro runs get 31      download what run 31 kept

` + "`maestro help concepts`" + ` says what builds, runs and lines are.`,
	Version: Version,
	// Execute prints the error once; a failed command isn't a usage mistake.
	SilenceErrors: true,
	SilenceUsage:  true,
	// Bare invocation shows who is signed in instead of the usual help text.
	RunE: runWelcome,
	// Whatever talks to Maestro needs to know which server: when nothing says,
	// it is asked for first. maestro (bare) and login need no key beyond
	// that; logout, forges, help and completion need neither. Commands that
	// read a forge load its token themselves, and completions load the key
	// themselves too.
	PersistentPreRunE: func(cmd *cobra.Command, args []string) error {
		if !needsServer(cmd) {
			return nil
		}
		if err := ensureServer(os.Stdin, os.Stderr); err != nil {
			return err
		}
		// maestro itself, or maestro login.
		if !cmd.HasParent() || (cmd.Name() == "login" && !cmd.Parent().HasParent()) {
			return nil
		}

		mk, err := config.LoadMaestroKey(maestroapi.BaseURL())
		var other *config.OtherServerError
		if errors.As(err, &other) {
			return fmt.Errorf("%w\n\nRun `maestro login --server %s` to sign in there, which forgets the key for %s", err, other.Asked, other.Stored)
		}
		if err != nil {
			return fmt.Errorf("%w\n\nRun `maestro login` to sign in", err)
		}
		maestroKey = mk
		return nil
	},
}

// needsServer is whether cmd talks to Maestro. logout, forges, help and
// completion don't, nor anything under them: `maestro completion bash` is
// named bash, and must work on a computer that never signed in, as when a
// package is built.
func needsServer(cmd *cobra.Command) bool {
	for c := cmd; c != nil; c = c.Parent() {
		switch c.Name() {
		case "logout", "help", "completion", "forges", cobra.ShellCompRequestCmd, cobra.ShellCompNoDescRequestCmd:
			if c.Parent() == nil || !c.Parent().HasParent() {
				return false
			}
		}
	}
	return true
}

var serverFlag string

// server is the Maestro server to talk to, and what chose it: --server, then
// MAESTRO_ORQ_URL, then the one kept in the config ("config"), asked for
// the first time or signed in to. "" when nothing says: there is no default,
// so that nothing is ever sent to a server the user didn't name. A server is
// a URL; https:// is assumed when it says no scheme.
func server() (url, from string) {
	if serverFlag != "" {
		return normalizeServer(serverFlag), "--server"
	}
	if env := os.Getenv("MAESTRO_ORQ_URL"); env != "" {
		return normalizeServer(env), "MAESTRO_ORQ_URL"
	}
	if stored, err := config.Server(); err == nil && stored != "" {
		return stored, "config"
	}
	return "", ""
}

func normalizeServer(raw string) string {
	raw = strings.TrimSpace(raw)
	if !strings.Contains(raw, "://") {
		raw = "https://" + raw
	}
	return strings.TrimRight(raw, "/")
}

// checkServer is a server's URL as typed, made whole (https:// when it says
// no scheme), or why it isn't one.
func checkServer(raw string) (string, error) {
	normalized := normalizeServer(raw)
	parsed, err := url.Parse(normalized)
	if err != nil || parsed.Host == "" || strings.ContainsAny(normalized, " \t") || (parsed.Scheme != "https" && parsed.Scheme != "http") {
		return "", fmt.Errorf("%q isn't a server's URL, such as https://maestro.example.org", strings.TrimSpace(raw))
	}
	return normalized, nil
}

// ensureServer asks for the Maestro server when nothing names one, and
// keeps it in the config, so that it is asked once. Without a terminal to
// ask at, it says how to name one instead.
func ensureServer(in *os.File, out io.Writer) error {
	if url, _ := server(); url != "" {
		return nil
	}
	if !term.IsTerminal(int(in.Fd())) {
		return errors.New("no Maestro server is set\n\nPass --server <url>, set MAESTRO_ORQ_URL, or run `maestro login --server <url>` once")
	}
	url, err := askServer(bufio.NewReader(in), out)
	if err != nil {
		return err
	}
	if err := config.SaveServer(url); err != nil {
		return err
	}
	fmt.Fprintln(out, dimStyle.Render("Kept. `maestro login --server <url>` changes it."))
	return nil
}

// askServer asks for a server's URL until it is given one.
func askServer(in *bufio.Reader, out io.Writer) (string, error) {
	fmt.Fprintln(out, "No Maestro server is set yet.")
	for {
		fmt.Fprint(out, "Maestro server URL (e.g. https://maestro.example.org): ")
		line, readErr := in.ReadString('\n')
		if raw := strings.TrimSpace(line); raw != "" {
			url, err := checkServer(raw)
			if err == nil {
				return url, nil
			}
			fmt.Fprintln(out, warnStyle.Render(err.Error()))
		}
		if readErr != nil {
			if readErr == io.EOF {
				fmt.Fprintln(out)
				return "", errors.New("no Maestro server given")
			}
			return "", fmt.Errorf("reading input: %w", readErr)
		}
	}
}

func init() {
	rootCmd.PersistentFlags().StringVar(&serverFlag, "server", "", "the Maestro server to talk to (default: the one kept, asked for the first time)")
	// Asked at every request, so that what a flag says is what is used,
	// completions included.
	maestroapi.Server = func() string {
		url, _ := server()
		return url
	}
	rootCmd.PersistentFlags().BoolVar(&noColor, "no-color", false, "print without colours (as NO_COLOR=1 does)")
	// After the flags are read, before any command runs.
	cobra.OnInitialize(applyNoColor)
}

// writeJSON is what --json prints: the same things a table shows, as the
// server names them, for scripts.
func writeJSON(w io.Writer, v any) error {
	encoder := json.NewEncoder(w)
	encoder.SetIndent("", "  ")
	return encoder.Encode(v)
}

// How maestro exits: 0 when all went well, exitFailed when something it did
// failed (a run included, when it was watched to its end), exitUsage when it
// was asked wrongly, exitInterrupted when ctrl-c stopped it before the end.
const (
	exitFailed      = 1
	exitUsage       = 2
	exitInterrupted = 130
)

// exitError is a failure with an exit status of its own.
type exitError struct {
	code int
	err  error
}

func (e *exitError) Error() string { return e.err.Error() }
func (e *exitError) Unwrap() error { return e.err }

func Execute() {
	cmd, err := execute()
	if err != nil {
		os.Exit(report(os.Stderr, cmd, err))
	}
}

var prepared sync.Once

// commandGroups is how maestro --help lists the commands: by what they are
// for, in the order they are first needed.
var commandGroups = []struct {
	id, title string
	commands  []string
}{
	{"start", "Get started:", []string{"login", "build", "run"}},
	{"work", "Follow your work:", []string{"runs", "builds", "queue", "usage"}},
	{"forges", "Forges and repos:", []string{"forges", "repos"}},
	{"cluster", "Machines and scheduling:", []string{"top", "machines", "settings", "export"}},
	{"setup", "Setup:", []string{"logout", "completion", "help"}},
}

// groupCommands puts each command in its group, once all are added.
func groupCommands() {
	for _, g := range commandGroups {
		rootCmd.AddGroup(&cobra.Group{ID: g.id, Title: g.title})
		for _, name := range g.commands {
			switch name {
			case "help":
				rootCmd.SetHelpCommandGroupID(g.id)
			case "completion":
				rootCmd.SetCompletionCommandGroupID(g.id)
			default:
				if c, _, err := rootCmd.Find([]string{name}); err == nil && c != rootCmd {
					c.GroupID = g.id
				}
			}
		}
	}
}

// execute runs the command asked for, once every command has been added.
func execute() (*cobra.Command, error) {
	prepared.Do(func() {
		groupCommands()
		asUsage(rootCmd)
		rootCmd.SetFlagErrorFunc(func(_ *cobra.Command, err error) error { return usageError(err) })
	})
	return rootCmd.ExecuteC()
}

// report says what went wrong, if there is anything to say, and returns
// the status to exit with.
func report(w io.Writer, cmd *cobra.Command, err error) int {
	// Whoever pressed ctrl-c knows; the command said what goes on, if anything.
	if errors.Is(err, tui.ErrInterrupted) {
		return exitInterrupted
	}
	fmt.Fprintln(w, errorStyle.Render("Error:"), err)
	code := exitFailed
	var exit *exitError
	if errors.As(err, &exit) {
		code = exit.code
	}
	// A mistyped command or flag, not a command that failed: how it is asked.
	if strings.HasPrefix(err.Error(), "unknown ") {
		code = exitUsage
	}
	if code == exitUsage {
		if cmd == nil {
			cmd = rootCmd
		}
		if !strings.HasPrefix(err.Error(), "unknown command") {
			fmt.Fprintln(w, "\nUsage: "+cmd.UseLine())
		}
		fmt.Fprintf(w, "Run '%s --help' for more.\n", cmd.CommandPath())
	}
	return code
}

// runWelcome asks Maestro and every configured forge who the stored
// credentials belong to. Nothing it finds out is an error: whatever can't be
// reached is reported in place.
func runWelcome(cmd *cobra.Command, args []string) error {
	ctx, cancel := context.WithTimeout(cmd.Context(), 5*time.Second)
	defer cancel()

	w := tui.Welcome{}
	w.Server, w.ServerFrom = server()
	var wg sync.WaitGroup

	key, err := config.LoadMaestroKey(w.Server)
	var other *config.OtherServerError
	if errors.As(err, &other) {
		w.SignedInTo = other.Stored
	}
	if err == nil {
		w.SignedIn = true
		wg.Go(func() {
			w.Email, w.AccountErr = maestroapi.VerifyKey(ctx, key)
		})
	}

	forges, err := config.Forges()
	if err != nil {
		return err
	}
	w.Forges = make([]tui.ForgeIdentity, len(forges))
	for i, f := range forges {
		id := &w.Forges[i]
		id.Name = f.Name
		wg.Go(func() {
			active, err := openForge(f)
			if err != nil {
				id.Err = config.ErrNoForgeToken
				return
			}
			id.Host = active.Client.Host()
			user, err := active.Client.CurrentUser(ctx)
			if err != nil {
				id.Err = err
				return
			}
			id.Login = user.Login
		})
	}
	wg.Wait()

	width, _, _ := term.GetSize(int(os.Stdout.Fd()))
	fmt.Print(tui.RenderWelcome(w, width))
	return nil
}
