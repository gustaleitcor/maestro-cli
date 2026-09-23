package cmd

import (
	"github.com/spf13/cobra"

	"maestro-cli/internal/githubapi"
	"maestro-cli/tui"
)

var repoCmd = &cobra.Command{
	Use:   "repo",
	Short: "Repository-related commands",
}

var repoListCmd = &cobra.Command{
	Use:   "list",
	Short: "List repositories visible to the authenticated user",
	RunE:  runRepoList,
}

func init() {
	rootCmd.AddCommand(repoCmd)
	repoCmd.AddCommand(repoListCmd)
}

func runRepoList(cmd *cobra.Command, args []string) error {
	client := githubapi.NewClient(githubToken)
	return tui.RunRepoList(cmd.Context(), client)
}
