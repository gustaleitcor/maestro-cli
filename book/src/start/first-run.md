# Your first run

This page takes a small program from a repo to its results: three
containers, each with other parameters, each writing a file you download at
the end. It assumes you have [signed in](login.md) and [added a
forge](forges.md).

## 1. A repo with a program

Make a repo on your forge with three files. The program reads its
parameters from its arguments and writes a result:

```python
# main.py
import json, sys, time

size, seed = int(sys.argv[1]), int(sys.argv[2])
time.sleep(5)                      # stands in for real work
result = {"size": size, "seed": seed, "value": size * seed}
print("done:", result)             # goes to stdout.log
with open("result.json", "w") as f:
    json.dump(result, f)           # kept, because maestro.toml says so
```

A Dockerfile that builds it into an image:

```dockerfile
# Dockerfile
FROM docker.io/library/python:3.12-slim
WORKDIR /work
COPY main.py /app/main.py
ENTRYPOINT ["python", "/app/main.py"]
CMD ["10", "1"]
```

And a `maestro.toml`, which says how to run it: one container per line of
`parameters`, each line's words given to the program as its arguments in
place of `CMD`, and which files to keep:

```toml
# maestro.toml
parameters = ["10 1", "10 2", "20 1"]
outputs    = ["result.json"]
cpus       = 1
memory     = 256
```

Push it. (You can also fork
[maestro-template](https://github.com/gustaleitcor/maestro-template), a
stress test of the scheduler, described in [A worked
example](../run/example.md).)

## 2. Build it, run it, follow it

```sh
maestro build my-first-run --watch
```

The server clones the repo, builds the image (the build log streams as it
goes) and gives the build a number, say `#12`. `--watch` then runs it at
once and follows the run until every line is over: three lines, each
`queued`, then `staged`, `assigned`, `running`, and `succeeded`.

The same, one step at a time:

```sh
maestro build my-first-run     # build #12
maestro run 12                 # run 31 of build #12: 3 lines
maestro runs show 31 --watch   # follow it
```

## 3. Look at what it left

```sh
maestro runs show 31           # each line: its args, where it ran, how it ended
maestro runs logs 31 2         # what line 2 printed
maestro runs get 31            # download every line's outputs and logs into ./31/
```

```text
31/
├── 1/
│   ├── result.json
│   ├── stderr.log
│   └── stdout.log
├── 2/
│   └── …
└── 3/
    └── …
```

The Maestro page shows the same: open My runs, then the run, then a line,
and its files open beside it.

## 4. Change something

Edit `maestro.toml` to add lines, push, and build again. Or, without
pushing, send your local copy with the run:

```sh
maestro run 12 -c maestro.toml --watch
```

A run with `-c` reuses build #12's image with other lines, in seconds; see
[Trying a change without pushing it](../run/local-files.md).

## Where next

- [maestro.toml, key by key](../run/maestro-toml.md): machines, cores,
  memory, environment, outputs.
- [Computing lines with a generator](../run/generators.md), when there are
  too many lines to write out.
- [Inside the container](../run/container.md): what your program can read
  about its line, and how many threads it should use.
- [The queue, explained](../watch/queue.md), when a line waits longer than
  you thought.
