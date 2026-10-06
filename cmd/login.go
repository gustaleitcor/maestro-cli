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
	Long: `Prints a link to the Maestro page and a short code to approve there.
The link can be opened in any browser, on this machine or another.`,
	RunE: runLogin,
}

func init() {
	rootCmd.AddCommand(loginCmd)
}

func runLogin(cmd *cobra.Command, args []string) error {
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

	fmt.Println("\n" + warnStyle.Render("No git forge is configured yet.") + " Maestro reads your repositories from one")
	fmt.Println("(GitHub, a Forgejo instance such as Codeberg, or GitLab) with a read-only token.")
	answer, err := readLine(stdin, "Add one now? (y/n)", "y")
	if err != nil {
		return err
	}
	if !strings.HasPrefix(strings.ToLower(answer), "y") {
		fmt.Println(dimStyle.Render("Skipped. Add one later with: maestro forge add"))
		return nil
	}
	return addForge(ctx, stdin, "", "", "")
}

// loginMaestroInBrowser has the user approve a short code on the Maestro
// page, then receives a key minted for this CLI.
func loginMaestroInBrowser(ctx context.Context) error {
	login, err := maestroapi.StartCLILogin(ctx)
	if err != nil {
		return fmt.Errorf("starting Maestro login: %w", err)
	}

	fmt.Printf("\nTo sign in to Maestro, open:\n\n  %s\n\nand approve this code:\n\n  %s\n\n", accentStyle.Render(login.VerificationURL), accentStyle.Render(login.UserCode))
	fmt.Println(dimStyle.Render("Waiting for approval..."))

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
			if err := config.SaveMaestroKey(result.Key); err != nil {
				return fmt.Errorf("saving Maestro key: %w", err)
			}
			fmt.Println(successStyle.Render("Maestro: logged in as " + result.Email + "."))
			return nil
		case "denied":
			return fmt.Errorf("the login was denied on the Maestro page")
		default:
			return fmt.Errorf("the login expired; run `maestro login` again")
		}
	}
	return fmt.Errorf("the login expired; run `maestro login` again")
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
		masked, err := readMasked()
		if err != nil {
			return "", fmt.Errorf("reading input: %w", err)
		}
		value = strings.TrimSpace(masked)
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
