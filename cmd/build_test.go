package cmd

import (
	"context"
	"encoding/json"
	"fmt"
	"net/http"
	"net/http/httptest"
	"os"
	"path/filepath"
	"strings"
	"testing"

	"maestro-cli/internal/config"
	"maestro-cli/internal/forge"
	"maestro-cli/internal/maestroapi"
)

func TestSplitRepoURL(t *testing.T) {
	cases := []struct {
		in, host, path string
		ok             bool
	}{
		{"https://github.com/octo/app", "github.com", "octo/app", true},
		{"https://github.com/octo/app.git", "github.com", "octo/app", true},
		{"https://codeberg.org/octo/app/", "codeberg.org", "octo/app", true},
		{"https://gitlab.com/grp/sub/app/-/tree/main", "gitlab.com", "grp/sub/app", true},
		{"https://git.example.com:8443/octo/app", "git.example.com:8443", "octo/app", true},
		{"ssh://git@github.com/octo/app", "", "", false},
		{"https:///octo/app", "", "", false},
	}
	for _, tc := range cases {
		host, path, err := splitRepoURL(tc.in)
		if (err == nil) != tc.ok || host != tc.host || path != tc.path {
			t.Errorf("splitRepoURL(%q) = %q, %q, %v", tc.in, host, path, err)
		}
	}
}

func TestSplitRepoPath(t *testing.T) {
	cases := []struct {
		kind, in, owner, repo string
		ok                    bool
	}{
		{forge.GitHub, "octo/app", "octo", "app", true},
		{forge.GitHub, "octo/app/tree/main", "octo", "app", true},
		{forge.Forgejo, "octo/app", "octo", "app", true},
		{forge.GitLab, "octo/app", "octo", "app", true},
		{forge.GitLab, "grp/sub/deep/app", "grp/sub/deep", "app", true},
		{forge.GitHub, "app", "", "", false},
		{forge.GitLab, "grp//app", "", "", false},
	}
	for _, tc := range cases {
		owner, repo, err := splitRepoPath(tc.kind, tc.in)
		if (err == nil) != tc.ok || owner != tc.owner || repo != tc.repo {
			t.Errorf("splitRepoPath(%s, %q) = %q, %q, %v", tc.kind, tc.in, owner, repo, err)
		}
	}
}

// forgejoStub is a Forgejo that has the given owner/name repos and nothing else.
func forgejoStub(t *testing.T, login string, repos ...string) string {
	t.Helper()
	srv := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		switch {
		case r.URL.Path == "/api/v1/user":
			json.NewEncoder(w).Encode(map[string]any{"login": login})
		case r.URL.Path == "/api/v1/user/repos":
			if r.URL.Query().Get("page") != "1" {
				w.Write([]byte("[]"))
				return
			}
			var list []map[string]any
			for _, full := range repos {
				_, name, _ := strings.Cut(full, "/")
				list = append(list, map[string]any{"name": name, "full_name": full})
			}
			json.NewEncoder(w).Encode(list)
		default:
			for _, full := range repos {
				if r.URL.Path == "/api/v1/repos/"+full {
					json.NewEncoder(w).Encode(map[string]any{"full_name": full})
					return
				}
			}
			w.WriteHeader(http.StatusNotFound)
		}
	}))
	t.Cleanup(srv.Close)
	return srv.URL
}

func TestBuildFindsTheForge(t *testing.T) {
	dir := t.TempDir()
	t.Setenv("XDG_CONFIG_HOME", dir)
	t.Setenv("HOME", dir)
	t.Setenv("MAESTRO_FORCE_FILE_STORE", "1")

	home := forgejoStub(t, "ana", "ana/app", "team/shared")
	work := forgejoStub(t, "ana-w", "ana-w/api", "team/shared")
	for name, baseURL := range map[string]string{"home": home, "work": work} {
		if err := config.AddForge(config.Forge{Name: name, Kind: forge.Forgejo, BaseURL: baseURL}, "tok"); err != nil {
			t.Fatal(err)
		}
	}
	ctx := context.Background()

	// Without --forge, the repo is looked for on every forge.
	for arg, want := range map[string]string{"ana/app": "home ana/app", "api": "work ana-w/api", "app": "home ana/app"} {
		active, owner, repo, err := parseRepoArg(ctx, arg, "")
		if err != nil || active.Name+" "+owner+"/"+repo != want {
			t.Errorf("parseRepoArg(%q) = %v %s/%s, %v; want %s", arg, active, owner, repo, err, want)
		}
	}
	if _, _, _, err := parseRepoArg(ctx, "team/shared", ""); err == nil || !strings.Contains(err.Error(), "several forges") {
		t.Errorf("a repo on both forges = %v", err)
	}
	if active, _, _, err := parseRepoArg(ctx, "team/shared", "work"); err != nil || active.Name != "work" {
		t.Errorf("--forge work = %v, %v", active, err)
	}
	if _, _, _, err := parseRepoArg(ctx, "ana/nope", ""); err == nil {
		t.Error("a repo on no forge was found")
	}

	// Completion offers both forges' repos, saying where each lives.
	buildForge = ""
	got, _ := completeRepos(buildCmd, nil, "")
	joined := strings.Join(got, "\n")
	if len(got) != 4 {
		t.Errorf("completions = %q, want one per repo", got)
	}
	for _, want := range []string{"ana/app\thome", "ana-w/api\twork", "team/shared\thome", "team/shared\twork"} {
		if !strings.Contains(joined, want) {
			t.Errorf("completions %q lack %q", got, want)
		}
	}
}

func TestBuildPublicRepoNeedsNoForge(t *testing.T) {
	dir := t.TempDir()
	t.Setenv("XDG_CONFIG_HOME", dir)
	t.Setenv("HOME", dir)
	t.Setenv("MAESTRO_FORCE_FILE_STORE", "1")
	ctx := context.Background()

	for arg, want := range map[string]string{
		"https://github.com/ana/app":    "github ana/app",
		"https://gitlab.com/team/sub/x": "gitlab team/sub/x",
		"https://codeberg.org/ana/app":  "forgejo ana/app",
	} {
		active, owner, repo, err := parseRepoArg(ctx, arg, "")
		if err != nil || active.Token != "" || active.Name+" "+owner+"/"+repo != want {
			t.Errorf("parseRepoArg(%q) = %v %s/%s, %v; want %s", arg, active, owner, repo, err, want)
		}
	}
	// Anonymous means a full URL: a name or owner/name needs a forge.
	if _, _, _, err := parseRepoArg(ctx, "ana/app", ""); err == nil {
		t.Error("owner/name was served without a forge")
	}
	// A host that isn't one of the three public ones still needs a forge.
	if _, _, _, err := parseRepoArg(ctx, "https://git.example.com/ana/app", ""); err == nil {
		t.Error("an unknown host was served without a forge")
	}
}

// A repo the token can't see isn't said to lack a Dockerfile.
func TestCheckBuildFile(t *testing.T) {
	client, err := forge.New(forge.Forgejo, forgejoStub(t, "ana", "ana/app"), "tok")
	if err != nil {
		t.Fatal(err)
	}
	ctx := context.Background()
	if err := checkBuildFile(ctx, client, "ana", "app", "main"); err == nil || !strings.Contains(err.Error(), "has no Dockerfile") {
		t.Errorf("a repo without one: %v", err)
	}
	if err := checkBuildFile(ctx, client, "raquel", "private", "main"); err == nil || !strings.Contains(err.Error(), "raquel/private not found") {
		t.Errorf("a repo the token can't see: %v", err)
	}
}

// The files -c, -g and -f name are read whatever they are called here; the
// generator is sent with its own name, and nothing empty or too large goes.
func TestReadLocalFiles(t *testing.T) {
	dir := t.TempDir()
	write := func(name, content string) string {
		path := filepath.Join(dir, name)
		os.WriteFile(path, []byte(content), 0o644)
		return path
	}
	files, err := readLocalFiles(localFlags{config: write("bench.toml", "cpus = 2"), generator: write("lines.py", "def genvrs(): ..."), dockerfile: write("Dockerfile.dev", "FROM x")})
	want := maestroapi.LocalFiles{Config: "cpus = 2", Generator: "def genvrs(): ...", GeneratorName: "lines.py", Dockerfile: "FROM x"}
	if err != nil || files != want {
		t.Fatalf("got %+v, %v", files, err)
	}
	if files, err := readLocalFiles(localFlags{}); err != nil || !files.Empty() {
		t.Errorf("no flags: %+v, %v", files, err)
	}
	for name, flags := range map[string]localFlags{
		"missing":   {config: filepath.Join(dir, "nope.toml")},
		"empty":     {config: write("empty.toml", "")},
		"too large": {generator: write("big.py", strings.Repeat("#", maxLocalFile+1))},
		"directory": {dockerfile: dir},
	} {
		if _, err := readLocalFiles(flags); err == nil {
			t.Errorf("%s: read", name)
		}
	}
}

// A run sent files carries them and the forge's token; one without sends
// the request it always did.
func TestStartRunSendsFiles(t *testing.T) {
	var body map[string]json.RawMessage
	var token string
	srv := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		body, token = nil, r.Header.Get("X-Forge-Token")
		json.NewDecoder(r.Body).Decode(&body)
		w.WriteHeader(http.StatusCreated)
		fmt.Fprint(w, `{"id": 7, "local_files": ["maestro.toml"]}`)
	}))
	defer srv.Close()
	t.Setenv("MAESTRO_ORQ_URL", srv.URL)

	run, err := maestroapi.StartRun(context.Background(), "key", 81, maestroapi.LocalFiles{Config: "cpus = 2"}, "tok")
	if err != nil || run.ID != 7 || token != "tok" || string(body["files"]) != `{"config":"cpus = 2"}` {
		t.Errorf("with files: %+v, %v, token %q, body %s", run, err, token, body["files"])
	}
	if _, err := maestroapi.StartRun(context.Background(), "key", 81, maestroapi.LocalFiles{}, "tok"); err != nil || token != "" || body["files"] != nil {
		t.Errorf("without: %v, token %q, files %s", err, token, body["files"])
	}
}
