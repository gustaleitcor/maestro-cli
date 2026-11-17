# Comparing policies offline

`sim` is a program in the maestro-orq repository. It runs orq's own
scheduler over a workload on a simulated clock, so you can compare policies
without using the machines. It also measures how lines were served, whether
they came from a simulation or from a real Maestro. It can replay the lines
of a real week, to check how closely the simulator follows your machines.

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
{"dispatcher": "pq", "share_weight": -1}
```

Without `-config`, every queue is FIFO. The names are the settings of
[Policies and the score](../how/score.md):

| Kind | Names |
| --- | --- |
| Policies | `user`, `dispatcher`, `machine` |
| Score | `cpu_weight`, `memory_weight`, `aging_weight`, `share_weight`, `horizon_minutes` |
| Share | `share`, `half_life_hours` |
| The machine queue and the draw | `depth`, `seed` |

`-user-max-cpus` is the ceiling on the cores each user holds, as "Cores
per user" sets it on the Maestro page. It is 0, no ceiling, by default.
`sim run`, `sim replay` and `sim grid` all take it, and `sim grid` takes one
value.

A file from before the three queues still works. Its `select` is read as
`dispatcher`, and its `allocate` and `staged_capacity` are ignored.

A simulated line starts the moment it is chosen. A real line first waits
for its image to be copied and its container to be created, which the
simulation leaves out.

## Measuring

`sim evaluate` reads lines from `sim run` or from
[`maestro export`](../use/usage.md#maestro-export) and reports:

- the mean wait, and how much of it was spent queued and assigned
- the mean time in the system
- the mean bounded slowdown
- the utilization of cores and of memory
- Jain's index of the users' slowdowns
- the objective Z = F + λ_J·Φ_J + λ_O·Φ_O

An export from before the three queues has a `staged` column, which
`sim evaluate` skips. Its lines' time queued then runs to when they were
assigned.

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

## Comparing two files of lines

`sim compare` measures two files of lines, as `sim evaluate` does, and
prints them side by side. The last column is the second file's value less
the first's.

```sh
go run ./cmd/sim compare -cpus 30 -memory 61440 week.csv replayed.csv
```

```text
MEASURE             week.csv   replayed.csv  DIFFERENCE
lines               296        296           0
wq                  13535.196  13125.912     -409.284
queued              12677.507  12269.348     -408.159
...
```

It takes the same flags as `sim evaluate`. Times are in seconds. The
utilizations, Φ_J and Φ_O are fractions between 0 and 1.

## Running every combination

`sim grid` runs one workload under many settings and writes a CSV with one
row for each. You give it lists, separated by commas, and it runs every
combination of them:

| Flag | A list of |
| --- | --- |
| `-user` | user queue policies |
| `-dispatcher` | dispatcher queue policies |
| `-machine` | machine queue policies |
| `-share` | share measures |
| `-seed` | seeds |

A list you leave out keeps the value in the `-config` file. It also takes
`-machines`, `-config`, `-tick` and `-user-max-cpus` as `sim run` does, and `-theta`,
`-lambda-j` and `-lambda-o` as `sim evaluate` does.

```sh
go run ./cmd/sim grid -machines Q1:32:65536:2:4096 -config base.json \
  -dispatcher fifo,pq -share drf,decay -seed 1,2,3,4,5 < workload.csv > grid.csv
```

This runs 2 × 2 × 5 = 20 simulations. Each row starts with the settings
`user`, `dispatcher`, `machine`, `share` and `seed`. Every measure
`sim compare` names follows, under the same name. The cores and memory that
utilization is measured against are what the machines have for lines.

On the same commit, the same arguments write the same file, byte for byte.
The simulations run at once, but the rows always come in the same order:
the user policy changes slowest and the seed fastest.

## Checking the simulator against a real week

`sim replay` reads an export and runs its lines again. Each line that ran
becomes a line of the workload, with its user, group, cores and memory.
It arrives when it was submitted, and it runs as long as it ran, from
started to finished. A line that never ran is left out, and `sim replay`
says how many it left out. It takes the same flags as `sim run`.

To see how closely the simulator follows your machines:

1. Pick a week in which nobody changed the scheduler's settings. The
   `policy_set` column of the export says which settings each line was
   assigned under, so every line of the week should have the same one.
2. Export the week:

   ```sh
   maestro export --all --since 2026-11-02 --until 2026-11-09 -o week.csv
   ```

3. Write the machines as they were that week, with what each kept back,
   for `-machines`. Save the scheduler's settings from that week as a JSON
   file for `-config`, and give its cores per user to `-user-max-cpus`.
4. Replay the week, and compare the replay with what happened:

   ```sh
   go run ./cmd/sim replay -machines Q1:32:65536:2:4096,Q2:32:65536:2:4096 -config week.json < week.csv > replayed.csv
   go run ./cmd/sim compare -cpus 60 -memory 122880 week.csv replayed.csv
   ```

The smaller the differences, the better the simulator predicts your
machines. Some differences come from what the simulation leaves out:

- A real line waits for its image to be copied and its container to be
  created before it starts. A simulated line starts at once.
- The export doesn't say which machines a line was allowed to use, so the
  replay lets each line use any machine.
- A machine that was down for part of the week is up for all of it in the
  replay.
- siro and the choice among machines with room draw at random. The seed
  repeats the simulation's draws, but not the real cluster's.
- Two runs of one user submitted within the same second become one run.
  That changes only F, which is measured over runs.

With `@speed`, `sim replay` multiplies a line's time by the speed of the
machine it ran on, so that the line runs as long again on that machine.

Maestro's tests check that a simulation's own lines, replayed on the same
machines with the same settings and seed, give every line the same wait.
A difference between a replay and a real week is something the simulator
doesn't model.
