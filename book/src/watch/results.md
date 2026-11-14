# What a line leaves behind

When a container ends, the server keeps what it printed and the outputs it
wrote, then removes the container. On the server they are:

```text
/maestro/data/<user>/<run>/<line>/stdout.log
/maestro/data/<user>/<run>/<line>/stderr.log
/maestro/data/<user>/<run>/<line>/<file>
/maestro/data/<user>/<run>/shared/<file>     (with outputs_layout = "shared")
```

and `maestro runs get <run>` downloads them the same way, without the first
two levels:

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

A run downloaded from the Maestro page, as a zip, is laid out the same.

## Logs

stdout and stderr are kept apart, each in its own file, in the line's folder
with its outputs. Both are made even when a stream printed nothing. An
output named `stdout.log` or `stderr.log` isn't kept: the log always is.

The server creates each container with Podman's file log driver
(`k8s-file`), whatever the machine's default, so logs come back on machines
whose default is journald too.

Runs from before logs were kept apart have both in one `<line>.log` beside
the line's folder, and are still read as they are.

## Outputs

Each path in `outputs` is copied out of the container, a directory with all
it holds. Only regular files and directories are kept: symbolic links,
devices and the like are skipped. An output the line didn't write is listed
as missing.

With `outputs_layout = "shared"`, every line's outputs go into one tree of
the run's instead; two lines that write the same path overwrite each other,
the one kept last winning.

## Limits

Logs and outputs together are kept up to 1 GB per line, unless your
administrator set otherwise (`RUN_OUTPUT_MAX_MB`). Past it, the rest is
dropped, and the line notes that its logs or outputs were cut.

What runs keep stays until you delete the run (`maestro runs rm`); removing
an image to make room for a new build keeps its runs and their files.
