package cmd

import (
	"os"

	"github.com/charmbracelet/lipgloss"
)

// Colors for the plain, line-by-line output. lipgloss drops them when the
// output isn't a terminal or NO_COLOR is set.
var (
	successStyle = lipgloss.NewStyle().Foreground(lipgloss.Color("42"))
	warnStyle    = lipgloss.NewStyle().Foreground(lipgloss.Color("214"))
	accentStyle  = lipgloss.NewStyle().Bold(true).Foreground(lipgloss.Color("39"))
	dimStyle     = lipgloss.NewStyle().Foreground(lipgloss.Color("241"))

	// Errors go to stderr, which may be a terminal when stdout isn't.
	errorStyle = lipgloss.NewRenderer(os.Stderr).NewStyle().Bold(true).Foreground(lipgloss.Color("196"))
)
