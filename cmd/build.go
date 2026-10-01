package cmd

import (
	"context"
	"fmt"
	"net/url"
	"strings"
	"time"

	"github.com/spf13/cobra"

	"maestro-cli/internal/forge"
	"maestro-cli/internal/maestroapi"
)

var (
	buildRef   string
	buildForge string
)

var buildCmd = &cobra.Command{
	Use:   "build <repo>",
	Short: "Build a container image from a repo on one of your forges",
	Long: `Builds a container image from a repository.

<repo> is a name (one of your own repos), owner/name, or the repo's full
URL. A URL picks the forge by its host; otherwise --forge does, and may be
left out when only one forge is configured.

  maestro build my-app
  maestro build some-org/my-app --ref v1.2.0
  maestro build https://codeberg.org/some-org/my-app
  maestro build group/subgroup/my-app --forge gitlab`,
	Args: cobra.ExactArgs(1),
	RunE: runBuild,

	ValidArgsFunction: completeRepos,
}

func init() {
	buildCmd.Flags().StringVar(&buildRef, "ref", "", "branch, tag, or commit SHA to build (default: repo's default branch)")
	addForgeFlag(buildCmd, &buildForge)
	rootCmd.AddCommand(buildCmd)
}

func runBuild(cmd *cobra.Command, args []string) error {
	active, owner, repo, err := parseRepoArg(cmd.Context(), args[0], buildForge)
	if err != nil {
		return err
	}

	ref := buildRef
	if ref == "" {
		r, err := active.Client.GetRepo(cmd.Context(), owner, repo)
		if err != nil {
			return fmt.Errorf("resolving default branch: %w", err)
		}
		ref = r.DefaultBranch
	}

	fmt.Printf("Building %s/%s/%s@%s...\n", active.Client.Host(), owner, repo, ref)

	final, err := maestroapi.TriggerBuild(cmd.Context(), maestroKey, active.Token, maestroapi.BuildRequest{
		Forge: active.Client.Kind(),
		Host:  active.Client.Host(),
		Owner: owner,
		Repo:  repo,
		Ref:   ref,
	}, func(line string) { fmt.Print(line) })
	if err != nil {
		return err
	}
	if final.Status != "success" {
		return fmt.Errorf("build failed: %s", final.Error)
	}

	fmt.Printf("\nBuild succeeded: %s\n", final.ImageID)
	return nil
}

// parseRepoArg accepts "name", "owner/name", or a full repo URL. A URL
// selects the forge by host; a bare name resolves the owner to the
// authenticated user. On GitLab the owner may be a nested group path.
func parseRepoArg(ctx context.Context, arg, forgeName string) (active *activeForge, owner, repo string, err error) {
	if strings.Contains(arg, "://") {
		host, path, err := splitRepoURL(arg)
		if err != nil {
			return nil, "", "", err
		}
		if active, err = selectForgeByHost(host, forgeName); err != nil {
			return nil, "", "", err
		}
		if owner, repo, err = splitRepoPath(active.Client.Kind(), path); err != nil {
			return nil, "", "", fmt.Errorf("invalid repo URL %q: %w", arg, err)
		}
		return active, owner, repo, nil
	}

	if active, err = selectForge(forgeName); err != nil {
		return nil, "", "", err
	}
	arg = strings.Trim(strings.TrimSuffix(arg, ".git"), "/")
	if !strings.Contains(arg, "/") {
		if arg == "" {
			return nil, "", "", fmt.Errorf("invalid repo: expected name, owner/name, or a repo URL")
		}
		user, err := active.Client.CurrentUser(ctx)
		if err != nil {
			return nil, "", "", fmt.Errorf("resolving authenticated user: %w", err)
		}
		return active, user.Login, arg, nil
	}
	if owner, repo, err = splitRepoPath(active.Client.Kind(), arg); err != nil {
		return nil, "", "", fmt.Errorf("invalid repo %q: %w", arg, err)
	}
	return active, owner, repo, nil
}

// splitRepoURL returns the host and the repo path of a browser or clone URL.
func splitRepoURL(raw string) (host, path string, err error) {
	parsed, err := url.Parse(raw)
	if err != nil || parsed.Host == "" || (parsed.Scheme != "https" && parsed.Scheme != "http") {
		return "", "", fmt.Errorf("invalid repo URL %q: expected https://host/owner/name", raw)
	}
	path = strings.Trim(parsed.Path, "/")
	// GitLab puts everything that isn't the project path behind "/-/".
	path, _, _ = strings.Cut(path, "/-/")
	return parsed.Host, strings.TrimSuffix(path, ".git"), nil
}

// splitRepoPath splits owner/name. GitHub and Forgejo owners are a single
// segment, so anything after owner/name (like /tree/main in a browser URL)
// is dropped; a GitLab owner is everything before the last segment.
func splitRepoPath(kind, path string) (owner, repo string, err error) {
	parts := strings.Split(path, "/")
	if len(parts) < 2 {
		return "", "", fmt.Errorf("expected owner/name")
	}
	for _, part := range parts {
		if part == "" {
			return "", "", fmt.Errorf("expected owner/name")
		}
	}
	if kind != forge.GitLab {
		return parts[0], parts[1], nil
	}
	owner, repo, _ = forge.SplitFullName(path)
	return owner, repo, nil
}

// completeRepos offers "owner/name" for every repo the user can see, plus the
// bare name for repos they own (parseRepoArg resolves a bare name to them),
// most recently updated first.
func completeRepos(cmd *cobra.Command, args []string, toComplete string) ([]string, cobra.ShellCompDirective) {
	if len(args) > 0 {
		return nil, cobra.ShellCompDirectiveNoFileComp
	}

	ctx, cancel := context.WithTimeout(context.Background(), 10*time.Second)
	defer cancel()

	active, err := selectForge(buildForge)
	if err != nil {
		return nil, cobra.ShellCompDirectiveNoFileComp
	}
	user, err := active.Client.CurrentUser(ctx)
	if err != nil {
		return nil, cobra.ShellCompDirectiveNoFileComp
	}
	repos, err := active.Client.ListRepos(ctx)
	if err != nil {
		return nil, cobra.ShellCompDirectiveNoFileComp
	}

	// Tabs separate a completion from its description; newlines end it.
	clean := strings.NewReplacer("\t", " ", "\n", " ", "\r", " ")

	var completions []string
	for _, r := range repos {
		candidates := []string{r.FullName}
		if r.FullName == user.Login+"/"+r.Name {
			candidates = append(candidates, r.Name)
		}
		for _, c := range candidates {
			if !strings.HasPrefix(c, toComplete) {
				continue
			}
			if r.Description != "" {
				c += "\t" + clean.Replace(r.Description)
			}
			completions = append(completions, c)
		}
	}
	// ListRepos already returns most recently updated first; keep that order
	// instead of letting the shell sort alphabetically.
	return completions, cobra.ShellCompDirectiveNoFileComp | cobra.ShellCompDirectiveKeepOrder
}
