package sandbox

import (
	"os"
	"os/exec"
	"path/filepath"
	"testing"
)

// runShellCheck parses a script with `bash -n`, which reports syntax errors
// without running anything.
func runShellCheck(t *testing.T, script string) (string, error) {
	t.Helper()
	path := filepath.Join(t.TempDir(), "script.sh")
	if err := os.WriteFile(path, []byte(script), 0o600); err != nil {
		t.Fatal(err)
	}
	out, err := exec.Command("bash", "-n", path).CombinedOutput()
	return string(out), err
}
