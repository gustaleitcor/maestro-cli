package cmd

import (
	"github.com/spf13/cobra"

	"maestro-cli/tui"
)

var imageCmd = &cobra.Command{
	Use:   "image",
	Short: "Container image commands",
}

var imageListCmd = &cobra.Command{
	Use:   "list",
	Short: "List the built images",
	Args:  cobra.NoArgs,
	RunE:  runImageList,
}

func init() {
	rootCmd.AddCommand(imageCmd)
	imageCmd.AddCommand(imageListCmd)
}

func runImageList(cmd *cobra.Command, args []string) error {
	return tui.RunImageList(cmd.Context(), maestroKey)
}
