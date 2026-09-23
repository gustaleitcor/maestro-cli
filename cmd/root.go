package cmd

import (
	"fmt"
	"os"

	"github.com/spf13/cobra"

	"maestro-cli/internal/config"
)

var githubToken string
var maestroKey string

var rootCmd = &cobra.Command{
	Use:   "maestro",
	Short: "Maestro CLI — orchestration and repository tooling",
	// login, help, and completion shouldn't require credentials to run.
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
