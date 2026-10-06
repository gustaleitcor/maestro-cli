package forge

import (
	"context"
	"errors"
	"fmt"
	"net/url"
	"time"

	gh "github.com/google/go-github/v66/github"
)

type github struct {
	client *gh.Client
	host   string
}

func (g *github) Kind() string { return GitHub }

func (g *github) Host() string { return g.host }

func (g *github) CurrentUser(ctx context.Context) (*User, error) {
	u, _, err := g.client.Users.Get(ctx, "")
	if err != nil {
		return nil, g.explain(err)
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
			return nil, g.explain(err)
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
		return nil, g.explain(err)
	}
	converted := githubRepo(r)
	return &converted, nil
}

func (g *github) HasFile(ctx context.Context, owner, repo, ref, path string) (bool, error) {
	_, _, _, err := g.client.Repositories.GetContents(ctx, owner, repo, path, &gh.RepositoryContentGetOptions{Ref: ref})
	if err != nil {
		err = g.explain(err)
	}
	return missing(err)
}

// explain replaces go-github's "GET <url>: 401 Bad credentials []" with the
// same wording the other forges' errors get.
func (g *github) explain(err error) error {
	var rateLimit *gh.RateLimitError
	if errors.As(err, &rateLimit) {
		return fmt.Errorf("%s is rate limiting this token; try again at %s", g.host, rateLimit.Rate.Reset.Local().Format(time.Kitchen))
	}
	var abuse *gh.AbuseRateLimitError
	if errors.As(err, &abuse) {
		return statusError(g.host, 429, abuse.Message)
	}
	var response *gh.ErrorResponse
	if errors.As(err, &response) && response.Response != nil {
		return statusError(g.host, response.Response.StatusCode, response.Message)
	}
	return unreachableError(g.host, err)
}

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
