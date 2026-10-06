package cmd

import (
	"bufio"
	"fmt"
	"os"
	"strings"

	"github.com/spf13/cobra"

	"maestro-cli/internal/config"
)

var logoutYes bool

var logoutCmd = &cobra.Command{
	Use:   "logout",
	Short: "Sign out and forget everything the CLI has stored",
	Long: `Forgets the Maestro key, every forge and its token, on this machine only.
Revoke the key and tokens on the Maestro page and on each forge.`,
	Args:              cobra.NoArgs,
	ValidArgsFunction: cobra.NoFileCompletions,
	RunE:              runLogout,
}

func init() {
	logoutCmd.Flags().BoolVarP(&logoutYes, "yes", "y", false, "don't ask for confirmation")
	rootCmd.AddCommand(logoutCmd)
}

func runLogout(cmd *cobra.Command, args []string) error {
	forges, err := config.Forges()
	if err != nil {
		return err
	}
	_, keyErr := config.LoadMaestroKey()
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
