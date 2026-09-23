// Package cmd contains the Maestro CLI's Cobra commands.
package cmd

import (
	"fmt"
	"os"

	"github.com/spf13/cobra"

	"maestro-cli/internal/config"
)

// githubToken and maestroKey hold the loaded credentials for the current
// command run, set by rootCmd's PersistentPreRunE and read by subcommands
// that hit the GitHub or Maestro APIs.
var githubToken string
var maestroKey string

var rootCmd = &cobra.Command{
	Use:   "maestro",
	Short: "Maestro CLI — orchestration and repository tooling",
	// Skip the auth check for `login` itself, and for help/completion,
	// which shouldn't require credentials to run.
	PersistentPreRunE: func(cmd *cobra.Command, args []string) error {
		switch cmd.Name() {
		case "login", "help", "completion":
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
