package config

import (
	"os"
	"path/filepath"
	"testing"
)

// isolate points the config at an empty directory and keeps the test away
// from the real keyring.
func isolate(t *testing.T) string {
	t.Helper()
	dir := t.TempDir()
	t.Setenv("XDG_CONFIG_HOME", dir)
	t.Setenv("HOME", dir)
	t.Setenv(fileEnvFlag, "1")
	return dir
}

func TestForgesRoundTrip(t *testing.T) {
	isolate(t)

	if forges, err := Forges(); err != nil || len(forges) != 0 {
		t.Fatalf("Forges() on a fresh config = %v, %v", forges, err)
	}

	if err := AddForge(Forge{Name: "codeberg", Kind: "forgejo", BaseURL: "https://codeberg.org"}, "tok-1"); err != nil {
		t.Fatal(err)
	}
	if err := AddForge(Forge{Name: "work", Kind: "gitlab", BaseURL: "https://git.example.com"}, "tok-2"); err != nil {
		t.Fatal(err)
	}
	// Same name again replaces rather than duplicates.
	if err := AddForge(Forge{Name: "codeberg", Kind: "forgejo"}, "tok-3"); err != nil {
		t.Fatal(err)
	}
	if err := SaveMaestroKey("mk_test"); err != nil {
		t.Fatal(err)
	}

	forges, err := Forges()
	if err != nil || len(forges) != 2 || forges[0].Name != "codeberg" || forges[0].BaseURL != "" || forges[1].Kind != "gitlab" {
		t.Fatalf("Forges() = %+v, %v", forges, err)
	}
	if token, err := ForgeToken("codeberg"); err != nil || token != "tok-3" {
		t.Errorf("ForgeToken(codeberg) = %q, %v", token, err)
	}
	if token, err := ForgeToken("work"); err != nil || token != "tok-2" {
		t.Errorf("ForgeToken(work) = %q, %v", token, err)
	}

	if removed, err := RemoveForge("work"); err != nil || !removed {
		t.Fatalf("RemoveForge(work) = %v, %v", removed, err)
	}
	if removed, err := RemoveForge("work"); err != nil || removed {
		t.Errorf("RemoveForge(work) twice = %v, %v", removed, err)
	}
	if _, err := ForgeToken("work"); err == nil {
		t.Error("ForgeToken(work) should fail after removal")
	}
	if key, err := LoadMaestroKey(); err != nil || key != "mk_test" {
		t.Errorf("forge changes clobbered the Maestro key: %q, %v", key, err)
	}

	if err := AddForge(Forge{Name: "Bad Name", Kind: "github"}, "t"); err == nil {
		t.Error("AddForge should reject a name with spaces and capitals")
	}
}

func TestLegacyGitHubTokenMigrates(t *testing.T) {
	dir := isolate(t)

	base, err := os.UserConfigDir()
	if err != nil {
		t.Fatal(err)
	}
	if err := os.MkdirAll(filepath.Join(base, appDirName), 0o700); err != nil {
		t.Fatal(err)
	}
	legacy := `{"github_token":"ghp_old","maestro_key":"mk_old"}`
	if err := os.WriteFile(filepath.Join(base, appDirName, configFile), []byte(legacy), 0o600); err != nil {
		t.Fatal(err)
	}

	forges, err := Forges()
	if err != nil || len(forges) != 1 || forges[0].Name != "github" || forges[0].Kind != "github" {
		t.Fatalf("Forges() after upgrade = %+v, %v (config dir %s)", forges, err, dir)
	}
	if token, err := ForgeToken("github"); err != nil || token != "ghp_old" {
		t.Errorf("ForgeToken(github) = %q, %v", token, err)
	}
	if _, err := LoadGitHubToken(); err == nil {
		t.Error("the legacy github_token should be gone after migrating")
	}
	if key, err := LoadMaestroKey(); err != nil || key != "mk_old" {
		t.Errorf("migration clobbered the Maestro key: %q, %v", key, err)
	}

	// A second call finds the forge and has nothing left to migrate.
	if forges, err := Forges(); err != nil || len(forges) != 1 {
		t.Errorf("Forges() second call = %+v, %v", forges, err)
	}
}

func TestReset(t *testing.T) {
	isolate(t)

	if err := AddForge(Forge{Name: "codeberg", Kind: "forgejo"}, "tok-1"); err != nil {
		t.Fatal(err)
	}
	if err := SaveMaestroKey("mk_test"); err != nil {
		t.Fatal(err)
	}

	if err := Reset(); err != nil {
		t.Fatal(err)
	}
	if forges, err := Forges(); err != nil || len(forges) != 0 {
		t.Fatalf("Forges() after Reset = %v, %v", forges, err)
	}
	if _, err := LoadMaestroKey(); err != ErrNoMaestroKey {
		t.Fatalf("LoadMaestroKey() after Reset = %v, want ErrNoMaestroKey", err)
	}
	if _, err := ForgeToken("codeberg"); err == nil {
		t.Fatal("ForgeToken() after Reset still returns a token")
	}
	// Resetting an already empty config is fine.
	if err := Reset(); err != nil {
		t.Fatal(err)
	}
}
