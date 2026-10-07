package tui

import (
	"bytes"
	"context"
	"fmt"
	"io"
	"os"
	"os/signal"
	"strings"
	"time"

	"github.com/charmbracelet/x/ansi"
	"golang.org/x/term"
)

// A watcher draws something again and again where it was first drawn. On a
// terminal it rewrites the screen in place, without switching to another
// one, so what it showed last stays when it ends. Anywhere else it prints a
// new copy only when what it would print changed.
type watcher struct {
	out, errOut io.Writer
	interval    time.Duration
	// size is the terminal's width and height; nil when out isn't one.
	size func() (width, height int)
	now  func() time.Time
}

// Watch runs render every interval until it reports being done or ctrl-c is
// pressed, which is a way of ending it and not a failure. A first render
// that fails is returned; a later one keeps what was last drawn and says so.
func Watch(ctx context.Context, interval time.Duration, render func(context.Context, io.Writer) (done bool, err error)) error {
	ctx, stop := signal.NotifyContext(ctx, os.Interrupt)
	defer stop()

	w := watcher{out: os.Stdout, errOut: os.Stderr, interval: interval, now: time.Now}
	if fd := int(os.Stdout.Fd()); term.IsTerminal(fd) {
		w.size = func() (int, int) {
			width, height, err := term.GetSize(fd)
			if err != nil {
				return 80, 24
			}
			return width, height
		}
	}
	return w.run(ctx, render)
}

const (
	clearScreen = "\x1b[H\x1b[2J"
	homeCursor  = "\x1b[H"
	clearLine   = "\x1b[K"
	clearBelow  = "\x1b[J"
	hideCursor  = "\x1b[?25l"
	showCursor  = "\x1b[?25h"
)

func (w watcher) run(ctx context.Context, render func(context.Context, io.Writer) (bool, error)) error {
	onTerminal := w.size != nil
	if onTerminal {
		fmt.Fprint(w.out, hideCursor)
		defer fmt.Fprint(w.out, showCursor)
	}

	var shown string // the last body drawn
	var shownAt time.Time
	var lastFailure string
	first := true
	for {
		var body bytes.Buffer
		done, err := render(ctx, &body)
		if ctx.Err() != nil {
			return nil
		}
		if err != nil && first {
			return err
		}
		if err == nil {
			changed := body.String() != shown || first
			shown, shownAt, lastFailure = body.String(), w.now(), ""
			if !onTerminal && changed {
				if !first {
					fmt.Fprintln(w.out)
				}
				fmt.Fprintln(w.out, strings.TrimRight(shown, "\n"))
			}
		} else if !onTerminal && err.Error() != lastFailure {
			lastFailure = err.Error()
			fmt.Fprintln(w.errOut, topWarn.Render("update failed: "+lastFailure))
		}
		if onTerminal {
			w.draw(first, w.status(shownAt, done, err), shown)
		}
		first = false
		if done && err == nil {
			return nil
		}

		select {
		case <-ctx.Done():
			return nil
		case <-time.After(w.interval):
		}
	}
}

func (w watcher) status(shownAt time.Time, done bool, err error) string {
	at := shownAt.Format("15:04:05")
	switch {
	case err != nil:
		return topWarn.Render("update failed: "+strings.Join(strings.Fields(err.Error()), " ")) + topDim.Render(fmt.Sprintf(" · showing %s · ctrl-c to stop", at))
	case done:
		return topDim.Render(fmt.Sprintf("updated %s · done", at))
	}
	return topDim.Render(fmt.Sprintf("every %s · updated %s · ctrl-c to stop", w.interval, at))
}

// draw writes the status and as much of the body as fits over what was
// there: each line cleared to its end, and everything below the last. A
// line is cut at the terminal's width, since a wrapped one would push the
// rest down and off the place it is redrawn at.
func (w watcher) draw(first bool, status, body string) {
	width, height := w.size()
	lines := append([]string{status}, strings.Split(strings.TrimRight(body, "\n"), "\n")...)
	// The last row stays free: writing a newline on it would scroll.
	if room := max(height-1, 1); len(lines) > room {
		lines = lines[:room]
	}

	var frame strings.Builder
	if first {
		frame.WriteString(clearScreen)
	} else {
		frame.WriteString(homeCursor)
	}
	for _, line := range lines {
		frame.WriteString(ansi.Truncate(line, width, "…"))
		frame.WriteString(clearLine + "\n")
	}
	frame.WriteString(clearBelow)
	io.WriteString(w.out, frame.String())
}
