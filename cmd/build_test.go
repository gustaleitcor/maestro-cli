package cmd

import (
	"context"
	"encoding/json"
	"net/http"
	"net/http/httptest"
	"strings"
	"testing"

	"maestro-cli/internal/config"
	"maestro-cli/internal/forge"
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
