# Watching the machines

```sh
maestro machines         # every machine, and whether the server can reach it
maestro top              # how loaded every machine is, refreshed as you watch
maestro top Q1           # one machine in detail
maestro machines stats   # the same table once (--json for scripts)
```

## Listing the machines

```text
NAME  STATUS       CPUS         MEMORY  PODMAN  DESCRIPTION
Q1    ready        16 (2 kept)  64G     5.2.1   the big one
Q2    ready        8            16G     4.9.3
Q3    unreachable  -            -       -
```

`maestro machines` lists the machines an administrator added. The names are
the ones [`machines`](../repo/maestro-toml.md#machines) takes in
`maestro.toml`. `16 (2 kept)` means the machine has 16 cores and keeps 2 of
them back for its own system, so lines can use 14.

The list answers at once, because the server reports what it last saw of
each machine and doesn't open a connection while you wait. If that reading
is older than 30 seconds, or older than 10 for a machine it couldn't reach,
the server checks the machine again in the background, and the next list
has the new status.

## `maestro top`

`maestro top` is a full-screen view in the manner of `top` or `btop`, with
one row for each machine: CPU, memory, GPUs, load, and under `CORES` how
much of the machine Maestro's lines are using.

`6/14 +3` under `CORES` means lines are using 6 of the 14 cores the machine
has for them, and 3 more lines are waiting there. The machine's own page
shows the same for memory.

Press enter on a machine, or name it as in `maestro top Q1`, to see each of
its cores, its swap, disks, network, GPU memory and temperatures, and the
containers running on it. Each container shows whose it is and the
repository it was built from. The view refreshes every 2 seconds, and `-n`
sets another interval of at least 1 second.

Everyone who is signed in sees the machines and the containers of their own
runs. Administrators see every container. They also see the server orq runs
on, listed as `orq`.

## Where the numbers come from

The server reads a machine by running one small shell script over SSH. The
script reads `/proc` and `df`, and `nvidia-smi` when the machine has it. The
server asks Podman for the containers' usage. GPUs are read only when they
are NVIDIA's and the machine runs Linux.

Some machines let the server's SSH user reach the Podman socket and nothing
else. Such a machine still appears with its containers, and says why it has
no other numbers.

The server keeps each reading for 2 seconds and shares it among everyone
watching. However many `maestro top` windows are open, each machine gets one
SSH connection every couple of seconds. The cores lines take and the lines
waiting come from the server's own records, so they are always current.
