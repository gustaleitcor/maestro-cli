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
	repoListScreen screen = iota
	imageListScreen
)

// RepoForge is one forge the repo list can show, under its configured name.
type RepoForge struct {
	Name   string
	Client forge.Forge
}

type model struct {
	screen  screen
	ctx     context.Context
	spinner spinner.Model
	loading bool
	err     error

	// The repo list switches between forges; current indexes forges, and
	// repoCache keeps what each one already answered.
	forges    []RepoForge
	current   int
	repoCache map[int][]forge.Repo

	maestroKey string

	repos  []forge.Repo
	images []maestroapi.Image
	table  table.Model

	width, height int
}

func newModel(ctx context.Context, s screen) model {
	sp := spinner.New()
	sp.Spinner = spinner.Dot

	return model{
		screen:  s,
		ctx:     ctx,
		spinner: sp,
		loading: true,
	}
}

// RunRepoList shows the repos of forges[start]; the left and right arrows
// move to the other forges.
func RunRepoList(ctx context.Context, forges []RepoForge, start int) error {
	m := newModel(ctx, repoListScreen)
	m.forges = forges
	m.current = start
	m.repoCache = map[int][]forge.Repo{}
	_, err := tea.NewProgram(m, tea.WithAltScreen()).Run()
	return err
}

func RunImageList(ctx context.Context, maestroKey string) error {
	m := newModel(ctx, imageListScreen)
	m.maestroKey = maestroKey
	_, err := tea.NewProgram(m, tea.WithAltScreen()).Run()
	return err
}
