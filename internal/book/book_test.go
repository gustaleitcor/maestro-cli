package book

import (
	"archive/tar"
	"bytes"
	"compress/gzip"
	"os"
	"path/filepath"
	"testing"
)

// archive is a tar.gz of entries, as book/build.sh packs one: "./" first,
// names starting "./", a name ending "/" a directory.
func archive(t *testing.T, entries map[string]string, order ...string) []byte {
	t.Helper()
	var buf bytes.Buffer
	gz := gzip.NewWriter(&buf)
	tw := tar.NewWriter(gz)
	for _, name := range order {
		hdr := &tar.Header{Name: name, Mode: 0o644, Size: int64(len(entries[name])), Typeflag: tar.TypeReg}
		if name[len(name)-1] == '/' {
			hdr = &tar.Header{Name: name, Mode: 0o755, Typeflag: tar.TypeDir}
		}
		if err := tw.WriteHeader(hdr); err != nil {
			t.Fatal(err)
		}
		if hdr.Typeflag == tar.TypeReg {
			tw.Write([]byte(entries[name]))
		}
	}
	tw.Close()
	gz.Close()
	return buf.Bytes()
}

func TestUnpackInto(t *testing.T) {
	a := archive(t, map[string]string{"./index.html": "<h1>Maestro</h1>", "./watch/queue.html": "queue"},
		"./", "./index.html", "./watch/", "./watch/queue.html")
	dir := filepath.Join(t.TempDir(), "maestro", "book", "abc")
	older := filepath.Join(filepath.Dir(dir), "old")
	if err := os.MkdirAll(older, 0o755); err != nil {
		t.Fatal(err)
	}
	if err := unpackInto(a, dir); err != nil {
		t.Fatal(err)
	}
	if got, _ := os.ReadFile(filepath.Join(dir, "watch", "queue.html")); string(got) != "queue" {
		t.Errorf("watch/queue.html holds %q", got)
	}
	if _, err := os.Stat(older); err == nil {
		t.Error("an older book was left beside the new one")
	}
	// Already there, as when another maestro unpacked it first.
	if err := unpackInto(a, dir); err != nil {
		t.Errorf("unpacking over a whole book: %v", err)
	}
	if left, _ := filepath.Glob(filepath.Join(filepath.Dir(dir), ".unpacking-*")); len(left) > 0 {
		t.Errorf("left %q behind", left)
	}
}

func TestUnpackStaysInside(t *testing.T) {
	a := archive(t, map[string]string{"../evil": "x"}, "../evil")
	if err := unpack(a, t.TempDir()); err == nil {
		t.Error("an entry outside the book was unpacked")
	}
}

// Whatever this binary carries, Dir either finds the book or says it has
// none: never half of one.
func TestDir(t *testing.T) {
	t.Setenv("XDG_CACHE_HOME", t.TempDir())
	t.Setenv("HOME", t.TempDir())
	dir, err := Dir()
	if err == ErrMissing {
		t.Skip("built without book/build.sh")
	}
	if err != nil {
		t.Fatal(err)
	}
	if _, err := os.Stat(filepath.Join(dir, "index.html")); err != nil {
		t.Error(err)
	}
}
