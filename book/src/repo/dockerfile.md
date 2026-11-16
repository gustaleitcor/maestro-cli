# The minimum: a Dockerfile

Maestro needs one file from a repository: a Dockerfile at its root. The
Dockerfile says how to build an image that holds your program, and Maestro
starts containers from that image.

## The template's Dockerfile

```dockerfile
FROM docker.io/library/python:3.12-slim

# Relative `outputs` are looked for here.
WORKDIR /work

COPY requirements.txt /app/requirements.txt
RUN pip install --no-cache-dir -r /app/requirements.txt
COPY . /app

# Your program. Each line of `parameters` replaces CMD and reaches it as
# arguments; -u prints as it goes, so logs can be followed.
ENTRYPOINT ["python", "-u", "/app/main.py"]

# Used when maestro.toml has no lines.
CMD ["-t", "1", "out.txt"]
```

Three of its instructions decide how Maestro runs the image.

| Instruction | What Maestro does with it |
| --- | --- |
| `ENTRYPOINT` | The program every container starts. Maestro never changes it. |
| `CMD` | The arguments the program gets when a run supplies none. A line of parameters replaces it. |
| `WORKDIR` | The directory the program starts in. A relative path among the files to keep is looked for here. |

So the template's containers run `python -u /app/main.py` followed by either
a line of parameters or, when there are none, `-t 1 out.txt`.

Write `ENTRYPOINT` as a list, as above. Written as plain text
(`ENTRYPOINT python /app/main.py`), it runs through a shell that drops the
arguments, and every line would run the program with none.

The rest of the Dockerfile is yours. Any language works, because Maestro
only starts the container. A compiled program can be built in one stage and
copied into a small image:

```dockerfile
FROM docker.io/library/gcc:14 AS build
WORKDIR /src
COPY . .
RUN make solver

FROM docker.io/library/debian:bookworm-slim
WORKDIR /work
COPY --from=build /src/solver /usr/local/bin/solver
ENTRYPOINT ["solver"]
CMD ["--help"]
```

## What a Dockerfile alone gets you

A repository with a Dockerfile and no `maestro.toml` builds and runs. The
build log says what its runs will do:

```text
No maestro.toml; runs will use the default: 1 container running the image's own command on any machine, with 1 CPU and 512 MB each, keeping only its logs.
```

A run of that build is one line:

```text
Run 32 of ada/maestro-template@main (build #13): succeeded, on any machine
1 line(s): 1 succeeded

LINE  PARAMETERS         MACHINE  STATUS     FILES
1     (image's command)  Q1       succeeded  -
```

That is one container, started with the image's own `CMD`, with 1 core and
512 MB, on any machine. Maestro keeps what it printed and no files, so the
`out.txt` the program wrote is gone with the container.

This is enough to run one long job on the lab's machines. To run the program
many times with other arguments, to give it more cores or memory, or to keep
its files, add a [`maestro.toml`](maestro-toml.md).

You can see this with your copy of the template: delete `maestro.toml`,
push, and build. Put the file back afterwards, because the next chapter
starts from it.

## Where Maestro looks for the Dockerfile

Maestro builds the `Containerfile` at the root of the repository, or else the
`Dockerfile` there. To build from a file somewhere else, name it in
`maestro.toml` with [`dockerfile`](maestro-toml.md#dockerfile).

The root of the repository is the build's context wherever the Dockerfile
is, so the paths in `COPY` start at the root.

A repository with neither file doesn't reach the server. `maestro build`
stops and says that it has no Dockerfile.

## What goes into the image

`COPY . /app` copies the repository into the image, except what
`.dockerignore` lists. The template's `.dockerignore` leaves out caches, the
README, and also `maestro.toml` and `genvrs.py`. Maestro reads those two from
the repository when it builds. The program never needs them.

Maestro's clone has no `.git` directory. Git keeps the forge token there,
and the token would end up in the image. A `RUN git rev-parse HEAD` in a
Dockerfile therefore fails. The Maestro page shows which commit a line's
image was built from.

A smaller image starts sooner. The first time a line runs on a machine,
Maestro copies the image there, and the line waits for the copy.

## How long a build takes

Every build runs every step of the Dockerfile again. Maestro keeps no layers
from earlier builds, so installing dependencies takes the same time each
time. A build that runs longer than the server allows, 30 minutes unless the
administrator changed it, fails.

Changing only the parameters doesn't need a build. `maestro run <build> -c
maestro.toml` runs an image you already have on other lines, as
[Trying a change without pushing it](local-files.md) explains.

## Printing as the program runs

The template starts Python with `-u`, which turns off output buffering.
Without it, Python holds back what the program prints until a buffer fills,
and `maestro runs logs <run> <line> -f` shows nothing for a long time.
In other languages, flush standard output after each line you want to see.
