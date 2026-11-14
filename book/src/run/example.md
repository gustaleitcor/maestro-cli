# A worked example: the stress test

[maestro-template](https://github.com/gustaleitcor/maestro-template) is a
repo to fork: a stress test of Maestro's scheduler, which uses every key of
`maestro.toml` and a generator with `Config`. Its containers do no useful
work. Each takes up the cores and memory it asked for (or only reserves
them) for a while, and writes `report.json` with what it used. What it tests
is how Maestro places, queues and interleaves a thousand lines of very
different sizes.

| File | What it is |
| --- | --- |
| `Dockerfile` | Python and `main.py`, nothing to install |
| `maestro.toml` | where it may run, one line of `parameters` per mode, and the default `cpus`, `memory`, `env` and `outputs` |
| `genvrs.py` | 996 more lines, which makes 1000, a run's most |
| `main.py` | the program: `sleep`, `burn`, `hold` or `mix` for some seconds |

## maestro.toml

Abridged; the template's own file says what each key does.

```toml
machines   = ["q1", "q2", "q3", "q4", "q5", "q6", "q7", "q8", "q9", "q10", "q11", "q12"]

# One container per line. A line replaces the image's CMD, so main.py gets it
# as its arguments.
parameters = [
  "sleep 30",
  "burn 30",
  "hold 30 --fraction 0.5",
  "mix 30",
]

# 996 more lines, computed. Its class Config changes outputs, cpus, memory
# and env per line.
generator = "genvrs.py"

cpus   = 1
memory = 512
env    = { STRESS_LINE = "parameters", STRESS_SHAPE = "default" }

# Lines run with --samples also keep samples.csv, which genvrs.py adds.
outputs = ["report.json"]
```

## The generator

`genvrs.py` gives each line a shape (its cores and memory, through
`Config`), a mode and a time, drawn from a fixed seed, so every build of a
commit gives the same lines:

```python
import random
from collections.abc import Generator

LINES = 996
SEED = 2026

class Config:
    outputs: list[str] | None = None
    cpus: int | None = None
    memory: int | None = None
    env: dict[str, str] | None = None

def genvrs() -> Generator[str]:
    rng = random.Random(SEED)
    for n in range(1, LINES + 1):
        shape = rng.choices(SHAPES, weights=[s.weight for s in SHAPES])[0]
        mode = rng.choices(list(MODES), weights=list(MODES.values()))[0]
        seconds = rng.choice((10, 20, 30, 45, 60, 90, 120))
        Config.cpus = shape.cpus
        Config.memory = shape.memory
        Config.env = {"STRESS_LINE": str(n), "STRESS_SHAPE": shape.name}
        line = f"{mode} {seconds}"
        # One in ten also keeps its samples: an output maestro.toml lacks.
        if rng.random() < 0.1:
            line += " --samples"
            Config.outputs = ["report.json", "samples.csv"]
        else:
            Config.outputs = None
        yield line
```

(Abridged too: the real one defines `SHAPES` and `MODES`, keeps the largest
lines longer, gives `hold` and `mix` a `--fraction`, makes every hundredth
line fail on purpose so that a run has failed lines to show, and prints how
many lines of each shape it made, which goes to the build log.)

## The Dockerfile

The program needs only Python's standard library. Its `ENTRYPOINT` is the
program, so each line's words reach it as arguments; `CMD` is what runs when
there are no lines.

```dockerfile
FROM docker.io/library/python:3.12-slim
ENV PYTHONDONTWRITEBYTECODE=1 \
    PYTHONUNBUFFERED=1
WORKDIR /work
COPY main.py /app/main.py
ENTRYPOINT ["python", "/app/main.py"]
CMD ["sleep", "10"]
```

## Running it

```sh
maestro build <your fork> --watch     # build, run once, and follow it
maestro run <build>                   # again
maestro queue                         # where the lines wait
maestro top q1                        # one machine, as it fills
maestro runs get <run>                # the reports and logs
```

Every run of a build is the same lines. A different `SEED` in `genvrs.py`
draws others, with no new build: `maestro run <build> -g genvrs.py` sends
it from your copy, and `-c` a `maestro.toml` with, say, other machines.
