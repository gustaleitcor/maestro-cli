# Generating lines with Python

Writing lines by hand works for a few of them. A grid of parameters, or one
line for each instance file, is easier to compute. A generator is a Python
script in the repository that defines a function called `genvrs`. Every
string the function yields becomes one line, split into words the same way a
line of [`parameters`](maestro-toml.md#parameters) is.

## Turn on the template's generator

The template already has one, `genvrs.py`:

```python
def genvrs():
    for seconds in (10, 20, 30):
        yield f"-t {seconds} out.txt"
```

Its `maestro.toml` has the line that enables it commented out. Remove the
`#`:

```toml
generator = "genvrs.py"
```

Push and build. The build log now counts five containers:

```text
Read maestro.toml: 5 lines (3 from genvrs.py) on any machine, with 1 core and 512 MB each, keeping out.txt.
```

The two lines of `parameters` come first, then the three the generator
yielded. If you delete `parameters`, the generator's lines are the whole run.

## A grid of parameters

```python
import itertools
from collections.abc import Generator

def genvrs() -> Generator[str]:
    for t, seed in itertools.product([10, 20, 30], range(3)):
        yield f"-t {t} --seed {seed} out.txt"
```

This yields nine lines, from `-t 10 --seed 0 out.txt` to
`-t 30 --seed 2 out.txt`.

## One line for each file

The script can read the repository's files, so it can list a directory of
instances. This is a shortened version of what the template's `knapsack`
branch does:

```python
from pathlib import Path

INSTANCES = Path(__file__).parent / "instances"

def genvrs():
    for path in sorted(INSTANCES.glob("*.txt")):
        yield f"{path.stem} --time-limit 60"
```

## Lines that need different settings

By default every generated line runs with the `cpus`, `memory`, `env`,
`outputs` and `time` of `maestro.toml`. To change them for some lines, define a class
called `Config` above `genvrs` and assign to its attributes before yielding.
Each line runs with what the class holds at the moment the line is yielded.

```python
class Config:
    outputs: list[str] | None = None     # None: what maestro.toml says
    cpus: int | None = 4                 # 4 cores, whatever maestro.toml says
    memory: int | None = None
    env: dict[str, str] | None = None
    time: str | None = None

def genvrs() -> Generator[str]:
    yield "-t 10 small"          # 4 cores, maestro.toml's memory
    Config.memory = 4096
    for seed in range(3):
        Config.env = {"SEED": str(seed)}
        yield "-t 10 large"      # 4 cores, 4096 MB, SEED 0, 1 and 2
    Config.memory = None
    yield "-t 10 medium"         # 4 cores, maestro.toml's memory again
```

`None` means "what `maestro.toml` says, or the default if it says nothing".
The class may hold only these five attributes. `machines` is the same for
every line and is set in `maestro.toml`.

## Where and when the script runs

Maestro runs the script once, during the build, and the build keeps the
lines it yielded. Every run of that build has the same lines. A script that
draws random numbers should seed them (`random.Random(2026)`), so that
building the same commit twice gives the same lines.

The script runs inside the Maestro server, in a sandbox, and not in your
image. The sandbox is a Python 3.14 with the standard library and nothing
else. That has consequences:

- Packages from `requirements.txt` aren't there, and neither is anything
  that needs compiling, such as numpy.
- The script can read the repository's files and import its other modules.
  It can't write files.
- It has no network and no environment variables.
- It gets 256 MB of memory and 30 seconds.

What the script prints goes to the build log, which makes `print` a good way
to check what it computed.

## What fails the build

The build fails, and the log says why, when the script:

- raises an exception, or runs past its 30 seconds
- doesn't define `genvrs`
- yields nothing
- yields something that isn't a string
- brings the run to more than 5000 lines, counting those of `parameters`
- sets a `Config` attribute to a value `maestro.toml` wouldn't accept, or
  sets one that isn't a setting

```text
genvrs yielded a int as item 4; every item must be a string
Config.machines isn't a setting; Config may only hold outputs, cpus, memory, env, time
```

## Trying a generator without building

`maestro run <build> -g genvrs.py` runs an image you already have on the
lines of a script from your computer. Nothing is pushed or built. See
[Trying a change without pushing it](local-files.md).
