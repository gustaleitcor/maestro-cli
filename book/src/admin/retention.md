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

### Backing up the database

The database is `MAESTRO_ORQ_DB`, which the compose file puts at
`/data/maestro-orq.db` in the `maestro-orq-data` volume.

When `DB_BACKUP_DIR` is set, orq backs the database up there on its own.
It writes the first copy a minute after it starts, and another every 24
hours after that. Each copy is named after the day it was made, in UTC,
such as `maestro-orq-2026-11-16.db`, and a second copy on the same day
replaces the first. orq writes it with SQLite's `VACUUM INTO`, which is safe
while orq runs. It keeps the 7 newest copies and removes the older ones.
`DB_BACKUPS_KEPT` changes the number, and `0` keeps every copy. The compose
file sets `DB_BACKUP_DIR` to `/backups`, in the `maestro-orq-backups`
volume. That volume is on the same server as the database, so copy its
files off the server too.

You can also back the database up yourself. orq opens the database in WAL
mode, so the latest writes sit in `maestro-orq.db-wal` beside it until
SQLite moves them into the database. A copy of `maestro-orq.db` alone can
miss them. Back the database up in one of two ways:

- While orq runs, use SQLite's own backup, which reads both files
  consistently: `sqlite3 maestro-orq.db ".backup maestro-orq-backup.db"`.
- Stop orq, then copy `maestro-orq.db` together with `maestro-orq.db-wal`
  and `maestro-orq.db-shm`, if they are there.

### Restoring a backup

1. Stop orq:

   ```sh
   podman compose stop orq
   ```

2. Put the database aside with its `-wal` and `-shm` files, and copy the
   backup in its place. Leave no `-wal` file of the old database beside
   the copy, because SQLite would read it as part of the copy.

   ```sh
   podman compose run --rm --entrypoint sh orq -c '
     mkdir -p /data/before-restore &&
     mv /data/maestro-orq.db* /data/before-restore/ &&
     cp /backups/maestro-orq-2026-11-16.db /data/maestro-orq.db'
   ```

3. Start orq:

   ```sh
   podman compose start orq
   ```

   If the copy was made by an older orq, orq brings the database up to
   date as it starts, as it does after an update.

The copy holds the database and nothing else. The logs and outputs of runs
aren't in it, and a restore leaves `/maestro/data` as it is. A restore
loses everything orq wrote to the database after the copy was made:

- Runs started after the copy aren't in it, and orq doesn't list them.
  Their files stay in `/maestro/data` and count toward their owners'
  `USER_DATA_MAX_GB`. orq gives their numbers to the next runs. Once orq
  is back, `maestro runs --all` shows the newest run it knows. Move the
  directories of runs numbered above it, `/maestro/data/<user>/<run>`, out
  of `/maestro/data` before new runs take those numbers.
- A line that was running when the copy was made, and ended after, is
  ended as `error`, with "the container disappeared from <machine>". Its
  container was removed after orq collected it, so orq can't collect it
  again.
- A run deleted after the copy is listed again, without its files.
- A line that was waiting when the copy was made starts again, even if it
  ran after the copy.
- Builds, machines, settings, groups and Maestro keys changed after the
  copy go back to what they were. A key issued after the copy stops
  working, and a key revoked after it works again.

## Files, in `/maestro/data`

orq keeps the logs and outputs of runs under `DATA_DIR`, which is
`/maestro/data` by default, as `<user>/<run>/<line>/<file>`. Each line
keeps at most `RUN_OUTPUT_MAX_MB` of logs and outputs, 1024 by default.

The files stay until the run is deleted. A user deletes a run with
`maestro runs rm`. orq deletes runs on its own too, once you set "Days a
run is kept" (`run_retention_days`) under Admin › Settings › Limits. It is
0 by default, which keeps every run until someone deletes it, so size the
volume for that.

With a number of days set, the runner looks once an hour for runs whose
last line ended more than that many days ago. It deletes each of them with
its lines, the record of its files and its directory under
`/maestro/data`, as `maestro runs rm` does, and its log names the run. Two
kinds of run never expire:

- A run with a line still waiting or running. Its days start when its last
  line ends.
- A run its owner, or an administrator, kept with `maestro runs keep`. It
  stays until someone deletes it, or lets it expire again with
  `maestro runs keep <run> --off`.

Lowering the number deletes the runs it makes old within the hour.
`maestro runs --all` shows, in its `EXPIRES` column, the runs that expire
within a week.

A run that a newer run retries expires on its own days. The retry has its
own copy of each line, of its machines and of the files sent from the CLI,
so it runs and keeps its files without the run it came from. Its
`retry_of` still names that run after it is gone.

An expired run's lines are gone from `maestro export`, as those of a run
deleted with `maestro runs rm` are. What each user used is kept apart, so
the scheduler's shares and `maestro usage` don't change.

`USER_DATA_MAX_GB` limits what all of a user's runs keep, and is off by
default. orq adds up the user's files when they start a run. Once they keep
that much or more, it refuses the run with "your runs keep 52 GB of the 50 GB
allowed; delete one with `maestro runs rm <run>`". orq checks only when a
run starts, so a run already started can take the user past the limit, by
at most `RUN_OUTPUT_MAX_MB` for each of its lines.

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
| Runs (`maestro runs`, completing a run, the page) | 2 s | Their owner or an administrator starts, cancels, keeps or deletes one |
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
