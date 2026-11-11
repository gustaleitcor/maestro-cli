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
	  "users": [{"id": 1, "name": "Ada", "email": "ada@example.com", "group_id": 2, "lines": {"queued": 2, "running": 1}, "share": 0.25}],
	  "groups": [{"id": 2, "name": "lab", "staged": 1, "capacity": 4}],
	  "scheduler": {"user": "fifo", "select": "pq", "allocate": "fifo", "machine": "best-fit", "share": "drf", "share_weight": -1}
	}`), &q); err != nil {
		t.Fatal(err)
	}
	var out bytes.Buffer
	if err := writeQueue(&out, &q); err != nil {
		t.Fatal(err)
	}
	got := out.String()
	parts := strings.SplitN(got, "\n\n", 3)
	wantTable(t, parts[0], [][]string{
		{"USER", "GROUP", "QUEUED", "STAGED", "ASSIGNED", "STARTING", "RUNNING", "SHARE"},
		{"Ada #1", "lab", "2", "0", "0", "0", "1", "25%"},
		{"everyone", "5", "2", "3", "0", "4"},
	})
	wantTable(t, parts[1], [][]string{{"GROUP", "STAGED"}, {"lab", "1/4"}})
	if !strings.Contains(got, "chosen by pq among each group's users") || !strings.Contains(got, "start by best-fit") || !strings.Contains(got, "share (drf) by -1") {
		t.Errorf("no summary of the scheduler: %q", got)
	}
}
