package cmd

import (
	"fmt"
	"io"
	"os"
	"strings"
	"text/tabwriter"

	"github.com/spf13/cobra"

	"maestro-cli/internal/maestroapi"
)

var keysAsJSON bool

var keysCmd = &cobra.Command{
	Use:   "keys",
	Short: "List and revoke your Maestro keys",
	Long: `Lists the Maestro keys you have on the server, newest first. Each
` + "`maestro login`" + ` issues one, on whichever computer you run it. A key is shown by
its prefix, the first 10 characters, and never in full. The one this
computer uses is marked.`,
	Example: `  maestro keys
  maestro keys rm mk_Q3xv9aZ`,
	Args:              nounArgs,
	ValidArgsFunction: cobra.NoFileCompletions,
	RunE:              runKeysList,
}

var keysRmCmd = &cobra.Command{
	Use:     "rm <prefix>",
	Aliases: []string{"revoke"},
	Short:   "Revoke one of your Maestro keys",
	Long: `Revokes the key that starts with <prefix>, as ` + "`maestro keys`" + ` shows it. The
key stops working at once, on every computer that holds it. It asks first.

Revoke a key this way when you no longer trust a computer you signed in on.
To end the key of the computer you are on, ` + "`maestro logout`" + ` does it and
forgets the key too.`,
	Example:           `  maestro keys rm mk_Q3xv9aZ`,
	Args:              cobra.ExactArgs(1),
	ValidArgsFunction: cobra.NoFileCompletions,
	RunE:              runKeysRm,
}

func init() {
	keysCmd.Flags().BoolVar(&keysAsJSON, "json", false, "print the keys as JSON")
	addYesFlag(keysRmCmd)
	keysCmd.AddCommand(keysRmCmd)
	rootCmd.AddCommand(keysCmd)
}

func runKeysList(cmd *cobra.Command, args []string) error {
	keys, err := maestroapi.ListKeys(cmd.Context(), maestroKey)
	if err != nil {
		return err
	}
	if keysAsJSON {
		return writeJSON(os.Stdout, keys)
	}
	return writeKeysList(os.Stdout, keys, maestroKey)
}

// writeKeysList marks the key that current, the key this computer uses,
// starts with.
func writeKeysList(out io.Writer, keys []maestroapi.Key, current string) error {
	if len(keys) == 0 {
		_, err := fmt.Fprintln(out, "No Maestro keys. Sign in with `maestro login` to get one.")
		return err
	}
	w := tabwriter.NewWriter(out, 0, 0, 2, ' ', 0)
	fmt.Fprintln(w, "KEY\tCREATED")
	for _, k := range keys {
		fmt.Fprintf(w, "%s\t%s", k.Prefix, k.CreatedAt.Local().Format("2006-01-02 15:04"))
		if strings.HasPrefix(current, k.Prefix) {
			fmt.Fprint(w, "\tthis computer")
		}
		fmt.Fprintln(w)
	}
	return w.Flush()
}

func runKeysRm(cmd *cobra.Command, args []string) error {
	prefix := args[0]
	question := fmt.Sprintf("Revoke key %s? Every computer that uses it will have to sign in again.", prefix)
	if strings.HasPrefix(maestroKey, prefix) {
		question = fmt.Sprintf("Revoke key %s? It is the key this computer uses, so you will have to run `maestro login` again.", prefix)
	}
	if ok, err := confirm(question); err != nil || !ok {
		if err == nil {
			fmt.Println(dimStyle.Render(fmt.Sprintf("Kept key %s.", prefix)))
		}
		return err
	}
	if err := maestroapi.RevokeKey(cmd.Context(), maestroKey, prefix); err != nil {
		return err
	}
	fmt.Println(successStyle.Render(fmt.Sprintf("Revoked key %s.", prefix)))
	return nil
}
