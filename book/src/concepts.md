# The words Maestro uses

A handful of words name everything Maestro does. Each has a page of its own
later; this is how they fit together.

**Repo.** A git repository of yours, on GitHub, GitLab, Codeberg or another
Forgejo or Gitea. It holds your program, a Dockerfile, and usually a
`maestro.toml`, which says how to run it.

**Forge.** Where a repo lives, as Maestro reaches it: the forge's API and a
read-only token of yours, which the CLI keeps and hands to the server when it
needs to clone. Forges never sign anyone in. See [Git forges](start/forges.md).

**Build.** `maestro build <repo>` clones a repo at a commit and builds a
container image from it. Each build has a number, shown as `#12`. A build
also reads `maestro.toml` and computes the lines, so every run of a build is
the same lines. See [Builds and images](watch/builds.md).

**Image.** What a build makes: your program and all it needs, ready to start
as a container. You keep a few at a time (10 unless your administrator says
otherwise); the one you ran least recently goes when a new build needs room.

**Run.** `maestro run <build>` runs a build's image on the machines. Each run
has a number of its own, shown as `run 31`. A build number and a run number
are never the same thing: `run` takes a build, `runs show` takes a run. See
[Runs and lines](watch/runs.md).

**Line.** One container of a run. `maestro.toml` lists lines of parameters
(and may name a generator that computes more); a run starts one container
per line, with that line's arguments in place of the image's `CMD`. Without
a `maestro.toml`, a run is a single line that runs the image's own command.
A run has at most 1000 lines.

**Machine.** A computer whose Podman runs the containers. Maestro reaches it
over SSH; a line runs on one machine, the first allowed one with room for
its cores and memory. See [Watching the machines](watch/top.md).

**Waiting.** A line waits at three levels before it runs:

| Level | What it waits for |
| --- | --- |
| `queued` | to be chosen from among its user's lines |
| `staged` | chosen, waiting for a machine |
| `assigned` | given to a machine, waiting for room on it while its image is copied there |

then it is `starting` and `running`, and ends `succeeded`, `failed` (its
program exited with an error), `error` (it couldn't run, which is Maestro's
trouble rather than your program's) or `cancelled`. See [The queue,
explained](watch/queue.md).

**Group.** Users are in groups, and groups take turns at the machines, so a
lab of ten doesn't crowd out a lab of two. Until an administrator makes
others, everyone is in one. See [Groups](admin/groups.md).

**Outputs.** The files `maestro.toml` names, copied out of each container
when it ends, beside what it printed (`stdout.log` and `stderr.log`).
`maestro runs get` downloads them. See [What a line leaves
behind](watch/results.md).

**orq, view, CLI.** The three programs Maestro is made of: orq is the server,
the view is the Maestro page in your browser, and the CLI is `maestro`. See
[How the parts fit](architecture.md).
