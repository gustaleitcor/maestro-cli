package cmd

import (
	"bytes"
	"strings"
	"testing"
	"time"

	"maestro-cli/internal/maestroapi"
)

func TestWriteKeysList(t *testing.T) {
	var out bytes.Buffer
	if err := writeKeysList(&out, nil, "mk_abcdefghijkl"); err != nil || !strings.Contains(out.String(), "No Maestro keys") {
		t.Errorf("no keys: %q, %v", out.String(), err)
	}

	out.Reset()
	created := time.Date(2026, 10, 7, 9, 30, 0, 0, time.Local)
	expires := created.AddDate(0, 0, 90)
	keys := []maestroapi.Key{{Prefix: "mk_zzzzzzz", CreatedAt: created}, {Prefix: "mk_abcdefg", CreatedAt: created, ExpiresAt: expires}}
	if err := writeKeysList(&out, keys, "mk_abcdefghijkl"); err != nil {
		t.Fatal(err)
	}
	wantTable(t, strings.TrimSuffix(out.String(), "\n"), [][]string{
		{"KEY", "CREATED", "EXPIRES"},
		{"mk_zzzzzzz", "2026-10-07", "09:30", "never"},
		{"mk_abcdefg", "2026-10-07", "09:30", "2027-01-05", "09:30", "this", "computer"},
	})
}
