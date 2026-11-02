package validate

import (
	"os"
	"path/filepath"
	"runtime"
	"strings"
	"testing"
)

func TestFrontmatterAnchorsValidate(t *testing.T) {
	base := t.TempDir()
	root := filepath.Join(base, "project")
	if err := os.Mkdir(root, 0o755); err != nil {
		t.Fatal(err)
	}
	specsDir := filepath.Join(root, "specs")
	writeSpec(t, specsDir, "sample", "")
	if err := os.WriteFile(filepath.Join(root, "present.py"), []byte("pass\n"), 0o644); err != nil {
		t.Fatal(err)
	}
	frontmatter := "id: property\nparent: sample\ncreated: 2026-01-01T00:00:00Z\npriority: 1\nstatus: not_started\n"
	writeAssertion(t, specsDir, "sample", "property", frontmatter+"anchors: [present.py]\n")
	result, err := Run(specsDir)
	if err != nil || !result.Passed() {
		t.Fatalf("present anchor: result %v, error %v", result, err)
	}
	writeAssertion(t, specsDir, "sample", "property", frontmatter+"anchors: [missing.py, ../outside.py, /tmp/other.py, specs/sample]\n")
	result, err = Run(specsDir)
	if err != nil {
		t.Fatal(err)
	}
	if len(result.Failures) != 4 {
		t.Fatalf("failures = %v", result.Failures)
	}
	for _, anchor := range []string{"missing.py", "../outside.py", "/tmp/other.py", "specs/sample"} {
		found := false
		for _, failure := range result.Failures {
			if strings.Contains(failure.String(), "property.md: anchor \""+anchor+"\"") {
				found = true
			}
		}
		if !found {
			t.Errorf("missing failure for %s: %v", anchor, result.Failures)
		}
	}
	if runtime.GOOS == "windows" {
		return
	}
	outside := filepath.Join(base, "outside")
	if err := os.Mkdir(outside, 0o755); err != nil {
		t.Fatal(err)
	}
	if err := os.WriteFile(filepath.Join(outside, "secret.py"), []byte("pass\n"), 0o644); err != nil {
		t.Fatal(err)
	}
	for name, target := range map[string]string{
		"outside-link.py":  filepath.Join("..", "outside", "secret.py"),
		"outside-dir":      filepath.Join("..", "outside"),
		"inside-link.py":   filepath.Join(root, "present.py"),
		"dangling-link.py": "missing.py",
	} {
		if err := os.Symlink(target, filepath.Join(root, name)); err != nil {
			t.Fatal(err)
		}
	}
	writeAssertion(t, specsDir, "sample", "property", frontmatter+"anchors: [outside-link.py, outside-dir/secret.py, inside-link.py, dangling-link.py]\n")
	result, err = Run(specsDir)
	if err != nil {
		t.Fatal(err)
	}
	if len(result.Failures) != 3 {
		t.Fatalf("symlink failures = %v", result.Failures)
	}
	for _, anchor := range []string{"outside-link.py", "outside-dir/secret.py"} {
		found := false
		for _, failure := range result.Failures {
			if strings.Contains(failure.String(), "anchor \""+anchor+"\" resolves outside the project") {
				found = true
			}
		}
		if !found {
			t.Errorf("missing outside-project failure for %s: %v", anchor, result.Failures)
		}
	}
	foundDangling := false
	for _, failure := range result.Failures {
		if strings.Contains(failure.String(), "anchor \"dangling-link.py\" does not name a file") {
			foundDangling = true
		}
	}
	if !foundDangling {
		t.Errorf("missing dangling-link failure: %v", result.Failures)
	}
}
