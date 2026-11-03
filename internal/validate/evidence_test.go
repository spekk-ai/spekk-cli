package validate

import (
	"os"
	"os/exec"
	"path/filepath"
	"reflect"
	"testing"
)

func TestEvidenceValidation(t *testing.T) {
	root := t.TempDir()
	specsDir := filepath.Join(root, "specs")
	writeSpec(t, specsDir, "rules", "")
	writeAssertion(t, specsDir, "rules", "known-id", "id: known-id\nparent: rules\ncreated: 2026-01-01T00:00:00Z\npriority: 1\nstatus: done\n")
	if out, err := exec.Command("git", "-C", root, "init", "-q").CombinedOutput(); err != nil {
		t.Fatalf("git init: %v: %s", err, out)
	}
	write := func(path, content string) {
		t.Helper()
		full := filepath.Join(root, path)
		if err := os.MkdirAll(filepath.Dir(full), 0o755); err != nil {
			t.Fatal(err)
		}
		if err := os.WriteFile(full, []byte(content), 0o644); err != nil {
			t.Fatal(err)
		}
	}
	write("checks_test.go", "// spekk: missing-id evidence=property\nfunc TestOne(t *testing.T) {}\n// spekk: known-id evidence=magic\nfunc TestTwo(t *testing.T) {}\n")
	write("test_rules.py", "@spekk('missing-python', evidence='proof')\ndef test_rule(): pass\n@spekk('known-id', evidence='invalid')\ndef test_other(): pass\n@spekk('known-id')\ndef helper(): pass\n@spekk('missing-async')\nasync def test_async(): pass\n")
	write("testdata/skipped_test.go", "// spekk: skipped-id evidence=invalid\nfunc TestSkip(t *testing.T) {}\n")
	write(".gitignore", "ignored_test.go\n")
	write("ignored_test.go", "// spekk: ignored-id\nfunc TestSkip(t *testing.T) {}\n")
	result, err := Run(specsDir)
	if err != nil {
		t.Fatal(err)
	}
	var failures []string
	for _, f := range result.Failures {
		failures = append(failures, f.String())
	}
	want := []string{
		"checks_test.go:1: evidence marker names unknown property missing-id",
		"checks_test.go:3: unknown evidence level magic",
		"test_rules.py:1: evidence marker names unknown property missing-python",
		"test_rules.py:3: unknown evidence level invalid",
		"test_rules.py:7: evidence marker names unknown property missing-async",
	}
	if !reflect.DeepEqual(failures, want) {
		t.Fatalf("failures = %v, want %v", failures, want)
	}
	if !reflect.DeepEqual(result.Warnings, []string{"test_rules.py:5: @spekk decorator is not on a pytest test function"}) {
		t.Fatalf("warnings = %v", result.Warnings)
	}
	write("checks_test.go", "// spekk: known-id evidence=property\nfunc TestOne(t *testing.T) {}\n")
	write("test_rules.py", "@spekk('known-id')\nasync def test_rule(): pass\n")
	result, err = Run(specsDir)
	if err != nil || !result.Passed() || len(result.Warnings) != 0 {
		t.Fatalf("valid markers: %#v, %v", result, err)
	}
}
