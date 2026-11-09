package cmd

import (
	"bytes"
	"encoding/json"
	"strings"
	"testing"

	"maestro-cli/internal/maestroapi"
)

func TestWriteQueue(t *testing.T) {
	var q maestroapi.Queue
	if err := json.Unmarshal([]byte(`{
	  "totals": {"queued": 5, "staged": 2, "assigned": 3, "starting": 0, "running": 4},
	  "users": [{"id": 1, "name": "Ada", "email": "ada@example.com", "lines": {"queued": 2, "running": 1}, "share": 0.25}],
	  "scheduler": {"user": "fifo", "select": "pq", "allocate": "fifo", "place": "tetris", "machine": "fifo", "share": "drf", "share_weight": -1}
	}`), &q); err != nil {
		t.Fatal(err)
	}
	var out bytes.Buffer
	if err := writeQueue(&out, &q); err != nil {
		t.Fatal(err)
	}
	got := out.String()
	wantTable(t, strings.SplitN(got, "\n\n", 2)[0], [][]string{
		{"USER", "QUEUED", "STAGED", "ASSIGNED", "STARTING", "RUNNING", "SHARE"},
		{"Ada #1", "2", "0", "0", "0", "1", "25%"},
		{"everyone", "5", "2", "3", "0", "4"},
	})
	if !strings.Contains(got, "chosen by pq among users") || !strings.Contains(got, "share (drf) by -1") {
		t.Errorf("no summary of the scheduler: %q", got)
	}
}
