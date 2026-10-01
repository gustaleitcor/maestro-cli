package tui

import (
	"context"
	"strings"
	"testing"

	tea "github.com/charmbracelet/bubbletea"

	"maestro-cli/internal/forge"
)

type fakeForge struct {
	host  string
	repos []forge.Repo
	calls *int
}

func (f fakeForge) Kind() string { return "fake" }
func (f fakeForge) Host() string { return f.host }
func (f fakeForge) CurrentUser(context.Context) (*forge.User, error) {
	return &forge.User{}, nil
}
func (f fakeForge) ListRepos(context.Context) ([]forge.Repo, error) {
	*f.calls++
	return f.repos, nil
}
func (f fakeForge) GetRepo(context.Context, string, string) (*forge.Repo, error) {
	return nil, nil
}
func (f fakeForge) HasFile(context.Context, string, string, string, string) (bool, error) {
	return false, nil
}

// press sends a key and feeds back whatever the fetch it started answers.
func press(t *testing.T, m model, key tea.KeyType) model {
	t.Helper()
	next, _ := m.Update(tea.KeyMsg{Type: key})
	m = next.(model)
	if m.loading {
		next, _ = m.Update(fetchRepos(m.ctx, m.current, m.forges[m.current].Client)())
		m = next.(model)
	}
	return m
}

func TestArrowsSwitchForge(t *testing.T) {
	var calls [3]int
	m := newModel(context.Background(), repoListScreen)
	m.forges = []RepoForge{
		{"github", fakeForge{"github.com", []forge.Repo{{FullName: "me/a"}}, &calls[0]}},
		{"codeberg", fakeForge{"codeberg.org", []forge.Repo{{FullName: "me/b"}, {FullName: "me/c"}}, &calls[1]}},
		{"work", fakeForge{"git.example.com", nil, &calls[2]}},
	}
	m.repoCache = map[int][]forge.Repo{}
	next, _ := m.Update(fetchRepos(m.ctx, 0, m.forges[0].Client)())
	m = next.(model)

	m = press(t, m, tea.KeyRight)
	if m.current != 1 || len(m.repos) != 2 || !strings.Contains(m.View(), "Repositories on codeberg.org (2)") {
		t.Fatalf("right: current=%d repos=%d view=%q", m.current, len(m.repos), m.View())
	}
	m = press(t, m, tea.KeyRight)
	m = press(t, m, tea.KeyRight) // wraps around to the first
	if m.current != 0 || m.repos[0].FullName != "me/a" {
		t.Fatalf("right x3: current=%d repos=%v", m.current, m.repos)
	}
	m = press(t, m, tea.KeyLeft) // and back to the last
	if m.current != 2 || m.loading {
		t.Fatalf("left: current=%d loading=%v", m.current, m.loading)
	}
	// Each forge is only asked once, however often it is shown.
	if calls != [3]int{1, 1, 1} {
		t.Fatalf("ListRepos calls = %v, want one each", calls)
	}

	// An answer from a forge no longer on screen is kept, not shown.
	next, _ = m.Update(reposLoadedMsg{forge: 1, repos: []forge.Repo{{FullName: "me/late"}}})
	m = next.(model)
	if m.current != 2 || len(m.repos) != 0 {
		t.Fatalf("stale answer changed the view: current=%d repos=%v", m.current, m.repos)
	}
}
