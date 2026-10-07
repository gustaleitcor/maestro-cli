package cmd

import (
	"bytes"
	"strings"
	"testing"

	"maestro-cli/internal/config"
	"maestro-cli/internal/forge"
)

func TestWriteForgesList(t *testing.T) {
	var out bytes.Buffer
	if err := writeForgesList(&out, nil); err != nil || !strings.Contains(out.String(), "maestro forges add") {
		t.Errorf("no forges: %q, %v", out.String(), err)
	}

	// A public instance's URL isn't stored, and is shown all the same.
	out.Reset()
	err := writeForgesList(&out, []config.Forge{
		{Name: "github", Kind: forge.GitHub},
		{Name: "work", Kind: forge.Forgejo, BaseURL: "https://git.example.com"},
	})
	if err != nil {
		t.Fatal(err)
	}
	wantTable(t, out.String(), [][]string{
		{"NAME", "KIND", "URL"},
		{"github", "github", forge.DefaultBaseURL(forge.GitHub)},
		{"work", "forgejo", "https://git.example.com"},
	})
}
