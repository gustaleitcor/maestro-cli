package cmd

import (
	"strings"
	"testing"
)

// The names commands had before still reach them.
func TestOldNamesStillWork(t *testing.T) {
	for old, now := range map[string]string{
		"image":         "builds",
		"image list":    "builds list",
		"builds remove": "builds rm",
		"machine":       "machines",
		"machine list":  "machines list",
		"machine stats": "machines stats",
		"repo":          "repos",
		"repo list":     "repos list",
		"forge":         "forges",
		"forge add":     "forges add",
		"forge remove":  "forges rm",
		"runs remove":   "runs rm",
	} {
		was, _, err := rootCmd.Find(strings.Fields(old))
		if err != nil {
			t.Errorf("maestro %s: %v", old, err)
			continue
		}
		is, _, err := rootCmd.Find(strings.Fields(now))
		if err != nil {
			t.Errorf("maestro %s: %v", now, err)
			continue
		}
		if was != is || is.CommandPath() != "maestro "+now {
			t.Errorf("maestro %s is %q, want %q", old, was.CommandPath(), "maestro "+now)
		}
	}
}
