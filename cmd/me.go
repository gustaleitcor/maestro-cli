package cmd

import (
	"github.com/spf13/cobra"

	"maestro-cli/tui"
)

var meForge string

var meCmd = &cobra.Command{
	Use:   "me",
	Short: "Show who a forge token belongs to",
	RunE:  runMe,
}

func init() {
	addForgeFlag(meCmd, &meForge)
	rootCmd.AddCommand(meCmd)
}

func runMe(cmd *cobra.Command, args []string) error {
	active, err := selectForge(meForge)
	if err != nil {
		return err
	}
	return tui.RunMe(cmd.Context(), active.Client)
}
