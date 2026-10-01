package cmd

import (
	"bufio"
	"context"
	"fmt"
	"io"
	"os"
	"os/exec"
	"runtime"
	"strings"
	"time"

	"github.com/spf13/cobra"
	"golang.org/x/term"

	"maestro-cli/internal/config"
	"maestro-cli/internal/maestroapi"
)

var (
	loginWithKey   bool
	loginNoBrowser bool
)

var loginCmd = &cobra.Command{
	Use:   "login",
	Short: "Sign the CLI in to Maestro",
	Long: `Signs in by approving a short code on the Maestro page in your browser.
Without a browser, use --with-key to paste a key generated on that page.`,
	RunE: runLogin,
}

func init() {
	loginCmd.Flags().BoolVar(&loginWithKey, "with-key", false, "paste a Maestro key instead of using the browser")
	loginCmd.Flags().BoolVar(&loginNoBrowser, "no-browser", false, "print the approval URL without opening a browser")
	rootCmd.AddCommand(loginCmd)
}

func runLogin(cmd *cobra.Command, args []string) error {
	stdin := bufio.NewReader(os.Stdin)

	var err error
	if loginWithKey {
		err = loginMaestroWithKey(cmd.Context(), stdin)
	} else {
		err = loginMaestroInBrowser(cmd.Context())
	}
	if err != nil {
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
	if !loginNoBrowser {
		openBrowser(login.VerificationURL)
	}
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

// openBrowser is best effort: the URL is already on screen for when there
// is no browser to open.
func openBrowser(target string) {
	var cmd *exec.Cmd
	switch runtime.GOOS {
	case "darwin":
		cmd = exec.Command("open", target)
	case "windows":
		cmd = exec.Command("rundll32", "url.dll,FileProtocolHandler", target)
	default:
		cmd = exec.Command("xdg-open", target)
	}
	if err := cmd.Start(); err == nil {
		go cmd.Wait()
	}
}

func loginMaestroWithKey(ctx context.Context, stdin *bufio.Reader) error {
	_, err := config.LoadMaestroKey()
	hasExisting := err == nil

	key, err := readSecret(stdin, "Maestro key", hasExisting)
	if err != nil {
		return err
	}
	if key == "" {
		fmt.Println(dimStyle.Render("Maestro: keeping existing key."))
		return nil
	}

	fmt.Println(dimStyle.Render("Validating Maestro key..."))
	email, err := maestroapi.VerifyKey(ctx, key)
	if err != nil {
		return fmt.Errorf("Maestro key validation failed: %w", err)
	}

	if err := config.SaveMaestroKey(key); err != nil {
		return fmt.Errorf("saving Maestro key: %w", err)
	}
	fmt.Println(successStyle.Render("Maestro: logged in as " + email + "."))
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

// readMasked reads a line from the terminal, echoing a * for each character
// typed or pasted instead of the character itself.
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
