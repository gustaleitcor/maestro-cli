package cmd

import (
	"bytes"
	"encoding/json"
	"strings"
	"testing"

	"maestro-cli/internal/maestroapi"
)

func TestWriteUsage(t *testing.T) {
	var usage []maestroapi.Usage
	if err := json.Unmarshal([]byte(`[{"id": 1, "name": "Ada", "days": [
	  {"day": "2026-11-08", "cpu_hours": 2, "memory_gb_hours": 1.5},
	  {"day": "2026-11-09", "cpu_hours": 1, "memory_gb_hours": 0.5}]}]`), &usage); err != nil {
		t.Fatal(err)
	}
	var out bytes.Buffer
	writeUsage(&out, usage, 30, false)
	wantTable(t, strings.SplitN(out.String(), "\n\n", 2)[0], [][]string{
		{"USER", "CORE-HOURS", "GB-HOURS", "DAYS"},
		{"Ada", "3.0", "2.0", "2"},
	})
	out.Reset()
	writeUsage(&out, usage, 30, true)
	wantTable(t, strings.SplitN(out.String(), "\n\n", 2)[0], [][]string{
		{"USER", "DAY", "CORE-HOURS", "GB-HOURS"},
		{"Ada", "2026-11-08", "2.0", "1.5"},
		{"Ada", "2026-11-09", "1.0", "0.5"},
	})
	out.Reset()
	writeUsage(&out, nil, 7, false)
	if !strings.Contains(out.String(), "Nothing used in the last 7 day(s)") {
		t.Errorf("nothing used: %q", out.String())
	}
}
