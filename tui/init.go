package tui

import (
	"context"

	tea "github.com/charmbracelet/bubbletea"

	"maestro-cli/internal/forge"
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

func (m model) Init() tea.Cmd {
	return tea.Batch(m.spinner.Tick, fetchRepos(m.ctx, m.current, m.forges[m.current].Client))
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
