package cmd

import (
	"github.com/spf13/cobra"

	"maestro-cli/tui"
)

var repoListForge string

var repoCmd = &cobra.Command{
	Use:   "repo",
	Short: "Repository-related commands",
}

var repoListCmd = &cobra.Command{
	Use:   "list",
	Short: "List the repositories a forge token can see",
	RunE:  runRepoList,
}

func init() {
	addForgeFlag(repoListCmd, &repoListForge)
	rootCmd.AddCommand(repoCmd)
	repoCmd.AddCommand(repoListCmd)
}

func runRepoList(cmd *cobra.Command, args []string) error {
	active, err := selectForge(repoListForge)
	if err != nil {
		return err
	}
	return tui.RunRepoList(cmd.Context(), active.Client)
}
