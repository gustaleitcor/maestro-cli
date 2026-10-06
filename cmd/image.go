package cmd

import (
	"fmt"

	"github.com/spf13/cobra"

	"maestro-cli/internal/maestroapi"
	"maestro-cli/tui"
)

var imageCmd = &cobra.Command{
	Use:   "image",
	Short: "Container image commands",
}

var imageListCmd = &cobra.Command{
	Use:               "list",
	Short:             "List the built images",
	Args:              cobra.NoArgs,
	ValidArgsFunction: cobra.NoFileCompletions,
	RunE:              runImageList,
}

var imageRmCmd = &cobra.Command{
	Use:   "rm <build>",
	Short: "Remove the image a build produced",
	Long: `Removes the image of a build, from the server and from the machines.
Each user may keep a limited number of images; this frees a place for a new
build. The build's record and its runs, with their files, are kept.

<build> is the build number shown by ` + "`maestro image list`" + `, with or without #.`,
	Args:              cobra.ExactArgs(1),
	ValidArgsFunction: completeBuilds,
	RunE:              runImageRm,
}

func init() {
	rootCmd.AddCommand(imageCmd)
	imageCmd.AddCommand(imageListCmd, imageRmCmd)
}

func runImageRm(cmd *cobra.Command, args []string) error {
	buildID, err := parseID(args[0], "build")
	if err != nil {
		return err
	}
	if err := maestroapi.RemoveImage(cmd.Context(), maestroKey, buildID); err != nil {
		return err
	}
	fmt.Println(successStyle.Render(fmt.Sprintf("Removed the image of build #%d.", buildID)))
	return nil
}

func runImageList(cmd *cobra.Command, args []string) error {
	return tui.RunImageList(cmd.Context(), maestroKey)
}
