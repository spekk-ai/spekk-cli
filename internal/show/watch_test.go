package show

import (
	"encoding/json"
	"os"
	"os/exec"
	"path/filepath"
	"strings"
	"testing"
	"time"

	"github.com/spekk-ai/spekk-cli/internal/parser"
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

	// watchSpecs captures its baseline snapshot synchronously before returning,
	// so the modification below is guaranteed to come after it — no sleep needed.
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

	// Baseline snapshot is taken synchronously in watchSpecs; the new file below
	// therefore always postdates it.
	writeFile(t, filepath.Join(specsDir, "new.md"), "# New")

	select {
	case <-changed:
		// ok
	case <-time.After(3 * time.Second):
		t.Fatal("watcher did not detect new file within timeout")
	}
}

// TestWatchRefsDetectsBranchChange verifies the cross-branch ref watcher fires
// when a new branch is created — a git-state change that moves no working-tree
// .md file and would be missed by the file watcher alone.
func TestWatchRefsDetectsBranchChange(t *testing.T) {
	repo := initGitRepo(t)

	// chdir into repo so the crossbranch chokepoint (which shells out to git in
	// the current directory) sees this repo.
	prev, err := os.Getwd()
	if err != nil {
		t.Fatal(err)
	}
	if err := os.Chdir(repo); err != nil {
		t.Fatal(err)
	}
	defer os.Chdir(prev)

	changed := make(chan struct{}, 1)
	stop := watchRefs(func() {
		select {
		case changed <- struct{}{}:
		default:
		}
	})
	defer stop()

	// watchRefs records its baseline ref fingerprint synchronously before
	// returning, so creating the branch now is always seen as a later change.
	runGit(t, repo, "branch", "feature-x")

	select {
	case <-changed:
		// ok
	case <-time.After(5 * time.Second):
		t.Fatal("ref watcher did not detect new branch within timeout")
	}
}

// TestWatcherStopIdempotent ensures the watcher stop functions can be called
// more than once without panicking on a double close(done). RunWatch may invoke
// a stopper on both its error branch and the normal fall-through, so the stoppers
// must be idempotent.
func TestWatcherStopIdempotent(t *testing.T) {
	s1 := watchSpecs(t.TempDir(), func() {})
	s1()
	s1() // must not panic on a second close

	s2 := watchRefs(func() {})
	s2()
	s2() // must not panic on a second close
}

// initGitRepo creates a throwaway git repo with one commit and returns its path.
func initGitRepo(t *testing.T) string {
	t.Helper()
	dir := t.TempDir()
	runGit(t, dir, "init")
	runGit(t, dir, "config", "user.email", "test@example.com")
	runGit(t, dir, "config", "user.name", "Test")
	runGit(t, dir, "config", "commit.gpgsign", "false")
	writeFile(t, filepath.Join(dir, "README.md"), "# Test")
	runGit(t, dir, "add", "README.md")
	runGit(t, dir, "commit", "-m", "init")
	return dir
}

func runGit(t *testing.T, dir string, args ...string) {
	t.Helper()
	cmd := exec.Command("git", args...)
	cmd.Dir = dir
	if out, err := cmd.CombinedOutput(); err != nil {
		t.Fatalf("git %s: %v\n%s", strings.Join(args, " "), err, out)
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
