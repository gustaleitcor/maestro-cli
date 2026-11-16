# Usage and export

```sh
maestro usage                            # what your lines used, over 30 days
maestro usage --daily --days 7           # a row for each of the last 7 days
maestro export --since 7d -o lines.csv   # what happened to each line, as CSV
```

## `maestro usage`

```text
USER  CORE-HOURS  GB-HOURS  DAYS
Ada   54.8        187.5     3

Over the last 30 day(s), counted as lines end.
```

`maestro usage` adds up the core-hours and memory GB-hours your lines used.
A line counts when it ends, with what it asked for and how long it ran. A
line with 4 cores that ran for 30 minutes is 2 core-hours.

`--days` sets how far back to look, and `--daily` prints a row for each day:

```text
USER  DAY         CORE-HOURS  GB-HOURS
Ada   2026-11-14  12.5        25.0
Ada   2026-11-15  2.0         1.5
Ada   2026-11-16  40.2        161.0
```

Administrators see every user, or one user with `--user <email>`. The
Maestro page charts the same numbers under Usage.

Usage can count against you in the queue. An administrator can set the
scheduler to weigh what each user used recently when it decides whose line
goes next, with older usage counting less. This is the `decay` measure in
[Policies and the score](../how/score.md#a-users-share).

## `maestro export`

`maestro export` writes a CSV with one row for each line of the runs you
queued over a period. A row has the line's run and number, its cores and
memory, the machine it ran on, its status, and the times it was submitted,
staged, assigned, started and finished. It also records the scheduler
settings the line was staged under and the group you were in then.

```sh
maestro export --since 2026-11-01 --until 2026-11-08 > lines.csv
maestro export --since 30d -o month.csv
```

`--since` and `--until` take a day, a full timestamp, or an age such as
`12h`, `7d` or `2w`. Without `--since` the export starts a week ago, and
without `--until` it ends now. Administrators export everyone's lines with
`--all`.

`-o` writes the file only after the export succeeded, so a failed export
leaves no partial file behind.

The export answers the question of how well the queue served you. The Usage
page's "Lines over a period" section downloads the same file and measures
it. An administrator comparing policies can measure it from a terminal with
`sim evaluate`, as [Comparing policies offline](../admin/sim.md) describes.
