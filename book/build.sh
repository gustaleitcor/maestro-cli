#!/bin/sh
# Builds the book and packs it as internal/book/site/book.tar.gz, which the
# next `go build` embeds and `maestro docs` opens offline. goreleaser runs it
# before every release; run it yourself before a `go build` that should
# carry the book.
#
# It uses mdbook and mdbook-mermaid from PATH when they are there, and else
# downloads the versions below into ~/.cache/maestro-book, checked against
# their checksums. Those are built for x86_64 Linux only: elsewhere, install
# both (`cargo install mdbook@0.4.40 mdbook-mermaid@0.14.0`).
set -eu

MDBOOK=0.4.40
MDBOOK_SHA256=9ef07fd288ba58ff3b99d1c94e6d414d431c9a61fdb20348e5beb74b823d546b
MERMAID=0.14.0
MERMAID_SHA256=37364965fb190cf9929d93c930c6b6c69b2fed05f7658ea2d312882ed59c645e

cd "$(dirname "$0")/.."

# fetch <url> <sha256>: unpacks a release tarball's binary into $tools.
fetch() {
	tmp="$tools/download.tgz"
	curl -fsSL -o "$tmp" "$1"
	echo "$2  $tmp" | sha256sum -c - >/dev/null || {
		echo "book/build.sh: $1 isn't what its checksum says" >&2
		exit 1
	}
	tar -xzf "$tmp" -C "$tools"
	rm "$tmp"
}

if ! command -v mdbook >/dev/null || ! command -v mdbook-mermaid >/dev/null; then
	tools="${XDG_CACHE_HOME:-$HOME/.cache}/maestro-book/mdbook-$MDBOOK-mermaid-$MERMAID"
	if [ ! -x "$tools/mdbook" ] || [ ! -x "$tools/mdbook-mermaid" ]; then
		if [ "$(uname -s)-$(uname -m)" != "Linux-x86_64" ]; then
			echo "book/build.sh: install mdbook $MDBOOK and mdbook-mermaid $MERMAID (cargo install mdbook@$MDBOOK mdbook-mermaid@$MERMAID)" >&2
			exit 1
		fi
		mkdir -p "$tools"
		echo "book/build.sh: downloading mdbook $MDBOOK and mdbook-mermaid $MERMAID into $tools" >&2
		fetch "https://github.com/rust-lang/mdBook/releases/download/v$MDBOOK/mdbook-v$MDBOOK-x86_64-unknown-linux-gnu.tar.gz" "$MDBOOK_SHA256"
		fetch "https://github.com/badboy/mdbook-mermaid/releases/download/v$MERMAID/mdbook-mermaid-v$MERMAID-x86_64-unknown-linux-gnu.tar.gz" "$MERMAID_SHA256"
	fi
	PATH="$tools:$PATH"
fi

go run . gen-docs book/src/reference/cli.md
mdbook-mermaid install book >/dev/null 2>&1
rm -rf book/dist
# mdbook-mermaid is built against an older mdbook, and says so on every build.
mdbook build book 2>&1 | grep -v "built against version" >&2 || true
[ -f book/dist/index.html ] || { echo "book/build.sh: the book didn't build" >&2; exit 1; }

mkdir -p internal/book/site
tar -C book/dist -czf internal/book/site/book.tar.gz .
echo "book/build.sh: packed $(du -h internal/book/site/book.tar.gz | cut -f1) into internal/book/site/book.tar.gz" >&2
