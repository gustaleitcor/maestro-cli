# When something goes wrong

## Signing in

**"no Maestro server is set".** The CLI has no default server. Run
`maestro login --server https://maestro.example.org` once, or pass
`--server`, or set `MAESTRO_ORQ_URL`. On a terminal, the CLI asks instead.

**"…sign in there".** The command is pointed at a server other than the one
your key was made by (by `--server` or `MAESTRO_ORQ_URL`). Run `maestro
login --server <that server>`, which forgets the key for the other.

**The keyring asks for a password, or there is none.** Sign in with
`maestro login --store-in-file`.

## Building

**The repo isn't found, or can't be cloned.** Check `maestro repos` lists
it. A GitHub token must be a classic one with the `repo` scope; GitHub and
GitLab tokens expire in 30 days by default, and `maestro forges add <name>`
replaces one. A self-hosted forge must be in the server's
`ALLOWED_FORGE_HOSTS`.

**The build fails.** The end of its log says why. The usual causes:

- the Dockerfile itself fails;
- `maestro.toml` is invalid, or has a key Maestro doesn't know (a misspelt
  one included);
- the generator raised, yielded nothing or something that isn't a string,
  gave more than 1000 lines, or set a `Config` key that isn't `outputs`,
  `cpus`, `memory` or `env`;
- the generator imports a package that isn't in Python's standard library,
  such as numpy: it runs inside the server, not in your image. See
  [Computing lines with a generator](../run/generators.md).

**The Dockerfile can't find a file.** The build's context is the root of the
repo, even with `dockerfile = "docker/Dockerfile"`: `COPY` names files from
the root.

**It can't ask git for the commit.** The clone has no `.git`, so that the
forge token doesn't end up in the image.

## Running

**The run is refused as too large.** One of its lines asks for more `cpus`
or `memory` than any machine it may use has for lines, even empty. Ask for
less, or allow more `machines`; `maestro machines` shows what each has.

**"the image of build #N was removed".** You keep a few images at a time,
and the one run least recently went to make room for a newer build. Build
the commit again.

**Lines wait for long.** See [Why is my line
waiting?](../watch/queue.md#why-is-my-line-waiting).

**A line ends `failed`.** Its program exited with an error: `maestro runs
logs <run> <line>` says why. Exit code 137 means it was killed for taking
more memory than its `memory`.

**A line ends `error`.** It couldn't start on its machine, or lost its
container when the machine restarted. That is the server's trouble; the
Maestro page says what happened, in the line's status. Run it again, and
tell your administrator if it keeps happening.

**The program is slow, or slows others down.** It may start a thread per
core of the machine rather than per core it was given; size it by
`MAESTRO_CPUS`. See [How many threads](../run/container.md#how-many-threads).

## Results

**An output is listed as missing.** The line didn't write it where
`maestro.toml` says. A relative path is from the image's `WORKDIR` (`/`
without one).

**Logs are empty until the line ends, or come in bursts.** The program
buffers what it prints. In Python, set `PYTHONUNBUFFERED=1` in the image or
print with `flush=True`.

**Logs or outputs end early, with a note that they were cut.** A line keeps
at most 1 GB of logs and outputs (`RUN_OUTPUT_MAX_MB`).

## Machines (administrators)

**A machine shows as unreachable.** Its error says why: SSH refused, the host
key changed, the Podman socket isn't there (`systemctl --user enable --now
podman.socket`, and `loginctl enable-linger` so it stays up), or a proxy
command while `MACHINE_PROXY_COMMANDS` is off. Lines waiting there go to
other machines, and orq tries it again every 30 seconds.

**A machine runs lines but `maestro top` shows nothing for it.** Its SSH user
may only reach the Podman socket, not run commands; the machine says so.

**Lines take more cores than they asked for.** The machine's Podman can't
limit CPU; see [Holding lines to their
cores](../admin/machines.md#holding-lines-to-their-cores).

**Logs came back empty on an older orq.** orq now creates containers with
Podman's file log driver, whatever the machine's default; logs of runs from
before that, on a machine whose default is journald, may be empty.
