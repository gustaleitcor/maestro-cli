// Package book is the Maestro book, built by book/build.sh and carried in
// the binary, so that `maestro docs` opens it without a server or a
// network. No other package holds files the binary carries.
package book

import (
	"archive/tar"
	"bytes"
	"compress/gzip"
	"crypto/sha256"
	"embed"
	"encoding/hex"
	"errors"
	"fmt"
	"io"
	"io/fs"
	"os"
	"path/filepath"
	"strings"
)

// site holds book.tar.gz once book/build.sh has run, and only .gitkeep
// before, so that a build without the book still compiles.
//
//go:embed all:site
var site embed.FS

// ErrMissing is what a maestro built without running book/build.sh says.
var ErrMissing = errors.New("this maestro was built without its book")

// Dir is where the book is on disk, unpacked from the binary into the
// user's cache the first time, once for each book: a new maestro with
// another book unpacks it beside the old one.
func Dir() (string, error) {
	archive, err := site.ReadFile("site/book.tar.gz")
	if errors.Is(err, fs.ErrNotExist) {
		return "", ErrMissing
	}
	if err != nil {
		return "", err
	}
	cache, err := os.UserCacheDir()
	if err != nil {
		return "", err
	}
	sum := sha256.Sum256(archive)
	dir := filepath.Join(cache, "maestro", "book", hex.EncodeToString(sum[:6]))
	if _, err := os.Stat(filepath.Join(dir, "index.html")); err == nil {
		return dir, nil
	}
	return dir, unpackInto(archive, dir)
}

// unpackInto unpacks archive beside dir and then moves it there, so that a
// book cut short is never taken for a whole one, and two maestros unpacking
// at once don't mix theirs.
func unpackInto(archive []byte, dir string) error {
	if err := os.MkdirAll(filepath.Dir(dir), 0o755); err != nil {
		return err
	}
	tmp, err := os.MkdirTemp(filepath.Dir(dir), ".unpacking-")
	if err != nil {
		return err
	}
	defer os.RemoveAll(tmp)
	if err := unpack(archive, tmp); err != nil {
		return fmt.Errorf("unpacking the book: %w", err)
	}
	if err := os.Rename(tmp, dir); err != nil {
		// Another maestro got there first, with the same book.
		if _, statErr := os.Stat(filepath.Join(dir, "index.html")); statErr == nil {
			return nil
		}
		return err
	}
	removeOthers(dir)
	return nil
}

// removeOthers removes the books an older maestro unpacked beside dir, so
// that updating doesn't leave one behind each time. What can't be removed
// stays: it is only a cache.
func removeOthers(dir string) {
	entries, err := os.ReadDir(filepath.Dir(dir))
	if err != nil {
		return
	}
	for _, e := range entries {
		if e.IsDir() && e.Name() != filepath.Base(dir) && !strings.HasPrefix(e.Name(), ".unpacking-") {
			os.RemoveAll(filepath.Join(filepath.Dir(dir), e.Name()))
		}
	}
}

// unpack writes the regular files and directories of a tar.gz into dir.
// Anything else is skipped, and no entry may land outside dir.
func unpack(archive []byte, dir string) error {
	gz, err := gzip.NewReader(bytes.NewReader(archive))
	if err != nil {
		return err
	}
	tr := tar.NewReader(gz)
	for {
		hdr, err := tr.Next()
		if err == io.EOF {
			return nil
		}
		if err != nil {
			return err
		}
		name := filepath.FromSlash(strings.TrimPrefix(hdr.Name, "./"))
		if name == "" || name == "." {
			continue
		}
		if !filepath.IsLocal(name) {
			return fmt.Errorf("%q would land outside the book", hdr.Name)
		}
		target := filepath.Join(dir, name)
		switch hdr.Typeflag {
		case tar.TypeDir:
			if err := os.MkdirAll(target, 0o755); err != nil {
				return err
			}
		case tar.TypeReg:
			if err := os.MkdirAll(filepath.Dir(target), 0o755); err != nil {
				return err
			}
			f, err := os.OpenFile(target, os.O_CREATE|os.O_WRONLY|os.O_TRUNC, 0o644)
			if err != nil {
				return err
			}
			if _, err := io.Copy(f, tr); err != nil {
				f.Close()
				return err
			}
			if err := f.Close(); err != nil {
				return err
			}
		}
	}
}
