package cmd

import (
	"bytes"
	"strings"
	"testing"

	"maestro-cli/internal/maestroapi"
)

func TestWriteMachinesList(t *testing.T) {
	var out bytes.Buffer
	if err := writeMachinesList(&out, nil); err != nil || !strings.Contains(out.String(), "No machines yet") {
		t.Errorf("no machines: %q, %v", out.String(), err)
	}

	out.Reset()
	err := writeMachinesList(&out, []maestroapi.Machine{
		{Name: "Q1", Status: "ready", Slots: 4, PodmanVersion: "5.2.1", Description: "the big one"},
		{Name: "Q2", Status: "unreachable", Slots: 2},
	})
	if err != nil {
		t.Fatal(err)
	}
	wantTable(t, out.String(), [][]string{
		{"NAME", "STATUS", "SLOTS", "PODMAN", "DESCRIPTION"},
		{"Q1", "ready", "4", "5.2.1", "the big one"},
		{"Q2", "unreachable", "2", "-"},
	})
}
