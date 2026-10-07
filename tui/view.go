package tui

import (
	"fmt"
	"strconv"

	"github.com/charmbracelet/bubbles/table"
	"github.com/charmbracelet/lipgloss"

	"maestro-cli/internal/forge"
)

var (
	labelStyle = lipgloss.NewStyle().Bold(true).Foreground(lipgloss.Color("241")).Width(9)
	helpStyle  = lipgloss.NewStyle().Foreground(lipgloss.Color("241"))
	warnStyle  = lipgloss.NewStyle().Foreground(lipgloss.Color("214"))
	errorStyle = lipgloss.NewStyle().Foreground(lipgloss.Color("196")).Bold(true)
	logoStyle  = lipgloss.NewStyle().Bold(true).Foreground(lipgloss.Color("62"))

	currentForgeStyle = lipgloss.NewStyle().Bold(true).Foreground(lipgloss.Color("229")).Background(lipgloss.Color("57"))
	otherForgeStyle   = lipgloss.NewStyle().Bold(false).Foreground(lipgloss.Color("241"))

	headerStyle = lipgloss.NewStyle().Bold(true).Padding(0, 1).
			BorderStyle(lipgloss.NormalBorder()).BorderBottom(true).BorderForeground(lipgloss.Color("240"))
	footerStyle = lipgloss.NewStyle().Foreground(lipgloss.Color("241")).Padding(0, 1).
			BorderStyle(lipgloss.NormalBorder()).BorderTop(true).BorderForeground(lipgloss.Color("240"))
)

const cellPad = 2 // bubbles/table's Cell/Header padding on each side of every column

const tableChromeLines = 6 // page header + footer bars, plus their joining newlines

const tableHeaderLines = 2 // the table's own column-header row plus its border

func (m model) View() string {
	return m.repoListView()
}

// repoListView keeps the header and footer up while loading or failing, so
// the forge being shown, and the way to another one, are always in sight.
func (m model) repoListView() string {
	title := "Repositories on " + m.forges[m.current].Client.Host()
	if !m.loading && m.err == nil {
		title += fmt.Sprintf(" (%d)", len(m.repos))
	}
	keys := "↑/↓ navigate · q to quit"
	if len(m.forges) > 1 {
		title += "  "
		for i, f := range m.forges {
			style := otherForgeStyle
			if i == m.current {
				style = currentForgeStyle
			}
			title += style.Render(" " + f.Name + " ")
		}
		keys = "←/→ switch forge · " + keys
	}

	var body string
	switch {
	case m.err != nil:
		body = "\n  " + errorStyle.Render(fmt.Sprintf("Error: %v", m.err)) + "\n"
	case m.loading:
		body = fmt.Sprintf("\n  %s Loading...\n", m.spinner.View())
	default:
		body = m.table.View()
	}
	return bar(headerStyle, m.width, title) + "\n" + body + "\n" + bar(footerStyle, m.width, keys)
}

func bar(s lipgloss.Style, width int, text string) string {
	if width > 0 {
		s = s.Width(width - s.GetHorizontalFrameSize())
	}
	return s.Render(text)
}

func buildRepoTable(repos []forge.Repo, width, height int) table.Model {
	columns := []table.Column{
		{Title: "NAME"},
		{Title: "VISIBILITY", Width: 10},
		{Title: "LANGUAGE", Width: 14},
		{Title: "STARS", Width: 7},
		{Title: "UPDATED", Width: 10},
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

	return newListTable(columns, rows, width, height)
}

// newListTable gives the first column whatever width the others leave over,
// and caps the height so the header and footer bars always stay on screen.
func newListTable(columns []table.Column, rows []table.Row, width, height int) table.Model {
	if width <= 0 {
		width = 80
	}

	fixed := 0
	for _, c := range columns[1:] {
		fixed += c.Width
	}
	columns[0].Width = max(width-fixed-cellPad*len(columns), 8)
	totalWidth := columns[0].Width + fixed + cellPad*len(columns)

	maxTableHeight := 20 + tableHeaderLines
	if height > 0 {
		maxTableHeight = max(height-tableChromeLines, tableHeaderLines+1)
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
