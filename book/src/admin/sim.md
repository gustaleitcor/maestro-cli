# Comparing policies offline

`sim` is a program in the maestro-orq repository. It runs orq's own
scheduler over a workload on a simulated clock, so you can compare policies
without using the machines. It also measures how lines were served, whether
they came from a simulation or from a real Maestro.

```sh
cd maestro-orq
go run ./cmd/sim generate -users 4 -groups 2 -hours 8 > workload.csv
go run ./cmd/sim run -machines Q1:32:65536:2:4096 -config drf.json < workload.csv > lines.csv
go run ./cmd/sim evaluate -cpus 30 -memory 61440 -lambda-j 1h -lambda-o 1h < lines.csv
```

## A workload

`sim generate` writes a workload: runs of lines that users submit over a
number of hours. The `group` column puts each user in a group. A workload
without that column has every user in group 1.

## The machines

`-machines` takes a list separated by commas. Each machine is written
`name:cpus:memory[:reserve_cpus:reserve_memory][@speed]`, with memory in MB.

| Example | Meaning |
| --- | --- |
| `Q1:32:65536:2:4096` | 32 cores and 64 GB, with 2 cores and 4 GB kept back |
| `Q2:8:16384@1.5` | 8 cores and 16 GB. Lines run 1.5 times as fast on it. |

With `@speed`, how long a line runs depends on the machine it lands on.

## The scheduler's settings

`-config` names a JSON file with the scheduler's settings, in the form
`GET /api/settings` returns them under `scheduler`. This one is DRF:

```json
{"select": "pq", "share_weight": -1}
```

Without `-config`, every level is FIFO. The names are the settings of
[Policies and the score](../how/score.md):

| Kind | Names |
| --- | --- |
| Policies | `user`, `select`, `allocate`, `machine` |
| Score | `cpu_weight`, `memory_weight`, `aging_weight`, `share_weight`, `horizon_minutes` |
| Share | `share`, `half_life_hours` |
| Limits and the draw | `staged_capacity`, `depth`, `seed` |

A simulated line starts the moment it is chosen. A real line first waits
for its image to be copied and its container to be created, which the
simulation leaves out.

## Measuring

`sim evaluate` reads lines from `sim run` or from
[`maestro export`](../use/usage.md#maestro-export) and reports:

- the mean wait, and how much of it was spent at each level
- the mean time in the system
- the mean bounded slowdown
- the utilization of cores and of memory
- Jain's index of the users' slowdowns
- the objective Z = F + λ_J·Φ_J + λ_O·Φ_O

The objective has three parts.

| Part | What it is |
| --- | --- |
| F | The mean time from a run's submission to the end of its last line |
| Φ_J | The gap between the largest and the smallest dominant share, among users who have unfinished lines, averaged over time |
| Φ_O | The fraction of the machines that is idle while some line waits, averaged over time |

`-lambda-j` and `-lambda-o` say how much time one whole unit of Φ_J and of
Φ_O is worth.

The Usage page on the Maestro page measures a real period the same way,
under "Lines over a period". It uses what the machines have for lines at
the moment you ask.
