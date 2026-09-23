package cmd

import (
	"github.com/spf13/cobra"

	"maestro-cli/internal/githubapi"
	"maestro-cli/tui"
)

var meCmd = &cobra.Command{
	Use:   "me",
	Short: "Show the authenticated GitHub user",
	RunE:  runMe,
}

func init() {
	rootCmd.AddCommand(meCmd)
}

func runMe(cmd *cobra.Command, args []string) error {
	client := githubapi.NewClient(githubToken)
	return tui.RunMe(cmd.Context(), client)
}
