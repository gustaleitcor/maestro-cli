package forge

import (
	"context"
	"net/url"

	gh "github.com/google/go-github/v66/github"
)

type github struct {
	client *gh.Client
	host   string
}

// newGitHub talks to github.com, or to a GitHub Enterprise Server when base
// points anywhere else.
func newGitHub(base *url.URL, token string) (Forge, error) {
	client := gh.NewClient(nil)
	if token != "" {
		client = client.WithAuthToken(token)
	}
	if base.Host != "github.com" {
		var err error
		if client, err = client.WithEnterpriseURLs(base.String(), base.String()); err != nil {
			return nil, err
		}
	}
	return &github{client: client, host: base.Host}, nil
}

func (g *github) Kind() string { return GitHub }
func (g *github) Host() string { return g.host }

func (g *github) CurrentUser(ctx context.Context) (*User, error) {
	u, _, err := g.client.Users.Get(ctx, "")
	if err != nil {
		return nil, err
	}
	return &User{
		Login:       u.GetLogin(),
		Name:        u.GetName(),
		Email:       u.GetEmail(),
		Bio:         u.GetBio(),
		PublicRepos: u.GetPublicRepos(),
		Followers:   u.GetFollowers(),
		Following:   u.GetFollowing(),
		HTMLURL:     u.GetHTMLURL(),
	}, nil
}

func (g *github) ListRepos(ctx context.Context) ([]Repo, error) {
	opts := &gh.RepositoryListByAuthenticatedUserOptions{
		Sort:        "updated",
		ListOptions: gh.ListOptions{PerPage: 100},
	}

	var all []Repo
	for {
		repos, resp, err := g.client.Repositories.ListByAuthenticatedUser(ctx, opts)
		if err != nil {
			return nil, err
		}
		for _, r := range repos {
			all = append(all, githubRepo(r))
		}
		if resp.NextPage == 0 {
			break
		}
		opts.Page = resp.NextPage
	}
	return all, nil
}

func (g *github) GetRepo(ctx context.Context, owner, repo string) (*Repo, error) {
	r, _, err := g.client.Repositories.Get(ctx, owner, repo)
	if err != nil {
		return nil, err
	}
	converted := githubRepo(r)
	return &converted, nil
}

func githubRepo(r *gh.Repository) Repo {
	return Repo{
		Name:          r.GetName(),
		FullName:      r.GetFullName(),
		Description:   r.GetDescription(),
		Private:       r.GetPrivate(),
		Fork:          r.GetFork(),
		Language:      r.GetLanguage(),
		Stars:         r.GetStargazersCount(),
		UpdatedAt:     r.GetUpdatedAt().Time,
		HTMLURL:       r.GetHTMLURL(),
		CloneURL:      r.GetCloneURL(),
		DefaultBranch: r.GetDefaultBranch(),
	}
}
