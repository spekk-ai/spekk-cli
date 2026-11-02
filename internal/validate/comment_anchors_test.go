package validate

import (
	"os"
	"os/exec"
	"path/filepath"
	"strings"
	"testing"
)

func TestCommentAnchorsValidateGitSources(t *testing.T) {
	root := t.TempDir()
	specsDir := filepath.Join(root, "specs")
	writeSpec(t, specsDir, "rules", "")
	writeAssertion(t, specsDir, "rules", "known-id", "id: known-id\nparent: rules\ncreated: 2026-01-01T00:00:00Z\npriority: 1\nstatus: done\n")
	if out, err := exec.Command("git", "-C", root, "init", "-q").CombinedOutput(); err != nil {
		t.Fatalf("git init: %v: %s", err, out)
	}
	write := func(path string, data []byte) {
		t.Helper()
		full := filepath.Join(root, path)
		if err := os.MkdirAll(filepath.Dir(full), 0o755); err != nil {
			t.Fatal(err)
		}
		if err := os.WriteFile(full, data, 0o644); err != nil {
			t.Fatal(err)
		}
	}
	write("tracked.py", []byte("# spekk: known-id\n"))
	if out, err := exec.Command("git", "-C", root, "add", "tracked.py").CombinedOutput(); err != nil {
		t.Fatalf("git add: %v: %s", err, out)
	}
	write("source.py", []byte("x = 1 # spekk: known-id\n// spekk: known-id\n-- spekk: known-id\n# spekk: the binary is installed\n# spekk:managed hash=abc\n// spekk: known-id evidence=property\n# spekk: missing-id\n// spekk: Balance\n"))
	write("checks_test.go", []byte("// spekk: missing-marker\nfunc TestRule(t *testing.T) {}\n"))
	write("guide.md", []byte("# spekk: missing-doc\n"))
	write("testdata/sample.py", []byte("# spekk: missing-fixture\n"))
	write("binary.py", []byte("\x00# spekk: missing-binary\n"))
	write(".gitignore", []byte("ignored.py\n"))
	write("ignored.py", []byte("# spekk: missing-ignored\n"))

	result, err := Run(specsDir)
	if err != nil {
		t.Fatal(err)
	}
	if len(result.Warnings) != 0 || len(result.Failures) != 2 {
		t.Fatalf("validation = %#v", result)
	}
	got := result.Failures[0].String() + "\n" + result.Failures[1].String()
	if !strings.Contains(got, "source.py:7: anchor names unknown property missing-id") || !strings.Contains(got, "source.py:8: anchor names unknown property Balance") {
		t.Fatalf("failures = %s", got)
	}
}

func TestCommentAnchorsOutsideGitWarnAndPass(t *testing.T) {
	root := t.TempDir()
	specsDir := filepath.Join(root, "specs")
	writeSpec(t, specsDir, "rules", "")
	writeAssertion(t, specsDir, "rules", "known-id", "id: known-id\nparent: rules\ncreated: 2026-01-01T00:00:00Z\npriority: 1\nstatus: done\n")
	if err := os.WriteFile(filepath.Join(root, "source.py"), []byte("# spekk: missing-id\n"), 0o644); err != nil {
		t.Fatal(err)
	}
	result, err := Run(specsDir)
	if err != nil {
		t.Fatal(err)
	}
	if !result.Passed() || len(result.Warnings) != 1 || !strings.Contains(result.Warnings[0], "outside a git work tree") {
		t.Fatalf("validation = %#v", result)
	}
}
