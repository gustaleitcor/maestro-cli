package cmd

import (
	"bufio"
	"context"
	"errors"
	"fmt"
	"io"
	"os"
	"strings"

	"github.com/spf13/cobra"
	"golang.org/x/term"

	"maestro-cli/internal/config"
	"maestro-cli/internal/maestroapi"
	"maestro-cli/internal/netfail"
)

var logoutYes bool

var logoutCmd = &cobra.Command{
	Use:   "logout",
	Short: "Revoke your Maestro key and forget everything the CLI has stored",
	Long: `Revokes the Maestro key on the server that issued it, then forgets the
key, the server and every forge with its token on your computer.

If the server can't be reached, it still forgets the key, and says that the
key is still valid there. Revoke it later with ` + "`maestro keys rm <prefix>`" + ` from
a computer that is signed in.

A forge token stays valid on its forge. Revoke it there if you no longer
trust this computer.`,
	Args:              cobra.NoArgs,
	ValidArgsFunction: cobra.NoFileCompletions,
	RunE:              runLogout,
}

func init() {
	logoutCmd.Flags().BoolVarP(&logoutYes, "yes", "y", false, "don't ask first")
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
		return false, usageError(errors.New("there is no terminal to ask at; add --yes to go on without asking"))
	}
	fmt.Printf("%s [y/N] ", question)
	line, err := bufio.NewReader(os.Stdin).ReadString('\n')
	if err != nil && err != io.EOF {
		return false, fmt.Errorf("couldn't read what you typed: %w", err)
	}
	return strings.HasPrefix(strings.ToLower(strings.TrimSpace(line)), "y"), nil
}

func runLogout(cmd *cobra.Command, args []string) error {
	forges, err := config.Forges()
	if err != nil {
		return err
	}
	key, keyErr := config.LoadMaestroKey("")
	if keyErr != nil && len(forges) == 0 {
		fmt.Println(dimStyle.Render("Nothing to forget. You aren't signed in and have no forges."))
		return nil
	}

	if !logoutYes {
		fmt.Println(warnStyle.Render("This forgets:"))
		if keyErr == nil {
			fmt.Println("  - your Maestro key, after the server revokes it")
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

	if keyErr == nil {
		revokeOnLogout(cmd.Context(), os.Stdout, key)
	}
	if err := config.Reset(); err != nil {
		return err
	}
	fmt.Println(successStyle.Render("Signed out.") + " Run `maestro login` to sign in again.")
	return nil
}

// revokeOnLogout revokes key on the server that issued it, and says so when
// it couldn't. Logging out goes on either way.
func revokeOnLogout(ctx context.Context, out io.Writer, key string) {
	// The key goes only to the server that issued it, whatever --server or
	// MAESTRO_ORQ_URL say.
	if stored, err := config.Server(); err == nil && stored != "" {
		defer func(old string) { serverFlag = old }(serverFlag)
		serverFlag = stored
	}
	server := maestroapi.BaseURL()
	prefix := key[:min(len(key), 10)]
	later := fmt.Sprintf("Revoke it with `maestro keys rm %s` from a computer that is signed in.", prefix)

	var err error
	if server == "" {
		err = netfail.ErrUnreachable
	} else {
		err = maestroapi.RevokeCurrentKey(ctx, key)
	}
	switch {
	case err == nil:
		fmt.Fprintf(out, "Revoked key %s on %s.\n", prefix, server)
	case errors.Is(err, netfail.ErrUnreachable):
		fmt.Fprintln(out, warnStyle.Render(fmt.Sprintf("The server couldn't be reached, so key %s is still valid there.", prefix))+" "+later)
	default:
		fmt.Fprintln(out, warnStyle.Render(fmt.Sprintf("The server didn't revoke key %s: %v", prefix, err))+"\n"+later)
	}
}
