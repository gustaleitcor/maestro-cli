# Your own copy of the template

The quickstart ran the template as it is. To change what runs, you need the
template in a repository you can push to. The chapters after this one all
change that copy.

## Make the copy

On GitHub, open
[maestro-template](https://github.com/gustaleitcor/maestro-template) and
press Fork. For another forge, or for a private copy, create an empty
repository there and push the template into it:

```sh
git clone https://github.com/gustaleitcor/maestro-template my-experiment
cd my-experiment
git remote set-url origin <the address of your empty repository>
git push -u origin main
```

## Let Maestro read it

A public copy builds the way the template did, by its address:

```sh
maestro build https://github.com/<you>/maestro-template --watch
```

A private copy needs a forge. A forge is how Maestro reaches the place your
repositories live: its address and a read-only token of yours.

```sh
maestro forges add
```

The command asks which kind of forge it is (`github`, `forgejo` or `gitlab`),
tells you which token to create and where, and checks the token before it
keeps it. [Git forges](../use/forges.md) has the details.

With a forge added you can also name a repository by its short name, and
`maestro repos` lists the ones your token can see:

```sh
maestro build maestro-template --watch          # a repository of yours
maestro build some-org/solver --ref v1.2.0      # someone else's, at a tag
```

Without `--ref`, a build takes the newest commit of the default branch.

## What the copy holds

| File | What it is | Where this book covers it |
| --- | --- | --- |
| `Dockerfile` | Builds the image. Your program is its `ENTRYPOINT`. | [The minimum: a Dockerfile](../repo/dockerfile.md) |
| `maestro.toml` | Optional. Lists the lines of parameters and the files to keep. | [maestro.toml](../repo/maestro-toml.md) |
| `genvrs.py` | Optional. Computes lines in place of writing them out. | [Generating lines with Python](../repo/generator.md) |
| `main.py` | A stand-in program. It waits `-t` seconds and writes one line to a file. | Replace it with your program. |
| `requirements.txt` | The Python packages the Dockerfile installs. | |
| `.dockerignore` | What stays out of the image. | [The minimum: a Dockerfile](../repo/dockerfile.md#what-goes-into-the-image) |

## The loop you will repeat

1. Edit a file.
2. Commit and push.
3. `maestro build <your copy> --watch`.

A build clones what the forge has, so a change you haven't pushed isn't in
it. When you only want to try other parameters, you can skip the push and
the build. [Trying a change without pushing it](../repo/local-files.md) shows
how.
