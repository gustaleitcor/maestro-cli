# Describing a run: maestro.toml

A repo says how its image is run with a `maestro.toml` at its root, beside
its Dockerfile. Every key is optional:

```toml
machines   = ["Q1", "Q2", "Q3"]              # leave out for any machine
parameters = ["-t 10 out.txt", "-t 20 out.txt"] # one container per line
generator  = "genvrs.py"                     # more lines, computed
outputs    = ["out.txt"]                     # copied out when each finishes
cpus       = 2                               # cores per container; default 1
memory     = 2048                            # MB per container; default 512
env        = { SEED = "42" }                 # variables of each container's environment
dockerfile = "docker/Dockerfile"             # what the image is built from
outputs_layout = "per-line"                  # or "shared"
```

The server reads `maestro.toml` when it builds the repo, and keeps it with the
build: every run of a build is the same lines, run the same way. An invalid
one fails the build, and the build log says why. A name it doesn't know
fails it too, so that a misspelt key isn't ignored.

Without a `maestro.toml`, or for any key left out of one, a run takes the
defaults: a single container with the image's own `CMD`, on any machine,
with 1 core and 512 MB, keeping only what it prints. The build log says which
one a build got.

## `parameters`

```toml
parameters = ["sleep 30", "burn 30", "hold 30 --fraction 0.5"]
```

One container per line. A line replaces the image's `CMD`, so the program
the image's `ENTRYPOINT` starts gets it as its arguments. No shell runs it:
the line is split into words at spaces, and quotes group words (`"a b" c` is
two words), nothing else. There are no variables, pipes or globs; a program
that wants them can be a shell script.

Without `parameters` or a `generator`, a run is one line, with the image's
own `CMD`. A run has at most 1000 lines.

## `generator`

```toml
generator = "genvrs.py"
```

A Python script in the repo that computes more lines, after those of
`parameters`, and can change `cpus`, `memory`, `env` and `outputs` from one
line to the next. See [Computing lines with a generator](generators.md).

## `machines`

```toml
machines = ["Q1", "Q2"]
```

Where the lines may run: names from `maestro machines`. Without it, any
machine. It is the same for every line; a generator can't change it.

## `cpus` and `memory`

```toml
cpus   = 4      # whole cores, at least 1; default 1
memory = 8192   # MB, at least 6; default 512
```

What each container may use. There is always a limit: a container that
takes more memory than it asked for is killed, and its line fails.

They also decide where and when a line runs: a machine starts a line only
while its cores and memory fit in what the machine has, less what is kept
back for its own system and what the lines already there take. A run with a
line larger than every machine it may use, even empty, is refused, so asking
for less lets a line start sooner. `maestro machines` shows what each machine
has for lines.

A limit on cores doesn't change how many cores a program *sees*; see
[Inside the container](container.md) for sizing threads.

## `env`

```toml
env = { SEED = "42", MODE = "fast" }
```

Variables of each container's environment, besides the `MAESTRO_` ones the
server sets (see [Inside the container](container.md)). Names may not start
with `MAESTRO_`.

The environment is kept with the run and shown by `podman inspect` on the
machine, so `env` is no place for secrets.

## `outputs`

```toml
outputs = ["report.json", "results/", "plots/summary.png"]
```

Files copied out of each container when it ends, as paths in the container:
a relative path is from the image's `WORKDIR` (`/` without one), and a
directory is copied with all it holds. An output the line didn't write is
listed as missing, which is no error. What a container prints is always
kept, as `stdout.log` and `stderr.log`, outputs or not. See [What a line
leaves behind](../watch/results.md).

## `outputs_layout`

```toml
outputs_layout = "shared"   # default: "per-line"
```

Where the outputs are kept. With `per-line`, the default, each line's go in
a directory of its own, `<run>/<line>/<path>`. With `shared`, every line's
go in one tree of the run's, `<run>/shared/<path>`, as a program that has
its lines fill in one results tree expects: `results/<month>/<day>/...` from
all of them, downloaded as one.

Two lines writing the same path overwrite each other, the one kept last
winning, so each line should write paths of its own. Logs stay per line
either way, and `maestro runs get` and the run's download give the shared
tree once, under `shared/`.

## `dockerfile`

```toml
dockerfile = "docker/Dockerfile"
```

The file the image is built from, a path from the root of the repo. The root
stays the build's context: the Dockerfile's `COPY` lines name files from the
root, not from beside it. Without it the server builds the `Containerfile` at
the root, or else the `Dockerfile`. It must be a file inside the repo; a
symbolic link out of it isn't followed, and a missing one fails the build.

## What is cloned

The server clones the repo's files without `.git`, for the generator and for
the image alike: git keeps the forge token there, and it would end up in the
image. So a Dockerfile can't ask git for the commit it is built from; a
line's Details, on the Maestro page, say which commit its image was built
from.
