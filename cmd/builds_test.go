package cmd

import (
	"bytes"
	"encoding/json"
	"reflect"
	"strings"
	"testing"
	"time"

	"maestro-cli/internal/maestroapi"
)

func TestBuildRows(t *testing.T) {
	rows := buildRows(
		[]maestroapi.Build{{ID: 12, Status: "success"}, {ID: 11, Status: "success"}, {ID: 10, Status: "error"}},
		[]maestroapi.Image{{BuildID: 12, Size: 2048}, {BuildID: 3, Size: 1}},
	)
	if len(rows) != 3 {
		t.Fatalf("got %d rows", len(rows))
	}
	if !rows[0].HasImage || rows[0].ImageSize != 2048 {
		t.Errorf("the build with an image: %+v", rows[0])
	}
	if rows[1].HasImage || rows[2].HasImage {
		t.Errorf("builds without an image have one: %+v", rows[1:])
	}
	if rows := buildRows(nil, nil); rows == nil || len(rows) != 0 {
		t.Errorf("no builds: %#v, want an empty list", rows)
	}
}

func TestWriteBuildsList(t *testing.T) {
	var out bytes.Buffer
	if err := writeBuildsList(&out, nil, 5); err != nil || !strings.Contains(out.String(), "No builds yet") {
		t.Errorf("no builds: %q, %v", out.String(), err)
	}

	out.Reset()
	created := time.Date(2026, 10, 7, 9, 30, 0, 0, time.Local)
	rows := []buildRow{
		{Build: maestroapi.Build{ID: 13, Repo: "lab/sim", Ref: "dev", Status: "building", CreatedAt: created}},
		{Build: maestroapi.Build{ID: 12, Repo: "lab/sim", Ref: "main", Status: "success", CreatedAt: created}, HasImage: true, ImageSize: 3 << 20},
		{Build: maestroapi.Build{ID: 11, Repo: "lab/fit", Ref: "v2", Status: "success", CreatedAt: created}},
		{Build: maestroapi.Build{ID: 10, Repo: "lab/fit", Ref: "v1", Status: "error", Error: "STEP 1/2: FROM nope\nno such image\n", CreatedAt: created}},
	}
	if err := writeBuildsList(&out, rows, 5); err != nil {
		t.Fatal(err)
	}
	got := out.String()
	table, under, _ := strings.Cut(got, "\n\n")
	wantTable(t, table, [][]string{
		{"BUILD", "REPO", "REF", "STATUS", "IMAGE", "CREATED"},
		{"#13", "lab/sim", "dev", "building", "-", "2026-10-07 09:30"},
		{"#12", "lab/sim", "main", "success", "3.0", "2026-10-07 09:30"},
		{"#11", "lab/fit", "v2", "success", "removed"},
		{"#10", "lab/fit", "v1", "error", "-"},
	})
	// Only the last line of why it failed, and only the images still there.
	if want := "#10: no such image\n1 of the 5 images allowed; the oldest goes when a new build needs room.\n"; under != want {
		t.Errorf("under the table: %q, want %q", under, want)
	}

	out.Reset()
	writeBuildsList(&out, rows[:3], 0)
	if strings.Contains(out.String(), "allowed") || strings.Contains(out.String(), "#10:") || !strings.HasSuffix(out.String(), "\n") {
		t.Errorf("no failure and no limit known: %q", out.String())
	}
}

// --json is a build as the server names it, with what the table adds.
func TestBuildsJSON(t *testing.T) {
	var out bytes.Buffer
	rows := buildRows([]maestroapi.Build{{ID: 12, Repo: "lab/sim", Ref: "main", Status: "success"}, {ID: 11, Status: "error", Error: "no"}}, []maestroapi.Image{{BuildID: 12, Size: 2048}})
	if err := writeJSON(&out, rows); err != nil {
		t.Fatal(err)
	}
	var got []map[string]any
	if err := json.Unmarshal(out.Bytes(), &got); err != nil || len(got) != 2 {
		t.Fatalf("%q: %v", out.String(), err)
	}
	for key, want := range map[string]any{"id": 12.0, "repo": "lab/sim", "ref": "main", "status": "success", "has_image": true, "image_size": 2048.0} {
		if !reflect.DeepEqual(got[0][key], want) {
			t.Errorf("%s is %v, want %v", key, got[0][key], want)
		}
	}
	if got[1]["has_image"] != false || got[1]["error"] != "no" {
		t.Errorf("the build that failed: %v", got[1])
	}
	if _, has := got[1]["image_size"]; has {
		t.Errorf("a build without an image has a size: %v", got[1])
	}

	out.Reset()
	if writeJSON(&out, buildRows(nil, nil)); strings.TrimSpace(out.String()) != "[]" {
		t.Errorf("no builds: %q", out.String())
	}
}
