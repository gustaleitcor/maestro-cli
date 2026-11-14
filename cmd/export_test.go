package cmd

import (
	"fmt"
	"net/http"
	"net/http/httptest"
	"os"
	"path/filepath"
	"testing"
	"time"
)

func TestExportWritesWhatTheServerSends(t *testing.T) {
	var asked string
	srv := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		asked = r.URL.RequestURI()
		fmt.Fprint(w, "run,line\n1,1\n")
	}))
	defer srv.Close()
	t.Setenv("MAESTRO_ORQ_URL", srv.URL)

	file := filepath.Join(t.TempDir(), "lines.csv")
	exportSince, exportUntil, exportAll, exportOut = "2026-11-01", "2026-11-08", true, file
	t.Cleanup(func() { exportSince, exportUntil, exportAll, exportOut = "", "", false, "" })
	if err := exportCmd.RunE(background(), nil); err != nil {
		t.Fatal(err)
	}
	if asked != "/api/lines/export?all=true&since=2026-11-01&until=2026-11-08" {
		t.Errorf("asked %q", asked)
	}
	if got, _ := os.ReadFile(file); string(got) != "run,line\n1,1\n" {
		t.Errorf("wrote %q", got)
	}
}

// An export that failed leaves no file behind to be taken for one.
func TestExportThatFailsLeavesNoFile(t *testing.T) {
	srv := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		w.WriteHeader(http.StatusForbidden)
		fmt.Fprint(w, `{"error": "administrators only"}`)
	}))
	defer srv.Close()
	t.Setenv("MAESTRO_ORQ_URL", srv.URL)

	file := filepath.Join(t.TempDir(), "lines.csv")
	exportAll, exportOut = true, file
	t.Cleanup(func() { exportAll, exportOut = false, "" })
	if err := exportCmd.RunE(background(), nil); err == nil {
		t.Fatal("no error from a refused export")
	}
	if _, err := os.Stat(file); !os.IsNotExist(err) {
		t.Errorf("left %s: %v", file, err)
	}
}

func TestRelativeTime(t *testing.T) {
	now := time.Date(2026, 11, 14, 12, 0, 0, 0, time.UTC)
	for in, want := range map[string]string{
		"12h": "2026-11-14T00:00:00Z", "7d": "2026-11-07T12:00:00Z", "2w": "2026-10-31T12:00:00Z",
		"2026-11-01": "2026-11-01", "": "", "d": "d", "7m": "7m",
	} {
		if got := relativeTime(in, now); got != want {
			t.Errorf("relativeTime(%q) = %q, want %q", in, got, want)
		}
	}
}
