# The Maestro page

The Maestro page is the same runs and files as the CLI, in a browser. It
opens on My runs once you have signed in.

## The sidebar

The sidebar leads to **My runs**, **Queue**, **Usage** and **Docs** (this
book), and for administrators to **All users** and the Admin pages
(**Machines**, **Groups**, **Settings**). On a narrow screen it is a rail of
icons. At its foot are who is signed in, Sign out, and the theme: the page
follows the system's light or dark theme until the toggle picks one, which
the browser then keeps.

The page is in the address, so a link to a run, a line or a file opens it
for anyone allowed to see it.

## My runs

Your runs, newest first, narrowed to active or finished ones, or searched by
run or repository. Administrators have everyone's under All users.

A run's lines show the cores and memory each asked for, and can be sorted by
line, args, those, where they are or how many files they kept: click a
column's heading, and again to reverse it. Each click on Status puts the
next status the lines have at the top, from those that are over: cancelled,
then error, failed, and on round to queued.

A line that ends `failed` ran and its program exited with an error; one that
ends `error` is the server's trouble, most often a line that couldn't start
on its machine, or one whose machine restarted while it ran there and so
lost its container, which the page says and draws apart, in amber.
Cancelling a run cancels the lines it stops; one that had already ended
keeps how it ended.

## A line

A line's page opens on its files. Its **Details** button shows all it runs
with, as its container got it: its cores and memory, its command, its
environment (its `env`, then the server's `MAESTRO_` variables), the files
kept from it, its container's name, the image and commit, and when it was
queued, staged, assigned, started and finished, with how long it waited and
ran.

A line's files show as folders, not as a list of long paths: paths that
share a prefix go under a folder of it, a chain of folders that each hold
only the next shows as one (`results/7/1/0.01_0.01`), and a folder folds
with a click. A run whose outputs are shared also has its tree on its own
page, every line's files merged, each saying which line wrote it.

## Files

A file, or either of a line's logs, opens in a pane beside the line's files
(a running line's logs are read again every 5 seconds), or in a tab of its
own, and can be downloaded. A line or a whole run downloads as a zip, laid
out as `maestro runs get` leaves it.

Only png, jpeg, gif and webp images, PDFs and text are shown; the rest can
only be downloaded. Text is highlighted by its extension (Python, JSON,
TOML, YAML, shell, C/C++, Go, Rust, Julia, R, SQL, Markdown, HTML/XML and
more), and some of it is drawn, with its source a click away:

- a CSV as a table;
- an SVG as an image, which runs none of its scripts;
- an HTML page, such as a chart, in a sandbox.

The sandbox runs the page's scripts, so that its charts draw, in an origin
of its own: it can't read your session, call the server as you, or move the
Maestro page. Any preview goes full screen, and a drawn one opens in a tab
of its own drawn the same way, sandbox included.

So a program that writes its results as an HTML chart, an SVG or a CSV can
be read straight from the page.

## Queue and Usage

**Queue** is what `maestro queue` says, read again every 5 seconds: how many
lines wait at each level, yours and everyone's, your group and your share.
See [The queue, explained](../watch/queue.md).

**Usage** is the core-hours and memory GB-hours your lines used each day.
Its "Lines over a period" downloads what `maestro export` writes, and
measures it as `sim evaluate` does, against what the machines have for lines
now. See [Usage and export](../watch/usage.md).

## Admin

For administrators: **Machines** to add and edit them ([Adding
machines](../admin/machines.md)), **Groups** to split users into groups
([Groups](../admin/groups.md)), and **Settings** for the scheduler's
policies and the server's limits ([Scheduler
policies](../admin/policies.md)).

## Approving the CLI

`maestro login` prints a link to the page with a code; the page asks you to
approve it, and the CLI gets its key. See [Signing in](../start/login.md).
