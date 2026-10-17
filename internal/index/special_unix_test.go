//go:build unix

package index_test

import (
	"os"
	"path/filepath"
	"strings"
	"syscall"
	"testing"
	"time"

	"github.com/spekk-ai/spekk-cli/internal/docs"
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
	outside := filepath.Join(t.TempDir(), "outside.txt")
	must(t, os.WriteFile(outside, []byte("Not in the repository.\n"), 0o644))
	must(t, os.Symlink(outside, filepath.Join(root, "raw", "outside.txt")))

	// The forced rebuild with the entries reads every stub and document,
	// so it also reaches the readers that the manifest does not.
	withEntries := filepath.Join(t.TempDir(), "with.db")
	done := make(chan error, 1)
	go func() {
		if _, err := index.Refresh(specsDir, dbPath, false); err != nil {
			done <- err
			return
		}
		_, err := index.BuildIndex(specsDir, withEntries, true)
		done <- err
	}()
	select {
	case err := <-done:
		must(t, err)
	case <-time.After(10 * time.Second):
		t.Fatal("Refresh and a forced rebuild did not return in 10 s")
	}
	want := docRows(t, forced)
	for _, db := range []string{dbPath, withEntries} {
		if got := docRows(t, db); got != want {
			t.Errorf("rows of %s differ from a forced rebuild without the special files:\n%s\nwant:\n%s", filepath.Base(db), got, want)
		}
	}
	// docs status must not tell an agent to add one of them.
	c, err := docs.Scan(root)
	must(t, err)
	if len(c.NoStub) != 0 {
		t.Errorf("docs status lists %v as raw files to add", c.NoStub)
	}
	// Nor must docs add copy the outside file into the corpus.
	var out strings.Builder
	if failed := docs.Add(root, []string{filepath.Join(root, "raw", "outside.txt")}, docs.AddOptions{}, &out); failed != 1 {
		t.Errorf("docs add of the outside symlink: %d failed, want 1:\n%s", failed, out.String())
	}
	for _, rel := range []string{"formatted/outside.md", "formatted/outside.txt"} {
		if _, err := os.Lstat(filepath.Join(root, rel)); !os.IsNotExist(err) {
			t.Errorf("docs add of the outside symlink wrote %s", rel)
		}
	}
}
