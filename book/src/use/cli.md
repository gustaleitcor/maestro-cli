# How the CLI behaves

A few rules hold for every `maestro` command. Knowing them saves looking up
each command separately.

## Listing and removing

Anything there can be several of has a command in the plural. Given nothing
else, it lists them: `maestro builds`, `maestro runs`, `maestro machines`,
`maestro repos` and `maestro forges`.

Removing is always `rm`, as in `maestro runs rm 31`. It asks before it
removes. A script has no terminal to answer at, so there it needs `--yes`.

The names older versions used still work: `image`, `machine`, `repo` and
`forge` for the plurals, and `remove` for `rm`.

## Build numbers and run numbers

A build's number is written `#12` and a run's is written `run 31`. They
count separately, so build #12 and run 12 are unrelated. `maestro run`
takes a build, and `maestro runs show` takes a run. `maestro builds run 12`
does the same as `maestro run 12`.

## Output for scripts

Every list takes `--json`. It prints the same things the table shows, under
the names the server uses.

`maestro build` and `maestro run` take `--json` too, and `-q`, which prints
only the number of what they made. With either flag, standard output holds
the answer alone, and progress messages go to standard error:

```sh
run=$(maestro run 12 -q) && maestro runs show "$run" --watch
```

`--json` and `-q` print once, so neither works with `--watch`. A command
takes only one of the two.

## Exit status

| Status | Meaning |
| --- | --- |
| 0 | The command did what it was asked. |
| 1 | Something failed. This includes a run followed to its end with `--watch` that failed or was cancelled. |
| 2 | The command was written wrongly. `maestro` says so before it contacts the server. |
| 130 | Ctrl-C stopped it. |

## Colours

`--no-color`, or `NO_COLOR=1` in the environment, prints without colours.
Output sent to a pipe or a file has none in any case.

## Help

`maestro --help` lists the commands grouped by purpose, and
`maestro <command> --help` describes one. Each help text ends with the
`maestro docs` topic that explains the command in this book.
`maestro help concepts` prints a short version of the
[glossary](../reference/glossary.md) in the terminal.
