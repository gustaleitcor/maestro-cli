# maestro.toml keys

`maestro.toml` sits at the root of the repository. The file and every key
in it are optional. [maestro.toml: how the image is run](../repo/maestro-toml.md)
introduces the keys one at a time, and this page is the lookup table.

## The keys

| Key | Type | Default | What it sets |
| --- | --- | --- | --- |
| [`parameters`](../repo/maestro-toml.md#parameters) | List of strings | One line that runs the image's own `CMD` | The lines of a run. Each string becomes the arguments of one container. |
| [`generator`](../repo/maestro-toml.md#generator) | String | None | A Python script in the repository whose `genvrs` function yields more lines |
| [`outputs`](../repo/maestro-toml.md#outputs) | List of strings | None, so only the logs are kept | The files and directories to keep from each container |
| [`outputs_layout`](../repo/maestro-toml.md#outputs_layout) | `"per-line"` or `"shared"` | `"per-line"` | Whether each line's outputs go in a directory of their own or all into one tree for the run |
| [`cpus`](../repo/maestro-toml.md#cpus-and-memory) | Whole number | 1 | The cores each container can use, and the cores a machine must have free to start it |
| [`memory`](../repo/maestro-toml.md#cpus-and-memory) | Whole number, in MB | 512 | The memory each container can use, and the memory a machine must have free to start it |
| [`env`](../repo/maestro-toml.md#env) | Table of strings | None | Environment variables set in every container |
| [`machines`](../repo/maestro-toml.md#machines) | List of strings | Any machine | The only machines the lines can run on |
| [`dockerfile`](../repo/maestro-toml.md#dockerfile) | String | `Containerfile` at the root, or else `Dockerfile` | The file the image is built from |

## Limits

| What | Limit |
| --- | --- |
| Lines in a run, from `parameters` and the generator together | 5000 |
| `cpus` | 1 to 1024 |
| `memory` | 6 to 16777216 MB |
| `env`, all names and values together | 64 KiB |
| An `env` name | Letters, digits and `_`. It can't start with a digit or with `MAESTRO_`. |
| A path in `outputs` | A file or directory inside the container. It can't be `/` or the working directory itself, and it can't begin with `..`. |
| `generator` and `dockerfile` | A file inside the repository |
| The generator script | 1 MiB |
| A file sent with `-c`, `-g` or `-f` | 1 MiB |

Maestro also refuses a run when one of its lines asks for more `cpus` or
`memory` than any machine it may use has for lines, even with nothing else
running there.

## How a line is split into words

Maestro splits each line of `parameters`, and each string a generator
yields, at spaces and tabs. No shell is involved.

| Written | Meaning |
| --- | --- |
| `'two words'` | One argument. Everything inside single quotes is taken as written. |
| `"two words"` | One argument. Inside double quotes, a backslash keeps the next character as written, so `\"` is a quote. |
| `two\ words` | One argument. A backslash keeps the next character as written. |
| `$HOME`, `*`, `~` | Passed to the program as typed, because no shell expands them |

A line with a quote left open, or one that ends in a backslash, is an
error.

## Per-line settings from a generator

A generator's `Config` class can set `outputs`, `cpus`, `memory` and `env`
for the lines it yields, within the same limits.
[Lines that need different settings](../repo/generator.md#lines-that-need-different-settings)
shows how.

## A file with every key

```toml
parameters = [
  "--seed 1 results/",
  "--seed 2 results/",
]
generator      = "genvrs.py"
outputs        = ["results/", "summary.json"]
outputs_layout = "per-line"
cpus           = 4
memory         = 8192
env            = { MODE = "fast" }
machines       = ["Q1", "Q2"]
dockerfile     = "docker/Dockerfile"
```

## Errors

An unknown key fails the build and lists the keys Maestro knows:

```text
maestro.toml: unknown key "paramters" (known: machines, parameters, generator, dockerfile, outputs, outputs_layout, cpus, memory, env)
```

The other messages name the key and the value at fault, as these do:

```text
maestro.toml: cpus is 0; give between 1 and 1024 cores
maestro.toml: memory is 4; give between 6 and 16777216 MB
maestro.toml: outputs_layout "merged" must be "per-line" (each line's outputs apart, the default) or "shared" (every line's in one tree)
```
