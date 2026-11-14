# Usage and export

```sh
maestro usage                         # core-hours and memory GB-hours, per day
maestro export --since 7d -o lines.csv
```

## `maestro usage`

The core-hours and memory GB-hours your lines used each day, counted as
they end: a line of 4 cores that ran for 30 minutes is 2 core-hours.
Administrators see every user, or one with `--user <email>`. The Maestro
page has the same under Usage.

## `maestro export`

`maestro export --since 2026-11-01` (or `--since 7d`) writes, as CSV, what
happened to each of your lines over a period: when it was submitted,
staged, assigned, started and finished, where, under which scheduler
settings and in which group. Administrators get everyone's with `--all`.
`-o` writes a file only once the export worked, so a failed one leaves no
half-written file behind.

It is the raw material for asking how well the queue served you, or for an
administrator to compare policies: the Maestro page's Usage, under "Lines
over a period", downloads the same file and measures it, and `sim evaluate`
does the same from a terminal. See [Comparing policies
offline](../admin/sim.md).
