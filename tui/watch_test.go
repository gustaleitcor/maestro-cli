package tui

import (
	"bytes"
	"context"
	"errors"
	"fmt"
	"io"
	"strings"
	"testing"
	"time"

	"github.com/charmbracelet/x/ansi"
)

// frames gives a render that writes each of its frames in turn, the last
// one being the one it is done at. A frame starting with "!" is a failure.
func frames(list ...string) (func(context.Context, io.Writer) (bool, error), *int) {
	calls := 0
	return func(_ context.Context, w io.Writer) (bool, error) {
		frame := list[min(calls, len(list)-1)]
		calls++
		if strings.HasPrefix(frame, "!") {
			return false, errors.New(frame[1:])
		}
		fmt.Fprint(w, frame)
		return calls >= len(list), nil
	}, &calls
}

func testWatcher(out, errOut *bytes.Buffer, width, height int) watcher {
	w := watcher{out: out, errOut: errOut, interval: time.Millisecond, now: func() time.Time { return time.Date(2026, 1, 2, 15, 4, 5, 0, time.UTC) }}
	if width > 0 {
		w.size = func() (int, int) { return width, height }
	}
	return w
}

func TestWatchRedrawsInPlaceUntilDone(t *testing.T) {
	var out, errOut bytes.Buffer
	render, calls := frames("RUN  STATUS\n1    queued\n", "RUN  STATUS\n1    running\n", "RUN  STATUS\n1    succeeded\n")
	if err := testWatcher(&out, &errOut, 80, 24).run(context.Background(), render); err != nil {
		t.Fatal(err)
	}
	if *calls != 3 {
		t.Errorf("rendered %d times, want 3", *calls)
	}

	got := out.String()
	if !strings.HasPrefix(got, hideCursor+clearScreen) || !strings.HasSuffix(got, showCursor) {
		t.Errorf("doesn't start on a cleared screen with the cursor hidden, or doesn't give the cursor back: %q", got)
	}
	// The first frame clears the screen; the others only go back to its top.
	if n := strings.Count(got, clearScreen); n != 1 {
		t.Errorf("cleared the screen %d times", n)
	}
	if n := strings.Count(got, homeCursor); n != 3 {
		t.Errorf("went to the top %d times, want 3", n)
	}
	if strings.Contains(got, "\x1b[?1049h") {
		t.Error("switched to the other screen")
	}
	if n := strings.Count(got, clearBelow); n != 3 {
		t.Errorf("cleared below what it drew %d times, want 3", n)
	}
	for _, line := range strings.Split(strings.TrimSuffix(strings.TrimSuffix(got, showCursor), clearBelow), "\n") {
		if line != "" && !strings.HasSuffix(line, clearLine) {
			t.Errorf("a line isn't cleared to its end: %q", line)
		}
	}

	plain := ansi.Strip(got)
	for _, want := range []string{"every 1ms · updated 15:04:05 · ctrl-c to stop", "1    queued", "1    running", "1    succeeded", "updated 15:04:05 · done"} {
		if !strings.Contains(plain, want) {
			t.Errorf("lacks %q in %q", want, plain)
		}
	}
}

func TestWatchFitsTheTerminal(t *testing.T) {
	var out, errOut bytes.Buffer
	render, _ := frames(strings.Repeat("a line much wider than the terminal is\n", 30))
	if err := testWatcher(&out, &errOut, 20, 6).run(context.Background(), render); err != nil {
		t.Fatal(err)
	}
	lines := strings.Split(strings.TrimRight(ansi.Strip(out.String()), "\n"), "\n")
	if len(lines) != 5 {
		t.Errorf("drew %d lines in a terminal of 6, want 5 and the last left free", len(lines))
	}
	for _, line := range lines {
		if ansi.StringWidth(line) > 20 {
			t.Errorf("%q is wider than 20", line)
		}
	}
}

func TestWatchKeepsWhatItShowedWhenAnUpdateFails(t *testing.T) {
	var out, errOut bytes.Buffer
	render, calls := frames("first\n", "!maestro-orq is\nunreachable", "second\n")
	if err := testWatcher(&out, &errOut, 80, 24).run(context.Background(), render); err != nil {
		t.Fatal(err)
	}
	if *calls != 3 {
		t.Errorf("rendered %d times, want 3", *calls)
	}
	shown := strings.Split(ansi.Strip(out.String()), "\x1b")
	plain := strings.Join(shown, "")
	failed := strings.Index(plain, "update failed: maestro-orq is unreachable · showing 15:04:05 · ctrl-c to stop")
	if failed < 0 {
		t.Fatalf("doesn't say an update failed: %q", plain)
	}
	if after := plain[failed:]; !strings.HasPrefix(after[strings.Index(after, "\n")+1:], "first\n") {
		t.Errorf("the failed update didn't keep what was shown: %q", after)
	}
}

func TestWatchReturnsAFirstFailure(t *testing.T) {
	var out, errOut bytes.Buffer
	render, calls := frames("!no run 9")
	err := testWatcher(&out, &errOut, 80, 24).run(context.Background(), render)
	if err == nil || err.Error() != "no run 9" || *calls != 1 {
		t.Errorf("err = %v after %d renders", err, *calls)
	}
	if strings.Contains(out.String(), clearScreen) {
		t.Error("cleared the screen for nothing")
	}
}

func TestWatchStopsWhenTheContextEnds(t *testing.T) {
	var out, errOut bytes.Buffer
	ctx, cancel := context.WithCancel(context.Background())
	calls := 0
	err := testWatcher(&out, &errOut, 80, 24).run(ctx, func(_ context.Context, w io.Writer) (bool, error) {
		if calls++; calls == 2 {
			cancel()
			return false, context.Canceled
		}
		fmt.Fprintln(w, "still going")
		return false, nil
	})
	if err != nil || calls != 2 {
		t.Errorf("err = %v after %d renders; ctrl-c is not a failure", err, calls)
	}
	if strings.Contains(out.String(), "update failed") {
		t.Error("being stopped was drawn as a failed update")
	}
}

// Into a pipe there is no screen to redraw: a copy, and another only when
// something changed.
func TestWatchIntoAPipePrintsOnlyWhatChanged(t *testing.T) {
	var out, errOut bytes.Buffer
	render, calls := frames("queued\n", "queued\n", "!orq is away", "!orq is away", "queued\n", "running\n", "running\n", "succeeded\n")
	if err := testWatcher(&out, &errOut, 0, 0).run(context.Background(), render); err != nil {
		t.Fatal(err)
	}
	if *calls != 8 {
		t.Errorf("rendered %d times, want 8", *calls)
	}
	if got, want := out.String(), "queued\n\nrunning\n\nsucceeded\n"; got != want {
		t.Errorf("printed %q, want %q", got, want)
	}
	if got := ansi.Strip(errOut.String()); got != "update failed: orq is away\n" {
		t.Errorf("stderr = %q", got)
	}
}
