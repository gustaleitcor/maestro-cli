package cmd

import (
	"context"
	"encoding/json"
	"errors"
	"fmt"
	"io"
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
	Use:     "maestro",
	Short:   "Maestro CLI — orchestration and repository tooling",
	Version: Version,
	// Execute prints the error once; a failed command isn't a usage mistake.
	SilenceErrors: true,
	SilenceUsage:  true,
	// Bare invocation shows who is signed in instead of the usual help text.
	RunE: runWelcome,
	// maestro (bare), login, logout, forges, help, and completion shouldn't require a
	// Maestro key to run. Commands that read a forge load it themselves, and
	// completions that need the key load it themselves too.
	PersistentPreRunE: func(cmd *cobra.Command, args []string) error {
		switch cmd.Name() {
		case "maestro", "login", "logout", "help", "completion", cobra.ShellCompRequestCmd, cobra.ShellCompNoDescRequestCmd:
			return nil
		}
		if cmd == forgesCmd || cmd.Parent() == forgesCmd {
			return nil
		}

		mk, err := config.LoadMaestroKey()
		if err != nil {
			return fmt.Errorf("%w\n\nRun `maestro login` to authenticate", err)
		}
		maestroKey = mk
		return nil
	},
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
	cmd, err := rootCmd.ExecuteC()
	if err != nil {
		os.Exit(report(os.Stderr, cmd, err))
	}
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
	// A mistyped command or flag, not a command that failed.
	if strings.HasPrefix(err.Error(), "unknown ") {
		fmt.Fprintln(w, "Run 'maestro --help' for usage.")
	}
	return code
}

// runWelcome asks Maestro and every configured forge who the stored
// credentials belong to. Nothing it finds out is an error: whatever can't be
// reached is reported in place.
func runWelcome(cmd *cobra.Command, args []string) error {
	ctx, cancel := context.WithTimeout(cmd.Context(), 5*time.Second)
	defer cancel()

	w := tui.Welcome{Server: maestroapi.BaseURL()}
	var wg sync.WaitGroup

	if key, err := config.LoadMaestroKey(); err == nil {
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
