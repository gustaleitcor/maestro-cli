# Adding machines

A machine is a host whose Podman runs users' containers. orq connects to it
over SSH and talks to the Podman socket through that connection. The
machine needs an SSH server and Podman's API socket, and no Maestro
software.

## Preparing a machine

Run this on the machine, as the user the containers will run as:

```sh
systemctl --user enable --now podman.socket
loginctl enable-linger "$USER"     # keep the socket up without a session
ls /run/user/$(id -u)/podman/podman.sock
```

Make an SSH key pair without a passphrase for orq, and add the public key
to that user's `~/.ssh/authorized_keys`.

Let the user run commands over SSH. orq reads how loaded a machine is with
a small shell script that uses `/proc`, `df` and `nvidia-smi`. A user who
may only reach the socket still runs lines, but `maestro top` then shows
the machine's containers and nothing else.

### Holding lines to their cores

Some machines can't limit how much CPU a container uses. This happens with
rootless Podman when the user's systemd doesn't pass down the `cpu` cgroup
controller, which is the default on Ubuntu 22.04. On such a machine lines
still count their `cpus` toward the machine's capacity, but a line that
uses more cores gets them and slows the others down.

To get the limit, delegate the controllers to users once:

```sh
sudo mkdir -p /etc/systemd/system/user@.service.d
printf '[Service]\nDelegate=cpu cpuset io memory pids\n' |
  sudo tee /etc/systemd/system/user@.service.d/delegate.conf
sudo systemctl daemon-reload
```

Then have the Podman user sign out of every session, or restart the
machine, so that the user's systemd starts again with the new setting.
`podman info` should now list `cpu` under `cgroupControllers`.

## Adding it to Maestro

On the Maestro page, open Admin › Machines and fill in:

- the host, the SSH port and the user
- the socket path, such as `/run/user/1000/podman/podman.sock`
- the private key
- how many cores and how much memory to keep back for the machine's own
  system
- the host key, or nothing (see below)

When you save, orq connects and asks Podman for its version and for the
machine's cores and memory. It saves the machine only if that works, and it
refuses a reserve that would leave nothing for lines. orq asks again every
time it checks the machine, so it notices when a machine gains or loses
cores or memory.

### The host key

If you leave the host key empty, orq pins the key the machine presents on
that first connection and shows its fingerprint. Compare the fingerprint
with the output of `ssh-keygen -lf /etc/ssh/ssh_host_ed25519_key.pub` on the
machine. You can instead paste the line that `ssh-keyscan -t ed25519 <host>`
prints.

### The secret that protects the keys

orq stores private keys encrypted with `MACHINE_SECRET` and never sends
them back to the page. `MACHINE_SECRET` must be at least 32 characters, and
`openssl rand -base64 32` makes a good one. While it is empty, machines are
turned off. If you change it, orq can no longer read the keys it stored.

## What orq does on a machine

### Copying images

The first time a line needs an image on a machine, orq copies it there by
streaming `podman save` into `podman load` over SSH, gzipped. The image
stays on the machine for the next run.

### Starting containers

orq starts each container with the line's cores, memory and environment,
and a limit on processes, which is `RUN_PIDS` and 512 by default.

It always uses Podman's file log driver (`k8s-file`), whatever the machine's
default is. Under journald, the user Podman runs as may not be allowed to
read the journal, and the logs would come back empty.

### Collecting results

When a container ends, orq takes back its logs and outputs and then removes
it.

### Sweeping

orq sweeps each machine when it starts, every 10 minutes, and whenever an
image is removed.

orq labels everything it creates with `maestro.orq`, set to a name it gave
itself the first time it started. Its images also carry their build, and
its containers carry their run and line. The sweep removes two things: a
container of orq's own that no line has on that machine, and an image of
orq's own that orq no longer has and no line needs there.

The sweep touches nothing else on the machine. The machine's own images and
containers stay, and so do those of another orq that uses the same machine.

Images and containers that an orq made before it labelled what it made have
no `maestro.orq` label. Every orq takes them for its own, as it did then.

## When a machine can't be reached

A machine orq can't reach gets no new lines. The lines waiting on it, or
about to start there, go back to `queued` so another machine can take them.
orq tries the machine again every 30 seconds. Lines already running on it
are left alone.

orq doesn't keep SSH connections open. The runner, a check of the machine,
and `maestro top` each open their own. `maestro top` opens one every couple
of seconds while someone is watching.

## Who sees what

Users see the list of machines and whether orq can reach each one
(`maestro machines`), how loaded each is (`maestro top`), and the containers
of their own runs. Administrators see every container. They also see the
server orq runs on, listed as `orq`.
