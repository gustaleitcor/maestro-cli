# The Maestro page

The Maestro page shows the same runs and files as the CLI, in a browser.
Open your Maestro server's address and [sign in](login.md). The page opens
on My runs.

## The sidebar

The sidebar has My runs, Queue, Usage and Docs, which opens this book.
Administrators also get All users and the admin pages: Machines, Groups and
Settings. On a narrow screen the sidebar shrinks to a rail of icons.

At the foot of the sidebar are the account you signed in with, Sign out, and
the theme toggle. The page follows your system's light or dark theme until
you use the toggle, and the browser then remembers your choice.

The address bar always holds the page you are on. You can send someone a
link to a run, a line or a file, and it opens for anyone allowed to see it.

## My runs

My runs lists your runs, newest first. You can narrow the list to active or
finished runs, and search it by run number or repository. Administrators
find everyone's runs under All users.

Opening a run shows its lines, with the cores and memory each one asked for.
You can sort them by line number, arguments, cores, memory, machine or
number of files kept. Click a column's heading to sort by it, and click
again to reverse the order. The Status column works differently. Each click moves the next
status to the top, starting with the ones that are over (cancelled, then
error, then failed) and going round to queued.

The page draws a line that ended in `error` in amber and says what happened,
because the cause was on the server's side and the line is worth running
again. A `failed` line ran, and its program exited with an error.
[How a line ends](runs.md#how-a-line-ends) lists every status.

Cancelling a run cancels the lines that hadn't ended. A line that had
already ended keeps its status.

## A line

A line's page opens on its files. The Details button shows everything the
line's container ran with:

- its cores and memory
- its command
- its environment, with the run's `env` first and then the `MAESTRO_`
  variables
- the files kept from it
- the name of its container, its image and the commit the image was built
  from
- when it was queued, staged, assigned, started and finished, and how long
  it waited and ran

The page shows files as folders. Paths that share a prefix go under one
folder, and a chain of folders that each hold only the next one shows as a
single row, such as `results/7/1/0.01_0.01`. Click a folder to fold it.

A run whose outputs are [shared](../repo/maestro-toml.md#outputs_layout)
also has a page for the whole tree, with every line's files merged and each
file marked with the line that wrote it.

## Files

Click a file or one of a line's two logs to open it in a pane beside the
list. You can also open it in its own tab or download it. While a line
runs, the page reads its logs again every 5 seconds.

A line, or a whole run, downloads as a zip laid out the way
[`maestro runs get`](results.md#the-layout-of-a-download) leaves it.

The page shows text, PDFs and images (png, jpeg, gif and webp). Any other
file can only be downloaded. Text is highlighted by its extension. Python,
JSON, TOML, YAML, shell, C and C++, Go, Rust, Julia, R, SQL, Markdown, HTML
and XML are among the languages it knows.

Three kinds of file are drawn, with the source one click away:

| File | Shown as |
| --- | --- |
| CSV | A table |
| SVG | An image. None of its scripts run. |
| HTML | The page itself, in a sandbox |

The sandbox runs the HTML page's scripts, so a chart in it draws. It gives
the page an origin of its own, so the page can't read your session, call the
server as you, or navigate the Maestro page away. Any preview can go full
screen. A drawn file opened in its own tab is drawn the same way, sandbox
included.

A program that writes its results as an HTML chart, an SVG or a CSV can
therefore be read on the page without a download.

## Queue and Usage

Queue shows what `maestro queue` prints, and reads it again every 5 seconds:
how many lines wait at each level, yours and everyone's, your group, and
your share of the machines. [The queue](../how/queue.md) explains the
levels.

Usage charts the core-hours and memory GB-hours your lines used each day.
Its "Lines over a period" section downloads the file `maestro export`
writes and measures how the queue served those lines, as
[Usage and export](usage.md) describes.

## Admin pages

Administrators have three more pages.

| Page | What it is for | Chapter |
| --- | --- | --- |
| Machines | Adding and editing machines | [Adding machines](../admin/machines.md) |
| Groups | Putting users in groups | [Groups](../admin/groups.md) |
| Settings | The queue's policies and the server's limits | [Choosing the policies](../admin/policies.md) |

## Approving the CLI

`maestro login` prints a link to the page with a code in it. The page asks
you to approve the code, and the CLI then gets its key. See
[Signing in](login.md#in-the-terminal).
