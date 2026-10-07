package cmd

import (
	"bytes"
	"strings"
	"testing"
	"time"
)

func TestWriteReposList(t *testing.T) {
	var out bytes.Buffer
	if err := writeReposList(&out, nil); err != nil || !strings.Contains(out.String(), "No repositories") {
		t.Errorf("no repos: %q, %v", out.String(), err)
	}

	out.Reset()
	updated := time.Date(2026, 10, 7, 9, 30, 0, 0, time.UTC)
	err := writeReposList(&out, []repoRow{
		{Forge: "home", Repo: "ana/app", Language: "Go", UpdatedAt: updated},
		{Forge: "work", Repo: "team/sub/api", Private: true, UpdatedAt: updated},
	})
	if err != nil {
		t.Fatal(err)
	}
	wantTable(t, out.String(), [][]string{
		{"FORGE", "REPO", "VISIBILITY", "LANGUAGE", "UPDATED"},
		{"home", "ana/app", "public", "Go", "2026-10-07"},
		{"work", "team/sub/api", "private", "-", "2026-10-07"},
	})
}
