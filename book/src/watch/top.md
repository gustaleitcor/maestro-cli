# Watching the machines

```sh
maestro machines         # every machine, and whether the server can reach it
maestro top              # how loaded every machine is, as you watch
maestro top Q1           # one machine, in detail
maestro machines stats   # the table, once (--json for scripts)
```

## `maestro top`

`maestro top` shows how loaded every machine is, like `top` or `btop`, one
machine to a row: CPU, memory, GPUs, load, and under `CORES` how much of it
Maestro's lines are using.

`6/14 +3` reads: lines take six of the fourteen cores the machine has for
them (what it has, less what is kept back for its own system), and three
lines wait there. The machine's own page says the same of memory.

Press enter on a machine (or name it, `maestro top Q1`) for each core, swap,
disks, network, GPU memory, temperatures, and the containers running on it,
with whose each is and the repo it was built from. `-n` changes how often it
refreshes, every 2 seconds by default.

Everyone signed in sees the machines and the containers of their own runs.
Administrators see every container, and the server orq runs on, as `orq`.

## Where the numbers come from

The server reads a machine by running one small shell script over the SSH
connection it already has (`/proc`, `df`, and `nvidia-smi` when there is
one), and asks Podman for the containers' usage. Only NVIDIA GPUs are read,
and only on Linux. A machine whose SSH user may only reach the Podman socket
still shows, with its containers, and says why it has nothing else.

Readings are kept for 2 seconds and shared, so any number of open windows
costs each machine one SSH connection every couple of seconds. What lines
take and the queue come from the server's own records, and are never that
old.

## `maestro machines`

Lists the machines and whether the server can reach them. Checking opens an
SSH connection, so the server answers with what it last saw and never makes
a list wait for a machine: an answer older than 30 seconds (10 for a machine
it couldn't reach) is still given, and the machine is looked at again behind
it, so the next list has the news.
