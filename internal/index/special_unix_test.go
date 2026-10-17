//go:build unix

package index_test

import (
	"os"
	"path/filepath"
	"syscall"
	"testing"
	"time"

	"github.com/spekk-ai/spekk-cli/internal/index"
)

// A committed symlink to a device or a FIFO in the corpus is not a file to
// read: before the fix, the hash of raw/zero.bin read /dev/zero forever.
func TestRefreshSkipsFilesThatAreNotRegular(t *testing.T) {
	specsDir := makeSpecs(t)
	root := filepath.Dir(specsDir)
	writeFile(t, root, "docs/a.md", "# A\n\nThe aardvark.\n")
	writeFile(t, root, "raw/r.txt", "The raw file.\n")
	writeFile(t, root, "formatted/r.md", "---\ntitle: R\npath: raw/r.txt\ntext: formatted/r.txt\nsummary: A raw document about the rhea, long enough for a summary.\n---\n")
	writeFile(t, root, "formatted/r.txt", "The rhea.\n")
	dbPath := filepath.Join(root, ".spekk", "index.db")
	_, err := index.BuildIndex(specsDir, dbPath, false)
	must(t, err)
	forced := filepath.Join(t.TempDir(), "forced.db")
	_, err = index.BuildIndex(specsDir, forced, true)
	must(t, err)

	must(t, os.Symlink("/dev/zero", filepath.Join(root, "raw", "zero.bin")))
	must(t, syscall.Mkfifo(filepath.Join(root, "raw", "pipe.txt"), 0o644))
	must(t, syscall.Mkfifo(filepath.Join(root, "formatted", "pipe.md"), 0o644))
	must(t, syscall.Mkfifo(filepath.Join(root, "docs", "pipe.md"), 0o644))

	done := make(chan error, 1)
	go func() {
		_, err := index.Refresh(specsDir, dbPath, false)
		done <- err
	}()
	select {
	case err := <-done:
		must(t, err)
	case <-time.After(10 * time.Second):
		t.Fatal("Refresh did not return in 10 s")
	}
	if got, want := docRows(t, dbPath), docRows(t, forced); got != want {
		t.Errorf("rows differ from a forced rebuild without the special files:\n%s\nwant:\n%s", got, want)
	}
}
