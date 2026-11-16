# Contributing

Each part of Maestro lives in its own repository and is developed and
versioned on its own. The
[maestro](https://github.com/gustaleitcor/maestro) repository holds them as
git submodules.

| Repository | What it holds |
| --- | --- |
| `maestro-orq` | The server, in Go with Gin and SQLite |
| `maestro-cli` | The `maestro` command and this book, in Go with cobra and Bubble Tea, and mdBook |
| `maestro-view` | The Maestro page, in Vue |
| `maestro-template` | The repository users fork to start |
| `maestro` | The compose files that run the parts together |

[CONVENTIONS.md](https://github.com/gustaleitcor/maestro/blob/main/CONVENTIONS.md),
at the root of the maestro repository, says where code goes in each part.
Read it before you add a file.
[Deploying Maestro](admin/deploy.md#a-development-stack) describes the
development stack.

## This book

The book is in `maestro-cli/book` and is written with
[mdBook](https://rust-lang.github.io/mdBook/). Every `maestro` binary
carries a built copy, which `maestro docs` opens offline.

```sh
cd maestro-cli
sh book/build.sh                 # build it, for the next go build to embed
mdbook serve book                # or edit it with a live preview, on :3000
```

`book/build.sh` does three things:

1. It writes [Every command](reference/cli.md) from the CLI itself, with
   `maestro gen-docs`.
2. It builds the book.
3. It packs the result where `go build` embeds it.

The script uses `mdbook` and `mdbook-mermaid` when they are installed.
Otherwise it downloads the versions it names into your cache. goreleaser
runs it before every release.

### When you move or rename a page

`cmd/docs.go` holds the topics of `maestro docs`, each with the page it
opens. A page that moves in the book has to move there too. `go test ./cmd`
fails while a topic points at a page that `SUMMARY.md` doesn't list.

### When you change what Maestro does

Document the change in this book, in the same change as the code.
