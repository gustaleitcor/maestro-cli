# Fuller examples

The template's default branch is as small as a working repository can be.
Two other branches of the
[same repository](https://github.com/gustaleitcor/maestro-template) use more
of what the previous chapters covered. `--ref` builds a branch:

```sh
maestro build https://github.com/gustaleitcor/maestro-template --ref knapsack
```

Both branches name the machines of the lab they were written for in
`machines`. On another Maestro, a run of them is refused because those
machines don't exist. Fork the repository and remove the `machines` line, or
start the run with a `maestro.toml` of your own:
`maestro run <build> -c my.toml`.

## `knapsack`: a solver job

This branch solves knapsack problems with the
[mip](https://www.python-mip.com) solver, one container for each instance
and capacity.

| File | What it shows |
| --- | --- |
| `maestro.toml` | Two lines of `parameters`, a `generator`, `cpus`, `memory`, `time`, `retries`, an `env` variable, and `outputs_layout = "shared"` for one `results/` folder |
| `genvrs.py` | Reads every file in `instances/` and yields three lines for each, with the capacity scaled by 0.25, 0.5 and 2 |
| `main.py` | Solves one instance, with as many solver threads as `MAESTRO_CPUS` says, and stops searching before `MAESTRO_TIME` runs out |
| `requirements.txt` | The solver, installed by the Dockerfile |

The generator gives larger instances more than the rest. An instance with
more than 30 items gets 1024 MB and 2 cores through `Config` and 60 seconds
to search. A smaller one keeps the 1 core and 512 MB of `maestro.toml` and
gets 10 seconds. The generator also prints one line for each instance, which
shows up in the build log.

Every line may run for 2 minutes (`time = "2m"`). `main.py` reads that from
`MAESTRO_TIME` and ends its search 15 seconds before, so the line
`large --time-limit 120` stops on its own and still writes its best
solution:

```python
seconds = args.time_limit
if (limit := int(os.environ.get("MAESTRO_TIME", "0"))) > 0:
    seconds = min(seconds, max(1, limit - 15))
```

A solve can start over from the beginning, so each line has `retries = 2`
and goes back to the queue if its machine is lost.

Each line writes `results/<instance>-<capacity>.json`. With
`outputs_layout = "shared"`, every line's file goes into one tree for the
run, and `maestro runs get` puts them all in `<run>/shared/results/`. Each
line's logs stay in `<run>/<line>/`. The names differ by instance and
capacity, so no line overwrites another's.

With three instance files, a run has eleven lines: two from `parameters` and
nine from the generator.

## `stress`: a thousand lines

This branch is a stress test of the queue. Its containers do no useful work.
Each one sleeps, keeps its cores busy, fills part of its memory, or does
both, for between ten seconds and four minutes, and writes a `report.json`
with what it used.

`maestro.toml` lists four lines and `genvrs.py` adds 996, which makes 1000.
A run could have 5000, but ten runs of 1000 show more of how the queue takes
turns. The generator draws each line's size from ten shapes, from 1 core and
64 MB up to 10 cores and 10 GB, with a fixed seed so that every build yields
the same lines. One line in ten also keeps a `samples.csv`, and every
hundredth line exits with code 3 on purpose, so the run has failed lines to
look at.

Every line may run for 10 minutes (`time`), far more than any needs. Each
has `retries = 2`, so restarting a machine in the middle of a run sends its
lines back to the queue instead of ending them as `error`.

It is a good run to have going while you read
[The queue](../how/queue.md):

```sh
maestro queue                       # how many lines wait in each queue
maestro top                         # each machine as it fills
maestro runs show <run> --failed    # the lines that exited with 3
maestro runs retry <run>            # run them again
```
