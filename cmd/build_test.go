package cmd

import (
	"testing"

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
