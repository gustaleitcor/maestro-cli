// Package githubapi wraps go-github with a small, presentation-agnostic
// surface: functions here return plain structs and never print anything,
// so the same calls can back a plain CLI command today and a Bubble Tea
// tea.Cmd later without any duplicated fetch logic.
package githubapi

import (
	"context"

	"github.com/google/go-github/v66/github"
)

func NewClient(token string) *github.Client {
	return github.NewClient(nil).WithAuthToken(token)
}

type User struct {
	Login       string
	Name        string
	Email       string
	Bio         string
	PublicRepos int
	Followers   int
	Following   int
	HTMLURL     string
}

func GetAuthenticatedUser(ctx context.Context, client *github.Client) (*User, error) {
	u, _, err := client.Users.Get(ctx, "")
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
