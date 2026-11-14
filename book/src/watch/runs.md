# Runs and lines

```sh
maestro run 12                  # run build #12's image: run 31
maestro run 12 --watch          # and follow it until it is over
maestro runs                    # your newest 20 runs
maestro runs show 31            # its lines: args, where, how each ended
maestro runs logs 31 2          # what line 2 printed
maestro runs get 31             # download every line's outputs and logs
maestro runs cancel 31          # stop it
maestro runs rm 31              # delete it and its files
```

`maestro run <build>` queues one container per line, spread over the
machines: each line runs once, on whichever allowed machine has room for it
first. It prints the run's number; `-q` prints only that, for scripts.

## Following a run

`maestro run <build> --watch`, `maestro runs show <run> --watch` and
`maestro runs list --watch` draw themselves again every 2 seconds
(`--interval`) in place, without taking over the terminal. `show --watch`
ends by itself when the run does, says on its status line how it ended, and
exits 1 if it failed; ctrl-c ends either. Piped somewhere, they print a new
copy only when something changed. `--json` prints once, so it can't be used
with `--watch`.

`maestro build <repo> --watch` goes from a repo to its results in one
command: it builds, runs, and follows the run.

## Finding runs and lines

- `maestro runs list` shows your newest 20 runs (`--limit`, 0 for all), of a
  `--status` or a `--repo` if asked, and how long ago each was created.
  `--all` gives an administrator everyone's, with whose each is.
- `maestro runs show` without a run shows your newest. `--failed` shows only
  the lines that failed, errored or were cancelled.

## Logs

`maestro runs logs <run> <line>` prints the line's stdout to stdout and its
stderr to stderr, so `2>/dev/null` or `>/dev/null` leaves just one. The line
may be left out of a run of one.

`-f` follows a line as `tail -f` does: one still waiting is waited for, and
what a running line prints comes as it is printed, until the line ends.

Once a line is over, all of its stdout comes before all of its stderr: how
the two were mixed as the container ran isn't kept.

## Downloading

`maestro runs get <run>` downloads every line's outputs and logs into
`./<run>/`, laid out as described in [What a line leaves
behind](results.md). `--line` downloads some lines only, and a run already
downloaded is downloaded over only with `--force`.

## Cancelling and removing

`maestro runs cancel <run>` stops the lines still waiting or running; those
already over keep how they ended. `maestro runs rm <run>` deletes the run and
everything it kept, and asks first (`--yes` without a terminal).

## How a line ends

| Status | |
| --- | --- |
| `succeeded` | its program exited 0 |
| `failed` | its program exited with an error, or was killed for taking more memory than it asked for |
| `error` | it couldn't run: most often it couldn't start on its machine, or its machine restarted while it ran and lost its container |
| `cancelled` | its run was cancelled before it ended |

A line that ends `error` is the server's trouble, not your program's; the
Maestro page draws it apart, in amber, and says why.

A machine the server can't reach gets no lines until it can: lines waiting
or about to start there go back to `staged`, for another machine, rather
than ending in an error. Lines already running there are left alone, and
looked at again once it is back.
