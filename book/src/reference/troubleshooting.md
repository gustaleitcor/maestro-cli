# When something goes wrong

## Signing in

### "no Maestro server is set"

The CLI has no built-in server. In a terminal it asks for the address the
first time. Where it can't ask, as in a script, name the server one of
three ways: run `maestro login --server https://maestro.example.org` once,
pass `--server`, or set `MAESTRO_ORQ_URL`.

### A message that says to "sign in there"

`--server` or `MAESTRO_ORQ_URL` points the command at a server other than
the one that issued your key. Run the `maestro login --server` command the
message gives. It forgets the key you had for the other server.

### "this Maestro key expired; run `maestro login`"

A Maestro key works for 90 days after it was issued, unless your
administrator set another length (`KEY_TTL`). Run `maestro login` for a new
key. `maestro keys` shows when each of your keys expires.

### The keyring asks for a password every time, or there is no keyring

Sign in with `maestro login --store-in-file`, which keeps the key in the
CLI's config file.

## Building

### The repository isn't found, or can't be cloned

Check that `maestro repos` lists it. If it doesn't:

- On GitHub, the token must be a classic one with the `repo` scope.
- GitHub and GitLab tokens expire after 30 days by default.
  `maestro forges add` with the same name replaces an expired one.
- A self-hosted forge must be in the server's `ALLOWED_FORGE_HOSTS`, which
  an administrator sets.

### The build fails

The last lines of the build log say why. The usual causes are:

- a step of the Dockerfile fails
- `maestro.toml` is invalid, or has a key Maestro doesn't know, which
  includes a misspelt one
- the generator raised an exception, yielded nothing, yielded something
  that isn't a string, or brought the run to more than 1000 lines
- the generator set a `Config` attribute other than `outputs`, `cpus`,
  `memory` and `env`
- the generator imports a package outside Python's standard library, such
  as numpy

On the last cause: the package may well be installed in your image, but the
generator runs inside the server, where only the standard library exists.
See [Where and when the script runs](../repo/generator.md#where-and-when-the-script-runs).

### The Dockerfile can't find a file

The build's context is the root of the repository, even when `maestro.toml`
says `dockerfile = "docker/Dockerfile"`. Paths in `COPY` start at the root.

### A step that runs git fails

Maestro's clone has no `.git` directory, so that the forge token doesn't
end up in the image. A step such as `RUN git rev-parse HEAD` can't work.

### The build is refused because of the image limit

```text
you have 10 of the 10 images allowed (counting builds still running), and runs are using the rest; remove one with `maestro builds rm <build>`
```

A build normally makes room by removing the image you ran least recently.
It can't when runs are still using every image you have. Wait for a run to
end, or cancel one, and build again.

## Running

### The run is refused because a line is too large

```text
line 3 asks for 64 cpus and 512 MB, which no machine it may use has for lines even with nothing else on it (see `maestro machines list`)
```

The line asks for more `cpus` or `memory` than any machine it may use has
for lines. Ask for less, or allow more machines in `machines`.
`maestro machines` shows what each machine has.

### "the image of build #12 was removed; build it again"

You keep a limited number of images, and this one went to make room for a
newer build, or you removed it with `maestro builds rm`. Build the commit
again.

### Lines wait for a long time

See [Why a line is waiting](../how/queue.md#why-a-line-is-waiting).

### A line ends `failed`

Its program exited with a code other than 0. `maestro runs logs <run>
<line>` shows what it printed. Exit code 137 usually means the kernel
killed it for using more memory than its `memory`.

### A line ends `error`

The line couldn't start on its machine, or it lost its container when the
machine restarted. The cause is on the server's side, and the line's status
on the Maestro page says what happened. Run it again, and tell your
administrator if it keeps happening.

### The program is slow, or slows down other people's lines

It probably starts one thread for each core of the machine, when it should
start one for each core it was given. Size it from `MAESTRO_CPUS`, as
[How many threads to start](../repo/container.md#how-many-threads-to-start)
describes.

## Results

### An output is listed as missing

The program didn't write the file where `maestro.toml` says. A relative
path in `outputs` starts at the image's `WORKDIR`, or at `/` when the image
has none.

### Logs stay empty until the line ends, or arrive in bursts

The program buffers what it prints. In Python, start it with `-u`, set
`PYTHONUNBUFFERED=1` in the image, or print with `flush=True`.

### Logs or outputs stop early, with a note that they were cut

A line keeps at most 1 GB of logs and outputs, unless the administrator
changed `RUN_OUTPUT_MAX_MB`.

## Machines, for administrators

### A machine shows as unreachable

The machine's error says why. The usual reasons are:

- SSH refused the connection.
- The host key changed.
- The Podman socket isn't there. Run `systemctl --user enable --now
  podman.socket` on the machine, and `loginctl enable-linger` so that the
  socket stays up.
- The machine uses a proxy command while `MACHINE_PROXY_COMMANDS` is off.

Lines waiting on the machine go to other machines, and orq tries it again
every 30 seconds.

### A machine runs lines but `maestro top` shows no numbers for it

The machine's SSH user can reach the Podman socket and can't run commands.
The machine says so in `maestro top`.
[Preparing a machine](../admin/machines.md#preparing-a-machine) explains
what orq runs there.

### Lines use more cores than they asked for

The machine's Podman can't limit CPU. See
[Holding lines to their cores](../admin/machines.md#holding-lines-to-their-cores).

### Old runs have empty logs

orq creates containers with Podman's file log driver, whatever the
machine's default is. Runs made by an orq from before that change, on a
machine whose default is journald, may have empty logs.
