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
		{Name: "Q1", Status: "ready", CPUs: 16, MemoryMB: 65536, ReserveCPUs: 1, PodmanVersion: "5.2.1", Description: "the big one"},
		{Name: "Q2", Status: "unreachable"},
	})
	if err != nil {
		t.Fatal(err)
	}
	wantTable(t, out.String(), [][]string{
		{"NAME", "STATUS", "CPUS", "MEMORY", "PODMAN", "DESCRIPTION"},
		{"Q1", "ready", "16 (1 kept)", "64G", "5.2.1", "the big one"},
		{"Q2", "unreachable", "-", "-", "-"},
	})
}
