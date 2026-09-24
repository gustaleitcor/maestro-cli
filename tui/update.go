package tui

import (
	"github.com/charmbracelet/bubbles/spinner"
	tea "github.com/charmbracelet/bubbletea"
)

func (m model) Update(msg tea.Msg) (tea.Model, tea.Cmd) {
	switch msg := msg.(type) {
	case tea.KeyMsg:
		switch msg.String() {
		case "q", "esc", "ctrl+c":
			return m, tea.Quit
		}

	case tea.WindowSizeMsg:
		m.width, m.height = msg.Width, msg.Height
		switch {
		case m.screen == repoListScreen && m.repos != nil:
			m.table = buildRepoTable(m.repos, m.width, m.height)
		case m.screen == imageListScreen && m.images != nil:
			m.table = buildImageTable(m.images, m.width, m.height)
		}
		return m, nil

	case userLoadedMsg:
		m.loading = false
		m.user = msg.user
		return m, nil

	case reposLoadedMsg:
		m.loading = false
		m.repos = msg.repos
		m.table = buildRepoTable(m.repos, m.width, m.height)
		return m, nil

	case imagesLoadedMsg:
		m.loading = false
		m.images = msg.images
		m.table = buildImageTable(m.images, m.width, m.height)
		return m, nil

	case fetchErrMsg:
		m.loading = false
		m.err = msg.err
		return m, nil

	case spinner.TickMsg:
		if !m.loading {
			return m, nil
		}
		var cmd tea.Cmd
		m.spinner, cmd = m.spinner.Update(msg)
		return m, cmd
	}

	if (m.screen == repoListScreen || m.screen == imageListScreen) && !m.loading {
		var cmd tea.Cmd
		m.table, cmd = m.table.Update(msg)
		return m, cmd
	}

	return m, nil
}
