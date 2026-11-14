# How the parts fit

Maestro is three programs and the machines they drive.

```mermaid
flowchart TB
    cli["maestro, the CLI"] -- "Maestro key" --> orq
    view["the Maestro page"] -- "session cookie" --> orq
    subgraph server["the Maestro server"]
        orq["orq: API, builds, scheduler"]
        orq --- store[("SQLite · Redis · /maestro/data")]
    end
    orq -- "sign in" --> idp["identity provider"]
    orq -- "clone" --> forge["git forges"]
    orq -- "SSH to each Podman socket" --> machines
    subgraph machines["the machines"]
        direction LR
        m1["Q1"] ~~~ m2["Q2"] ~~~ m3["…"]
    end
```

**orq** is the server, and the only part that decides anything. It signs
people in through the lab's identity provider, clones repos with the forge
tokens the CLI hands it, builds images with its own Podman, keeps every
build, run and line in SQLite, and runs the scheduler: the loop that picks
which waiting line goes where, copies images to machines, starts containers,
and collects what they leave. orq keeps everything about runs in its
database, so it can restart in the middle of one and pick up where it was.

**The view** is the Maestro page: a web page served beside orq, on the same
host, that shows your runs and their files, the queue and your usage, and
the admin pages. It holds no state of its own.

**The CLI**, `maestro`, does everything from a terminal, and keeps on your
computer only the server's URL, your Maestro key and your forge tokens.

**The machines** are computers with Podman and sshd, nothing else. orq
reaches each over SSH and talks to its Podman socket through that
connection: it copies images there, starts containers, reads how loaded the
machine is, and takes the outputs back when a container ends. Containers are
removed once their outputs are kept; images stay, so the next run of a build
starts without copying it again.

## The life of a run

```mermaid
sequenceDiagram
    actor you
    participant orq
    participant forge
    participant machine as a machine
    you->>orq: maestro build my-app
    orq->>forge: clone at a commit
    orq->>orq: read maestro.toml, run the generator, build the image
    orq-->>you: build #12
    you->>orq: maestro run 12
    orq->>orq: queue one line per line of parameters
    loop each line, when its turn comes and a machine has room
        orq->>machine: copy the image (once), start the container
        machine-->>orq: it ended: logs and outputs
        orq->>machine: remove the container
    end
    you->>orq: maestro runs get 31
    orq-->>you: every line's logs and outputs
```

The [queue](watch/queue.md) is where most of a line's life is spent on a busy
day; [Scheduler policies](admin/policies.md) says how its turns are decided.
