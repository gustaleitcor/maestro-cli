package tui

import (
	"context"

	"github.com/charmbracelet/bubbles/spinner"
	"github.com/charmbracelet/bubbles/table"
	tea "github.com/charmbracelet/bubbletea"

	"maestro-cli/internal/forge"
	"maestro-cli/internal/maestroapi"
)

type screen int

const (
	meScreen screen = iota
	repoListScreen
	imageListScreen
)

type model struct {
	screen  screen
	ctx     context.Context
	client  forge.Forge
	spinner spinner.Model
	loading bool
	err     error

	maestroKey string

	user   *forge.User
	repos  []forge.Repo
	images []maestroapi.Image
	table  table.Model

	width, height int
}

func newModel(ctx context.Context, client forge.Forge, s screen) model {
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

func RunMe(ctx context.Context, client forge.Forge) error {
	m := newModel(ctx, client, meScreen)
	_, err := tea.NewProgram(m, tea.WithAltScreen()).Run()
	return err
}

func RunRepoList(ctx context.Context, client forge.Forge) error {
	m := newModel(ctx, client, repoListScreen)
	_, err := tea.NewProgram(m, tea.WithAltScreen()).Run()
	return err
}

func RunImageList(ctx context.Context, maestroKey string) error {
	m := newModel(ctx, nil, imageListScreen)
	m.maestroKey = maestroKey
	_, err := tea.NewProgram(m, tea.WithAltScreen()).Run()
	return err
}
