package cmd

import (
	"context"
	"fmt"
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
	// maestro (bare), login, logout, forge, help, and completion shouldn't require a
	// Maestro key to run. Commands that read a forge load it themselves, and
	// completions that need the key load it themselves too.
	PersistentPreRunE: func(cmd *cobra.Command, args []string) error {
		switch cmd.Name() {
		case "maestro", "login", "logout", "help", "completion", cobra.ShellCompRequestCmd, cobra.ShellCompNoDescRequestCmd:
			return nil
		}
		if cmd == forgeCmd || cmd.Parent() == forgeCmd {
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

func Execute() {
	if err := rootCmd.Execute(); err != nil {
		fmt.Fprintln(os.Stderr, errorStyle.Render("Error:"), err)
		// A mistyped command or flag, not a command that failed.
		if strings.HasPrefix(err.Error(), "unknown ") {
			fmt.Fprintln(os.Stderr, "Run 'maestro --help' for usage.")
		}
		os.Exit(1)
	}
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
