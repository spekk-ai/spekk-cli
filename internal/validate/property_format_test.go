package validate

import (
	"os"
	"path/filepath"
	"strings"
	"testing"
)

func TestFrontmatterAnchorsValidate(t *testing.T) {
	root := t.TempDir()
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
}
