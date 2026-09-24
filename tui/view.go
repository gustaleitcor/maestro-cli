package tui

import (
	"fmt"
	"strconv"
	"strings"

	"github.com/charmbracelet/bubbles/table"
	"github.com/charmbracelet/lipgloss"

	"maestro-cli/internal/githubapi"
	"maestro-cli/internal/maestroapi"
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
	case imageListScreen:
		header := bar(headerStyle, m.width, fmt.Sprintf("Images (%d)", len(m.images)))
		if len(m.images) == 0 {
			footer := bar(footerStyle, m.width, "q to quit")
			return header + "\n\n  No images yet. Build one with: maestro build <repo>\n\n" + footer
		}
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

func buildImageTable(images []maestroapi.Image, width, height int) table.Model {
	columns := []table.Column{
		{Title: "REPO"},
		{Title: "REF", Width: 14},
		{Title: "BUILD", Width: 7},
		{Title: "IMAGE ID", Width: 12},
		{Title: "SIZE", Width: 9},
		{Title: "CREATED", Width: 16},
	}

	rows := make([]table.Row, 0, len(images))
	for _, img := range images {
		repo := img.Repo
		if repo == "" {
			repo = img.Tag // no build record left for it; the tag still says what it is
		}
		ref := img.Ref
		if ref == "" {
			ref = "-"
		}
		build := "-"
		if img.BuildID != 0 {
			build = "#" + strconv.FormatInt(img.BuildID, 10)
		}
		rows = append(rows, table.Row{
			repo,
			ref,
			build,
			shortID(img.ID),
			humanSize(img.Size),
			img.CreatedAt.Local().Format("2006-01-02 15:04"),
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

func shortID(id string) string {
	if len(id) > 12 {
		return id[:12]
	}
	return id
}

// humanSize formats bytes in decimal units, matching `podman images`.
func humanSize(b int64) string {
	const unit = 1000
	if b < unit {
		return fmt.Sprintf("%d B", b)
	}
	div, exp := int64(unit), 0
	for n := b / unit; n >= unit; n /= unit {
		div *= unit
		exp++
	}
	return fmt.Sprintf("%.1f %cB", float64(b)/float64(div), "kMGTPE"[exp])
}
