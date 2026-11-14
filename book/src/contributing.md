# Contributing

Maestro's parts each live in their own repo, as git submodules of
[maestro](https://github.com/gustaleitcor/maestro), and are developed and
versioned on their own:

| Repo | |
| --- | --- |
| `maestro-orq` | the server: Go, Gin, SQLite |
| `maestro-cli` | `maestro`, and this book: Go, cobra, Bubble Tea, mdBook |
| `maestro-view` | the Maestro page: Vue |
| `maestro-template` | a repo for users to fork |
| `maestro` | what ties them together: the compose files |

Where code goes in each is in
[CONVENTIONS.md](https://github.com/gustaleitcor/maestro/blob/main/CONVENTIONS.md),
at the root of the maestro repo; read it before adding a file.
[Deploying Maestro](admin/deploy.md#a-development-stack) has the
development stack.

## This book

The book is in `maestro-cli/book`, written with
[mdBook](https://rust-lang.github.io/mdBook/), and every `maestro` carries
it: `maestro docs` opens the copy built into the binary, offline.

```sh
cd maestro-cli
sh book/build.sh                 # build it, for the next go build to embed
mdbook serve book                # or edit it with a live preview, on :3000
```

`book/build.sh` writes [Every command](reference/cli.md) from the CLI itself
(`maestro gen-docs`), builds the book, and packs it where `go build` embeds
it. It uses `mdbook` and `mdbook-mermaid` when they are installed, and else
downloads the versions it names into your cache. goreleaser runs it before
every release.

A page moved or renamed must move in `cmd/docs.go` too, where the topics of
`maestro docs` say which page each opens; `go test ./cmd` fails while one
points at a page the book hasn't.

A change to what Maestro does is documented here, in the same change.
