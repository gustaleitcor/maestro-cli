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
	s := &maestroapi.Settings{ImagesPerUser: 10, Scheduler: &maestroapi.Scheduler{User: "fifo", Dispatcher: "pq", Machine: "fifo", Depth: 3, Share: "decay", HalfLifeHours: 168,
		Weights: []maestroapi.UserWeight{{User: 7, CPUs: 2, Memory: 1}}}}
	if err := writeSettings(&out, s); err != nil {
		t.Fatal(err)
	}
	if !regexp.MustCompile(`images per user\s+10\n`).MatchString(out.String()) {
		t.Errorf("no image limit:\n%s", out.String())
	}
	if !regexp.MustCompile(`cores per user\s+no ceiling\n`).MatchString(out.String()) {
		t.Errorf("no ceiling on cores:\n%s", out.String())
	}
	if !regexp.MustCompile(`runs kept\s+until someone deletes them\n`).MatchString(out.String()) {
		t.Errorf("runs aren't kept for ever:\n%s", out.String())
	}
	out.Reset()
	s.RunRetentionDays = 30
	writeSettings(&out, s)
	if !regexp.MustCompile(`runs kept\s+30 days after their last line ends\n`).MatchString(out.String()) {
		t.Errorf("runs aren't kept 30 days:\n%s", out.String())
	}
	s.RunRetentionDays = 0
	out.Reset()
	s.UserMaxCPUs = 16
	writeSettings(&out, s)
	if !regexp.MustCompile(`cores per user\s+16\n`).MatchString(out.String()) {
		t.Errorf("no ceiling of 16 cores:\n%s", out.String())
	}
	out.Reset()
	s.UserMaxCPUs = 0
	writeSettings(&out, s)
	for _, want := range []string{"among a group's heads", "pq", "decay", "168 h", "user 7: 2 cpu, 1 memory"} {
		if !strings.Contains(out.String(), want) {
			t.Errorf("settings lack %q:\n%s", want, out.String())
		}
	}
	if strings.Contains(out.String(), "staged") {
		t.Errorf("settings without staging speak of it:\n%s", out.String())
	}
	// A server from before the three queues is shown as it answers.
	out.Reset()
	writeSettings(&out, &maestroapi.Settings{ImagesPerUser: 5, Scheduler: &maestroapi.Scheduler{Select: "pq", Allocate: "fifo"}})
	for _, want := range []string{"among users", "pq", "staged lines, in turn between groups", "twice the machines, per group"} {
		if !strings.Contains(out.String(), want) {
			t.Errorf("an older server's settings lack %q:\n%s", want, out.String())
		}
	}
	// A server from before the scheduler has none to show.
	out.Reset()
	writeSettings(&out, &maestroapi.Settings{ImagesPerUser: 5})
	if strings.Contains(out.String(), "among") {
		t.Errorf("an old server's settings: %q", out.String())
	}
}
