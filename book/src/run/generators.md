# Computing lines with a generator

When there are too many lines to write out (every instance of a benchmark,
a grid of parameters, a hundred seeds of each), a Python script can compute
them. `generator` in `maestro.toml` names a script in the repo that defines
a generator called `genvrs`; every string it yields is one more line, split
as a line of `parameters` is, after those of `parameters` when there are
both:

```toml
generator = "genvrs.py"
```

```python
import itertools
from collections.abc import Generator

def genvrs() -> Generator[str]:
    for t, seed in itertools.product([10, 20, 30], range(3)):
        yield f"-t {t} --seed {seed} out.txt"
```

That is nine lines: `-t 10 --seed 0 out.txt` to `-t 30 --seed 2 out.txt`.

## Lines that differ: `Config`

Above `genvrs`, the script may define a class `Config` with the keys that
can differ from one line to the next: `outputs`, `cpus`, `memory` and `env`.
A key the class leaves out or sets to `None` has what `maestro.toml` says, or
its default (`[]`, `1`, `512`, `{}`); any other value overrides it.

`genvrs` changes them on the class, `None` going back to `maestro.toml`'s,
and each line runs with what the class holds when it is yielded, checked as
`maestro.toml` is:

```python
class Config:
    outputs: list[str] | None = None     # default: none
    cpus: int | None = 4                 # instead of maestro.toml's; default: 1
    memory: int | None = None            # MB; default: 512
    env: dict[str, str] | None = None    # default: none

def genvrs() -> Generator[str]:
    yield "-t 10 small"          # 4 CPUs, maestro.toml's memory
    Config.memory = 4096
    for seed in range(3):
        Config.env = {"SEED": str(seed)}
        yield "-t 10 large"      # 4 CPUs, 4096 MB, SEED 0, 1 and 2
    Config.memory = None
    yield "-t 10 medium"         # 4 CPUs, maestro.toml's memory again
```

`machines` is set in `maestro.toml` only, the same for every line. Any other
name in `Config`, `machines` included, fails the build, so that a misspelt
key isn't ignored. A run lists the outputs of all its lines together; each
line keeps only its own, or, with `outputs_layout = "shared"`, writes them
into the run's shared tree.

## When and where it runs

The server runs the script once, when it builds the repo, and the build
keeps the lines it gave: every run of that build is the same lines. To get
other lines without a new build, send another script with the run
(`maestro run 12 -g genvrs.py`); see [Trying a change without pushing
it](local-files.md).

It isn't run in your image but inside the server, by a CPython 3.14
compiled to WebAssembly. It has Python's standard library and the repo,
which it can read and not write. It starts at the root of the repo and
imports from beside itself, so it can `import` the repo's other modules and
open its files (`open("cases.csv")`).

Nothing else is there:

- no packages installed from `requirements.txt`, and nothing that needs
  compiling, such as numpy;
- nothing a symbolic link leads to outside the repo;
- no other file of the server, no network, no environment;
- 256 MB of memory and 30 seconds.

What it prints goes to the build log. A script that raises, yields something
that isn't a string, yields nothing, makes the build's lines more than 1000,
or gives a line an invalid `Config`, fails the build, and the log says why.

A script that draws at random should seed its generator (`random.Random(2026)`),
so that a build of the same commit gives the same lines; [the stress
test](example.md) does.
