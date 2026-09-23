package tui

import (
	"fmt"
	"strconv"
	"strings"

	"github.com/charmbracelet/bubbles/table"
	"github.com/charmbracelet/lipgloss"

	"maestro-cli/internal/githubapi"
)

var (
	labelStyle = lipgloss.NewStyle().Bold(true).Foreground(lipgloss.Color("241")).Width(11)
	helpStyle  = lipgloss.NewStyle().Foreground(lipgloss.Color("241"))
	errorStyle = lipgloss.NewStyle().Foreground(lipgloss.Color("196")).Bold(true)
	cardStyle  = lipgloss.NewStyle().Padding(1, 2).Border(lipgloss.RoundedBorder()).BorderForeground(lipgloss.Color("62"))

	headerStyle = lipgloss.NewStyle().Bold(true).Padding(0, 1).
			BorderStyle(lipgloss.NormalBorder()).BorderBottom(true).BorderForeground(lipgloss.Color("240"))
	footerStyle = lipgloss.NewStyle().Foreground(lipgloss.Color("241")).Padding(0, 1).
			BorderStyle(lipgloss.NormalBorder()).BorderTop(true).BorderForeground(lipgloss.Color("240"))
)

func (m model) View() string {
	if m.err != nil {
		return errorStyle.Render(fmt.Sprintf("Error: %v", m.err)) + "\n" + helpStyle.Render("press q to quit") + "\n"
	}

	if m.loading {
		return fmt.Sprintf("\n  %s Loading...\n", m.spinner.View())
	}

	switch m.screen {
	case meScreen:
		return renderUser(m.user) + "\n" + helpStyle.Render("press q to quit") + "\n"
	case repoListScreen:
		header := bar(headerStyle, m.width, fmt.Sprintf("Repositories (%d)", len(m.repos)))
		footer := bar(footerStyle, m.width, "↑/↓ navigate · q to quit")
		return header + "\n" + m.table.View() + "\n" + footer
	}
	return ""
}

func bar(s lipgloss.Style, width int, text string) string {
	if width > 0 {
		s = s.Width(width - s.GetHorizontalFrameSize())
	}
	return s.Render(text)
}

func renderUser(u *githubapi.User) string {
	var b strings.Builder
	fmt.Fprintln(&b, labelStyle.Render("Login:")+" "+u.Login)
	if u.Name != "" {
		fmt.Fprintln(&b, labelStyle.Render("Name:")+" "+u.Name)
	}
	if u.Bio != "" {
		fmt.Fprintln(&b, labelStyle.Render("Bio:")+" "+u.Bio)
	}
	fmt.Fprintln(&b, labelStyle.Render("Repos:")+fmt.Sprintf(" %d public", u.PublicRepos))
	fmt.Fprintln(&b, labelStyle.Render("Followers:")+fmt.Sprintf(" %d · Following: %d", u.Followers, u.Following))
	fmt.Fprint(&b, labelStyle.Render("Profile:")+" "+u.HTMLURL)
	return cardStyle.Render(b.String())
}

const cellPad = 2 // bubbles/table's Cell/Header padding on each side of every column

const tableChromeLines = 6 // page header + footer bars, plus their joining newlines
const tableHeaderLines = 2 // the table's own column-header row plus its border

func buildRepoTable(repos []githubapi.Repo, width, height int) table.Model {
	if width <= 0 {
		width = 80
	}

	const numCols = 5
	visW, langW, starsW, updW := 10, 14, 7, 10
	fixed := visW + langW + starsW + updW

	nameW := width - fixed - cellPad*numCols
	if nameW < 8 {
		nameW = 8
	}
	totalWidth := nameW + fixed + cellPad*numCols

	columns := []table.Column{
		{Title: "NAME", Width: nameW},
		{Title: "VISIBILITY", Width: visW},
		{Title: "LANGUAGE", Width: langW},
		{Title: "STARS", Width: starsW},
		{Title: "UPDATED", Width: updW},
	}

	rows := make([]table.Row, 0, len(repos))
	for _, r := range repos {
		visibility := "public"
		if r.Private {
			visibility = "private"
		}
		lang := r.Language
		if lang == "" {
			lang = "-"
		}
		rows = append(rows, table.Row{
			r.FullName,
			visibility,
			lang,
			strconv.Itoa(r.Stars),
			r.UpdatedAt.Format("2006-01-02"),
		})
	}

	maxTableHeight := 20 + tableHeaderLines
	if height > 0 {
		maxTableHeight = height - tableChromeLines
		if maxTableHeight < tableHeaderLines+1 {
			maxTableHeight = tableHeaderLines + 1
		}
	}
	tableHeight := min(len(rows)+tableHeaderLines, maxTableHeight)

	t := table.New(
		table.WithColumns(columns),
		table.WithRows(rows),
		table.WithFocused(true),
		table.WithHeight(tableHeight),
		table.WithWidth(totalWidth),
	)

	s := table.DefaultStyles()
	s.Header = s.Header.Bold(true).BorderStyle(lipgloss.NormalBorder()).BorderBottom(true)
	s.Selected = s.Selected.Foreground(lipgloss.Color("229")).Background(lipgloss.Color("57")).Bold(true)
	t.SetStyles(s)

	return t
}
