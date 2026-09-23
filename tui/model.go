// Package tui provides Bubble Tea screens for the Maestro CLI: a "me"
// profile view and a "repo list" table, each fetching its own data via
// internal/githubapi once the program starts.
package tui

import (
	"context"

	"github.com/charmbracelet/bubbles/spinner"
	"github.com/charmbracelet/bubbles/table"
	tea "github.com/charmbracelet/bubbletea"
	"github.com/google/go-github/v66/github"

	"maestro-cli/internal/githubapi"
)

type screen int

const (
	meScreen screen = iota
	repoListScreen
)

type model struct {
	screen  screen
	ctx     context.Context
	client  *github.Client
	spinner spinner.Model
	loading bool
	err     error

	user  *githubapi.User
	repos []githubapi.Repo
	table table.Model

	width, height int
}

func newModel(ctx context.Context, client *github.Client, s screen) model {
	sp := spinner.New()
	sp.Spinner = spinner.Dot

	return model{
		screen:  s,
		ctx:     ctx,
		client:  client,
		spinner: sp,
		loading: true,
	}
}

// RunMe starts a Bubble Tea program that fetches and displays the
// authenticated GitHub user.
func RunMe(ctx context.Context, client *github.Client) error {
	m := newModel(ctx, client, meScreen)
	_, err := tea.NewProgram(m, tea.WithAltScreen()).Run()
	return err
}

// RunRepoList starts a Bubble Tea program that fetches and displays the
// repositories visible to the authenticated user in a scrollable table.
func RunRepoList(ctx context.Context, client *github.Client) error {
	m := newModel(ctx, client, repoListScreen)
	_, err := tea.NewProgram(m, tea.WithAltScreen()).Run()
	return err
}
