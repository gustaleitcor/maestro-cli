package book

import (
	"io/fs"
	"testing"
)

// The binary carries a whole book: its first page, a chapter in a folder,
// and the styles and scripts a page needs beside it.
func TestSite(t *testing.T) {
	for _, name := range []string{"index.html", "start/quickstart.html", "css/general.css", "book.js"} {
		info, err := fs.Stat(Site(), name)
		if err != nil {
			t.Errorf("%s: %v", name, err)
			continue
		}
		if info.Size() == 0 {
			t.Errorf("%s is empty", name)
		}
	}
}
