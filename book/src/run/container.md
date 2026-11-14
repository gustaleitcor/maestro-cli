# Inside the container

Each line runs as a container of your image: its `ENTRYPOINT`, with the
line's words as its arguments in place of `CMD`, in the image's `WORKDIR`.
It gets the cores and memory the line asked for and nothing more, and is
removed once its logs and outputs are kept.

## What the server tells it

The server sets these variables in every container, after those of `env`:

| Variable | |
| --- | --- |
| `MAESTRO_RUN`, `MAESTRO_LINE` | The run's number, and the line's in it, from 1 |
| `MAESTRO_CPUS` | Its `cpus` |
| `MAESTRO_MEMORY` | Its `memory`, in MB |
| `MAESTRO_OUTPUTS` | Its `outputs`, one per line |

## How many threads

A limit on cores doesn't change how many cores a program sees: `nproc`,
`os.cpu_count()` and most libraries still count every core of the machine.
A program that starts a thread per core would start far more than it has,
and slow itself down. Size it by `MAESTRO_CPUS` instead:

```python
import os
threads = int(os.environ.get("MAESTRO_CPUS", "1"))
```

```sh
export OMP_NUM_THREADS="$MAESTRO_CPUS"   # OpenMP, and numpy's BLAS
```

Solvers usually take it as an option: Gurobi's `Threads`, CPLEX's
`threads`, OR-Tools' `num_workers`.

## Memory

A container that takes more memory than its `memory` is killed by the
kernel, and its line ends `failed`, with exit code 137. Ask for what the
program needs at its largest, with some room: a line that asks for less
starts sooner, but one that dies has to run again.

## Machines that can't hold cores

On a machine whose Podman can't limit CPU (a rootless one whose user
systemd doesn't hand down the `cpu` cgroup controller, which is Ubuntu
22.04's default) lines still count their `cpus` there, but nothing holds
them to it: one that takes more slows the others down. The administrator of
the machine can fix it once; see [Adding machines](../admin/machines.md).

## Files and logs

Write results to the paths `outputs` names; everything else in the
container is gone when it ends. Print as you go: what the program prints is
kept as `stdout.log` and `stderr.log`, and can be followed while the line
runs (`maestro runs logs <run> <line> -f`). Python buffers what it prints
when it isn't writing to a terminal, so set `PYTHONUNBUFFERED=1` in the
image, or print with `flush=True`, to see lines as they are printed.

Logs and outputs together are kept up to a limit per line (1 GB unless your
administrator set otherwise); past it, the rest is dropped, and the line
says so.

## Exit code

A line `succeeded` when its program exits 0 and `failed` when it exits with
anything else; `maestro runs show <run> --failed` lists those that didn't
succeed. A line that ends `error` never got to run, or lost its container
(its machine restarted while it ran): that is the server's trouble, not your
program's.
