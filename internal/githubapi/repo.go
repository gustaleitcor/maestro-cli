package githubapi

import (
	"context"
	"time"

	"github.com/google/go-github/v66/github"
)

type Repo struct {
	Name          string
	FullName      string
	Description   string
	Private       bool
	Fork          bool
	Language      string
	Stars         int
	UpdatedAt     time.Time
	HTMLURL       string
	CloneURL      string
	DefaultBranch string
}

func ListRepos(ctx context.Context, client *github.Client) ([]Repo, error) {
	opts := &github.RepositoryListByAuthenticatedUserOptions{
		Sort:        "updated",
		ListOptions: github.ListOptions{PerPage: 100},
	}

	var all []Repo
	for {
		repos, resp, err := client.Repositories.ListByAuthenticatedUser(ctx, opts)
		if err != nil {
			return nil, err
		}
		for _, r := range repos {
			all = append(all, Repo{
				Name:        r.GetName(),
				FullName:    r.GetFullName(),
				Description: r.GetDescription(),
				Private:     r.GetPrivate(),
				Fork:        r.GetFork(),
				Language:    r.GetLanguage(),
				Stars:       r.GetStargazersCount(),
				UpdatedAt:   r.GetUpdatedAt().Time,
				HTMLURL:     r.GetHTMLURL(),
				CloneURL:    r.GetCloneURL(),
			})
		}
		if resp.NextPage == 0 {
			break
		}
		opts.Page = resp.NextPage
	}
	return all, nil
}

// GetRepo fetches a single repo's metadata, primarily to resolve its default branch.
func GetRepo(ctx context.Context, client *github.Client, owner, repo string) (*Repo, error) {
	r, _, err := client.Repositories.Get(ctx, owner, repo)
	if err != nil {
		return nil, err
	}
	return &Repo{
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
	}, nil
}

type FileContent struct {
	Path    string
	SHA     string
	Size    int
	Content string
}

// ref empty means the repo's default branch.
func GetFileContent(ctx context.Context, client *github.Client, owner, repo, path, ref string) (*FileContent, error) {
	opts := &github.RepositoryContentGetOptions{Ref: ref}
	fileContent, _, _, err := client.Repositories.GetContents(ctx, owner, repo, path, opts)
	if err != nil {
		return nil, err
	}

	content, err := fileContent.GetContent()
	if err != nil {
		return nil, err
	}

	return &FileContent{
		Path:    fileContent.GetPath(),
		SHA:     fileContent.GetSHA(),
		Size:    fileContent.GetSize(),
		Content: content,
	}, nil
}
