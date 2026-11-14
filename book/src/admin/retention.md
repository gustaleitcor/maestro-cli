# What orq keeps, and for how long

## The record: SQLite

orq keeps everything about builds, runs and lines in its database, so it can
restart in the middle of a run: lines it was about to start are started
again, containers it started are picked up where they are, and those that
finished while it was off are collected as if it had been there. A build it
was making when it stopped fails, so it doesn't count against the image
limit, and an image left of it is removed.

## Files: `/maestro/data`

What runs leave behind (logs and outputs) is kept under `DATA_DIR`, by
default `/maestro/data`, as `<user>/<run>/<line>/<file>`, until the run is
deleted. Each line keeps at most `RUN_OUTPUT_MAX_MB` (1024) of logs and
outputs. Nothing is deleted on its own: plan the volume for it, and ask
users to `maestro runs rm` what they no longer need.

## Images

Each user keeps at most 10 built images (Admin › Settings changes the
number). A build that would make one more removes the image they ran least
recently, skipping any a run is still using, from orq's Podman and from the
machines; its build's record, runs and files are kept.

## Answers kept a while: Redis

Lists are asked for often, by tab completion above all, so orq answers the
slow ones from Redis. Nothing here is the record of anything: with Redis
emptied, the next answer is just read again. Sessions are kept there too, so
emptying it signs everyone out of the page (not the CLI, whose keys are in
the database).

| What | Remembered for | Forgotten sooner when |
| --- | --- | --- |
| A user's images (`maestro builds`, completing `maestro run`) | 60s | a build of theirs ends, or one of their images is removed |
| Runs (`maestro runs`, completing a run, the page) | 2s | their owner, or an administrator, starts, cancels or deletes one |
| A machine's status | 10 min, looked at again behind an answer older than 30s (10s if unreachable) | the machine is edited |
| A machine's load (`maestro top`) | 2s | |

So what you do yourself shows at once, and what the runner does to a run (a
line starting or ending) within 2 seconds. An image removed from Podman by
hand, behind orq's back, can stay listed for up to a minute. Builds are one
quick query and aren't remembered at all. The CLI keeps nothing itself: a
completion still costs one request to orq.

## On the machines

Containers are removed once their logs and outputs are kept. Images stay, so
the next run of a build starts without copying it again, until orq removes
the image itself (to make room for a newer build, or by `maestro builds
rm`); its sweep then removes it from every machine where no line needs it.
See [Adding machines](machines.md#what-orq-does-on-a-machine).
