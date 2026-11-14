# Comparing policies offline

`sim`, in maestro-orq, runs orq's own scheduler over a workload on a
simulated clock, so policies can be compared without the machines, and
measures how lines were served, simulated or real.

```sh
cd maestro-orq
go run ./cmd/sim generate -users 4 -groups 2 -hours 8 > workload.csv
go run ./cmd/sim run -machines Q1:32:65536:2:4096 -config drf.json < workload.csv > lines.csv
go run ./cmd/sim evaluate -cpus 30 -memory 61440 -lambda-j 1h -lambda-o 1h < lines.csv
```

## A workload

`generate` writes one: runs of lines submitted by users over some hours. A
workload's `group` column puts each user in a group (1 without it).

## Machines

`-machines` lists them, comma-separated, as
`name:cpus:memory[:reserve_cpus:reserve_memory][@speed]`, memory in MB:
`Q1:32:65536:2:4096` has 32 cores and 64 GB, of which 2 cores and 4 GB are
kept back. A machine written with `@speed`, as `Q2:8:16384@1.5`, runs lines
that much faster: a line's duration depends on the machine.

## A config

The scheduler's settings as JSON, as `GET /api/settings` shows them under
`scheduler`; for instance

```json
{"select": "pq", "share_weight": -1}
```

for DRF. Without one, FIFO everywhere. The names are those of [Scheduler
policies](policies.md): `user`, `select`, `allocate`, `machine`,
`cpu_weight`, `memory_weight`, `aging_weight`, `share_weight`,
`horizon_minutes`, `share`, `half_life_hours`, `staged_capacity`, `depth`,
`seed`.

A simulated line starts as soon as it is chosen; a real one first has its
image copied and its container created.

## Measuring

`evaluate` reads lines, from `sim run` or from [`maestro
export`](../watch/usage.md), and reports:

- the mean wait, and how much of it at each level;
- the mean time in the system;
- the mean bounded slowdown;
- the utilization of cores and memory;
- Jain's index of users' slowdowns;
- the objective Z = F + λ_J·Φ_J + λ_O·Φ_O.

F is the mean time from a run's submission to the end of its last line;
Φ_J the mean over time of the gap between the largest and the smallest
dominant share of the users with lines not finished; and Φ_O the mean over
time of the fraction of the machines idle while some line waits.
`-lambda-j` and `-lambda-o` say what a whole unit of each is worth.

The Maestro page's Usage, under "Lines over a period", measures a real
period the same way, against what the machines have for lines now.
