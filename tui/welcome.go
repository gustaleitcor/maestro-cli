package tui

import (
	"errors"
	"fmt"
	"strings"

	"github.com/charmbracelet/lipgloss"

	"maestro-cli/internal/netfail"
)

const logo = `███╗   ███╗ █████╗ ███████╗███████╗████████╗██████╗  ██████╗
████╗ ████║██╔══██╗██╔════╝██╔════╝╚══██╔══╝██╔══██╗██╔═══██╗
██╔████╔██║███████║█████╗  ███████╗   ██║   ██████╔╝██║   ██║
██║╚██╔╝██║██╔══██║██╔══╝  ╚════██║   ██║   ██╔══██╗██║   ██║
██║ ╚═╝ ██║██║  ██║███████╗███████║   ██║   ██║  ██║╚██████╔╝
╚═╝     ╚═╝╚═╝  ╚═╝╚══════╝╚══════╝   ╚═╝   ╚═╝  ╚═╝ ╚═════╝`

const logoGap = 3 // columns between the logo and the text beside it

type Welcome struct {
	Server string
	// SignedIn is whether a Maestro key is stored. Email is who it belongs
	// to; AccountErr is set instead when the server wouldn't say.
	SignedIn   bool
	Email      string
	AccountErr error
	Forges     []ForgeIdentity
}

type ForgeIdentity struct {
	Name  string
	Host  string
	Login string
	Err   error
}

// RenderWelcome puts the logo beside the user's info, or above it when the
// terminal is too narrow for both. A width of 0 means unknown.
func RenderWelcome(w Welcome, width int) string {
	art := logoStyle.Render(logo)
	info := renderWelcomeInfo(w)

	if width > 0 && lipgloss.Width(art)+logoGap+lipgloss.Width(info) > width {
		return art + "\n\n" + info + "\n"
	}
	gap := strings.Repeat(" ", logoGap)
	return lipgloss.JoinHorizontal(lipgloss.Center, art, gap, info) + "\n"
}

func renderWelcomeInfo(w Welcome) string {
	if !w.SignedIn && len(w.Forges) == 0 {
		return "Not signed in.\n\n" +
			"maestro login    Sign in to Maestro\n" +
			"maestro --help   More information"
	}

	// What couldn't be reached is said once, below, not on every row.
	var offline []error

	var b strings.Builder
	switch {
	case !w.SignedIn:
		fmt.Fprintln(&b, labelStyle.Render("Account:")+" not signed in; run `maestro login`")
	case errors.Is(w.AccountErr, netfail.ErrUnreachable):
		offline = append(offline, w.AccountErr)
		fmt.Fprintln(&b, labelStyle.Render("Account:")+" signed in, "+warnStyle.Render("server unreachable"))
	case w.AccountErr != nil:
		fmt.Fprintln(&b, labelStyle.Render("Account:")+" "+errorStyle.Render(firstLine(w.AccountErr))+"; run `maestro login`")
	default:
		fmt.Fprintln(&b, labelStyle.Render("Account:")+" "+w.Email)
	}
	fmt.Fprintln(&b, labelStyle.Render("Server:")+" "+w.Server)

	if len(w.Forges) == 0 {
		fmt.Fprintln(&b, labelStyle.Render("Forges:")+" none; run `maestro forges add`")
	}
	for i, f := range w.Forges {
		label := ""
		if i == 0 {
			label = "Forges:"
		}
		who := f.Login + " on " + f.Host
		switch {
		case errors.Is(f.Err, netfail.ErrUnreachable):
			offline = append(offline, f.Err)
			who = warnStyle.Render("unreachable")
		case f.Err != nil:
			who = errorStyle.Render(firstLine(f.Err))
		}
		fmt.Fprintln(&b, labelStyle.Render(label)+" "+f.Name+": "+who)
	}

	switch len(offline) {
	case 0:
	case 1:
		fmt.Fprintln(&b, "\n"+warnStyle.Render("Warning: "+offline[0].Error()))
	default:
		fmt.Fprintln(&b, "\n"+warnStyle.Render("Warning: check your internet connection"))
	}

	fmt.Fprint(&b, "\n"+helpStyle.Render("maestro --help for more information"))
	return b.String()
}

func firstLine(err error) string {
	line, _, _ := strings.Cut(err.Error(), "\n")
	return line
}
