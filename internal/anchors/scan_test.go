package anchors

import (
	"os"
	"os/exec"
	"path/filepath"
	"reflect"
	"testing"
)

func TestScanEvidenceAndAnchors(t *testing.T) {
	root := t.TempDir()
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
	write("checks_test.go", "// spekk: first-id evidence=property\n"+
		"// A later doc line keeps the marker in the block.\n"+
		"// spekk: second-id\n"+
		"func TestRule(t *testing.T) {}\n"+
		"// spekk: fuzz-id evidence=bounded\n"+
		"func FuzzRule(f *testing.F) {}\n"+
		"// spekk: anchor-id\n"+
		"\n"+
		"func TestOther(t *testing.T) {}\n"+
		"// spekk: anchor-id\n"+
		"func helper() {}\n")
	write("test_rules.py", "@spekk('first-id', evidence='contract')\n"+
		"@other\n"+
		"@spekk(\"second-id\")\n"+
		"def test_rule(): pass\n"+
		"\n"+
		"class TestRules:\n"+
		"    @other\n"+
		"    @spekk(\"first-id\", other=True, evidence=\"proof\")\n"+
		"    def test_method(self): pass\n"+
		"\n"+
		"@spekk(\"first-id\")\n"+
		"def helper(): pass\n"+
		"@spekk(\"second-id\")\n"+
		"async def test_async(): pass\n"+
		"\n"+
		"@spekk(\"first-id\")\n"+
		"async def helper(): pass\n")
	write("testdata/test_skip.py", `@spekk("missing-id")
def test_skip(): pass
`)
	write("guide.md", "// spekk: missing-id\n")
	write(".gitignore", "ignored_test.go\n")
	write("ignored_test.go", "// spekk: missing-id\nfunc TestIgnored(t *testing.T) {}\n")
	write("binary_test.go", "\x00// spekk: missing-id\nfunc TestBinary(t *testing.T) {}\n")
	got, inGit, err := ScanAll(root)
	if err != nil || !inGit {
		t.Fatalf("scan: %v, inGit %v", err, inGit)
	}
	var evidence []string
	for _, e := range got.Evidence {
		evidence = append(evidence, e.ID+":"+e.Level+":"+e.Language+":"+e.Path+":"+e.Test)
		if e.Test == "test_async" && e.Line != 13 {
			t.Fatalf("async evidence line = %d, want 13", e.Line)
		}
	}
	wantEvidence := []string{
		"first-id:property:go:checks_test.go:TestRule", "second-id:example:go:checks_test.go:TestRule", "fuzz-id:bounded:go:checks_test.go:FuzzRule",
		"first-id:contract:python:test_rules.py:test_rule", "second-id:example:python:test_rules.py:test_rule", "first-id:proof:python:test_rules.py:TestRules.test_method", "second-id:example:python:test_rules.py:test_async",
	}
	if !reflect.DeepEqual(evidence, wantEvidence) {
		t.Fatalf("evidence = %v, want %v", evidence, wantEvidence)
	}
	if len(got.Anchors) != 2 || got.Anchors[0].Line != 7 || got.Anchors[1].Line != 10 {
		t.Fatalf("anchors = %#v", got.Anchors)
	}
	if !reflect.DeepEqual(got.Warnings, []Warning{{Path: "test_rules.py", Line: 11}, {Path: "test_rules.py", Line: 16}}) {
		t.Fatalf("warnings = %#v", got.Warnings)
	}
}
