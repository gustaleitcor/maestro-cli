# What orq keeps, and for how long

orq keeps four kinds of data: its record of builds and runs in SQLite, the
files runs leave, the images, and short-lived answers in Redis.

## The record, in SQLite

orq keeps everything about builds, runs and lines in its database, which
lets it restart in the middle of a run. When it comes back:

- it starts again the lines it was about to start
- it picks up the containers it had started, where they are
- it collects the containers that finished while it was down, as if it had
  been running

A build that orq was making when it stopped is marked failed, so it doesn't
count against the user's image limit. orq removes any image that build left
behind.

## Files, in `/maestro/data`

orq keeps the logs and outputs of runs under `DATA_DIR`, which is
`/maestro/data` by default, as `<user>/<run>/<line>/<file>`. Each line
keeps at most `RUN_OUTPUT_MAX_MB` of logs and outputs, 1024 by default.

The files stay until someone deletes the run. orq deletes none of them on
its own, so size the volume for that, and ask users to remove the runs they
no longer need with `maestro runs rm`.

## Images

Each user keeps at most 10 built images. Admin › Settings changes the
number. A build that would exceed it removes the image the user ran least
recently, skipping any image a run is still using. The image goes from
orq's Podman and from the machines. The record of its build, the runs of
that build and their files are all kept.

## Short-lived answers, in Redis

Lists are requested often, mostly by tab completion, so orq answers the
slow ones from Redis. Redis holds no record that exists nowhere else. If
you empty it, orq reads each answer again the next time it is asked.

Sessions of the Maestro page are in Redis too, so emptying it signs
everyone out of the page. The CLI stays signed in, because its keys are in
the database.

| Answer | Kept for | Dropped sooner when |
| --- | --- | --- |
| A user's images (`maestro builds`, completing `maestro run`) | 60 s | A build of theirs ends, or one of their images is removed |
| Runs (`maestro runs`, completing a run, the page) | 2 s | Their owner or an administrator starts, cancels or deletes one |
| A machine's status | 10 min. An answer older than 30 s, or 10 s if the machine was unreachable, triggers a new check in the background. | The machine is edited |
| A machine's load (`maestro top`) | 2 s | |

In practice, what a user does shows at once, and what the runner does to a
run, such as a line starting or ending, shows within 2 seconds. An image
removed from Podman by hand, without orq knowing, can stay listed for up to
a minute.

The list of builds is one fast query and isn't cached. The CLI caches
nothing either, so each tab completion costs one request to orq.

## On the machines

orq removes a container once it has the container's logs and outputs.

An image stays on a machine, so the next run of its build starts without a
copy. It goes when orq removes the image itself, either to make room for a
newer build or because of `maestro builds rm`. orq's sweep then removes it
from every machine where no line needs it.
[Adding machines](machines.md#sweeping) describes the sweep.
