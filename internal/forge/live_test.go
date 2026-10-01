package forge

import (
	"context"
	"os"
	"testing"
	"time"
)

// TestLiveGetRepo reads one public repo from each default forge, without a
// token. It needs the network, so it only runs with MAESTRO_LIVE_TESTS=1.
func TestLiveGetRepo(t *testing.T) {
	if os.Getenv("MAESTRO_LIVE_TESTS") == "" {
		t.Skip("set MAESTRO_LIVE_TESTS=1 to query github.com, codeberg.org and gitlab.com")
	}

	cases := []struct{ kind, owner, repo, fullName string }{
		{GitHub, "octocat", "Hello-World", "octocat/Hello-World"},
		{Forgejo, "forgejo", "forgejo", "forgejo/forgejo"},
		{GitLab, "gitlab-org/gitlab-foss", "", ""},
		{GitLab, "gitlab-org/ci-cd/codequality", "", ""},
	}
	for _, tc := range cases {
		owner, repo, fullName := tc.owner, tc.repo, tc.fullName
		if tc.kind == GitLab {
			// The full path, split the way `maestro build` splits it.
			fullName = tc.owner
			owner, repo, _ = SplitFullName(fullName)
		}
		t.Run(tc.kind+"/"+fullName, func(t *testing.T) {
			f, err := New(tc.kind, "", "")
			if err != nil {
				t.Fatal(err)
			}
			ctx, cancel := context.WithTimeout(context.Background(), 30*time.Second)
			defer cancel()

			got, err := f.GetRepo(ctx, owner, repo)
			if err != nil {
				t.Fatal(err)
			}
			if got.FullName != fullName || got.DefaultBranch == "" || got.CloneURL == "" || got.Private {
				t.Errorf("GetRepo = %+v", got)
			}
			t.Logf("%s: %s, default branch %s, clone %s", f.Host(), got.FullName, got.DefaultBranch, got.CloneURL)
		})
	}
}
