package tui

import (
	"context"

	tea "github.com/charmbracelet/bubbletea"
	"github.com/google/go-github/v66/github"

	"maestro-cli/internal/githubapi"
)

type userLoadedMsg struct{ user *githubapi.User }
type reposLoadedMsg struct{ repos []githubapi.Repo }
type fetchErrMsg struct{ err error }

func (m model) Init() tea.Cmd {
	switch m.screen {
	case meScreen:
		return tea.Batch(m.spinner.Tick, fetchUser(m.ctx, m.client))
	case repoListScreen:
		return tea.Batch(m.spinner.Tick, fetchRepos(m.ctx, m.client))
	}
	return nil
}

func fetchUser(ctx context.Context, client *github.Client) tea.Cmd {
	return func() tea.Msg {
		u, err := githubapi.GetAuthenticatedUser(ctx, client)
		if err != nil {
			return fetchErrMsg{err}
		}
		return userLoadedMsg{u}
	}
}

func fetchRepos(ctx context.Context, client *github.Client) tea.Cmd {
	return func() tea.Msg {
		repos, err := githubapi.ListRepos(ctx, client)
		if err != nil {
			return fetchErrMsg{err}
		}
		return reposLoadedMsg{repos}
	}
}
