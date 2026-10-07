package tui

import (
	"github.com/charmbracelet/bubbles/spinner"
	tea "github.com/charmbracelet/bubbletea"

	"maestro-cli/internal/forge"
)

func (m model) Update(msg tea.Msg) (tea.Model, tea.Cmd) {
	switch msg := msg.(type) {
	case tea.KeyMsg:
		switch msg.String() {
		case "q", "esc", "ctrl+c":
			return m, tea.Quit
		case "left", "right":
			if len(m.forges) > 1 {
				step := 1
				if msg.String() == "left" {
					step = len(m.forges) - 1
				}
				return m.showForge((m.current + step) % len(m.forges))
			}
		}

	case tea.WindowSizeMsg:
		m.width, m.height = msg.Width, msg.Height
		if m.repos != nil {
			m.table = buildRepoTable(m.repos, m.width, m.height)
		}
		return m, nil

	case reposLoadedMsg:
		repos := msg.repos
		if repos == nil {
			repos = []forge.Repo{} // nil would read as not loaded yet
		}
		m.repoCache[msg.forge] = repos
		if msg.forge == m.current {
			m.loading = false
			m.repos = repos
			m.table = buildRepoTable(m.repos, m.width, m.height)
		}
		return m, nil

	case reposErrMsg:
		if msg.forge == m.current {
			m.loading = false
			m.err = msg.err
		}
		return m, nil

	case spinner.TickMsg:
		if !m.loading {
			return m, nil
		}
		var cmd tea.Cmd
		m.spinner, cmd = m.spinner.Update(msg)
		return m, cmd
	}

	if !m.loading {
		var cmd tea.Cmd
		m.table, cmd = m.table.Update(msg)
		return m, cmd
	}

	return m, nil
}

func (m model) showForge(index int) (tea.Model, tea.Cmd) {
	m.current = index
	m.err = nil
	if repos, ok := m.repoCache[index]; ok {
		m.loading = false
		m.repos = repos
		m.table = buildRepoTable(m.repos, m.width, m.height)
		return m, nil
	}
	m.loading = true
	m.repos = nil
	return m, tea.Batch(m.spinner.Tick, fetchRepos(m.ctx, index, m.forges[index].Client))
}
