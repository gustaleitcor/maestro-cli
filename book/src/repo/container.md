# Inside the container

Each line runs as one container of your image. Maestro starts the image's
`ENTRYPOINT` in its `WORKDIR`, with the line's words as arguments in place of
`CMD`. The container gets the cores and memory the line asked for. When it
ends, Maestro keeps its logs and outputs and removes it.

## What Maestro tells the program

Maestro sets these variables in every container, after those of
[`env`](maestro-toml.md#env):

| Variable | Value |
| --- | --- |
| `MAESTRO_RUN` | The run's number |
| `MAESTRO_LINE` | The line's number in the run, starting at 1 |
| `MAESTRO_CPUS` | The line's `cpus` |
| `MAESTRO_MEMORY` | The line's `memory`, in MB |
| `MAESTRO_TIME` | The seconds the line may run, from its `time` or the server's limit. `0` means no limit. |
| `MAESTRO_OUTPUTS` | The line's `outputs`, one path on each text line |

## How many threads to start

A limit on cores doesn't change how many cores a program sees. `nproc`,
`os.cpu_count()` and most libraries still report every core of the machine.
A program that starts one thread for each core it sees starts far more
threads than it has cores to run them on, and gets slower.

Read `MAESTRO_CPUS` and start that many. The template's `knapsack` branch
does it for its solver:

```python
import os

# As many threads as the container has cores; outside Maestro, 0 lets the
# solver choose.
model.threads = int(os.environ.get("MAESTRO_CPUS", "0"))
```

Most solvers have an option for this, such as Gurobi's `Threads`, CPLEX's
`threads` and `num_workers` in OR-Tools. Programs built on OpenMP read
`OMP_NUM_THREADS`, which a small start script can set from `MAESTRO_CPUS`
before it starts the program.

## What happens when memory runs out

When a container takes more memory than its `memory`, the kernel kills it.
The line ends `failed`, usually with exit code 137. Ask for what the program
needs at its largest, plus some room. A smaller request starts sooner, but a
line that is killed has to run again from the start.

## Machines that can't limit cores

Some machines can't enforce the limit on cores. This happens when Podman
runs without root and the system doesn't give it the `cpu` cgroup
controller, which is the default on Ubuntu 22.04. On such a machine a line
still counts as using its `cpus`, and `MAESTRO_CPUS` is still set, but a
program that starts more threads does take more cores and slows the lines
beside it. An administrator can fix the machine once, as
[Adding machines](../admin/machines.md#holding-lines-to-their-cores)
describes.

## Files and logs

Write results to the paths that `outputs` names. Everything else in the
container is gone when it ends.

Maestro keeps what the program prints as `stdout.log` and `stderr.log`, and
`maestro runs logs <run> <line> -f` shows it while the line runs. Python
buffers its output when it isn't writing to a terminal. Start it with `-u`,
set `PYTHONUNBUFFERED=1` in the image, or print with `flush=True`.

Maestro keeps up to 1 GB of logs and outputs for each line, unless your
administrator set another limit. It drops the rest and notes that on the
line.

## How the exit code becomes a status

A line is `succeeded` when its program exits with 0 and `failed` when it
exits with anything else. `maestro runs show <run> --failed` lists the lines
that didn't succeed.

A line that ends `error` never ran to the end for a reason outside your
program. Most often it couldn't start on its machine, or the machine
restarted while it ran and the container was lost. Run it again.
