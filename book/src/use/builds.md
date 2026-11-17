# Builds and images

A build turns one commit of a repository into a container image. Runs start
from builds, so a build comes first.

```sh
maestro build my-app                  # build the default branch
maestro build my-app --ref v1.2.0     # a branch, a tag or a commit
maestro build my-app --run            # build, then start a run
maestro build my-app --watch          # build, start a run, follow it to the end
maestro builds                        # the builds you have
maestro builds rm 12                  # remove a build's image
```

## What a build does

1. The server clones the repository at the commit you asked for.
2. It reads [`maestro.toml`](../repo/maestro-toml.md) and runs the
   [generator](../repo/generator.md), if there is one. The lines that result
   are stored with the build.
3. It builds the image from the Dockerfile.

The build log streams to your terminal while this happens. Ctrl-C stops the
build on the server too. When a build fails, the last lines of the log say
why. The usual causes are a step of the Dockerfile, an invalid
`maestro.toml`, and a generator that raised an exception.

A finished build has a number, written `#12`. `maestro run 12` starts a run
of its image.

You can have 2 builds running at once, unless your administrator set
another number. Maestro refuses a third with "you already have 2 builds
running; wait for one to end".

## Your builds

```text
BUILD  REPO                           REF   STATUS   IMAGE  CREATED
#14    ada/maestro-template           main  success  152M   2026-11-16 10:01
#13    ada/maestro-template           main  success  152M   2026-11-16 09:01
#12    gustaleitcor/maestro-template  main  success  152M   2026-11-16 08:01

3 of the 10 images allowed; the one run least recently goes when a new build needs room.
```

The `IMAGE` column shows the image's size, or `removed` when the build's
image is gone.

## How many images you keep

You keep at most 10 images, unless your administrator set another number.
When a new build would exceed it, Maestro removes the image you ran least
recently. An image you never ran counts by when it was built, and an image a
run is still using is never chosen.

Removing an image keeps the build's record, its runs and their files. To run
that commit again, build it again. `maestro builds rm <build>` removes an
image by hand to make room.

## Images on the machines

An image is built on the server. The first time one of its lines is given to
a machine, Maestro copies the image there, compressed on the way, and the
line stays `assigned` until the copy is done. The image then stays on that
machine, so later runs of the same build start without the copy.

A smaller image copies faster. A `-slim` base image and a `.dockerignore`
that leaves out what the program doesn't use both help.
