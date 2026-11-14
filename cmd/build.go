package cmd

import (
	"context"
	"errors"
	"fmt"
	"io/fs"
	"net/url"
	"os"
	"os/signal"
	"path/filepath"
	"slices"
	"strings"
	"sync"
	"time"

	"github.com/spf13/cobra"

	"maestro-cli/internal/config"
	"maestro-cli/internal/forge"
	"maestro-cli/internal/maestroapi"
	"maestro-cli/internal/netfail"
	"maestro-cli/tui"
)

var (
	buildRef   string
	buildForge string
	buildRun   bool
	buildLocal localFlags
)

// localFlags name files where the CLI runs to send in place of the repo's.
type localFlags struct {
	config, generator, dockerfile string
}

// maxLocalFile is the largest file sent, as the server takes no larger.
const maxLocalFile = 1 << 20

var buildCmd = &cobra.Command{
	Use:   "build <repo>",
	Short: "Build a container image from a repo on one of your forges",
	Long: `Builds a container image from a repo's Dockerfile (or Containerfile)
at the root of the repo, or from the one its maestro.toml names with
dockerfile = "path/to/Dockerfile". The root of the repo is the build's
context either way.

<repo> is a name, owner/name, or URL. The forge that has it is used; pass
--forge when several do. A public repo on GitHub, GitLab or Codeberg needs no
forge and no token: pass its full URL.

The build gets a number: ` + "`maestro run <build>`" + ` runs its image, and
` + "`maestro builds`" + ` shows the ones you have. With --run, the image is run as
soon as it is built; --watch implies --run, and follows the run (not the
build) until it is over.

To try a change without pushing it, send your own maestro.toml (-c), the
generator it names (-g) or the Dockerfile (-f) from here: the server builds
the repo with them in place of its own. Each may have any name here. A
change to maestro.toml or the generator alone needs no new build: send them
to ` + "`maestro run`" + ` instead, which runs the image you have.`,
	Example: `  maestro build my-app
  maestro build some-org/my-app --ref v1.2.0
  maestro build https://github.com/some-org/public-app
  maestro build my-app --watch
  maestro build my-app -c bench.toml -g genvrs.py -f Dockerfile
  build=$(maestro build my-app -q)`,
	Args: cobra.MatchAll(repoArg, flagCheck(checkWatch)),
	RunE: runBuild,

	ValidArgsFunction: completeRepos,
}

// repoArg is Args of build: the one repo to build.
func repoArg(cmd *cobra.Command, args []string) error {
	switch {
	case len(args) == 0:
		return fmt.Errorf("maestro build needs a repo: a name, owner/name or URL (see `maestro repos`)")
	case len(args) > 1:
		return fmt.Errorf("maestro build builds one repo at a time, not %d", len(args))
	}
	return nil
}

// buildFiles are what a repo can be built from. A maestro.toml may name a
// Dockerfile elsewhere, which only the server reads.
var buildFiles = []string{"Dockerfile", "Containerfile", "maestro.toml"}

func init() {
	buildCmd.Flags().StringVar(&buildRef, "ref", "", "branch, tag or commit to build (default: the default branch)")
	buildCmd.Flags().BoolVar(&buildRun, "run", false, "run the image once it is built")
	addWatchFlags(buildCmd, "run the image once it is built (as --run), and follow the run until it is over")
	addForgeFlag(buildCmd, &buildForge)
	buildCmd.Flags().StringVarP(&buildLocal.config, "config", "c", "", "a maestro.toml here to build with in place of the repo's")
	buildCmd.Flags().StringVarP(&buildLocal.generator, "generator", "g", "", "a generator script here, in place of the one maestro.toml names")
	buildCmd.Flags().StringVarP(&buildLocal.dockerfile, "dockerfile", "f", "", "a Dockerfile here to build from in place of the repo's")
	addAnswerFlags(buildCmd, "the build (and its run, with --run)")
	rootCmd.AddCommand(buildCmd)
}

func runBuild(cmd *cobra.Command, args []string) error {
	// Before the build, not after it.
	if err := checkWatch(); err != nil {
		return err
	}
	files, err := readLocalFiles(buildLocal)
	if err != nil {
		return err
	}
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

	if files.Dockerfile == "" {
		if err := checkBuildFile(cmd.Context(), active.Client, owner, repo, ref); err != nil {
			return err
		}
	}

	out := chatter()
	fmt.Fprintf(out, "Building %s/%s/%s@%s...\n", active.Client.Host(), owner, repo, ref)
	sayLocalFiles(buildLocal)

	// The server builds for as long as it is listened to: ctrl-c stops it.
	buildCtx, stop := signal.NotifyContext(cmd.Context(), os.Interrupt)
	final, err := maestroapi.TriggerBuild(buildCtx, maestroKey, active.Token, maestroapi.BuildRequest{
		Forge: active.Client.Kind(),
		Host:  active.Client.Host(),
		Owner: owner,
		Repo:  repo,
		Ref:   ref,
		Files: files,
	}, func(line string) { fmt.Fprint(out, line) })
	interrupted := buildCtx.Err() != nil
	stop()
	if interrupted {
		fmt.Fprintln(os.Stderr, "\n"+dimStyle.Render("Stopped: leaving cancels the build on the server too."))
		return tui.ErrInterrupted
	}
	if err != nil {
		return err
	}
	if final.Status != "success" {
		return fmt.Errorf("build failed: %s", final.Error)
	}
	if !files.Empty() && len(final.LocalFiles) == 0 {
		return fmt.Errorf("build #%d was built from the repo's files, not yours: this Maestro server is too old to take them; update it", final.BuildID)
	}

	succeeded := "Build succeeded:"
	if final.BuildID > 0 {
		succeeded = fmt.Sprintf("Build #%d succeeded:", final.BuildID)
	}
	fmt.Fprintf(out, "\n%s %s\n", successStyle.Render(succeeded), final.ImageID)

	if !buildRun && !runsWatch {
		return printAnswer(builtAnswer{Build: final}, final.BuildID)
	}
	if final.BuildID == 0 {
		return fmt.Errorf("this server doesn't say the build's number; run it with `maestro run <build>`")
	}
	run, err := startRun(cmd.Context(), final.BuildID)
	if err != nil {
		return err
	}
	if runsWatch {
		return showRun(cmd.Context(), run.ID, true)
	}
	return printAnswer(builtAnswer{Build: final, Run: run}, run.ID)
}

// builtAnswer is what build --json prints: the build, and the run of it
// that --run started.
type builtAnswer struct {
	Build *maestroapi.StreamEvent `json:"build"`
	Run   *maestroapi.Run         `json:"run,omitempty"`
}

// readLocalFiles reads the files flags name, each at most maxLocalFile.
func readLocalFiles(flags localFlags) (maestroapi.LocalFiles, error) {
	var files maestroapi.LocalFiles
	// Each is said as the flag that named it: "-c small.toml: no such file".
	for _, f := range []struct {
		flag, path string
		into       *string
	}{{"-c", flags.config, &files.Config}, {"-g", flags.generator, &files.Generator}, {"-f", flags.dockerfile, &files.Dockerfile}} {
		if f.path == "" {
			continue
		}
		named := f.flag + " " + f.path
		info, err := os.Stat(f.path)
		if errors.Is(err, fs.ErrNotExist) {
			return files, fmt.Errorf("%s: no such file", named)
		}
		if err != nil {
			return files, fmt.Errorf("%s: %w", named, err)
		}
		if !info.Mode().IsRegular() {
			return files, fmt.Errorf("%s: not a file", named)
		}
		if info.Size() > maxLocalFile {
			return files, fmt.Errorf("%s: larger than %d MiB, the most that can be sent", named, maxLocalFile>>20)
		}
		raw, err := os.ReadFile(f.path)
		if err != nil {
			return files, fmt.Errorf("%s: %w", named, err)
		}
		if len(raw) == 0 {
			return files, fmt.Errorf("%s: empty", named)
		}
		*f.into = string(raw)
	}
	if flags.generator != "" {
		files.GeneratorName = filepath.Base(flags.generator)
	}
	return files, nil
}

// sayLocalFiles says which files are sent from here, if any.
func sayLocalFiles(flags localFlags) {
	var sent []string
	for _, f := range []struct{ path, what string }{{flags.config, "maestro.toml"}, {flags.generator, "the generator"}, {flags.dockerfile, "the Dockerfile"}} {
		if f.path != "" {
			sent = append(sent, fmt.Sprintf("%s as %s", f.path, f.what))
		}
	}
	if len(sent) > 0 {
		fmt.Fprintln(chatter(), dimStyle.Render("Sending "+strings.Join(sent, ", ")+" from here."))
	}
}

// checkBuildFile stops a build that has nothing to build from, or a repo
// the token can't see, before it reaches the server. Only a clear "not
// there" stops it: when the forge can't say, the build goes ahead and the
// server decides.
func checkBuildFile(ctx context.Context, client forge.Forge, owner, repo, ref string) error {
	for _, name := range buildFiles {
		found, err := client.HasFile(ctx, owner, repo, ref, name)
		if err != nil {
			fmt.Fprintln(chatter(), warnStyle.Render("Warning: could not check for a Dockerfile: "+err.Error()))
			return nil
		}
		if found {
			return nil
		}
	}
	// A repo the token can't see has no files either: say that instead.
	if _, err := client.GetRepo(ctx, owner, repo); errors.Is(err, forge.ErrNotFound) {
		return fmt.Errorf("%s/%s not found, or your token can't see it\n\nIf it's private, add a token that can: maestro forges add", owner, repo)
	}
	return fmt.Errorf("%s/%s has no Dockerfile at %s\n\nMaestro builds the image from a Dockerfile (or Containerfile) at the root of\nthe repo, or from the one maestro.toml names with dockerfile = \"…\". Add one,\nor pick a branch that has it with --ref", owner, repo, ref)
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
		return nil, fmt.Errorf("no forge configured\n\nRun `maestro forges add` to add GitHub, a Forgejo instance, or GitLab")
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
	if slices.Contains(parts, "") {
		return "", "", fmt.Errorf("expected owner/name")
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
				description := descriptionCleaner.Replace(r.Description)
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
