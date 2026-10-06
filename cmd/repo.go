package cmd

import (
	"fmt"

	"github.com/spf13/cobra"

	"maestro-cli/internal/config"
	"maestro-cli/tui"
)

var repoListForge string

var repoCmd = &cobra.Command{
	Use:   "repo",
	Short: "Repository-related commands",
}

var repoListCmd = &cobra.Command{
	Use:   "list",
	Short: "List the repositories your forge tokens can see",
	Long: `Lists one forge's repositories. Left and right arrows switch forge;
--forge picks the one shown first.`,
	Args:              cobra.NoArgs,
	ValidArgsFunction: cobra.NoFileCompletions,
	RunE:              runRepoList,
}

func init() {
	addForgeFlag(repoListCmd, &repoListForge)
	rootCmd.AddCommand(repoCmd)
	repoCmd.AddCommand(repoListCmd)
}

// runRepoList opens every configured forge so the list can switch between
// them; --forge only picks which one it starts on.
func runRepoList(cmd *cobra.Command, args []string) error {
	configured, err := config.Forges()
	if err != nil {
		return err
	}
	if len(configured) == 0 {
		return fmt.Errorf("no forge configured\n\nRun `maestro forge add` to add GitHub, a Forgejo instance, or GitLab")
	}

	start := 0
	forges := make([]tui.RepoForge, 0, len(configured))
	for i, f := range configured {
		active, err := openForge(f)
		if err != nil {
			return err
		}
		forges = append(forges, tui.RepoForge{Name: f.Name, Client: active.Client})
		if f.Name == repoListForge {
			start = i
		}
	}
	if repoListForge != "" && configured[start].Name != repoListForge {
		return fmt.Errorf("no forge named %q (configured: %s)", repoListForge, forgeNames(configured))
	}
	return tui.RunRepoList(cmd.Context(), forges, start)
}
