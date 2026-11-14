package config

import (
	"errors"
	"os"
	"path/filepath"
	"testing"
)

// A key goes only to the server that minted it.
func TestKeyIsOnlyForItsServer(t *testing.T) {
	isolate(t)
	if err := SaveMaestroKey("https://maestro.example.org", "mk_a"); err != nil {
		t.Fatal(err)
	}
	if stored, err := Server(); err != nil || stored != "https://maestro.example.org" {
		t.Fatalf("Server() = %q, %v", stored, err)
	}

	for _, asked := range []string{"https://maestro.example.org", "https://MAESTRO.example.org/", ""} {
		if key, err := LoadMaestroKey(asked); err != nil || key != "mk_a" {
			t.Errorf("LoadMaestroKey(%q) = %q, %v; want mk_a", asked, key, err)
		}
	}

	key, err := LoadMaestroKey("https://evil.example.com")
	var other *OtherServerError
	if key != "" || !errors.As(err, &other) || other.Stored != "https://maestro.example.org" || other.Asked != "https://evil.example.com" {
		t.Errorf("LoadMaestroKey(another server) = %q, %v; want no key and OtherServerError", key, err)
	}
}

// A key stored before servers were, with no server beside it, goes to any:
// nothing says which it is for.
func TestKeyFromBeforeServersGoesAnywhere(t *testing.T) {
	dir := isolate(t)
	os.MkdirAll(filepath.Join(dir, appDirName), 0o700)
	if err := os.WriteFile(filepath.Join(dir, appDirName, configFile), []byte(`{"maestro_key":"mk_old"}`), 0o600); err != nil {
		t.Fatal(err)
	}
	if key, err := LoadMaestroKey("http://localhost:8080"); err != nil || key != "mk_old" {
		t.Errorf("LoadMaestroKey = %q, %v; want mk_old", key, err)
	}
}
