package forge

import (
	"context"
	"encoding/json"
	"fmt"
	"net/http"
	"net/http/httptest"
	"strings"
	"testing"
)

func TestForgejo(t *testing.T) {
	var pages []string
	srv := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		if got := r.Header.Get("Authorization"); got != "token s3cret" {
			t.Errorf("Authorization = %q, want %q", got, "token s3cret")
		}
		switch r.URL.Path {
		case "/api/v1/user":
			fmt.Fprint(w, `{"login":"ana","full_name":"Ana","email":"ana@example.com","followers_count":3,"following_count":1}`)
		case "/api/v1/user/repos":
			page := r.URL.Query().Get("page")
			pages = append(pages, page)
			// Two repos per page although 50 were asked for: a capped server.
			switch page {
			case "1":
				fmt.Fprint(w, `[{"name":"a","full_name":"ana/a","updated_at":"2026-01-01T00:00:00Z"},{"name":"b","full_name":"ana/b","private":true,"updated_at":"2026-03-01T00:00:00Z"}]`)
			case "2":
				fmt.Fprint(w, `[{"name":"c","full_name":"org/c","stars_count":7,"language":"Go","updated_at":"2026-02-01T00:00:00Z"}]`)
			default:
				fmt.Fprint(w, `[]`)
			}
		case "/api/v1/repos/ana/b":
			fmt.Fprint(w, `{"name":"b","full_name":"ana/b","default_branch":"trunk","clone_url":"https://x/ana/b.git"}`)
		default:
			w.WriteHeader(http.StatusNotFound)
			fmt.Fprint(w, `{"message":"not here"}`)
		}
	}))
	defer srv.Close()

	f, err := New(Forgejo, srv.URL+"/", "s3cret")
	if err != nil {
		t.Fatal(err)
	}
	ctx := context.Background()

	if f.Kind() != Forgejo || f.Host() != strings.TrimPrefix(srv.URL, "http://") {
		t.Errorf("Kind, Host = %q, %q", f.Kind(), f.Host())
	}

	user, err := f.CurrentUser(ctx)
	if err != nil || user.Login != "ana" || user.Followers != 3 || user.PublicRepos != -1 || user.HTMLURL != srv.URL+"/ana" {
		t.Errorf("CurrentUser = %+v, %v", user, err)
	}

	repos, err := f.ListRepos(ctx)
	if err != nil {
		t.Fatal(err)
	}
	var names []string
	for _, r := range repos {
		names = append(names, r.FullName)
	}
	if got := strings.Join(names, ","); got != "ana/b,org/c,ana/a" {
		t.Errorf("ListRepos = %s, want all three pages, most recently updated first", got)
	}
	if got := strings.Join(pages, ","); got != "1,2,3" {
		t.Errorf("pages requested = %s, want 1,2,3", got)
	}
	if !repos[0].Private || repos[1].Stars != 7 || repos[1].Language != "Go" {
		t.Errorf("repo fields not mapped: %+v", repos)
	}

	repo, err := f.GetRepo(ctx, "ana", "b")
	if err != nil || repo.DefaultBranch != "trunk" {
		t.Errorf("GetRepo = %+v, %v", repo, err)
	}

	if _, err := f.GetRepo(ctx, "ana", "missing"); err == nil || !strings.Contains(err.Error(), "not here") {
		t.Errorf("GetRepo(missing) error = %v, want the server's message", err)
	}
}

func TestGitLab(t *testing.T) {
	srv := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		if got := r.Header.Get("PRIVATE-TOKEN"); got != "glpat-x" {
			t.Errorf("PRIVATE-TOKEN = %q, want glpat-x", got)
		}
		if r.Header.Get("Authorization") != "" {
			t.Errorf("unexpected Authorization header")
		}
		switch r.URL.EscapedPath() {
		case "/api/v4/user":
			fmt.Fprint(w, `{"username":"ana","name":"Ana","web_url":"https://gitlab.example/ana"}`)
		case "/api/v4/projects":
			q := r.URL.Query()
			if q.Get("membership") != "true" {
				t.Errorf("membership = %q, want true", q.Get("membership"))
			}
			if q.Get("page") == "1" {
				w.Header().Set("X-Next-Page", "2")
				fmt.Fprint(w, `[{"path":"app","path_with_namespace":"grp/sub/app","visibility":"private","star_count":2,"last_activity_at":"2026-03-01T00:00:00Z"}]`)
				return
			}
			w.Header().Set("X-Next-Page", "")
			fmt.Fprint(w, `[{"path":"site","path_with_namespace":"ana/site","visibility":"public","forked_from_project":{"id":1}},{"path":"wiki","path_with_namespace":"ana/wiki","visibility":"internal"}]`)
		case "/api/v4/projects/grp%2Fsub%2Fapp":
			fmt.Fprint(w, `{"path":"app","path_with_namespace":"grp/sub/app","default_branch":"main","http_url_to_repo":"https://gitlab.example/grp/sub/app.git"}`)
		default:
			t.Errorf("unexpected request %s", r.URL.EscapedPath())
			w.WriteHeader(http.StatusNotFound)
		}
	}))
	defer srv.Close()

	g, err := New(GitLab, srv.URL, "glpat-x")
	if err != nil {
		t.Fatal(err)
	}
	ctx := context.Background()

	user, err := g.CurrentUser(ctx)
	if err != nil || user.Login != "ana" || user.Followers != -1 {
		t.Errorf("CurrentUser = %+v, %v", user, err)
	}

	repos, err := g.ListRepos(ctx)
	if err != nil {
		t.Fatal(err)
	}
	if len(repos) != 3 {
		t.Fatalf("ListRepos returned %d repos, want 3 across two pages", len(repos))
	}
	if repos[0].FullName != "grp/sub/app" || repos[0].Name != "app" || !repos[0].Private || repos[0].Stars != 2 {
		t.Errorf("nested project not mapped: %+v", repos[0])
	}
	if repos[1].Private || !repos[1].Fork {
		t.Errorf("public fork not mapped: %+v", repos[1])
	}
	if !repos[2].Private {
		t.Errorf("an internal project must not be shown as public: %+v", repos[2])
	}

	repo, err := g.GetRepo(ctx, "grp/sub", "app")
	if err != nil || repo.DefaultBranch != "main" {
		t.Errorf("GetRepo(nested group) = %+v, %v", repo, err)
	}
}

func TestGitHubEnterprise(t *testing.T) {
	srv := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		if got := r.Header.Get("Authorization"); got != "Bearer ghp_x" {
			t.Errorf("Authorization = %q, want Bearer ghp_x", got)
		}
		switch r.URL.Path {
		case "/api/v3/user":
			json.NewEncoder(w).Encode(map[string]any{"login": "ana", "public_repos": 4})
		case "/api/v3/user/repos":
			if r.URL.Query().Get("page") == "2" {
				json.NewEncoder(w).Encode([]map[string]any{{"name": "b", "full_name": "ana/b"}})
				return
			}
			w.Header().Set("Link", `<`+"http://"+r.Host+`/api/v3/user/repos?page=2>; rel="next"`)
			json.NewEncoder(w).Encode([]map[string]any{{"name": "a", "full_name": "ana/a", "private": true}})
		case "/api/v3/repos/ana/a":
			json.NewEncoder(w).Encode(map[string]any{"name": "a", "full_name": "ana/a", "default_branch": "main"})
		default:
			t.Errorf("unexpected request %s", r.URL.Path)
			w.WriteHeader(http.StatusNotFound)
		}
	}))
	defer srv.Close()

	g, err := New(GitHub, srv.URL, "ghp_x")
	if err != nil {
		t.Fatal(err)
	}
	ctx := context.Background()

	if g.Host() != strings.TrimPrefix(srv.URL, "http://") {
		t.Errorf("Host = %q", g.Host())
	}
	user, err := g.CurrentUser(ctx)
	if err != nil || user.Login != "ana" || user.PublicRepos != 4 {
		t.Errorf("CurrentUser = %+v, %v", user, err)
	}
	repos, err := g.ListRepos(ctx)
	if err != nil || len(repos) != 2 || !repos[0].Private || repos[1].FullName != "ana/b" {
		t.Errorf("ListRepos = %+v, %v", repos, err)
	}
	repo, err := g.GetRepo(ctx, "ana", "a")
	if err != nil || repo.DefaultBranch != "main" {
		t.Errorf("GetRepo = %+v, %v", repo, err)
	}
}

func TestNew(t *testing.T) {
	if f, err := New(GitHub, "", "t"); err != nil || f.Host() != "github.com" {
		t.Errorf("New(github, default) = %v, %v", f, err)
	}
	if f, err := New(Forgejo, "", "t"); err != nil || f.Host() != "codeberg.org" {
		t.Errorf("New(forgejo, default) = %v, %v", f, err)
	}
	if f, err := New(GitLab, "https://git.example.com:8443/", "t"); err != nil || f.Host() != "git.example.com:8443" {
		t.Errorf("New(gitlab, self-hosted) = %v, %v", f, err)
	}
	for _, bad := range []string{"git.example.com", "ftp://git.example.com", "://"} {
		if _, err := New(Forgejo, bad, "t"); err == nil {
			t.Errorf("New(forgejo, %q) should fail", bad)
		}
	}
	if _, err := New("svn", "https://x.example", "t"); err == nil {
		t.Error("New(svn) should fail")
	}
}

func TestSplitFullName(t *testing.T) {
	cases := []struct {
		in, owner, name string
		ok              bool
	}{
		{"ana/app", "ana", "app", true},
		{"grp/sub/app", "grp/sub", "app", true},
		{"app", "", "", false},
		{"/app", "", "", false},
		{"ana/", "", "", false},
	}
	for _, tc := range cases {
		owner, name, ok := SplitFullName(tc.in)
		if owner != tc.owner || name != tc.name || ok != tc.ok {
			t.Errorf("SplitFullName(%q) = %q, %q, %v", tc.in, owner, name, ok)
		}
	}
}
