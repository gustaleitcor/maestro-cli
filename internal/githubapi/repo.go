package githubapi

import (
	"context"
	"time"

	"github.com/google/go-github/v66/github"
)

// Repo is the subset of GitHub's repository fields Maestro cares about.
// Kept flat and small on purpose — this is the shape a bubbletea
// list.Item or table row will wrap later.
type Repo struct {
	Name        string
	FullName    string
	Description string
	Private     bool
	Fork        bool
	Language    string
	Stars       int
	UpdatedAt   time.Time
	HTMLURL     string
	CloneURL    string
}

// ListRepos fetches all repositories visible to the authenticated user
// (owned, collaborator, and org member repos), paginating as needed.
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

type FileContent struct {
	Path    string
	SHA     string
	Size    int
	Content string
}

// GetFileContent fetches and decodes a single file from a repository at
// the given ref (branch, tag, or commit SHA — empty string means the
// repo's default branch).
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
