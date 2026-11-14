# Trying a change without pushing it

`maestro build` and `maestro run` can send `maestro.toml`, its generator and,
to `build` only, the Dockerfile from where the CLI runs, in place of the
repo's. Each may have any name there:

```sh
maestro build my-app -c bench.toml -g genvrs.py -f Dockerfile  # a new image
maestro run 81 -c small.toml --watch                           # build #81's image, other lines
```

| Flag | Stands for |
| --- | --- |
| `-c` | `maestro.toml` at the root |
| `-g` | the generator it names, or, when it names none, a generator at the root under its own name |
| `-f` | the Dockerfile it names, or else `Dockerfile` at the root, built from even with a `Containerfile` there |

## Build or run?

**A build** puts the files over its clone before it reads anything, so the
image is built with them too. Use it when the Dockerfile changes, or what
the image holds.

**A run** doesn't build. The server clones the repo again at the commit its
build was made from (with the forge token the CLI has for it, when it's
private), puts the files over it, runs the generator there, and runs the
build's image on the lines that come of it: in seconds, and without copying
an image to the machines. So a run takes no Dockerfile, and the `dockerfile`
of a `maestro.toml` sent to it is of no matter. Use it to try other lines,
other cores and memory, or another seed, with the image you have.

Either fails as a build would when what was sent is invalid.

## Telling them apart

The lines of such a build or run aren't its commit's alone, so `maestro
builds` and `maestro runs show` say which files were sent. A file is at most
1 MiB.
