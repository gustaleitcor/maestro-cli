# Policies and the score

[The queue](queue.md) described the three queues a line passes on its way
to a machine. A policy picks who leaves each one. An administrator
picks the policies, and `maestro settings` shows the ones in force.

## The three policies

| Setting | The choice it makes | Queue |
| --- | --- | --- |
| `user` | Which of one user's queued lines is that user's head | User queue |
| `dispatcher` | Which of a group's heads goes to a machine in the group's turn | Dispatcher queue |
| `machine` | Which of the lines waiting on a machine starts | Machine queue |

All three are `fifo` until an administrator changes them. Lines then go in
the order they were queued, oldest run first, and on a machine the oldest
waiting line holds back the others until it fits.

## What a policy can be

The `user` and `dispatcher` policies choose before a line is on a machine.
At that point there is no free space to fit the line into, so these
policies only put lines in an order.

| Policy | Serves first |
| --- | --- |
| `fifo` | The oldest |
| `lcfs` | The newest |
| `siro` | One at random. The same `seed` gives the same choices. |
| `pq` | The one with the highest [score](#the-score). Among equal scores, the oldest. |

The `machine` policy can be any of those four, and it also has four that
look at the machine's free cores and memory.

| Policy | Serves first, among the lines that fit now |
| --- | --- |
| `fifo-skip` | The oldest |
| `best-fit` | The one that leaves the least free |
| `weighted-best-fit` | As `best-fit`, with the time a line has waited counting in its favour by ω3. A line that fits badly still starts eventually. |
| `tetris` | The one whose cores and memory are most like what the machine has free, so that less of either is stranded (Grandl et al., 2014) |

On a machine, `fifo` and `lcfs` are strict. They start their first choice
if it fits, and start nothing while it doesn't. `siro` and `pq` choose
among the lines that fit now, as the four policies in the second table do.

## The score

`pq` gives every candidate line a score and serves the highest.

> s = ω1·cores + ω2·memory + ω3·waiting + ω4·share

| Term | What it is |
| --- | --- |
| cores | The line's `cpus`, as a fraction of the cores all machines have for lines, summed |
| memory | The line's `memory`, as a fraction of the memory all machines have for lines, summed |
| waiting | The time since the line's run was queued, divided by the horizon. The horizon is an hour unless `horizon_minutes` says otherwise. |
| share | The user's share of the machines, as a fraction of every user's share added together |

The cores and memory are fractions of all the machines together, so they
are small when there are many machines. ω1 and ω2 then need to be larger
to weigh as much as waiting and share do.

An administrator sets the four weights, and a weight can be negative.

| Weight | Setting | Effect |
| --- | --- | --- |
| ω1 | `cpu_weight` | Below 0, lines that ask for few cores go first. |
| ω2 | `memory_weight` | Below 0, lines that ask for little memory go first. |
| ω3 | `aging_weight` | Above 0, a line's score grows while it waits, so no line waits forever. |
| ω4 | `share_weight` | Below 0, a user who holds less of the machines goes first. |

### A worked example

Take a lab whose machines have 32 cores for lines in all, with
`dispatcher = pq` and these weights: ω1 = −0.4, ω2 = 0, ω3 = 0.5 and
ω4 = −1. The horizon is an hour. Two users of a group each have a head.

| | Ada's line | Bo's line |
| --- | --- | --- |
| Cores asked for | 4, so cores = 0.125 | 8, so cores = 0.25 |
| Run queued | 30 minutes ago, so waiting = 0.5 | 6 minutes ago, so waiting = 0.1 |
| User's share | 0.75 | 0.25 |
| ω1·cores | −0.05 | −0.10 |
| ω3·waiting | +0.25 | +0.05 |
| ω4·share | −0.75 | −0.25 |
| Score | −0.55 | −0.30 |

Bo's line has the higher score, so it goes to a machine first. Ada's line has
waited longer and is smaller, but Ada already holds three times what Bo
holds, and with these weights the share outweighs the rest.

Giving Bo's line a machine raises Bo's share, because an assigned line
counts as held. The scheduler works the shares out again before the next
turn.

## A user's share

The `share` setting chooses how a user's share is measured.

| Measure | A user's share is |
| --- | --- |
| `drf` | The dominant share now. Take the cores the user's lines hold as a fraction of the cores the machines have for lines, and the same for memory. The share is the larger of the two fractions (Ghodsi et al., 2011). |
| `weighted-drf` | As `drf`, with each fraction divided by the user's weight for that resource. An administrator sets the weights for each user, and a user without any has 1 for both. |
| `asset` | The two fractions added together. It is there for comparison only, because it lacks the sharing-incentive property that `drf` has. |
| `decay` | The dominant share of what the user's lines have used in the past. Usage counts half as much after each half-life, which is a week unless `half_life_hours` says otherwise. Slurm's fair-share works this way. |

For `drf`, `weighted-drf` and `asset`, the lines a user holds are those that
are assigned, starting or running.

Suppose the machines of the lab in the example have 32 cores and 96 GB for
lines in total, and the measure is `drf`. Ada's lines hold 12 cores and
16 GB. That is
0.375 of the cores and 0.167 of the memory, so her dominant share is 0.375.
Bo's lines hold 3 cores and 12 GB, which is 0.094 and 0.125, so his is
0.125. The score uses each as a fraction of the two together, which gives
0.75 for Ada and 0.25 for Bo.

With `dispatcher = pq`, `share = drf` and ω4 = −1, the queue is the classic fair
queue of DRF, in which the user who holds the least goes next.

## The machine queue's length

| Setting | What it limits | Default |
| --- | --- | --- |
| `depth` | How many lines can wait on each machine | 3 |

A smaller `depth` leaves the choice of whose line goes next until closer to
the moment a machine has room.

When the dispatcher policy gives a line to a machine, the scheduler draws
the machine at random among those the line may use that have room in their
queue. It reserves nothing on the machine, and the machine policy decides
later whether the line fits. The `seed` setting makes that draw, and
`siro`, repeat the same choices.

## Seeing the settings

```text
images per user            10
cores per user             no ceiling
a user's head              fifo
among a group's heads      pq
on a machine               best-fit
waiting on a machine       3
score weights              cpu 0 · memory 0 · waiting 0.5 · share -1
waiting horizon            60 min
share measure              drf
half-life                  168 h
seed (siro, machine draw)  0
user weights               everyone 1
```

The three rows after the limits in `maestro settings` are the `user`,
`dispatcher` and `machine` policies, in that order.
[Choosing the policies](../admin/policies.md) shows an administrator where
to change them.
