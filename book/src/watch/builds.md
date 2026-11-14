# Running and watching

Once a repo builds, everything else is following what it does: [runs and
lines](runs.md), [what they leave](results.md), [where they wait](queue.md),
[the machines](top.md) and [what you used](usage.md). This page is about
builds.

## Builds and images

```sh
maestro build my-app                  # build #12
maestro build my-app --ref v1.2.0     # a branch, tag or commit
maestro build my-app --run            # and run it once it's built
maestro build my-app --watch          # and follow the run until it is over
maestro builds                        # the builds you have
maestro builds rm 12                  # remove a build's image
```

A build clones the repo at a commit (the default branch's newest unless
`--ref` says otherwise), reads `maestro.toml`, runs the generator, and builds
the image with the server's Podman. The build log streams as it goes;
ctrl-c during a build cancels it on the server. A build that fails says why
at the end of its log: most often the Dockerfile, an invalid `maestro.toml`
or a generator that raised.

The build gets a number, shown as `#12`: `maestro run 12` runs its image, and
`maestro builds run 12` is the same.

## How many images you keep

Each user keeps at most 10 built images (an administrator can change the
number on the Maestro page). A build that would make one more removes the
image you ran least recently, or built longest ago if you never ran it,
skipping any a run is still using; it is removed from the machines as well.
The build's record, runs and files are kept, so you lose nothing but the
image: build the commit again to run it again. `maestro builds rm <build>`
frees a place by hand.

## Images on the machines

The server copies an image to a machine the first time a line needs it
there, streaming `podman save` into `podman load` over SSH, gzipped on the
way, which takes an image's layers to about a third. It leaves the image
there when the run is over, so the next run of that build starts without
copying it again. A line is `assigned` while its image is copied.

Smaller images start sooner on a machine that hasn't seen them: a `-slim`
base, and nothing in the image the program doesn't need.
