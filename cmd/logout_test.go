package cmd

import (
	"bytes"
	"context"
	"net/http"
	"net/http/httptest"
	"os"
	"path/filepath"
	"strings"
	"testing"
)

// signedInAt is signedIn with the server that issued the key kept too.
func signedInAt(t *testing.T, key, server string) string {
	t.Helper()
	dir := t.TempDir()
	t.Setenv("XDG_CONFIG_HOME", dir)
	t.Setenv("MAESTRO_FORCE_FILE_STORE", "1")
	file := filepath.Join(dir, "maestro", "config.json")
	os.MkdirAll(filepath.Dir(file), 0o755)
	os.WriteFile(file, []byte(`{"maestro_key":"`+key+`","server":"`+server+`"}`), 0o600)
	return file
}

func TestLogoutRevokesTheKey(t *testing.T) {
	var revoked []string
	srv := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		if r.Method == http.MethodDelete && r.URL.Path == "/api/keys/current" {
			revoked = append(revoked, r.Header.Get("Authorization"))
			w.WriteHeader(http.StatusNoContent)
			return
		}
		w.WriteHeader(http.StatusNotFound)
	}))
	defer srv.Close()

	// The key goes to the server that issued it, not to the one
	// MAESTRO_ORQ_URL names.
	t.Setenv("MAESTRO_ORQ_URL", "http://127.0.0.1:1")
	file := signedInAt(t, "mk_abcdefghijkl", srv.URL)
	logoutYes = true
	t.Cleanup(func() { logoutYes = false })

	logoutCmd.SetContext(context.Background())
	if err := runLogout(logoutCmd, nil); err != nil {
		t.Fatal(err)
	}
	if len(revoked) != 1 || revoked[0] != "Bearer mk_abcdefghijkl" {
		t.Errorf("DELETE /api/keys/current got %q; want the key once", revoked)
	}
	if _, err := os.Stat(file); !os.IsNotExist(err) {
		t.Errorf("the config is still there after logout: %v", err)
	}
}

func TestLogoutUnreachable(t *testing.T) {
	srv := httptest.NewServer(http.NotFoundHandler())
	url := srv.URL
	srv.Close()
	signedInAt(t, "mk_abcdefghijkl", url)

	var out bytes.Buffer
	revokeOnLogout(context.Background(), &out, "mk_abcdefghijkl")
	if got := out.String(); !strings.Contains(got, "still valid") || !strings.Contains(got, "maestro keys rm mk_abcdefg") {
		t.Errorf("unreachable: %q; want it to say the key is still valid and how to revoke it", got)
	}
}
