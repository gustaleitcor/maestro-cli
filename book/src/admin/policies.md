# Choosing the policies

An administrator sets the scheduler up on the Maestro page, under Admin ›
Settings. [Policies and the score](../how/score.md) explains what each
policy and weight does.

A saved change applies from the scheduler's next pass, to lines that
haven't moved yet. Lines already staged or assigned stay where they are.

## Starting from a preset

The Scheduler section has four buttons under "Start from". Each one fills
in the fields listed for it and leaves the others alone. Nothing is saved
until you press "Save changes", and Discard puts back what was saved last.

| Preset | What it sets | The queue it gives |
| --- | --- | --- |
| FIFO (baseline) | All four policies to `fifo`, all four weights to 0 | Lines go in the order they were queued. This is how a new Maestro starts. |
| DRF | `select = pq`, `share = drf`, ω4 = −1, the other weights 0 | The user who holds the least of the machines now goes next. |
| Fair-share | `select = pq`, `share = decay`, ω4 = −1, the other weights 0 | The user who used the least recently goes next. |
| Fair, with aging | `select = pq`, `share = drf`, ω3 = 1, ω4 = −1, the other weights 0 | As DRF, and a line gains on the others the longer it waits. |

## The sections of the page

### Scheduler

This section holds the policy of each level.

| Label | Setting | Can be |
| --- | --- | --- |
| Within a user | `user` | `fifo`, `lcfs`, `siro`, `pq` |
| Among a group's users | `select` | `fifo`, `lcfs`, `siro`, `pq` |
| A group's staged lines | `allocate` | `fifo`, `lcfs`, `siro`, `pq` |
| On a machine | `machine` | Those four, and `fifo-skip`, `best-fit`, `weighted-best-fit`, `tetris` |

It also holds "Waiting on a machine" (`depth`, 3 by default) and the seed
that makes `siro` and the draw of a machine repeat.

### Fairness

This section holds the share measure, the four weights of the score, the
waiting horizon in minutes and the half-life in hours.

It also holds the user weights that `weighted-drf` divides by. Add a row
for each user who should count differently from the rest, with the user's
number and a weight for cores and for memory. Everyone without a row has 1
for both. `maestro queue` shows an administrator each user's number, as in
`Ada #7`.

### Limits

| Label | Setting | Default |
| --- | --- | --- |
| Images per user | `images_per_user` | 10, and at most 1000 |
| Staged at most, per group | `staged_capacity` | 0, which means twice the number of machines |

Lowering "Images per user" removes nobody's images at that moment. A user
who is over the new number loses images at their next build, which removes
as many of the ones they ran least recently as it takes to make room. The
build is refused only when runs are still using all the images that are
left.

## Checking what is in force

`maestro settings` prints the saved settings for any user, and
`maestro queue` ends with a one-line summary of them.

Each line records the settings it was staged under and the group its user
was in. An export that spans a change of settings therefore still says which
settings served which line.

If orq can't read the saved settings after an upgrade, it falls back to
FIFO at every level so that lines keep starting, and its log says why. Save
the settings again to fix it.

## Trying policies before choosing

[Comparing policies offline](sim.md) runs the same scheduler over a recorded
or generated workload, so you can compare two setups without waiting for the
lab to live through them.
