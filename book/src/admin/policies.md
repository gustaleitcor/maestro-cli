# Choosing the policies

An administrator sets the scheduler up on the Maestro page, under Admin ›
Settings. [Policies and the score](../how/score.md) explains what each
policy and weight does.

A saved change applies from the scheduler's next pass, to lines that
haven't moved yet. Lines already assigned stay where they are.

## Starting from a preset

The Scheduler section has four buttons under "Start from". Each one fills
in the fields listed for it and leaves the others alone. Nothing is saved
until you press "Save changes", and Discard puts back what was saved last.

| Preset | What it sets | The queue it gives |
| --- | --- | --- |
| FIFO (baseline) | All three policies to `fifo`, all four weights to 0 | Lines go in the order they were queued. This is how a new Maestro starts. |
| DRF | `dispatcher = pq`, `share = drf`, ω4 = −1, the other weights 0 | The user who holds the least of the machines now goes next. |
| Fair-share | `dispatcher = pq`, `share = decay`, ω4 = −1, the other weights 0 | The user who used the least recently goes next. |
| Fair, with aging | `dispatcher = pq`, `share = drf`, ω3 = 1, ω4 = −1, the other weights 0 | As DRF, and a line gains on the others the longer it waits. |

## The sections of the page

### Scheduler

This section holds the policy of each of the three queues.

| Label | Setting | Can be |
| --- | --- | --- |
| User queue | `user` | `fifo`, `lcfs`, `siro`, `pq` |
| Dispatcher queue | `dispatcher` | `fifo`, `lcfs`, `siro`, `pq` |
| Machine queue | `machine` | Those four, and `fifo-skip`, `best-fit`, `weighted-best-fit`, `tetris` |

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
| Cores per user | `user_max_cpus` | 0, which means no ceiling |
| Days a run is kept | `run_retention_days` | 0, which means until someone deletes it, and at most 3650 |

Lowering "Images per user" removes nobody's images at that moment. A user
who is over the new number loses images at their next build, which removes
as many of the ones they ran least recently as it takes to make room. The
build is refused only when runs are still using all the images that are
left.

"Cores per user" is a ceiling on the cores each user holds. A user's lines
assigned, starting and running count toward it, and their queued lines
don't. In a group's turn, a user's head isn't a candidate while its cores,
added to those, would be over the ceiling. The heads of other users go as
usual. Lowering it stops no line. It only holds back a user's next lines
until enough of theirs end. A line that asks for more cores than the
ceiling never starts, and while it is its user's head, their lines behind
it wait too. `maestro queue` tells a user when their next line waits for
the ceiling.

"Days a run is kept" has orq delete each run, with its files, that many
days after its last line ended. A run with a line still waiting or running
doesn't expire, and neither does one its owner kept with
`maestro runs keep`. Lowering it deletes the runs it makes old within the
hour. [What orq keeps](retention.md#files-in-maestrodata) says how.

## Checking what is in force

`maestro settings` prints the saved settings for any user, and
`maestro queue` ends with a one-line summary of them.

Each line records the settings it was assigned under and the group its user
was in. An export that spans a change of settings therefore still says which
settings served which line.

Settings saved before orq had three queues had a `select` policy, which
chose among a group's users. orq reads it as the `dispatcher` policy, so a
`pq` there stays `pq`. Their `allocate` and `staged_capacity` no longer set
anything.

If orq can't read the saved settings after an upgrade, it falls back to
FIFO in every queue so that lines keep starting, and its log says why. Save
the settings again to fix it.

## Trying policies before choosing

[Comparing policies offline](sim.md) runs the same scheduler over a recorded
or generated workload, so you can compare two setups without waiting for the
lab to live through them.
