package cmd

import (
	"errors"
	"fmt"
	"strings"

	"github.com/spf13/cobra"
)

// What a command is given is checked before anything else: cobra runs Args
// before PersistentPreRunE, so a mistake is said as itself, signed in or
// not, and before the server is asked anything. Every error from here is a
// usage mistake (exitUsage), which report follows with the command's usage.

// usageError is a command asked wrongly.
func usageError(err error) error {
	var exit *exitError
	if err == nil || errors.As(err, &exit) {
		return err
	}
	return &exitError{exitUsage, err}
}

// asUsage makes what every command's Args, and its flags, refuse a usage
// mistake. It runs once, when every command has been added.
func asUsage(c *cobra.Command) {
	if check := c.Args; check != nil {
		c.Args = func(cmd *cobra.Command, args []string) error {
			return usageError(check(cmd, args))
		}
	}
	for _, sub := range c.Commands() {
		asUsage(sub)
	}
}

// whereToFind says, after a number is missing or wrong, where the numbers are.
var whereToFind = map[string]string{
	"build": "see `maestro builds`",
	"run":   "see `maestro runs`",
	"line":  "see `maestro runs show <run>`",
}

// numberArgs is Args for a command that takes the numbers of whats, the
// first min of them required, and then up to max arguments in all; the
// ones past whats (a directory, say) aren't checked here.
func numberArgs(min, max int, whats ...string) cobra.PositionalArgs {
	return func(cmd *cobra.Command, args []string) error {
		if len(args) < min {
			what := whats[len(args)]
			return fmt.Errorf("%s needs %s %s number (%s)", cmd.CommandPath(), article(what), what, whereToFind[what])
		}
		if len(args) > max {
			return fmt.Errorf("%s takes at most %d argument(s), not %d", cmd.CommandPath(), max, len(args))
		}
		for i, what := range whats {
			if i >= len(args) {
				break
			}
			if _, err := parseID(args[i], what); err != nil {
				return fmt.Errorf("%w such as 12 (%s)", err, whereToFind[what])
			}
		}
		return nil
	}
}

func article(what string) string {
	if strings.ContainsRune("aeiou", rune(what[0])) {
		return "an"
	}
	return "a"
}

// flagCheck is Args that checks how flags go together, before the key.
func flagCheck(check func() error) cobra.PositionalArgs {
	return func(*cobra.Command, []string) error { return check() }
}

// nounArgs is Args of a command named after what there is several of: on
// its own it lists, and a word after it is a subcommand that isn't there.
func nounArgs(cmd *cobra.Command, args []string) error {
	if len(args) == 0 {
		return nil
	}
	msg := fmt.Sprintf("unknown command %q for %q", args[0], cmd.CommandPath())
	// As cobra does for the root, which it sets only there.
	if cmd.SuggestionsMinimumDistance <= 0 {
		cmd.SuggestionsMinimumDistance = 2
	}
	if suggestions := cmd.SuggestionsFor(args[0]); len(suggestions) > 0 {
		msg += "\n\nDid you mean this?\n\t" + strings.Join(suggestions, "\n\t") + "\n"
	}
	return errors.New(msg)
}
