package tui

import (
	"context"

	tea "github.com/charmbracelet/bubbletea"

	"maestro-cli/internal/forge"
	"maestro-cli/internal/maestroapi"
)

type userLoadedMsg struct{ user *forge.User }
type reposLoadedMsg struct{ repos []forge.Repo }
type imagesLoadedMsg struct{ images []maestroapi.Image }
type fetchErrMsg struct{ err error }

func (m model) Init() tea.Cmd {
	switch m.screen {
	case meScreen:
		return tea.Batch(m.spinner.Tick, fetchUser(m.ctx, m.client))
	case repoListScreen:
		return tea.Batch(m.spinner.Tick, fetchRepos(m.ctx, m.client))
	case imageListScreen:
		return tea.Batch(m.spinner.Tick, fetchImages(m.ctx, m.maestroKey))
	}
	return nil
}

func fetchUser(ctx context.Context, client forge.Forge) tea.Cmd {
	return func() tea.Msg {
		u, err := client.CurrentUser(ctx)
		if err != nil {
			return fetchErrMsg{err}
		}
		return userLoadedMsg{u}
	}
}

func fetchRepos(ctx context.Context, client forge.Forge) tea.Cmd {
	return func() tea.Msg {
		repos, err := client.ListRepos(ctx)
		if err != nil {
			return fetchErrMsg{err}
		}
		return reposLoadedMsg{repos}
	}
}

func fetchImages(ctx context.Context, maestroKey string) tea.Cmd {
	return func() tea.Msg {
		images, err := maestroapi.ListImages(ctx, maestroKey)
		if err != nil {
			return fetchErrMsg{err}
		}
		return imagesLoadedMsg{images}
	}
}
