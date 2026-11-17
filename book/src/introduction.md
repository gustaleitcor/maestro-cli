# What Maestro is

Maestro runs a program many times on a group of shared machines and keeps
what each run produces.

You keep the program in a git repository with a Dockerfile. Maestro clones
the repository, builds a container image from it, and starts one container
for each set of parameters you list. A hundred seeds become a hundred
containers. Each container gets the cores and memory it asked for, on
whichever machine has room for it first. When a container ends, Maestro keeps
what it printed and the files it wrote. You download them, or read them in
your browser.

The machines are shared, so everyone's containers wait in one queue. Users
take turns in it, and an administrator chooses the rules that decide whose
container starts next.

## What you need

- An account on a Maestro server. Your lab's administrator has its address.
- The `maestro` command on your computer.
- A git repository with a Dockerfile, on GitHub, GitLab, Codeberg or another
  Forgejo. The quickstart uses a public template, so you can start before you
  have one.

## How this book is organized

Each chapter adds one thing to what the previous one showed.

1. The [quickstart](start/quickstart.md) builds and runs a ready-made
   template, so you see a whole run before you learn its parts.
2. [The minimum: a Dockerfile](repo/dockerfile.md) starts from the least a
   repository needs. The next chapters add
   [`maestro.toml`](repo/maestro-toml.md), which runs the image many times,
   and a [Python script](repo/generator.md) that computes the parameters.
3. [Working with Maestro](use/install.md) covers the commands and the web
   page you use every day, from builds to downloaded results.
4. [How Maestro works](how/parts.md) explains the programs Maestro is made
   of, the queue, and the score that orders it.
5. [Running a Maestro](admin/deploy.md) is for administrators. It covers
   deploying the server, adding machines and choosing the queue's policies.
6. The reference lists [every command](reference/cli.md),
   [every `maestro.toml` key](reference/maestro-toml.md) and
   [every environment variable](reference/env.md), and has a
   [troubleshooting page](reference/troubleshooting.md) and a
   [glossary](reference/glossary.md).

If you want results and nothing more, read as far as the generator chapter.

## Opening a page from the terminal

`maestro docs` opens this book in your browser. The book is built into the
`maestro` command, so it opens offline and describes the version you have
installed.

```sh
maestro docs              # the first page
maestro docs toml         # maestro.toml
maestro docs queue        # why a line is waiting
maestro docs runs logs    # the reference entry for a command
```

Every command's `--help` ends with the topic that explains it.

`maestro` serves the book from your computer until you press Ctrl-C, and
only your computer can open it. Without a browser, as over SSH, it prints
the link and serves the book all the same. With `MAESTRO_DOCS_URL` set, it
opens the copy of the book at that address instead, such as
`http://localhost:3000`, where `mdbook serve` serves it while you work on
it.
