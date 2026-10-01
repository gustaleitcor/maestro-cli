package cmd

import (
	"bufio"
	"context"
	"fmt"
	"io"
	"net/url"
	"os"
	"slices"
	"strings"
	"text/tabwriter"

	"github.com/spf13/cobra"

	"maestro-cli/internal/config"
	"maestro-cli/internal/forge"
)

var (
	forgeAddKind string
	forgeAddURL  string
)

var forgeCmd = &cobra.Command{
	Use:   "forge",
	Short: "Manage the git forges Maestro reads repositories from",
	Long: `A forge is where your repositories live: GitHub, a Forgejo instance
(Codeberg and Gitea speak the same API), or GitLab. Maestro only reads from
it, to list repos and to clone the one you build, so a read-only token is
all it needs. Forges never sign you in to Maestro; 'maestro login' does.`,
}

var forgeAddCmd = &cobra.Command{
	Use:   "add [name]",
	Short: "Add a forge and its read-only token",
	Long: `Adds a forge, asking for whatever isn't given on the command line, then
checks the token against the forge before storing it.

  maestro forge add                                   # asks for everything
  maestro forge add --kind forgejo                    # Codeberg
  maestro forge add work --kind gitlab --url https://gitlab.example.com

Adding a forge under a name that already exists replaces it.`,
	Args: cobra.MaximumNArgs(1),
	RunE: func(cmd *cobra.Command, args []string) error {
		name := ""
		if len(args) == 1 {
			name = args[0]
		}
		return addForge(cmd.Context(), bufio.NewReader(os.Stdin), name, forgeAddKind, forgeAddURL)
	},
}

var forgeListCmd = &cobra.Command{
	Use:   "list",
	Short: "List the configured forges",
	Args:  cobra.NoArgs,
	RunE: func(cmd *cobra.Command, args []string) error {
		forges, err := config.Forges()
		if err != nil {
			return err
		}
		if len(forges) == 0 {
			fmt.Println("No forges configured. Add one with: maestro forge add")
			return nil
		}
		w := tabwriter.NewWriter(os.Stdout, 0, 0, 2, ' ', 0)
		fmt.Fprintln(w, "NAME\tKIND\tURL")
		for _, f := range forges {
			baseURL := f.BaseURL
			if baseURL == "" {
				baseURL = forge.DefaultBaseURL(f.Kind)
			}
			fmt.Fprintf(w, "%s\t%s\t%s\n", f.Name, f.Kind, baseURL)
		}
		return w.Flush()
	},
}

var forgeRemoveCmd = &cobra.Command{
	Use:   "remove <name>",
	Short: "Remove a forge and forget its token",
	Args:  cobra.ExactArgs(1),
	RunE: func(cmd *cobra.Command, args []string) error {
		removed, err := config.RemoveForge(args[0])
		if err != nil {
			return err
		}
		if !removed {
			return fmt.Errorf("no forge named %q; see `maestro forge list`", args[0])
		}
		fmt.Printf("Removed forge %s.\n", args[0])
		return nil
	},
	ValidArgsFunction: completeForgeNames,
}

func init() {
	forgeAddCmd.Flags().StringVar(&forgeAddKind, "kind", "", "forge kind: "+strings.Join(forge.Kinds, ", "))
	forgeAddCmd.Flags().StringVar(&forgeAddURL, "url", "", "base URL of a self-hosted forge (default: the kind's public instance)")
	forgeAddCmd.RegisterFlagCompletionFunc("kind", func(*cobra.Command, []string, string) ([]string, cobra.ShellCompDirective) {
		return forge.Kinds, cobra.ShellCompDirectiveNoFileComp
	})

	rootCmd.AddCommand(forgeCmd)
	forgeCmd.AddCommand(forgeAddCmd, forgeListCmd, forgeRemoveCmd)
}

func completeForgeNames(cmd *cobra.Command, args []string, toComplete string) ([]string, cobra.ShellCompDirective) {
	forges, err := config.Forges()
	if err != nil {
		return nil, cobra.ShellCompDirectiveNoFileComp
	}
	var names []string
	for _, f := range forges {
		names = append(names, f.Name)
	}
	return names, cobra.ShellCompDirectiveNoFileComp
}

// addForgeFlag gives a command the --forge flag every forge-reading
// command shares.
func addForgeFlag(cmd *cobra.Command, target *string) {
	cmd.Flags().StringVar(target, "forge", "", "which configured forge to use (see `maestro forge list`)")
	cmd.RegisterFlagCompletionFunc("forge", completeForgeNames)
}

// addForge asks for whatever of name, kind and baseURL is empty, then for
// the token, and only stores the forge once the token works.
func addForge(ctx context.Context, stdin *bufio.Reader, name, kind, baseURL string) error {
	var err error
	if kind == "" {
		if kind, err = readLine(stdin, "Forge kind ("+strings.Join(forge.Kinds, ", ")+")", forge.GitHub); err != nil {
			return err
		}
	}
	kind = strings.ToLower(kind)
	if !slices.Contains(forge.Kinds, kind) {
		return fmt.Errorf("unknown forge kind %q: use %s", kind, strings.Join(forge.Kinds, ", "))
	}

	if baseURL == "" {
		if baseURL, err = readLine(stdin, "Forge URL", forge.DefaultBaseURL(kind)); err != nil {
			return err
		}
	}
	if !strings.Contains(baseURL, "://") {
		baseURL = "https://" + baseURL
	}
	baseURL = strings.TrimRight(baseURL, "/")
	parsed, err := url.Parse(baseURL)
	if err != nil || parsed.Host == "" {
		return fmt.Errorf("invalid forge URL %q", baseURL)
	}

	if name == "" {
		// github.com -> github, codeberg.org -> codeberg, git.example.com -> git.example.com
		suggested := strings.ToLower(parsed.Hostname())
		if baseURL == forge.DefaultBaseURL(kind) {
			suggested, _, _ = strings.Cut(suggested, ".")
		}
		if name, err = readLine(stdin, "Name for this forge", suggested); err != nil {
			return err
		}
	}

	fmt.Printf("\n%s\n\n", forge.TokenHint(kind, baseURL))
	token, err := readSecret(stdin, "Token", false)
	if err != nil {
		return err
	}

	client, err := forge.New(kind, baseURL, token)
	if err != nil {
		return err
	}
	fmt.Println("Validating token...")
	user, err := client.CurrentUser(ctx)
	if err != nil {
		return fmt.Errorf("token validation failed: %w", err)
	}

	stored := config.Forge{Name: name, Kind: kind}
	if baseURL != forge.DefaultBaseURL(kind) {
		stored.BaseURL = baseURL
	}
	if err := config.AddForge(stored, token); err != nil {
		return fmt.Errorf("saving forge: %w", err)
	}
	fmt.Printf("Forge %s: reading %s as %s.\n", name, client.Host(), user.Login)
	return nil
}

func readLine(stdin *bufio.Reader, label, fallback string) (string, error) {
	if fallback != "" {
		fmt.Printf("%s [%s]: ", label, fallback)
	} else {
		fmt.Printf("%s: ", label)
	}
	line, err := stdin.ReadString('\n')
	if err != nil && err != io.EOF {
		return "", fmt.Errorf("reading input: %w", err)
	}
	if line = strings.TrimSpace(line); line != "" {
		return line, nil
	}
	if fallback == "" {
		return "", fmt.Errorf("no value provided")
	}
	return fallback, nil
}

// activeForge is a configured forge ready to use: its client, plus the
// token maestro-orq needs to clone from it.
type activeForge struct {
	config.Forge
	Token  string
	Client forge.Forge
}

func openForge(f config.Forge) (*activeForge, error) {
	token, err := config.ForgeToken(f.Name)
	if err != nil {
		return nil, fmt.Errorf("%w\n\nRun `maestro forge add %s --kind %s` to store one", err, f.Name, f.Kind)
	}
	client, err := forge.New(f.Kind, f.BaseURL, token)
	if err != nil {
		return nil, err
	}
	return &activeForge{Forge: f, Token: token, Client: client}, nil
}

// selectForge resolves --forge. Without it, the only configured forge is
// used; with several, the caller has to say which.
func selectForge(name string) (*activeForge, error) {
	forges, err := config.Forges()
	if err != nil {
		return nil, err
	}
	if len(forges) == 0 {
		return nil, fmt.Errorf("no forge configured\n\nRun `maestro forge add` to add GitHub, a Forgejo instance, or GitLab")
	}

	if name == "" {
		if len(forges) > 1 {
			return nil, fmt.Errorf("several forges are configured (%s); pick one with --forge", forgeNames(forges))
		}
		return openForge(forges[0])
	}
	for _, f := range forges {
		if f.Name == name {
			return openForge(f)
		}
	}
	return nil, fmt.Errorf("no forge named %q (configured: %s)", name, forgeNames(forges))
}

// selectForgeByHost finds the forge a repo URL belongs to. name, when set,
// narrows the search to that forge, which must then serve the host.
func selectForgeByHost(host, name string) (*activeForge, error) {
	forges, err := config.Forges()
	if err != nil {
		return nil, err
	}
	for _, f := range forges {
		if name != "" && f.Name != name {
			continue
		}
		baseURL := f.BaseURL
		if baseURL == "" {
			baseURL = forge.DefaultBaseURL(f.Kind)
		}
		if parsed, err := url.Parse(baseURL); err == nil && strings.EqualFold(parsed.Host, host) {
			return openForge(f)
		}
	}
	if name != "" {
		return nil, fmt.Errorf("forge %q does not serve %s", name, host)
	}
	return nil, fmt.Errorf("no configured forge serves %s\n\nRun `maestro forge add --url https://%s` to add it", host, host)
}

func forgeNames(forges []config.Forge) string {
	names := make([]string, 0, len(forges))
	for _, f := range forges {
		names = append(names, f.Name)
	}
	return strings.Join(names, ", ")
}
