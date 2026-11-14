# Adding machines

A machine is a host whose Podman runs users' containers. orq reaches it over
SSH and talks to the Podman socket through that connection, so the machine
needs nothing but sshd and Podman's API socket.

## Preparing a machine

On the machine, as the user the containers will run as:

```sh
systemctl --user enable --now podman.socket
loginctl enable-linger "$USER"     # keep the socket up without a session
ls /run/user/$(id -u)/podman/podman.sock
```

Make an SSH key pair without a passphrase for orq, and add its public half to
that user's `~/.ssh/authorized_keys`.

The user should be allowed to run commands over SSH, not only reach the
socket: orq reads how loaded the machine is with a small shell script
(`/proc`, `df`, `nvidia-smi`). Without it the machine still runs lines, but
`maestro top` has nothing to show for it but its containers.

### Holding lines to their cores

On a machine whose Podman can't limit CPU (a rootless one whose user systemd
doesn't hand down the `cpu` cgroup controller, which is Ubuntu 22.04's
default) lines still count their `cpus` there, but nothing holds them to it:
one that takes more slows the others down. To have the limit, give users
the controller once:

```sh
sudo mkdir -p /etc/systemd/system/user@.service.d
printf '[Service]\nDelegate=cpu cpuset io memory pids\n' |
  sudo tee /etc/systemd/system/user@.service.d/delegate.conf
sudo systemctl daemon-reload
```

and have the Podman user sign out of every session (or restart the machine),
so that its systemd starts again with it. `podman info` then lists `cpu`
under `cgroupControllers`.

## Adding it to Maestro

On the Maestro page, under Admin › Machines, give:

- the host, SSH port and user;
- the socket path, e.g. `/run/user/1000/podman/podman.sock`;
- the private key;
- how many cores and how much memory to keep back for the machine's own
  system;
- the host key, or nothing (see below).

Saving connects and asks Podman for its version, cores and memory, so only a
working machine is saved, and a reserve that leaves nothing for lines is
refused. orq asks again whenever it checks the machine, so a machine that
grows or shrinks is seen.

Leave the host key empty to pin the one the machine presents then, and
compare the fingerprint shown with
`ssh-keygen -lf /etc/ssh/ssh_host_ed25519_key.pub` on the machine; or paste
the line from `ssh-keyscan -t ed25519 <host>`.

Private keys are stored encrypted with `MACHINE_SECRET` and never sent back.
`MACHINE_SECRET` must be at least 32 characters (`openssl rand -base64 32`);
empty, machines are off. Changing it makes the stored keys unreadable.

## What orq does on a machine

- **Copies images** the first time a line needs one there, streaming
  `podman save` into `podman load` over SSH, gzipped, and leaves them there
  for the next run.
- **Starts containers** with the line's cores, memory, environment and a
  process limit (`RUN_PIDS`, 512 by default), with Podman's file log driver
  (`k8s-file`), whatever the machine's default: under journald, the user
  Podman runs as may not be allowed to read the journal, and the logs would
  come back empty.
- **Takes back** each container's logs and outputs when it ends, then removes
  it.
- **Sweeps** the machine when orq starts, every 10 minutes, and whenever an
  image goes. Everything orq makes is labelled with a name it gives itself
  the first time it starts (`maestro.orq`): its images, which also carry
  their build, and its containers, which also carry their run and line. A
  container of its own that no line has there is removed, and so is an image
  of its own that orq no longer has and no line needs there. Nothing else on
  a machine is touched: neither the machine's own images and containers nor
  those of another orq that uses it too.

A machine orq can't reach gets no lines until it can: those waiting or about
to start there go back to `staged`, for another machine, and orq tries it
again every 30 seconds. Lines already running there are left alone.

orq doesn't keep connections open: the runner, a check, and `maestro top`
(one every couple of seconds while someone watches it) each open their own.

## Who sees what

Users see the machines and whether orq can reach them (`maestro machines`),
how loaded each is (`maestro top`), and the containers of their own runs.
Administrators see every container, and the server orq runs on, as `orq`.
