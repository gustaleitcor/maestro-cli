# Trying a change without pushing it

Committing and pushing for every experiment with `maestro.toml` gets slow.
`maestro build` and `maestro run` can take three files from your computer and
use them in place of the repository's:

| Flag | Replaces |
| --- | --- |
| `-c <file>` | `maestro.toml` at the root of the repository |
| `-g <file>` | The generator that `maestro.toml` names. When it names none, the script is added at the root under its own name and used. |
| `-f <file>` | The Dockerfile that `maestro.toml` names, or else `Dockerfile` at the root. Only `maestro build` takes it. |

The file on your computer can have any name.

```sh
maestro build my-app -c bench.toml -g genvrs.py -f Dockerfile   # a new image
maestro run 81 -c small.toml --watch                            # build #81's image, other lines
```

## With a run: other lines, same image

`maestro run <build> -c` or `-g` doesn't build anything. The server clones
the repository again at the commit the build was made from, puts your files
over the clone, runs the generator there, and starts the build's image on
the lines that result. It takes seconds, because no image is built.

Use it to try other parameters, other `cpus` and `memory`, other machines or
another seed. For a private repository the CLI sends the forge token it has,
so the server can clone.

A run doesn't take `-f`, and it ignores a `dockerfile` key in the
`maestro.toml` you send, because it builds no image.

## With a build: a new image

`maestro build` puts your files over its clone before it reads anything, so
the image is built with them. Use it when the Dockerfile changes.

A change to the program itself still has to be pushed. These flags replace
only the three files above.

## How Maestro records it

The lines of such a build or run don't come from the commit alone, so
`maestro builds` and `maestro runs show` say which files you sent.

What you send is checked the way a build checks the repository's files, and
an invalid file fails the build or refuses the run. A file can be at most
1 MiB.
