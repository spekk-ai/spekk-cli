package show

import (
	"encoding/json"
	"path/filepath"
	"strings"
	"testing"
	"time"

	"github.com/spekk-dev/spekk-cli/internal/parser"
)

func TestScanMdFiles(t *testing.T) {
	dir := t.TempDir()
	writeFile(t, filepath.Join(dir, "spec.md"), "# Spec")
	writeFile(t, filepath.Join(dir, "sub", "nested.md"), "# Nested")
	writeFile(t, filepath.Join(dir, "readme.txt"), "text")

	files := scanMdFiles(dir)
	if len(files) != 2 {
		t.Fatalf("expected 2 md files, got %d", len(files))
	}
	if _, ok := files["spec.md"]; !ok {
		t.Error("missing spec.md")
	}
	if _, ok := files[filepath.Join("sub", "nested.md")]; !ok {
		t.Error("missing sub/nested.md")
	}
}

func TestScanMdFilesMissingDir(t *testing.T) {
	files := scanMdFiles("/nonexistent/path")
	if len(files) != 0 {
		t.Errorf("expected empty map for missing dir, got %d entries", len(files))
	}
}

func TestSnapshotChanged(t *testing.T) {
	now := time.Now()

	a := map[string]time.Time{"a.md": now}
	b := map[string]time.Time{"a.md": now}

	if snapshotChanged(a, b) {
		t.Error("identical snapshots should not be changed")
	}

	// Modified file
	c := map[string]time.Time{"a.md": now.Add(time.Second)}
	if !snapshotChanged(a, c) {
		t.Error("modified file should be detected")
	}

	// New file
	d := map[string]time.Time{"a.md": now, "b.md": now}
	if !snapshotChanged(a, d) {
		t.Error("new file should be detected")
	}

	// Deleted file
	e := map[string]time.Time{}
	if !snapshotChanged(a, e) {
		t.Error("deleted file should be detected")
	}
}

func TestWatchSpecsDetectsChange(t *testing.T) {
	dir := t.TempDir()
	specsDir := filepath.Join(dir, "specs")
	writeFile(t, filepath.Join(specsDir, "test.md"), "# Initial")

	changed := make(chan struct{}, 1)
	stop := watchSpecs(specsDir, func() {
		select {
		case changed <- struct{}{}:
		default:
		}
	})
	defer stop()

	// Wait for watcher to take initial snapshot
	time.Sleep(100 * time.Millisecond)

	// Modify a file
	writeFile(t, filepath.Join(specsDir, "test.md"), "# Modified")

	select {
	case <-changed:
		// ok
	case <-time.After(3 * time.Second):
		t.Fatal("watcher did not detect file change within timeout")
	}
}

func TestWatchSpecsDetectsNewFile(t *testing.T) {
	dir := t.TempDir()
	specsDir := filepath.Join(dir, "specs")
	writeFile(t, filepath.Join(specsDir, "existing.md"), "# Existing")

	changed := make(chan struct{}, 1)
	stop := watchSpecs(specsDir, func() {
		select {
		case changed <- struct{}{}:
		default:
		}
	})
	defer stop()

	time.Sleep(100 * time.Millisecond)

	// Add a new file
	writeFile(t, filepath.Join(specsDir, "new.md"), "# New")

	select {
	case <-changed:
		// ok
	case <-time.After(3 * time.Second):
		t.Fatal("watcher did not detect new file within timeout")
	}
}

func TestListenWithRetry(t *testing.T) {
	ln1, err := listenWithRetry(0, 0)
	if err != nil {
		t.Fatal(err)
	}
	defer ln1.Close()

	if ln1 == nil {
		t.Fatal("expected non-nil listener")
	}
}

func TestListenWithRetrySkipsUsedPort(t *testing.T) {
	// Bind a port
	ln1, err := listenWithRetry(13117, 0)
	if err != nil {
		t.Fatal(err)
	}
	defer ln1.Close()

	// Try same port with retry — should get next port
	ln2, err := listenWithRetry(13117, 5)
	if err != nil {
		t.Fatal(err)
	}
	defer ln2.Close()

	if ln1.Addr().String() == ln2.Addr().String() {
		t.Error("second listener should be on a different port")
	}
}

func TestSSEScriptInjection(t *testing.T) {
	dir := t.TempDir()
	specsDir := filepath.Join(dir, "specs")

	writeFile(t, filepath.Join(specsDir, "test-spec", "test-spec.md"), `---
id: test-spec
created: 2026-01-01T00:00:00Z
priority: 1
---

# Test Spec`)

	writeFile(t, filepath.Join(specsDir, "test-spec", "assertions", "test-a.md"), `---
id: test-a
parent: test-spec
created: 2026-01-01T00:00:00Z
priority: 1
status: not_started
---

# Test Assertion`)

	result, err := parser.ParseAllSpecs(specsDir)
	if err != nil {
		t.Fatal(err)
	}

	data := buildShowData(specsDir, result)
	jsonBytes, _ := json.Marshal(data)
	html := strings.Replace(templateHTML, "/*__SPEKK_DATA__*/", string(jsonBytes), 1)
	html = strings.Replace(html, "</body>", sseClientScript+"\n</body>", 1)

	if !strings.Contains(html, "EventSource") {
		t.Error("watch-mode HTML should contain EventSource SSE client")
	}
	if !strings.Contains(html, "spekkWatchState") {
		t.Error("watch-mode HTML should contain state preservation code")
	}
	if !strings.Contains(html, "test-spec") {
		t.Error("HTML should contain spec data")
	}
}
