# What a line leaves behind

When a container ends, Maestro keeps what it printed and the files that
[`outputs`](../repo/maestro-toml.md#outputs) names, then removes the
container. Everything else the program wrote inside it is gone.

## While a line runs

With [`outputs_every`](../repo/maestro-toml.md#outputs_every), Maestro also
copies the logs and outputs of a running line that often, into the same
place. Each copy takes the place of the one before, and the copy taken when
the line ends takes the place of the last. A line that loses its machine
keeps the last copy, and its note says when it was taken:

```text
q7 restarted while this line ran there, and its container was lost; its logs and outputs are those copied at 2026-11-17 14:30 UTC
```

Without `outputs_every`, a line that loses its machine keeps nothing.

## The layout of a download

`maestro runs get <run>` downloads a run into a directory named after it,
with one directory for each line:

```text
31/
├── 1/
│   ├── report.json
│   ├── stderr.log
│   └── stdout.log
├── 2/
│   └── …
└── shared/          (with outputs_layout = "shared")
    └── results/…
```

A run downloaded from the Maestro page comes as a zip with the same layout.

## Logs

Maestro keeps standard output and standard error apart, as `stdout.log` and
`stderr.log` in the line's directory. Both files exist even when the program
printed nothing to one of them.

The two names are reserved. If `outputs` lists a file called `stdout.log` or
`stderr.log`, Maestro keeps the log and drops the program's file.

Runs made before Maestro kept the two streams apart have both in one file,
`<line>.log`, beside the line's directory. Maestro still reads those.

## Outputs

Maestro copies each path in `outputs` out of the container. A directory comes
with everything in it. Only regular files and directories are kept, so
symbolic links, devices and similar entries are skipped.

An output the program didn't write is listed as missing, as in
`out.txt (missing)` under `FILES` in `maestro runs show`.

With [`outputs_layout = "shared"`](../repo/maestro-toml.md#outputs_layout),
the outputs of every line go into one tree for the run, under `shared/`.
When two lines write the same path, the one Maestro collects last overwrites
the other.

## How much is kept

Maestro keeps up to 1 GB of logs and outputs for each line, unless your
administrator set another limit with `RUN_OUTPUT_MAX_MB`. It drops what goes
over, and the line carries a note that its logs or outputs were cut.

## How long it is kept

A run's files stay until you delete the run with `maestro runs rm <run>`,
or until the run expires, if your administrator set a number of days.
[How long a run is kept](runs.md#how-long-a-run-is-kept) says when that is
and how to keep a run. Removing a build's image to make
room for a newer build keeps the build's runs and their files.

## Where the files are on the server

An administrator finds the same files under the server's data directory:

```text
/maestro/data/<user>/<run>/<line>/stdout.log
/maestro/data/<user>/<run>/<line>/stderr.log
/maestro/data/<user>/<run>/<line>/<file>
/maestro/data/<user>/<run>/shared/<file>     (with outputs_layout = "shared")
```

[What orq keeps, and for how long](../admin/retention.md) covers the volume
this needs.
