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
	  "totals": {"queued": 5, "assigned": 3, "starting": 0, "running": 4},
	  "users": [{"id": 1, "name": "Ada", "email": "ada@example.com", "group_id": 2, "lines": {"queued": 2, "running": 1}, "heads": 1, "share": 0.25}],
	  "groups": [{"id": 2, "name": "lab", "queued": 4, "heads": 2, "assigned": 3}],
	  "scheduler": {"user": "fifo", "dispatcher": "pq", "machine": "best-fit", "share": "drf", "share_weight": -1}
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
		{"USER", "GROUP", "QUEUED", "HEADS", "ASSIGNED", "STARTING", "RUNNING", "SHARE"},
		{"Ada #1", "lab", "2", "1", "0", "0", "1", "25%"},
		{"everyone", "5", "3", "0", "4"},
	})
	wantTable(t, parts[1], [][]string{{"GROUP", "QUEUED", "HEADS", "ASSIGNED"}, {"lab", "4", "2", "3"}})
	for _, want := range []string{"head is chosen by fifo", "pq picks one of its heads", "start by best-fit", "share (drf) by -1"} {
		if !strings.Contains(got, want) {
			t.Errorf("the summary of the scheduler lacks %q: %q", want, got)
		}
	}
	if strings.Contains(got, "most a user may hold") {
		t.Errorf("a caller under the ceiling is told of it: %q", got)
	}

	// A caller at the ceiling is told why their next line waits.
	q.UserMaxCPUs, q.AtCeiling = 16, true
	out.Reset()
	if err := writeQueue(&out, &q); err != nil {
		t.Fatal(err)
	}
	if !strings.Contains(out.String(), "more than 16 cores, the most a user may hold") {
		t.Errorf("a caller at the ceiling isn't told: %q", out.String())
	}
}

// A server from before the three queues is shown as it answers: staged
// lines, each group's out of its capacity, and four policies.
func TestWriteQueueOfAnOlderServer(t *testing.T) {
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
	if strings.Contains(got, "HEADS") || !strings.Contains(got, "chosen by pq among each group's users") || !strings.Contains(got, "share (drf) by -1") {
		t.Errorf("an older server's queue: %q", got)
	}
}
