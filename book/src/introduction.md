# What Maestro is

Maestro runs your program many times over, on a lab's shared machines, and
keeps what each run leaves behind.

You keep a program in a git repo, with a Dockerfile that builds it into a
container image. Maestro clones the repo, builds the image, and then runs it
as many containers as you have lines of parameters: a hundred seeds, a grid
of instances, a thousand cases of a benchmark. Each container gets the cores
and memory it asked for on whichever machine has room for it first, and when
it ends Maestro keeps what it printed and the files it wrote, for you to
download or look at in the browser.

The machines are shared, so the lines of everyone wait in one queue, and the
queue is fair: users and groups take turns, and an administrator chooses the
policies that decide whose line goes next.

```sh
maestro login                      # sign in, once
maestro build my-solver --watch    # build the repo, run it, follow the run
maestro runs get 31                # download what run 31 kept
```

## Who this book is for

- **If you run experiments**, read *Using Maestro*. [Your first
  run](start/first-run.md) goes from a repo to results in a few minutes, and
  [Describing a run](run/maestro-toml.md) is everything `maestro.toml` can
  say.
- **If you run a Maestro** for your lab, read *Running a Maestro*: deploying
  it, signing in through your identity provider, adding machines and choosing
  the scheduler's policies.
- **If you work on Maestro itself**, [Contributing](contributing.md) says
  where code goes.

## Finding your way

`maestro docs` opens this book; `maestro docs <topic>` opens a page of it,
and `maestro docs <command>` a command's entry in [Every
command](reference/cli.md). Every command's `--help` ends with the topic
that explains it:

```sh
maestro docs toml        # maestro.toml, key by key
maestro docs queue       # why a line is waiting
maestro docs runs logs   # what runs logs takes
```

`maestro help concepts` says, in the terminal, what [The words Maestro
uses](concepts.md) says here.

`maestro` carries this book, so `maestro docs` opens it offline, without
signing in: it describes what that version of `maestro` does.
`maestro docs --web` opens the copy your Maestro serves, if it serves one,
which you can open on another machine, as when you reach this one over SSH.
