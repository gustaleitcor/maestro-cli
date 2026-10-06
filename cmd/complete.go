package cmd

import (
	"context"
	"fmt"
	"strconv"
	"strings"
	"time"

	"github.com/spf13/cobra"

	"maestro-cli/internal/config"
	"maestro-cli/internal/maestroapi"
)

const completionTimeout = 10 * time.Second

// Tabs separate a completion from its description; newlines end it.
var descriptionCleaner = strings.NewReplacer("\t", " ", "\n", " ", "\r", " ")

// completion is one suggestion: the value, and what it is.
func completion(value any, description string) string {
	return fmt.Sprintf("%v\t%s", value, descriptionCleaner.Replace(description))
}

// withKey runs a completion that needs the Maestro server. Cobra doesn't run
// PersistentPreRunE for completions, so the key is loaded here; without one,
// or if the server can't be reached, nothing is offered.
func withKey(offer func(ctx context.Context, key string) []string) ([]string, cobra.ShellCompDirective) {
	key, err := config.LoadMaestroKey()
	if err != nil {
		return nil, cobra.ShellCompDirectiveNoFileComp
	}
	ctx, cancel := context.WithTimeout(context.Background(), completionTimeout)
	defer cancel()
	return offer(ctx, key), cobra.ShellCompDirectiveNoFileComp | cobra.ShellCompDirectiveKeepOrder
}

// completeBuilds offers the builds that still have an image, newest first.
func completeBuilds(cmd *cobra.Command, args []string, toComplete string) ([]string, cobra.ShellCompDirective) {
	if len(args) > 0 {
		return nil, cobra.ShellCompDirectiveNoFileComp
	}
	return withKey(func(ctx context.Context, key string) []string {
		images, err := maestroapi.ListImages(ctx, key)
		if err != nil {
			return nil
		}
		var offers []string
		for _, image := range images {
			id := strconv.FormatInt(image.BuildID, 10)
			if strings.HasPrefix(id, strings.TrimPrefix(toComplete, "#")) {
				offers = append(offers, completion(id, image.Repo+"@"+image.Ref))
			}
		}
		return offers
	})
}

// completeRuns offers the user's runs as the first argument, newest first.
func completeRuns(cmd *cobra.Command, args []string, toComplete string) ([]string, cobra.ShellCompDirective) {
	if len(args) > 0 {
		return nil, cobra.ShellCompDirectiveNoFileComp
	}
	return withKey(func(ctx context.Context, key string) []string { return runOffers(ctx, key, toComplete) })
}

// completeRunLines offers a run, then one of its lines.
func completeRunLines(cmd *cobra.Command, args []string, toComplete string) ([]string, cobra.ShellCompDirective) {
	switch len(args) {
	case 0:
		return completeRuns(cmd, args, toComplete)
	case 1:
		runID, err := parseID(args[0], "run")
		if err != nil {
			return nil, cobra.ShellCompDirectiveNoFileComp
		}
		return withKey(func(ctx context.Context, key string) []string {
			run, err := maestroapi.GetRun(ctx, key, runID)
			if err != nil {
				return nil
			}
			var offers []string
			for _, line := range run.Detail {
				id := strconv.FormatInt(line.Line, 10)
				if strings.HasPrefix(id, toComplete) {
					offers = append(offers, completion(id, line.Status+": "+strings.Join(line.Args, " ")))
				}
			}
			return offers
		})
	}
	return nil, cobra.ShellCompDirectiveNoFileComp
}

// completeRunThenDir offers a run, then a directory to download into.
func completeRunThenDir(cmd *cobra.Command, args []string, toComplete string) ([]string, cobra.ShellCompDirective) {
	if len(args) == 1 {
		return nil, cobra.ShellCompDirectiveFilterDirs
	}
	return completeRuns(cmd, args, toComplete)
}

func runOffers(ctx context.Context, key, toComplete string) []string {
	runs, err := maestroapi.ListRuns(ctx, key)
	if err != nil {
		return nil
	}
	var offers []string
	for _, run := range runs {
		id := strconv.FormatInt(run.ID, 10)
		if strings.HasPrefix(id, strings.TrimPrefix(toComplete, "#")) {
			offers = append(offers, completion(id, run.Status+": "+run.Repo+"@"+run.Ref))
		}
	}
	return offers
}

// completeMachines offers the machines, by name.
func completeMachines(cmd *cobra.Command, args []string, toComplete string) ([]string, cobra.ShellCompDirective) {
	if len(args) > 0 {
		return nil, cobra.ShellCompDirectiveNoFileComp
	}
	return withKey(func(ctx context.Context, key string) []string {
		machines, err := maestroapi.ListMachines(ctx, key)
		if err != nil {
			return nil
		}
		var offers []string
		for _, machine := range machines {
			if strings.HasPrefix(strings.ToLower(machine.Name), strings.ToLower(toComplete)) {
				offers = append(offers, completion(machine.Name, machine.Description))
			}
		}
		return offers
	})
}
