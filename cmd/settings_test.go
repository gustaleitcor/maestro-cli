package cmd

import (
	"bytes"
	"regexp"
	"strings"
	"testing"

	"maestro-cli/internal/maestroapi"
)

func TestWriteSettings(t *testing.T) {
	var out bytes.Buffer
	s := &maestroapi.Settings{ImagesPerUser: 10, Scheduler: &maestroapi.Scheduler{Select: "pq", Depth: 3, Share: "decay", HalfLifeHours: 168,
		Weights: []maestroapi.UserWeight{{User: 7, CPUs: 2, Memory: 1}}}}
	if err := writeSettings(&out, s); err != nil {
		t.Fatal(err)
	}
	if !regexp.MustCompile(`images per user\s+10\n`).MatchString(out.String()) {
		t.Errorf("no image limit:\n%s", out.String())
	}
	for _, want := range []string{"among users", "pq", "twice the machines", "decay", "168 h", "user 7: 2 cpu, 1 memory"} {
		if !strings.Contains(out.String(), want) {
			t.Errorf("settings lack %q:\n%s", want, out.String())
		}
	}
	// A server from before the scheduler has none to show.
	out.Reset()
	writeSettings(&out, &maestroapi.Settings{ImagesPerUser: 5})
	if strings.Contains(out.String(), "among users") {
		t.Errorf("an old server's settings: %q", out.String())
	}
}
