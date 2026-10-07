package cmd

import (
	"context"
	"fmt"
	"io"
	"os"
	"sync"
	"text/tabwriter"
	"time"

	"github.com/spf13/cobra"
	"golang.org/x/term"

	"maestro-cli/internal/config"
	"maestro-cli/tui"
)

var (
	repoListForge string
	reposAsJSON   bool
)

// repos was `repo` once, which still works. On its own it lists.
var reposCmd = &cobra.Command{
	Use:     "repos",
	Aliases: []string{"repo"},
	Short:   "Repository-related commands",
	Args:    cobra.NoArgs,
	RunE:    runReposList,
}

var reposListCmd = &cobra.Command{
	Use:   "list",
	Short: "List the repositories your forge tokens can see",
	Long: `Lists one forge's repositories. Left and right arrows switch forge;
--forge picks the one shown first.

Piped somewhere, or with --json, it prints every forge's repositories once
instead, or only those of --forge.`,
	Args:              cobra.NoArgs,
	ValidArgsFunction: cobra.NoFileCompletions,
	RunE:              runReposList,
}

func init() {
	for _, c := range []*cobra.Command{reposCmd, reposListCmd} {
		addForgeFlag(c, &repoListForge)
		c.Flags().BoolVar(&reposAsJSON, "json", false, "print the repositories as JSON")
	}
	rootCmd.AddCommand(reposCmd)
	reposCmd.AddCommand(reposListCmd)
}

// runReposList opens every configured forge so the list can switch between
// them; --forge only picks which one it starts on. Where there is no
// terminal to take over, it prints them instead.
func runReposList(cmd *cobra.Command, args []string) error {
	if reposAsJSON || !term.IsTerminal(int(os.Stdout.Fd())) {
		rows, err := loadRepos(cmd.Context(), repoListForge)
		if err != nil {
			return err
		}
		if reposAsJSON {
			return writeJSON(os.Stdout, rows)
		}
		return writeReposList(os.Stdout, rows)
	}

	configured, err := config.Forges()
	if err != nil {
		return err
	}
	if len(configured) == 0 {
		return fmt.Errorf("no forge configured\n\nRun `maestro forges add` to add GitHub, a Forgejo instance, or GitLab")
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

// repoRow is a repo and the forge it was found on.
type repoRow struct {
	Forge         string    `json:"forge"`
	Repo          string    `json:"repo"`
	Description   string    `json:"description,omitempty"`
	Private       bool      `json:"private"`
	Fork          bool      `json:"fork"`
	Language      string    `json:"language,omitempty"`
	Stars         int       `json:"stars"`
	UpdatedAt     time.Time `json:"updated_at"`
	URL           string    `json:"url"`
	CloneURL      string    `json:"clone_url"`
	DefaultBranch string    `json:"default_branch"`
}

// loadRepos asks every forge at once, or only the one named, and keeps them
// in the order they are configured. One that fails fails the list: half of
// it would read as all of it.
func loadRepos(ctx context.Context, forgeName string) ([]repoRow, error) {
	forges, err := forgesToSearch(forgeName)
	if err != nil {
		return nil, err
	}
	perForge := make([][]repoRow, len(forges))
	failed := make([]error, len(forges))
	var wg sync.WaitGroup
	for i, active := range forges {
		wg.Go(func() {
			repos, err := active.Client.ListRepos(ctx)
			if err != nil {
				failed[i] = fmt.Errorf("forge %s: %w", active.Name, err)
				return
			}
			for _, r := range repos {
				perForge[i] = append(perForge[i], repoRow{
					Forge: active.Name, Repo: r.FullName, Description: r.Description,
					Private: r.Private, Fork: r.Fork, Language: r.Language, Stars: r.Stars,
					UpdatedAt: r.UpdatedAt, URL: r.HTMLURL, CloneURL: r.CloneURL, DefaultBranch: r.DefaultBranch,
				})
			}
		})
	}
	wg.Wait()
	rows := []repoRow{}
	for i, err := range failed {
		if err != nil {
			return nil, err
		}
		rows = append(rows, perForge[i]...)
	}
	return rows, nil
}

func writeReposList(out io.Writer, rows []repoRow) error {
	if len(rows) == 0 {
		_, err := fmt.Fprintln(out, "No repositories: your forge tokens see none.")
		return err
	}
	w := tabwriter.NewWriter(out, 0, 0, 2, ' ', 0)
	fmt.Fprintln(w, "FORGE\tREPO\tVISIBILITY\tLANGUAGE\tUPDATED")
	for _, r := range rows {
		visibility := "public"
		if r.Private {
			visibility = "private"
		}
		language := r.Language
		if language == "" {
			language = "-"
		}
		fmt.Fprintf(w, "%s\t%s\t%s\t%s\t%s\n", r.Forge, r.Repo, visibility, language, r.UpdatedAt.Format("2006-01-02"))
	}
	return w.Flush()
}
