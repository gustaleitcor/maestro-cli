package cmd

import (
	"context"
	"fmt"
	"io"
	"os"
	"strings"
	"text/tabwriter"

	"github.com/spf13/cobra"

	"maestro-cli/internal/humanize"
	"maestro-cli/internal/maestroapi"
)

var buildsAsJSON bool

// builds is to build what runs is to run. It was `image` once, which still
// works. On its own it lists.
var buildsCmd = &cobra.Command{
	Use:     "builds",
	Aliases: []string{"image"},
	Short:   "Commands for what `maestro build` built",
	Args:    cobra.NoArgs,
	RunE:    runBuildsList,
}

var buildsListCmd = &cobra.Command{
	Use:   "list",
	Short: "List your builds, and whether each still has its image",
	Long: `Lists every build of yours, newest first: the ones that worked, the ones
that failed and why, and any still going. IMAGE is the size of the image a
build has, or "removed" once it is gone; only a build with an image can be
run. BUILD is the number ` + "`maestro run`" + ` and ` + "`maestro builds rm`" + ` take.`,
	Args:              cobra.NoArgs,
	ValidArgsFunction: cobra.NoFileCompletions,
	RunE:              runBuildsList,
}

var buildsRmCmd = &cobra.Command{
	Use:     "rm <build>",
	Aliases: []string{"remove"},
	Short:   "Remove the image a build produced",
	Long: `Removes the image of a build, from the server and from the machines.
Each user may keep a limited number of images; this frees a place for a new
build. The build's record and its runs, with their files, are kept.

<build> is the build number shown by ` + "`maestro builds list`" + `, with or without #.`,
	Args:              cobra.ExactArgs(1),
	ValidArgsFunction: completeBuilds,
	RunE:              runBuildsRm,
}

func init() {
	for _, c := range []*cobra.Command{buildsCmd, buildsListCmd} {
		c.Flags().BoolVar(&buildsAsJSON, "json", false, "print the builds as JSON")
	}
	rootCmd.AddCommand(buildsCmd)
	buildsCmd.AddCommand(buildsListCmd, buildsRmCmd)
}

func runBuildsRm(cmd *cobra.Command, args []string) error {
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

// buildRow is a build with what the server's Podman still has of it.
type buildRow struct {
	maestroapi.Build
	HasImage  bool  `json:"has_image"`
	ImageSize int64 `json:"image_size,omitempty"`
}

// buildRows says of each build whether its image is among images.
func buildRows(builds []maestroapi.Build, images []maestroapi.Image) []buildRow {
	sizes := make(map[int64]int64, len(images))
	for _, image := range images {
		sizes[image.BuildID] = image.Size
	}
	rows := make([]buildRow, 0, len(builds))
	for _, b := range builds {
		size, has := sizes[b.ID]
		rows = append(rows, buildRow{Build: b, HasImage: has, ImageSize: size})
	}
	return rows
}

func runBuildsList(cmd *cobra.Command, args []string) error {
	rows, limit, err := loadBuilds(cmd.Context())
	if err != nil {
		return err
	}
	if buildsAsJSON {
		return writeJSON(os.Stdout, rows)
	}
	return writeBuildsList(os.Stdout, rows, limit)
}

// loadBuilds also returns how many images a user may keep, or 0 when the
// server doesn't say.
func loadBuilds(ctx context.Context) ([]buildRow, int, error) {
	builds, err := maestroapi.ListBuilds(ctx, maestroKey)
	if err != nil {
		return nil, 0, err
	}
	images, err := maestroapi.ListImages(ctx, maestroKey)
	if err != nil {
		return nil, 0, err
	}
	limit := 0
	// Best effort: a server from before the limit has no settings.
	if settings, err := maestroapi.GetSettings(ctx, maestroKey); err == nil {
		limit = settings.ImagesPerUser
	}
	return buildRows(builds, images), limit, nil
}

func writeBuildsList(out io.Writer, rows []buildRow, limit int) error {
	if len(rows) == 0 {
		_, err := fmt.Fprintln(out, "No builds yet. Build one with: maestro build <repo>")
		return err
	}
	w := tabwriter.NewWriter(out, 0, 0, 2, ' ', 0)
	fmt.Fprintln(w, "BUILD\tREPO\tREF\tSTATUS\tIMAGE\tCREATED")
	kept := 0
	for _, b := range rows {
		image := "-"
		switch {
		case b.HasImage:
			image = humanize.Bytes(uint64(max(b.ImageSize, 0)))
			kept++
		case b.Status == "success":
			image = "removed"
		}
		fmt.Fprintf(w, "#%d\t%s\t%s\t%s\t%s\t%s\n", b.ID, b.Repo, b.Ref, b.Status, image, b.CreatedAt.Local().Format("2006-01-02 15:04"))
	}
	if err := w.Flush(); err != nil {
		return err
	}
	// Why a build failed is its last word: what came before is the log.
	for _, b := range rows {
		if b.Status == "error" && b.Error != "" {
			lines := strings.Split(strings.TrimSpace(b.Error), "\n")
			fmt.Fprintf(out, "\n#%d: %s", b.ID, strings.TrimSpace(lines[len(lines)-1]))
		}
	}
	if limit > 0 {
		fmt.Fprintf(out, "\n%s", dimStyle.Render(fmt.Sprintf("%d of the %d images allowed; the oldest goes when a new build needs room.", kept, limit)))
	}
	_, err := fmt.Fprintln(out)
	return err
}
