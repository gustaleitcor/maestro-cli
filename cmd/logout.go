package cmd

import (
	"bufio"
	"errors"
	"fmt"
	"io"
	"os"
	"strings"

	"github.com/spf13/cobra"
	"golang.org/x/term"

	"maestro-cli/internal/config"
)

var logoutYes bool

var logoutCmd = &cobra.Command{
	Use:   "logout",
	Short: "Sign out and forget everything the CLI has stored",
	Long: `Forgets the Maestro key and the server it is for, every forge and its
token, on this machine only.
Revoke the key and tokens on the Maestro page and on each forge.`,
	Args:              cobra.NoArgs,
	ValidArgsFunction: cobra.NoFileCompletions,
	RunE:              runLogout,
}

func init() {
	logoutCmd.Flags().BoolVarP(&logoutYes, "yes", "y", false, "don't ask for confirmation")
	rootCmd.AddCommand(logoutCmd)
}

// deleteYes is --yes of the commands that delete: rm of runs, builds and
// forges.
var deleteYes bool

func addYesFlag(c *cobra.Command) {
	c.Flags().BoolVarP(&deleteYes, "yes", "y", false, "don't ask first")
}

// confirm asks question at the terminal and says whether to go on. With
// --yes it doesn't ask; without a terminal to ask at, only --yes goes on,
// so that a script deletes nothing it didn't say to.
func confirm(question string) (bool, error) {
	if deleteYes {
		return true, nil
	}
	if !term.IsTerminal(int(os.Stdin.Fd())) {
		return false, usageError(errors.New("there is no terminal to ask at: add --yes to go on without asking"))
	}
	fmt.Printf("%s [y/N] ", question)
	line, err := bufio.NewReader(os.Stdin).ReadString('\n')
	if err != nil && err != io.EOF {
		return false, fmt.Errorf("reading input: %w", err)
	}
	return strings.HasPrefix(strings.ToLower(strings.TrimSpace(line)), "y"), nil
}

func runLogout(cmd *cobra.Command, args []string) error {
	forges, err := config.Forges()
	if err != nil {
		return err
	}
	_, keyErr := config.LoadMaestroKey("")
	if keyErr != nil && len(forges) == 0 {
		fmt.Println(dimStyle.Render("Nothing to forget: not signed in and no forges configured."))
		return nil
	}

	if !logoutYes {
		fmt.Println(warnStyle.Render("This forgets:"))
		if keyErr == nil {
			fmt.Println("  - the Maestro key")
		}
		if len(forges) > 0 {
			fmt.Printf("  - %d forge(s) and their tokens: %s\n", len(forges), forgeNames(forges))
		}
		answer, err := readLine(bufio.NewReader(os.Stdin), "Continue? (y/n)", "n")
		if err != nil {
			return err
		}
		if !strings.HasPrefix(strings.ToLower(answer), "y") {
			fmt.Println(dimStyle.Render("Cancelled."))
			return nil
		}
	}

	if err := config.Reset(); err != nil {
		return err
	}
	fmt.Println(successStyle.Render("Logged out.") + " Run `maestro login` to sign in again.")
	return nil
}
