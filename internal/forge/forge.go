// Package forge reads repositories from a git forge. A forge never signs
// anyone in to Maestro: it is only its API plus a read-only token, used to
// list repos and to hand maestro-orq something it can clone with.
package forge

import (
	"context"
	"fmt"
	"net/url"
	"strings"
	"time"
)

const (
	GitHub  = "github"
	Forgejo = "forgejo" // also Codeberg and Gitea, which share the API
	GitLab  = "gitlab"
)

// Kinds lists the supported forge kinds, in the order they are offered.
var Kinds = []string{GitHub, Forgejo, GitLab}

type Forge interface {
	Kind() string
	// Host is the forge's host[:port], as maestro-orq needs it to clone.
	Host() string
	CurrentUser(ctx context.Context) (*User, error)
	// ListRepos returns the repos the token can see, most recently updated first.
	ListRepos(ctx context.Context) ([]Repo, error)
	// GetRepo takes the owner as a path: on GitLab it may be a nested group.
	GetRepo(ctx context.Context, owner, repo string) (*Repo, error)
}

// User is the token's owner. The counters are -1 where a forge's API
// doesn't report them.
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

type Repo struct {
	Name          string
	FullName      string // owner/name; the owner part may itself contain slashes on GitLab
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

// DefaultBaseURL is where each kind is hosted when it isn't self-hosted.
func DefaultBaseURL(kind string) string {
	switch kind {
	case GitHub:
		return "https://github.com"
	case Forgejo:
		return "https://codeberg.org"
	case GitLab:
		return "https://gitlab.com"
	}
	return ""
}

// TokenHint tells the user which token to create, with the least access
// Maestro needs: listing repos and cloning them.
func TokenHint(kind, baseURL string) string {
	baseURL = strings.TrimRight(baseURL, "/")
	switch kind {
	case GitHub:
		return "Create a fine-grained personal access token at\n" +
			"  " + baseURL + "/settings/personal-access-tokens/new\n" +
			"with \"Repository access\" set to the repos Maestro may build, and under\n" +
			"\"Permissions\" grant Contents: Read-only. Nothing else is needed."
	case Forgejo:
		return "Create an access token at\n" +
			"  " + baseURL + "/user/settings/applications\n" +
			"with the scopes read:repository and read:user. Nothing else is needed."
	case GitLab:
		return "Create a personal access token at\n" +
			"  " + baseURL + "/-/user_settings/personal_access_tokens\n" +
			"with the scopes read_api and read_repository. Nothing else is needed."
	}
	return ""
}

// New builds the client for one configured forge. An empty baseURL means
// the kind's public instance; an empty token means anonymous access, which
// only sees what the forge shows to everyone.
func New(kind, baseURL, token string) (Forge, error) {
	if baseURL == "" {
		baseURL = DefaultBaseURL(kind)
	}
	base, err := url.Parse(strings.TrimRight(baseURL, "/"))
	if err != nil || base.Host == "" || (base.Scheme != "https" && base.Scheme != "http") {
		return nil, fmt.Errorf("invalid forge URL %q: expected something like https://git.example.com", baseURL)
	}

	switch kind {
	case GitHub:
		return newGitHub(base, token)
	case Forgejo:
		api := restAPI{base: base}
		if token != "" {
			api.header, api.value = "Authorization", "token "+token
		}
		return &forgejo{api: api}, nil
	case GitLab:
		return &gitlab{api: restAPI{base: base, header: "PRIVATE-TOKEN", value: token}}, nil
	}
	return nil, fmt.Errorf("unknown forge kind %q: use %s", kind, strings.Join(Kinds, ", "))
}

// SplitFullName splits "owner/name" at the last slash, so a nested GitLab
// group stays whole in the owner.
func SplitFullName(fullName string) (owner, name string, ok bool) {
	i := strings.LastIndex(fullName, "/")
	if i <= 0 || i == len(fullName)-1 {
		return "", "", false
	}
	return fullName[:i], fullName[i+1:], true
}
