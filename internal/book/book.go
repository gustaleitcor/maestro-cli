// Package book is the Maestro book, as book/build.sh last built it into
// site/. The built pages are in the repository, so a plain `go build`
// carries them and `maestro docs` opens them without a server or a network.
// No other package holds files the binary carries.
package book

import (
	"embed"
	"io/fs"
)

// site is the built book: index.html and everything beside it. A test in
// cmd fails when it is older than the book's sources, by sources.sha256,
// their checksum as book/build.sh last built them.
//
//go:embed all:site
var site embed.FS

// Site is the book's pages, with index.html at its root.
func Site() fs.FS {
	pages, err := fs.Sub(site, "site")
	if err != nil {
		// "site" is a valid path, so Sub has nothing to refuse.
		panic(err)
	}
	return pages
}
