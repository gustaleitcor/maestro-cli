package cmd

import (
	"bufio"
	"context"
	"fmt"
	"io"
	"os"
	"strings"

	"github.com/google/go-github/v66/github"
	"github.com/spf13/cobra"
	"golang.org/x/term"

	"maestro-cli/internal/config"
	"maestro-cli/internal/maestroapi"
)

var loginCmd = &cobra.Command{
	Use:   "login",
	Short: "Authenticate the CLI with a GitHub token and a Maestro key",
	Long: `Prompts for a GitHub personal access token and a Maestro key, validates
each against its own service, and stores them for future commands.

Maestro key:
  Generate one at https://maestro.logsad.com/

GitHub personal access token:
  Create a fine-grained token at
  https://github.com/settings/personal-access-tokens/new
  with "Repository access" set to All repositories, and under
  "Permissions" grant Contents: Read-only.
`,
	RunE: runLogin,
}

func init() {
	rootCmd.AddCommand(loginCmd)
}

func runLogin(cmd *cobra.Command, args []string) error {
	stdin := bufio.NewReader(os.Stdin)

	if err := loginGitHub(cmd.Context(), stdin); err != nil {
		return err
	}
	if err := loginMaestro(cmd.Context(), stdin); err != nil {
		return err
	}
	return nil
}

func loginGitHub(ctx context.Context, stdin *bufio.Reader) error {
	_, err := config.LoadGitHubToken()
	hasExisting := err == nil

	token, err := readSecret(stdin, "GitHub personal access token", hasExisting)
	if err != nil {
		return err
	}
	if token == "" {
		fmt.Println("GitHub: keeping existing token.")
		return nil
	}

	fmt.Println("Validating GitHub token...")
	user, err := validateGitHubToken(ctx, token)
	if err != nil {
		return fmt.Errorf("GitHub token validation failed: %w", err)
	}

	if err := config.SaveGitHubToken(token); err != nil {
		return fmt.Errorf("saving GitHub token: %w", err)
	}
	fmt.Printf("GitHub: logged in as %s.\n", user)
	return nil
}

func loginMaestro(ctx context.Context, stdin *bufio.Reader) error {
	_, err := config.LoadMaestroKey()
	hasExisting := err == nil

	key, err := readSecret(stdin, "Maestro key", hasExisting)
	if err != nil {
		return err
	}
	if key == "" {
		fmt.Println("Maestro: keeping existing key.")
		return nil
	}

	fmt.Println("Validating Maestro key...")
	email, err := maestroapi.VerifyKey(ctx, key)
	if err != nil {
		return fmt.Errorf("Maestro key validation failed: %w", err)
	}

	if err := config.SaveMaestroKey(key); err != nil {
		return fmt.Errorf("saving Maestro key: %w", err)
	}
	fmt.Printf("Maestro: logged in as %s.\n", email)
	return nil
}

// If hasExisting, a blank answer returns "" instead of erroring.
func readSecret(stdin *bufio.Reader, label string, hasExisting bool) (string, error) {
	if hasExisting {
		fmt.Printf("%s (leave blank to keep current): ", label)
	} else {
		fmt.Printf("%s: ", label)
	}

	var value string
	if term.IsTerminal(int(os.Stdin.Fd())) {
		raw, err := term.ReadPassword(int(os.Stdin.Fd()))
		fmt.Println()
		if err != nil {
			return "", fmt.Errorf("reading input: %w", err)
		}
		value = string(raw)
	} else {
		line, err := stdin.ReadString('\n')
		if err != nil && err != io.EOF {
			return "", fmt.Errorf("reading input: %w", err)
		}
		value = strings.TrimSpace(line)
	}

	if value == "" && !hasExisting {
		return "", fmt.Errorf("no value provided")
	}
	return value, nil
}

func validateGitHubToken(ctx context.Context, token string) (string, error) {
	client := github.NewClient(nil).WithAuthToken(token)
	u, _, err := client.Users.Get(ctx, "")
	if err != nil {
		return "", err
	}
	return u.GetLogin(), nil
}
