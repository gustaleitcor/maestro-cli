# Runs and lines

A run starts one container for each line of a build. Each line runs once, on
the first machine it is allowed on that has room for it.

```sh
maestro run 12                  # start a run of build #12
maestro run 12 --watch          # start it and follow it to the end
maestro runs                    # your newest runs
maestro runs show 31            # the lines of run 31
maestro runs logs 31 2          # what line 2 printed
maestro runs get 31             # download the outputs and logs
maestro runs cancel 31          # stop it
maestro runs cancel 31 --line 2 # stop only line 2
maestro runs retry 31           # run its failed lines again
maestro runs rm 31              # delete it and its files
```

## Listing your runs

```text
RUN  REPO                                BUILD  STATUS     LINES                CREATED
33   ada/maestro-template@main           #14    running    3 queued, 2 running  54m ago
32   ada/maestro-template@main           #13    succeeded  1 succeeded          1h ago
31   gustaleitcor/maestro-template@main  #12    succeeded  2 succeeded          2h ago
```

`maestro runs` shows your 20 newest runs. `--limit` changes the number, and
0 means all of them. `--status` and `--repo` narrow the list. An
administrator sees everyone's runs with `--all`.

## Looking at one run

```text
Run 33 of ada/maestro-template@main (build #14): failed, on any machine
2 line(s): 1 succeeded, 1 failed

LINE  PARAMETERS      MACHINE  STATUS           TIME  FILES
1     -t 10 out.txt   Q1       succeeded        2h    out.txt
2     -t ten out.txt  Q2       failed (exit 2)  2h    out.txt (missing)
```

`TIME` is the longest each line may run, from its
[`time`](../repo/maestro-toml.md#time) or the server's limit. It is `-` when
there is no limit.

`maestro runs show` with no number shows your newest run. `--failed` leaves
only the lines that failed, ended in an error, or were cancelled. A run made
by `maestro runs retry` says which run it retries, as in
`Run 34 of ada/maestro-template@main (build #14, a retry of run 33)`.

## Following a run

`--watch` redraws the output in place every 2 seconds, or as often as
`--interval` says. It works on `maestro run`, `maestro runs show` and
`maestro runs list`.

`maestro runs show <run> --watch` returns when the run is over. It says how
the run ended and exits with status 1 if the run failed or was cancelled, so
a script can wait on it. Ctrl-C stops the watching and leaves the run alone.
When the output goes to a pipe, `--watch` prints a new copy only when
something changed.

## Reading logs

`maestro runs logs <run> <line>` prints what the line wrote to standard
output on your standard output, and what it wrote to standard error on your
standard error. Redirect one away to read the other:

```sh
maestro runs logs 31 2 2>/dev/null     # standard output only
```

A run with a single line doesn't need the line number.

`-f` follows the line the way `tail -f` follows a file. It waits for a line
that hasn't started, prints what a running line prints as it comes, and
returns when the line ends. A server older than `-f` can't follow. The CLI
then prints what the line had printed so far and says so.

For a line that has ended, all of its standard output comes before all of
its standard error. Maestro stores the two apart and doesn't record how they
were interleaved.

## Downloading

`maestro runs get <run>` downloads every line's outputs and logs into a
directory named after the run. [What a line leaves behind](results.md)
describes the layout. `--line` picks lines, a second argument picks the
directory, and `--force` downloads over a run that is already there.

## Running lines again

`maestro runs retry <run>` queues a new run of the same build with some of
the run's lines. Each line runs with the arguments, cores, memory,
environment and outputs it had. It keeps its number too, so line 2 of the
new run is line 2 of the old one, and the new run's lines can skip numbers.

```sh
maestro runs retry 33                   # the lines that didn't succeed
maestro runs retry 33 --line 2 --watch  # line 2, whatever its status
```

Without `--line`, it takes the lines that failed, ended in an error or were
cancelled, as `--failed` does. With `--line`, again for more, it takes
exactly the lines you name, including one that succeeded. A line still
waiting or running can't run again, and the command says which status it
has. Like `maestro run`, it takes `--watch`, `-q` and `--json`.

The new run needs the build's image. When the image is gone, build the
commit again with `maestro build`.

On the Maestro page, a line's details have a "Run again" button that does
the same for that one line.

## Cancelling and deleting

`maestro runs cancel <run>` drops the lines that haven't started and stops
the ones that are running. Lines that had already ended keep their status.

With `--line`, again for more, it cancels only the lines you name and leaves
the rest of the run going. A line still waiting ends as `cancelled` at once.
A running line stops soon, and Maestro keeps its logs and outputs. A line
that is already over can't be cancelled, and the command says how it ended.
A run whose lines you cancelled this way ends as its other lines do, so it
can still end `succeeded`. On the Maestro page, a line's details have a
"Cancel" button that does the same for that line.

`maestro runs rm <run>` deletes the run with all its logs and files. It asks
first.

## How a line ends

| Status | What happened |
| --- | --- |
| `succeeded` | The program exited with 0. |
| `failed` | The program exited with another code, or the kernel killed it for using more memory than the line asked for. |
| `error` | The line couldn't run. Most often it couldn't start on its machine, or the machine restarted while it ran and the container was lost. |
| `cancelled` | You cancelled the line or its run before the line ended. |

A line runs for at most a week, unless your administrator set another
length. When a line runs longer, Maestro stops it, keeps its logs and
outputs, and ends it as `error` with the note "stopped after 168 hours, the
longest a line may run on this server". A line can ask for less with
[`time`](../repo/maestro-toml.md#time) in `maestro.toml`. Past that time,
Maestro stops it the same way, with the note "stopped after 2h, the time this line
asked for".

Any other `error` comes from the system and says nothing about your
program. The Maestro page shows such lines in amber with the reason, and
`maestro runs retry` runs them again.

A line whose machine restarts or is removed while it runs, or whose
container disappears, ends as `error` too. With
[`retries`](../repo/maestro-toml.md#retries) in `maestro.toml`, it goes back
to `queued` instead, as many times as `retries` says, and starts again from
the beginning. Its details on the Maestro page say how many attempts it
took. A line that exits with a code, that you cancel, or that runs out of
time never goes back.

A machine that stops answering doesn't turn its waiting lines into errors.
Maestro moves them back to `queued` so another machine can take them. Lines
already running on that machine are left alone and checked again when it
returns.
