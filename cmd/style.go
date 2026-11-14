package cmd

import (
	"os"

	"github.com/charmbracelet/lipgloss"
	"github.com/muesli/termenv"
)

// Colors for the plain, line-by-line output. lipgloss drops them when the
// output isn't a terminal, NO_COLOR is set, or --no-color is passed.
var (
	successStyle = lipgloss.NewStyle().Foreground(lipgloss.Color("42"))
	warnStyle    = lipgloss.NewStyle().Foreground(lipgloss.Color("214"))
	accentStyle  = lipgloss.NewStyle().Bold(true).Foreground(lipgloss.Color("39"))
	dimStyle     = lipgloss.NewStyle().Foreground(lipgloss.Color("241"))

	// Errors go to stderr, which may be a terminal when stdout isn't.
	stderrRenderer = lipgloss.NewRenderer(os.Stderr)
	errorStyle     = stderrRenderer.NewStyle().Bold(true).Foreground(lipgloss.Color("196"))
)

var noColor bool

// applyNoColor drops every colour, on stdout (and the full-screen views,
// which draw there) and on stderr, when --no-color asks it.
func applyNoColor() {
	if noColor {
		lipgloss.SetColorProfile(termenv.Ascii)
		stderrRenderer.SetColorProfile(termenv.Ascii)
	}
}
