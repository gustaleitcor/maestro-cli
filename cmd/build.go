package cmd

import (
	"context"
	"errors"
	"fmt"
	"net/url"
	"slices"
	"strings"
	"sync"
	"time"

	"github.com/spf13/cobra"

	"maestro-cli/internal/config"
	"maestro-cli/internal/forge"
	"maestro-cli/internal/maestroapi"
	"maestro-cli/internal/netfail"
)

var (
	buildRef   string
	buildForge string
)

var buildCmd = &cobra.Command{
	Use:   "build <repo>",
	Short: "Build a container image from a repo on one of your forges",
	Long: `Builds a container image from a repo's Dockerfile (or Containerfile),
which must be at the root of the repo.

<repo> is a name, owner/name, or URL. The forge that has it is used; pass
--forge when several do.

  maestro build my-app
  maestro build some-org/my-app --ref v1.2.0
  maestro build https://codeberg.org/some-org/my-app`,
	Args: cobra.ExactArgs(1),
	RunE: runBuild,

	ValidArgsFunction: completeRepos,
}

var buildFiles = []string{"Dockerfile", "Containerfile"}

func init() {
	buildCmd.Flags().StringVar(&buildRef, "ref", "", "branch, tag or commit to build (default: the default branch)")
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

	if err := checkBuildFile(cmd.Context(), active.Client, owner, repo, ref); err != nil {
		return err
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

	fmt.Printf("\n%s %s\n", successStyle.Render("Build succeeded:"), final.ImageID)
	return nil
}

// checkBuildFile stops a build that has nothing to build from before it
// reaches the server. Only a clear "not there" stops it: when the forge
// can't say, the build goes ahead and the server decides.
func checkBuildFile(ctx context.Context, client forge.Forge, owner, repo, ref string) error {
	for _, name := range buildFiles {
		found, err := client.HasFile(ctx, owner, repo, ref, name)
		if err != nil {
			fmt.Println(warnStyle.Render("Warning: could not check for a Dockerfile: " + err.Error()))
			return nil
		}
		if found {
			return nil
		}
	}
	return fmt.Errorf("%s/%s has no Dockerfile at %s\n\nMaestro builds the image from a Dockerfile (or Containerfile) at the root of\nthe repo. Add one, or pick a branch that has it with --ref", owner, repo, ref)
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

	arg = strings.Trim(strings.TrimSuffix(arg, ".git"), "/")
	if arg == "" {
		return nil, "", "", fmt.Errorf("invalid repo: expected name, owner/name, or a repo URL")
	}

	candidates, err := forgesToSearch(forgeName)
	if err != nil {
		return nil, "", "", err
	}
	if len(candidates) == 1 {
		owner, repo, err = splitRepoArg(ctx, candidates[0], arg)
		return candidates[0], owner, repo, err
	}

	// Several forges and no --forge: the repo says which one, by being there.
	type match struct {
		active      *activeForge
		owner, repo string
	}
	found := make([]*match, len(candidates))
	failed := make([]error, len(candidates)) // anything but "it isn't here"
	var wg sync.WaitGroup
	for i, c := range candidates {
		wg.Go(func() {
			owner, repo, err := splitRepoArg(ctx, c, arg)
			if err == nil {
				_, err = c.Client.GetRepo(ctx, owner, repo)
			}
			switch {
			case err == nil:
				found[i] = &match{c, owner, repo}
			case !errors.Is(err, forge.ErrNotFound):
				failed[i] = fmt.Errorf("forge %s: %w", c.Name, err)
			}
		})
	}
	wg.Wait()

	var matches []*match
	var names []string
	for _, m := range found {
		if m != nil {
			matches = append(matches, m)
			names = append(names, m.active.Name)
		}
	}
	switch len(matches) {
	case 0:
		// A forge that couldn't answer may well be the one that has it.
		failed = slices.DeleteFunc(failed, func(err error) bool { return err == nil })
		if len(failed) == len(candidates) && allUnreachable(failed) {
			return nil, "", "", fmt.Errorf("could not reach any of your forges: check your internet connection")
		}
		if len(failed) > 0 {
			return nil, "", "", failed[0]
		}
		return nil, "", "", fmt.Errorf("no configured forge has a repo %q", arg)
	case 1:
		return matches[0].active, matches[0].owner, matches[0].repo, nil
	}
	return nil, "", "", fmt.Errorf("%q is on several forges (%s); pick one with --forge", arg, strings.Join(names, ", "))
}

func allUnreachable(errs []error) bool {
	for _, err := range errs {
		if !errors.Is(err, netfail.ErrUnreachable) {
			return false
		}
	}
	return true
}

func forgesToSearch(name string) ([]*activeForge, error) {
	if name != "" {
		active, err := selectForge(name)
		if err != nil {
			return nil, err
		}
		return []*activeForge{active}, nil
	}

	configured, err := config.Forges()
	if err != nil {
		return nil, err
	}
	if len(configured) == 0 {
		return nil, fmt.Errorf("no forge configured\n\nRun `maestro forge add` to add GitHub, a Forgejo instance, or GitLab")
	}
	all := make([]*activeForge, 0, len(configured))
	for _, f := range configured {
		active, err := openForge(f)
		if err != nil {
			return nil, err
		}
		all = append(all, active)
	}
	return all, nil
}

func splitRepoArg(ctx context.Context, active *activeForge, arg string) (owner, repo string, err error) {
	if !strings.Contains(arg, "/") {
		user, err := active.Client.CurrentUser(ctx)
		if err != nil {
			return "", "", fmt.Errorf("resolving authenticated user: %w", err)
		}
		return user.Login, arg, nil
	}
	if owner, repo, err = splitRepoPath(active.Client.Kind(), arg); err != nil {
		return "", "", fmt.Errorf("invalid repo %q: %w", arg, err)
	}
	return owner, repo, nil
}

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

func completeRepos(cmd *cobra.Command, args []string, toComplete string) ([]string, cobra.ShellCompDirective) {
	if len(args) > 0 {
		return nil, cobra.ShellCompDirectiveNoFileComp
	}

	ctx, cancel := context.WithTimeout(context.Background(), 10*time.Second)
	defer cancel()

	forges, err := forgesToSearch(buildForge)
	if err != nil {
		return nil, cobra.ShellCompDirectiveNoFileComp
	}

	// Tabs separate a completion from its description; newlines end it.
	clean := strings.NewReplacer("\t", " ", "\n", " ", "\r", " ")

	// Every forge is asked at once; one that fails just offers nothing.
	perForge := make([][]string, len(forges))
	var wg sync.WaitGroup
	for i, active := range forges {
		wg.Go(func() {
			repos, err := active.Client.ListRepos(ctx)
			if err != nil {
				return
			}
			for _, r := range repos {
				c := r.FullName
				description := clean.Replace(r.Description)
				if len(forges) > 1 {
					// Say where it lives, since `maestro build` finds that out itself.
					description = strings.TrimSuffix(active.Name+": "+description, ": ")
				}
				if !strings.HasPrefix(c, toComplete) {
					continue
				}
				if description != "" {
					c += "\t" + description
				}
				perForge[i] = append(perForge[i], c)
			}
		})
	}
	wg.Wait()
	completions := slices.Concat(perForge...)

	// ListRepos already returns most recently updated first; keep that order
	// instead of letting the shell sort alphabetically.
	return completions, cobra.ShellCompDirectiveNoFileComp | cobra.ShellCompDirectiveKeepOrder
}
