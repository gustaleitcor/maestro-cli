package cmd

import (
	"fmt"
	"os"

	"github.com/spf13/cobra"

	"maestro-cli/internal/config"
)

var githubToken string
var maestroKey string

// Version is set via -ldflags at release build time (see .goreleaser.yaml).
var Version = "dev"

const welcomeText = `Welcome to Maestro — orchestration and repository tooling.

Get started:
  1. maestro login         Authenticate with GitHub and Maestro
  2. maestro me            Show your authenticated GitHub profile
  3. maestro repo list     Browse your repositories
  4. maestro build <repo>  Build a container image from a repo
  5. maestro image list    See the images you've built

Run 'maestro --help' to see all available commands.
`

var rootCmd = &cobra.Command{
	Use:     "maestro",
	Short:   "Maestro CLI — orchestration and repository tooling",
	Version: Version,
	// Bare invocation prints a welcome message instead of the usual help text.
	Run: func(cmd *cobra.Command, args []string) {
		fmt.Print(welcomeText)
	},
	// maestro (bare), login, help, and completion shouldn't require credentials to run.
	PersistentPreRunE: func(cmd *cobra.Command, args []string) error {
		switch cmd.Name() {
		case "maestro", "login", "help", "completion":
			return nil
		}

		gh, err := config.LoadGitHubToken()
		if err != nil {
			return fmt.Errorf("%w\n\nRun `maestro login` to authenticate", err)
		}
		mk, err := config.LoadMaestroKey()
		if err != nil {
			return fmt.Errorf("%w\n\nRun `maestro login` to authenticate", err)
		}
		githubToken = gh
		maestroKey = mk
		return nil
	},
}

func Execute() {
	if err := rootCmd.Execute(); err != nil {
		fmt.Fprintln(os.Stderr, "Error:", err)
		os.Exit(1)
	}
}
