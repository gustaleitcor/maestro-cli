# maestro.toml: how the image is run

`maestro.toml` is an optional file at the root of the repository, beside the
Dockerfile. It says how many containers a run starts and with which
arguments, how much of a machine each one may use, and which files to keep.

Every key is optional. A key you leave out keeps the default from the
[previous chapter](dockerfile.md#what-a-dockerfile-alone-gets-you): one
container, the image's own command, 1 core, 512 MB, any machine, no files.
A line runs for as long as the server lets it, a week unless your
administrator set another limit.

This is the template's file, with its comments removed:

```toml
parameters = [
  "-t 10 out.txt",
  "-t 20 --label 'long run' out.txt",
]

outputs = ["out.txt"]
```

Maestro reads `maestro.toml` when it builds the repository and keeps the
result with the build, so every run of a build has the same lines. To change
them, push the change and build again, or
[send the file with a run](local-files.md).

The sections below add one key at a time. Each can be tried on your copy of
the template: edit, push, and `maestro build <your copy> --watch`.

## `parameters`

Each string is one line, and a run starts one container for each line. The
line's words replace the image's `CMD`, so the program receives them as its
arguments. The template's two lines start these two containers:

```text
python -u /app/main.py -t 10 out.txt
python -u /app/main.py -t 20 --label 'long run' out.txt
```

Maestro splits a line into words at spaces. Single quotes, double quotes and
backslashes keep words together the way they do in a shell, so `'long run'`
reaches the program as one argument. Nothing else is special, because no
shell runs the line. A `$HOME` or a `*` reaches the program as typed. A
program that needs a shell can be a shell script.

Add a third line, and the next build says `3 lines`:

```toml
parameters = [
  "-t 10 out.txt",
  "-t 20 --label 'long run' out.txt",
  "-t 5 --label quick out.txt",
]
```

A run has at most 5000 lines. When a list gets long or follows a pattern, a
[generator](generator.md) can compute it.

## `outputs`

```toml
outputs = ["out.txt", "results/", "plots/summary.png"]
```

These are the files Maestro copies out of each container when it ends,
whether the program succeeded or failed. Each path is a path inside the
container. A relative one starts at the image's `WORKDIR`, which is `/work`
in the template. A directory is kept with everything in it.

A file the program didn't write is listed as missing, and the line's status
doesn't change because of it.

What the container prints is always kept, as `stdout.log` and `stderr.log`,
whatever `outputs` says. [What a line leaves behind](../use/results.md) shows
where everything ends up.

## `cpus` and `memory`

```toml
cpus   = 4      # whole cores; default 1
memory = 8192   # MB; default 512
```

These two numbers do two jobs.

They limit the container. It can use the processor time of `cpus` cores and
no more. If it takes more memory than `memory`, the kernel kills it, and the
line ends `failed`, usually with exit code 137.

They also decide where and when the line starts. A machine starts a line
only when that many cores and that much memory are free on it. A smaller
request fits on more machines and starts sooner. Maestro refuses a run when
one of its lines would not fit on any machine it may use, even an empty one:

```text
line 3 asks for 64 cpus and 512 MB, which no machine it may use has for lines even with nothing else on it (see `maestro machines list`)
```

`maestro machines` shows what each machine has. A program doesn't learn its
limit by counting the machine's cores, so read
[how many threads to start](container.md#how-many-threads-to-start) before
giving a multithreaded program more than one core.

## `time`

```toml
time = "2h"   # default: the server's limit, a week unless your administrator set another
```

This is the longest each line may run. Write it as a number and a unit,
`h`, `m` or `s`, such as `"90m"`, `"2h"` or `"1h30m"`. It must be at least
`"1m"`, and at most the server's limit. Maestro refuses a run whose lines
ask for more:

```text
line 1 asks for time "200h", longer than the 168 hours a line may run on this server; give 168h or less in maestro.toml
```

When a line runs longer than its `time`, Maestro stops it, keeps its logs
and outputs, and ends it as `error` with the note "stopped after 2h, the
time this line asked for". The program finds its time in seconds in
[`MAESTRO_TIME`](container.md#what-maestro-tells-the-program), so it can
stop on its own a little before and write what it has.

## `env`

```toml
env = { SEED = "42", MODE = "fast" }
```

These variables are set in every container's environment. A name is made of
letters, digits and `_`, and can't start with a digit. It also can't start
with `MAESTRO_`, which is reserved for the
[variables Maestro sets itself](container.md#what-maestro-tells-the-program).

Don't put secrets in `env`. Maestro stores the environment with the run and
shows it on the line's page.

## `machines`

```toml
machines = ["Q1", "Q2"]
```

The lines run only on these machines. The names come from `maestro machines`,
and upper or lower case doesn't matter. A name that doesn't exist refuses
the run.

Leave `machines` out unless the program needs particular hardware. With no
`machines`, a line starts on whichever machine has room first.

## `outputs_layout`

```toml
outputs_layout = "shared"   # default: "per-line"
```

With `per-line`, each line's outputs go in a directory of its own,
`<run>/<line>/<path>`. Two lines can both write `out.txt` and both files are
kept.

With `shared`, every line's outputs go into one tree for the whole run,
`<run>/shared/<path>`. This suits a program whose lines each fill in part of
one results directory, such as `results/<instance>/<seed>.json`. When two
lines write the same path, the one Maestro collects last overwrites the
other, so give each line paths of its own. Logs stay separate for each line
in both layouts.

## `dockerfile`

```toml
dockerfile = "docker/Dockerfile"
```

Maestro builds the image from this file. The path starts at the root of the
repository and must stay inside it. The root remains the build's context, so
`COPY` paths in that Dockerfile still start at the root.

## `generator`

```toml
generator = "genvrs.py"
```

This names a Python script in the repository that computes lines. They come
after the lines of `parameters`, and the script can give each of its lines
its own `cpus`, `memory`, `env`, `outputs` and `time`. The
[next chapter](generator.md) is about it.

## When the file is wrong

An invalid `maestro.toml` fails the build, and the last lines of the build
log say why. A misspelt key is an error too, so that a typo isn't silently
ignored:

```text
maestro.toml: unknown key "paramters" (known: machines, parameters, generator, dockerfile, outputs, outputs_layout, cpus, memory, env, time)
```

[maestro.toml keys](../reference/maestro-toml.md) lists every key with its
default and its limits on one page.
