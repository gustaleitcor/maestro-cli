package cmd

import (
	"bufio"
	"context"
	"fmt"
	"io"
	"os"
	"strings"
	"time"

	"github.com/spf13/cobra"
	"golang.org/x/term"

	"maestro-cli/internal/config"
	"maestro-cli/internal/maestroapi"
)

var loginCmd = &cobra.Command{
	Use:   "login",
	Short: "Sign the CLI in to Maestro",
	Long: `Prints a link to the Maestro page and a short code for you to approve
there. You can open the link in any browser, on your computer or another one.

With --server, it signs you in to that Maestro server, and every later
command talks to it. Without --server, it uses the server the CLI has kept,
and asks for one the first time. The CLI sends your key only to the server
that issued it.

The CLI keeps your key in the system keyring. Without --store-in-file,
` + "`maestro login`" + ` goes back to the keyring. If your keyring asks for a
password, or you have none, pass --store-in-file to keep secrets in the
config file instead. Only you can read that file.`,
	RunE: runLogin,
}

var storeInFile bool

// addStoreFlag adds --store-in-file to the commands that save a secret.
func addStoreFlag(cmd *cobra.Command) {
	cmd.Flags().BoolVar(&storeInFile, "store-in-file", false, "keep secrets in the config file from now on, instead of the system keyring")
}

// applyStoreFlag must run before anything is saved. Logging in and adding a
// forge are what choose where secrets live, so without the flag they go back
// to the keyring.
func applyStoreFlag() error {
	return config.UseFileStore(storeInFile)
}

func init() {
	addStoreFlag(loginCmd)
	rootCmd.AddCommand(loginCmd)
}

func runLogin(cmd *cobra.Command, args []string) error {
	if err := applyStoreFlag(); err != nil {
		return err
	}
	stdin := bufio.NewReader(os.Stdin)

	if err := loginMaestroInBrowser(cmd.Context()); err != nil {
		return err
	}
	return offerForge(cmd.Context(), stdin)
}

// offerForge only speaks up when there is no forge at all. A GitHub token
// stored by an older CLI already counts as one.
func offerForge(ctx context.Context, stdin *bufio.Reader) error {
	forges, err := config.Forges()
	if err != nil {
		return err
	}
	if len(forges) > 0 {
		return nil
	}

	fmt.Println("\n" + warnStyle.Render("You have no git forge yet.") + " Maestro reads your repositories from a forge")
	fmt.Println("(GitHub, GitLab, or a Forgejo server such as Codeberg) with a read-only token of yours.")
	answer, err := readLine(stdin, "Add one now? (y/n)", "y")
	if err != nil {
		return err
	}
	if !strings.HasPrefix(strings.ToLower(answer), "y") {
		fmt.Println(dimStyle.Render("Skipped. Add one later with `maestro forges add`."))
		return nil
	}
	return addForge(ctx, stdin, "", "", "")
}

// loginMaestroInBrowser has the user approve a short code on the Maestro
// page, then receives a key issued for this CLI.
func loginMaestroInBrowser(ctx context.Context) error {
	login, err := maestroapi.StartCLILogin(ctx)
	if err != nil {
		return fmt.Errorf("couldn't start signing in: %w", err)
	}

	fmt.Printf("\nTo sign in to Maestro, open:\n\n  %s\n\nand approve this code:\n\n  %s\n\n", accentStyle.Render(login.VerificationURL), accentStyle.Render(login.UserCode))
	fmt.Println(dimStyle.Render("Waiting for you to approve the code..."))

	interval := time.Duration(login.Interval) * time.Second
	if interval <= 0 {
		interval = 2 * time.Second
	}
	deadline := time.Now().Add(time.Duration(login.ExpiresIn) * time.Second)

	for time.Now().Before(deadline) {
		select {
		case <-ctx.Done():
			return ctx.Err()
		case <-time.After(interval):
		}

		result, err := maestroapi.PollCLILogin(ctx, login.DeviceCode)
		if err != nil {
			// A dropped connection shouldn't throw away a login the user
			// may be about to approve.
			continue
		}
		switch result.Status {
		case "pending":
		case "approved":
			if err := config.SaveMaestroKey(maestroapi.BaseURL(), result.Key); err != nil {
				return fmt.Errorf("couldn't save your Maestro key: %w", err)
			}
			fmt.Println(successStyle.Render("Signed in to " + maestroapi.BaseURL() + " as " + result.Email + "."))
			return nil
		case "denied":
			return fmt.Errorf("the code was denied on the Maestro page; run `maestro login` to get a new one")
		default:
			return fmt.Errorf("the code expired; run `maestro login` again")
		}
	}
	return fmt.Errorf("the code expired; run `maestro login` again")
}

// If hasExisting, a blank answer returns "" instead of erroring.
func readSecret(stdin *bufio.Reader, label string, hasExisting bool) (string, error) {
	if hasExisting {
		fmt.Printf("%s (leave it blank to keep the current one): ", label)
	} else {
		fmt.Printf("%s: ", label)
	}

	var value string
	if term.IsTerminal(int(os.Stdin.Fd())) {
		masked, err := readMasked()
		if err != nil {
			return "", fmt.Errorf("couldn't read what you typed: %w", err)
		}
		value = strings.TrimSpace(masked)
	} else {
		line, err := stdin.ReadString('\n')
		if err != nil && err != io.EOF {
			return "", fmt.Errorf("couldn't read what you typed: %w", err)
		}
		value = strings.TrimSpace(line)
	}

	if value == "" && !hasExisting {
		return "", fmt.Errorf("nothing was typed")
	}
	return value, nil
}

func readMasked() (string, error) {
	fd := int(os.Stdin.Fd())
	state, err := term.MakeRaw(fd)
	if err != nil {
		return "", err
	}
	defer term.Restore(fd, state)

	var value []byte
	buf := make([]byte, 256)
	for {
		n, err := os.Stdin.Read(buf)
		if err != nil {
			fmt.Print("\r\n")
			return "", err
		}
		for _, c := range buf[:n] {
			switch {
			case c == '\r' || c == '\n':
				fmt.Print("\r\n")
				return string(value), nil
			case c == 3: // ctrl+c: raw mode keeps the terminal from sending the signal
				fmt.Print("\r\n")
				return "", fmt.Errorf("interrupted")
			case c == 127 || c == 8: // backspace
				if len(value) > 0 {
					value = value[:len(value)-1]
					fmt.Print("\b \b")
				}
			case c == 21: // ctrl+u clears the line
				fmt.Print(strings.Repeat("\b \b", len(value)))
				value = value[:0]
			case c >= 32:
				value = append(value, c)
				fmt.Print("*")
			}
		}
	}
}
