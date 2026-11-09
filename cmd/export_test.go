package cmd

import (
	"fmt"
	"net/http"
	"net/http/httptest"
	"os"
	"path/filepath"
	"testing"
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
