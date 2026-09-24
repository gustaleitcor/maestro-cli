package cmd

import (
	"context"
	"fmt"
	"strings"
	"time"

	"github.com/google/go-github/v66/github"
	"github.com/spf13/cobra"

	"maestro-cli/internal/githubapi"
	"maestro-cli/internal/maestroapi"
)

var buildRef string

var buildCmd = &cobra.Command{
	Use:   "build <repo>",
	Short: "Build a container image from a GitHub repo",
	Args:  cobra.ExactArgs(1),
	RunE:  runBuild,

	ValidArgsFunction: completeRepos,
}

func init() {
	buildCmd.Flags().StringVar(&buildRef, "ref", "", "branch, tag, or commit SHA to build (default: repo's default branch)")
	rootCmd.AddCommand(buildCmd)
}

func runBuild(cmd *cobra.Command, args []string) error {
	client := githubapi.NewClient(githubToken)

	owner, repo, err := parseRepoArg(cmd.Context(), client, args[0])
	if err != nil {
		return err
	}

	ref := buildRef
	if ref == "" {
		r, err := githubapi.GetRepo(cmd.Context(), client, owner, repo)
		if err != nil {
			return fmt.Errorf("resolving default branch: %w", err)
		}
		ref = r.DefaultBranch
	}

	fmt.Printf("Building %s/%s@%s...\n", owner, repo, ref)

	final, err := maestroapi.TriggerBuild(cmd.Context(), maestroKey, githubToken, maestroapi.BuildRequest{
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

// parseRepoArg accepts "name", "owner/name", or a github.com URL; a bare
// name resolves owner to the authenticated user.
func parseRepoArg(ctx context.Context, client *github.Client, arg string) (owner, repo string, err error) {
	arg = strings.TrimSuffix(arg, ".git")
	arg = strings.TrimPrefix(arg, "https://github.com/")
	arg = strings.TrimPrefix(arg, "http://github.com/")

	switch parts := strings.Split(arg, "/"); len(parts) {
	case 1:
		user, err := githubapi.GetAuthenticatedUser(ctx, client)
		if err != nil {
			return "", "", fmt.Errorf("resolving authenticated user: %w", err)
		}
		return user.Login, parts[0], nil
	case 2:
		return parts[0], parts[1], nil
	default:
		return "", "", fmt.Errorf("invalid repo %q: expected name, owner/name, or a github.com URL", arg)
	}
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
	client := githubapi.NewClient(githubToken)

	user, err := githubapi.GetAuthenticatedUser(ctx, client)
	if err != nil {
		return nil, cobra.ShellCompDirectiveNoFileComp
	}
	repos, err := githubapi.ListRepos(ctx, client)
	if err != nil {
		return nil, cobra.ShellCompDirectiveNoFileComp
	}

	// Tabs separate a completion from its description; newlines end it.
	clean := strings.NewReplacer("\t", " ", "\n", " ", "\r", " ")

	var completions []string
	for _, r := range repos {
		candidates := []string{r.FullName}
		if strings.HasPrefix(r.FullName, user.Login+"/") {
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
