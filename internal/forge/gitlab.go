package forge

import (
	"context"
	"net/url"
	"time"
)

type gitlab struct {
	api restAPI
}

type gitlabProject struct {
	Path              string    `json:"path"`
	PathWithNamespace string    `json:"path_with_namespace"`
	Description       string    `json:"description"`
	Visibility        string    `json:"visibility"`
	ForkedFrom        *struct{} `json:"forked_from_project"`
	Stars             int       `json:"star_count"`
	LastActivityAt    time.Time `json:"last_activity_at"`
	WebURL            string    `json:"web_url"`
	HTTPURLToRepo     string    `json:"http_url_to_repo"`
	DefaultBranch     string    `json:"default_branch"`
}

func (g *gitlab) Kind() string { return GitLab }

func (g *gitlab) Host() string { return g.api.base.Host }

func (g *gitlab) CurrentUser(ctx context.Context) (*User, error) {
	var u struct {
		Username string `json:"username"`
		Name     string `json:"name"`
		Email    string `json:"email"`
		Bio      string `json:"bio"`
		WebURL   string `json:"web_url"`
	}
	if _, err := g.api.get(ctx, "/api/v4/user", nil, &u); err != nil {
		return nil, err
	}
	return &User{
		Login:       u.Username,
		Name:        u.Name,
		Email:       u.Email,
		Bio:         u.Bio,
		PublicRepos: -1,
		Followers:   -1,
		Following:   -1,
		HTMLURL:     u.WebURL,
	}, nil
}

func (g *gitlab) ListRepos(ctx context.Context) ([]Repo, error) {
	var all []Repo
	page := "1"
	for page != "" {
		var batch []gitlabProject
		query := url.Values{
			"membership": {"true"},
			"order_by":   {"last_activity_at"},
			"per_page":   {"100"},
			"page":       {page},
		}
		header, err := g.api.get(ctx, "/api/v4/projects", query, &batch)
		if err != nil {
			return nil, err
		}
		for _, p := range batch {
			all = append(all, p.repo())
		}
		page = header.Get("X-Next-Page")
	}
	return all, nil
}

// GetRepo addresses the project by its full path as a single escaped
// segment, which is how GitLab takes nested groups: group%2Fsubgroup%2Fname.
func (g *gitlab) GetRepo(ctx context.Context, owner, repo string) (*Repo, error) {
	var p gitlabProject
	if _, err := g.api.get(ctx, "/api/v4/projects/"+url.PathEscape(owner+"/"+repo), nil, &p); err != nil {
		return nil, err
	}
	converted := p.repo()
	return &converted, nil
}

func (g *gitlab) HasFile(ctx context.Context, owner, repo, ref, path string) (bool, error) {
	var file struct{}
	_, err := g.api.get(ctx, "/api/v4/projects/"+url.PathEscape(owner+"/"+repo)+"/repository/files/"+url.PathEscape(path), url.Values{"ref": {ref}}, &file)
	return missing(err)
}

func (p gitlabProject) repo() Repo {
	return Repo{
		Name:        p.Path,
		FullName:    p.PathWithNamespace,
		Description: p.Description,
		// "internal" is visible to every signed-in user but not to the
		// world, so it is not public either.
		Private:       p.Visibility != "public",
		Fork:          p.ForkedFrom != nil,
		Stars:         p.Stars,
		UpdatedAt:     p.LastActivityAt,
		HTMLURL:       p.WebURL,
		CloneURL:      p.HTTPURLToRepo,
		DefaultBranch: p.DefaultBranch,
	}
}
