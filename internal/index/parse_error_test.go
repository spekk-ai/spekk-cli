package index_test

import (
	"errors"
	"os"
	"path/filepath"
	"strings"
	"testing"

	"github.com/spekk-ai/spekk-cli/internal/index"
)

// writeSpecTree writes a parent spec plus the given assertion files, and
// returns the specs directory.
func writeSpecTree(t *testing.T, assertions map[string]string) string {
	t.Helper()
	specsDir := filepath.Join(t.TempDir(), "specs")
	assertDir := filepath.Join(specsDir, "demo", "assertions")
	if err := os.MkdirAll(assertDir, 0o755); err != nil {
		t.Fatal(err)
	}
	parent := "---\nid: demo\ntitle: Demo\npriority: 1\ncreated: 2026-08-01T00:00:00Z\n---\n\n# Demo\n"
	if err := os.WriteFile(filepath.Join(specsDir, "demo", "demo.md"), []byte(parent), 0o644); err != nil {
		t.Fatal(err)
	}
	for name, content := range assertions {
		if err := os.WriteFile(filepath.Join(assertDir, name), []byte(content), 0o644); err != nil {
			t.Fatal(err)
		}
	}
	return specsDir
}

const goodAssertion = "---\nid: a-one\nparent: demo\ncreated: 2026-08-01T00:00:00Z\npriority: 1\nstatus: not_started\n---\n\n# A one\n"

// A depends-on flow list stopped a repository. The parser reads it as one
// string and refuses it, and that failure applies to the whole tree. The
// build must mark it as a spec-tree failure and not as a database failure,
// because only a spec-tree failure is corrected by an edit to a file.
func TestBuildIndexMarksAnUnparseableTree(t *testing.T) {
	specsDir := writeSpecTree(t, map[string]string{
		"a-one.md": goodAssertion,
		"a-two.md": "---\nid: a-two\nparent: demo\ncreated: 2026-08-01T00:00:00Z\npriority: 1\nstatus: not_started\ndepends-on: [a-one, a-three]\n---\n\n# A two\n",
	})

	_, err := index.BuildIndex(specsDir, filepath.Join(t.TempDir(), "index.db"), false)
	if err == nil {
		t.Fatal("expected an error for a tree that does not parse")
	}
	if !errors.Is(err, index.ErrSpecsUnparseable) {
		t.Errorf("error must be identifiable as an unparseable tree, got: %v", err)
	}
	// The message keeps the file name and the value. The new text is an
	// addition, not a replacement.
	if !strings.Contains(err.Error(), "a-two.md") {
		t.Errorf("error must still name the offending file, got: %v", err)
	}
}

// The plain parser error gave none of three facts a person needs: the scope
// is the whole repository, no spekk command operates until the correction,
// and one command reports each problem.
func TestFormatErrorStatesScopeAndTheFix(t *testing.T) {
	err := index.FormatError(errWrapped())

	for _, want := range []string{
		"whole tree",       // scope: not just the one file
		"every branch",     // scope: not just this checkout
		"no spekk command", // consequence
		`"spekk validate"`, // the fix
	} {
		if !strings.Contains(err, want) {
			t.Errorf("guidance must mention %q, got:\n%s", want, err)
		}
	}
}

// A database failure applies to one checkout, and a person corrects it by a
// delete of index.db. The spec-tree guidance on that error would send the
// person to edit a file that is correct.
func TestFormatErrorLeavesOtherErrorsAlone(t *testing.T) {
	other := errors.New("cannot create schema: disk full")
	if got := index.FormatError(other); got != other.Error() {
		t.Errorf("a non-parse error must pass through unchanged, got: %q", got)
	}
	if index.FormatError(nil) != "" {
		t.Error("a nil error must format as the empty string")
	}
}

func errWrapped() error {
	specsDirErr := errors.New("Field 'depends-on' must be kebab-case in specs/demo/assertions/a-two.md")
	return errors.Join(index.ErrSpecsUnparseable, specsDirErr)
}
