package forge

import (
	"context"
	"net/url"
	"sort"
	"strconv"
	"time"
)

type forgejo struct {
	api restAPI
}

type forgejoRepo struct {
	Name          string    `json:"name"`
	FullName      string    `json:"full_name"`
	Description   string    `json:"description"`
	Private       bool      `json:"private"`
	Fork          bool      `json:"fork"`
	Language      string    `json:"language"`
	Stars         int       `json:"stars_count"`
	UpdatedAt     time.Time `json:"updated_at"`
	HTMLURL       string    `json:"html_url"`
	CloneURL      string    `json:"clone_url"`
	DefaultBranch string    `json:"default_branch"`
}

func (f *forgejo) Kind() string { return Forgejo }

func (f *forgejo) Host() string { return f.api.base.Host }

func (f *forgejo) CurrentUser(ctx context.Context) (*User, error) {
	var u struct {
		Login     string `json:"login"`
		FullName  string `json:"full_name"`
		Email     string `json:"email"`
		Bio       string `json:"description"`
		Followers int    `json:"followers_count"`
		Following int    `json:"following_count"`
		HTMLURL   string `json:"html_url"`
	}
	if _, err := f.api.get(ctx, "/api/v1/user", nil, &u); err != nil {
		return nil, err
	}
	htmlURL := u.HTMLURL
	if htmlURL == "" {
		htmlURL = f.api.base.String() + "/" + url.PathEscape(u.Login)
	}
	return &User{
		Login:       u.Login,
		Name:        u.FullName,
		Email:       u.Email,
		Bio:         u.Bio,
		PublicRepos: -1,
		Followers:   u.Followers,
		Following:   u.Following,
		HTMLURL:     htmlURL,
	}, nil
}

func (f *forgejo) ListRepos(ctx context.Context) ([]Repo, error) {
	var all []Repo
	// A server may cap the page size below what is asked for, so a short
	// page doesn't mean the last one; only an empty page does.
	for page := 1; ; page++ {
		var batch []forgejoRepo
		query := url.Values{"page": {strconv.Itoa(page)}, "limit": {"50"}}
		if _, err := f.api.get(ctx, "/api/v1/user/repos", query, &batch); err != nil {
			return nil, err
		}
		if len(batch) == 0 {
			break
		}
		for _, r := range batch {
			all = append(all, r.repo())
		}
	}
	sort.SliceStable(all, func(i, j int) bool { return all[i].UpdatedAt.After(all[j].UpdatedAt) })
	return all, nil
}

func (f *forgejo) GetRepo(ctx context.Context, owner, repo string) (*Repo, error) {
	var r forgejoRepo
	if _, err := f.api.get(ctx, "/api/v1/repos/"+url.PathEscape(owner)+"/"+url.PathEscape(repo), nil, &r); err != nil {
		return nil, err
	}
	converted := r.repo()
	return &converted, nil
}

func (f *forgejo) HasFile(ctx context.Context, owner, repo, ref, path string) (bool, error) {
	var file struct{}
	_, err := f.api.get(ctx, "/api/v1/repos/"+url.PathEscape(owner)+"/"+url.PathEscape(repo)+"/contents/"+url.PathEscape(path), url.Values{"ref": {ref}}, &file)
	return missing(err)
}

func (r forgejoRepo) repo() Repo {
	return Repo{
		Name:          r.Name,
		FullName:      r.FullName,
		Description:   r.Description,
		Private:       r.Private,
		Fork:          r.Fork,
		Language:      r.Language,
		Stars:         r.Stars,
		UpdatedAt:     r.UpdatedAt,
		HTMLURL:       r.HTMLURL,
		CloneURL:      r.CloneURL,
		DefaultBranch: r.DefaultBranch,
	}
}
