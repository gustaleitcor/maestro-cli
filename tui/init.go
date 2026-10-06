package tui

import (
	"context"

	tea "github.com/charmbracelet/bubbletea"

	"maestro-cli/internal/forge"
	"maestro-cli/internal/maestroapi"
)

// forge is the index of the forge that answered: by the time it does, the
// list may be showing another one.
type reposLoadedMsg struct {
	forge int
	repos []forge.Repo
}

type reposErrMsg struct {
	forge int
	err   error
}

type imagesLoadedMsg struct {
	images []maestroapi.Image
	limit  int
}

type fetchErrMsg struct{ err error }

func (m model) Init() tea.Cmd {
	switch m.screen {
	case repoListScreen:
		return tea.Batch(m.spinner.Tick, fetchRepos(m.ctx, m.current, m.forges[m.current].Client))
	case imageListScreen:
		return tea.Batch(m.spinner.Tick, fetchImages(m.ctx, m.maestroKey))
	}
	return nil
}

func fetchRepos(ctx context.Context, index int, client forge.Forge) tea.Cmd {
	return func() tea.Msg {
		repos, err := client.ListRepos(ctx)
		if err != nil {
			return reposErrMsg{index, err}
		}
		return reposLoadedMsg{index, repos}
	}
}

func fetchImages(ctx context.Context, maestroKey string) tea.Cmd {
	return func() tea.Msg {
		images, err := maestroapi.ListImages(ctx, maestroKey)
		if err != nil {
			return fetchErrMsg{err}
		}
		// Best effort: a server from before the limit has no settings.
		settings, err := maestroapi.GetSettings(ctx, maestroKey)
		if err != nil {
			return imagesLoadedMsg{images: images}
		}
		return imagesLoadedMsg{images: images, limit: settings.ImagesPerUser}
	}
}
