package tui

import (
	"context"

	"github.com/charmbracelet/bubbles/spinner"
	"github.com/charmbracelet/bubbles/table"
	tea "github.com/charmbracelet/bubbletea"

	"maestro-cli/internal/forge"
)

type RepoForge struct {
	Name   string
	Client forge.Forge
}

type model struct {
	ctx     context.Context
	spinner spinner.Model
	loading bool
	err     error

	// The repo list switches between forges; current indexes forges, and
	// repoCache keeps what each one already answered.
	forges    []RepoForge
	current   int
	repoCache map[int][]forge.Repo

	repos []forge.Repo
	table table.Model

	width, height int
}

func newModel(ctx context.Context) model {
	sp := spinner.New()
	sp.Spinner = spinner.Dot

	return model{
		ctx:     ctx,
		spinner: sp,
		loading: true,
	}
}

func RunRepoList(ctx context.Context, forges []RepoForge, start int) error {
	m := newModel(ctx)
	m.forges = forges
	m.current = start
	m.repoCache = map[int][]forge.Repo{}
	_, err := tea.NewProgram(m, tea.WithAltScreen()).Run()
	return err
}
