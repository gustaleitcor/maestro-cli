# Scheduler policies

Every waiting line passes four decisions on its way to a machine (see [The
queue, explained](../watch/queue.md)), and each is made by a policy an
administrator chooses on the Maestro page, under Admin › Settings.
`maestro settings` shows what is chosen now.

| Level | Policy | Decides |
| --- | --- | --- |
| among a user's lines | **user** | which of a user's queued lines is next |
| among a group's users | **select** | whose next line is staged, among the users of a group |
| a group's turn | **allocate** | which of a group's staged lines goes to a machine in its turn |
| on a machine | **machine** | which line waiting on a machine starts |

Until an administrator chooses, each is FIFO: lines are taken in the order
they were queued, oldest run first, and on a machine a line that doesn't
fit yet holds back those after it.

## The policies

The first three choose before a line is on a machine, where there is
nothing it could fit in, so they only order:

| Policy | Serves first |
| --- | --- |
| `fifo` | the oldest |
| `lcfs` | the newest |
| `siro` | one at random; the same `seed`, the same choices |
| `pq` | the highest score (below); the oldest of equals |

The machine's policy fits lines in what the machine has free, so it may also
be:

| Policy | Serves first, among the lines that fit now |
| --- | --- |
| `fifo-skip` | the oldest |
| `best-fit` | the one that leaves the least free |
| `weighted-best-fit` | best-fit, with how long a line has waited counting for it (by ω3), so that no line is passed over forever for fitting badly |
| `tetris` | the one whose cores and memory are most like what the machine has free, so that less is stranded (Grandl et al., 2014) |

## The score

`pq` serves the line with the highest score

> s = ω1·cores + ω2·memory + ω3·waiting + ω4·share

where *cores* and *memory* are the line's, as a fraction of the largest
machine's; *waiting* is how long since its run was queued, over a horizon
(`horizon_minutes`, an hour by default); and *share* is its user's share of
the machines, over everyone's. Weights may be negative: ω1 below 0 serves
small lines first, ω3 above 0 ages lines so none waits forever, and ω4 below
0 serves whoever holds less of the machines first.

## A user's share

What *share* measures is chosen too:

| Measure | A user's share is |
| --- | --- |
| `drf` | the dominant share now: the larger, of cores and memory, of the fraction of what the machines have for lines that the user's lines take (those starting or running, staged or assigned) |
| `weighted-drf` | DRF, with each resource's fraction over the user's weight for it, set per user |
| `asset` | the two fractions added up instead of the larger; for comparison only, as it isn't sharing-incentive |
| `decay` | fair-share: the dominant share of what the user's lines have used, each hour counting half as much a half-life later (`half_life_hours`, a week by default), as Slurm's fair-share does |

DRF, with `select = pq` and ω4 = −1, is the classic fair queue: whoever holds
the least goes next.

## Capacities

- **Staged per group** (`staged_capacity`): how many lines of each group may
  be staged at once; by default twice the machines.
- **Depth** (`depth`): how many lines may wait on each machine; 3 by
  default. Shallower means whose line goes next is decided later, closer to
  when there is room.

A staged line goes to a machine drawn among those it may use that have room
in their queue; nothing is kept for it there.

## Other settings

Admin › Settings also holds the server's limits, such as how many built
images each user keeps (10 by default).

Each line records the settings it was staged under, and its user's group, so
an export over a period in which settings changed still tells which served
which line. To compare policies before choosing one, see [Comparing
policies offline](sim.md).
