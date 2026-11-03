package validate

import (
	"os"
	"os/exec"
	"path/filepath"
	"runtime"
	"strings"
	"testing"
)

func TestCommentAnchorsSkipNonRegularSources(t *testing.T) {
	root := t.TempDir()
	specsDir := filepath.Join(root, "specs")
	writeSpec(t, specsDir, "rules", "")
	writeAssertion(t, specsDir, "rules", "known-id", "id: known-id\nparent: rules\ncreated: 2026-01-01T00:00:00Z\npriority: 1\nstatus: done\n")
	git := func(args ...string) string {
		t.Helper()
		out, err := exec.Command("git", append([]string{"-C", root}, args...)...).CombinedOutput()
		if err != nil {
			t.Fatalf("git %v: %v: %s", args, err, out)
		}
		return strings.TrimSpace(string(out))
	}
	git("init", "-q")
	check := func(t *testing.T) {
		t.Helper()
		result, err := Run(specsDir)
		if err != nil || !result.Passed() || len(result.Warnings) != 0 {
			t.Fatalf("validation = %#v, %v", result, err)
		}
	}

	if runtime.GOOS != "windows" {
		t.Run("outside file symlink", func(t *testing.T) {
			outside := filepath.Join(t.TempDir(), "secret.py")
			if err := os.WriteFile(outside, []byte("# spekk: missing-outside\n"), 0o644); err != nil {
				t.Fatal(err)
			}
			link := filepath.Join(root, "outside.py")
			if err := os.Symlink(outside, link); err != nil {
				t.Fatal(err)
			}
			check(t)
			if err := os.Remove(link); err != nil {
				t.Fatal(err)
			}
		})
		t.Run("device symlink", func(t *testing.T) {
			link := filepath.Join(root, "device.py")
			if err := os.Symlink("/dev/zero", link); err != nil {
				t.Fatal(err)
			}
			check(t)
			if err := os.Remove(link); err != nil {
				t.Fatal(err)
			}
		})
		t.Run("directory symlink", func(t *testing.T) {
			link := filepath.Join(root, "directory.py")
			if err := os.Symlink(t.TempDir(), link); err != nil {
				t.Fatal(err)
			}
			check(t)
			if err := os.Remove(link); err != nil {
				t.Fatal(err)
			}
		})
	}
	t.Run("gitlink directory", func(t *testing.T) {
		if err := os.MkdirAll(filepath.Join(root, "vendor", "sub"), 0o755); err != nil {
			t.Fatal(err)
		}
		git("update-index", "--add", "--cacheinfo", "160000,e69de29bb2d1d6434b8b29ae775ad8c2e48c5391,vendor/sub")
		check(t)
	})
}

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
	write("source.py", []byte("x = 1 # spekk: known-id\n// spekk: known-id\n-- spekk: known-id\n# spekk: the binary is installed\n# spekk:managed hash=abc\n// spekk: known-id evidence=property\n# spekk: missing-id\r\n// spekk: Balance\n"))
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
