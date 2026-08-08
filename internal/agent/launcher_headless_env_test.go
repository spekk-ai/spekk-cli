package agent

import (
	"os"
	"path/filepath"
	"strings"
	"testing"
)

func pathVar(env []string) string {
	for _, kv := range env {
		if eq := strings.Index(kv, "="); eq > 0 && strings.EqualFold(kv[:eq], "PATH") {
			return kv[eq+1:]
		}
	}
	return ""
}

func TestHeadlessChildEnv_PrependsExeDir(t *testing.T) {
	t.Setenv("PATH", "/usr/local/bin:/usr/bin")
	exe, err := os.Executable()
	if err != nil {
		t.Skip("no executable path in this environment")
	}
	dir := filepath.Dir(exe)

	got := pathVar(headlessChildEnv())
	parts := filepath.SplitList(got)
	if len(parts) == 0 || parts[0] != dir {
		t.Errorf("expected PATH to start with %q, got %q", dir, got)
	}
}

func TestHeadlessChildEnv_IdempotentWhenPresent(t *testing.T) {
	exe, err := os.Executable()
	if err != nil {
		t.Skip("no executable path in this environment")
	}
	dir := filepath.Dir(exe)
	t.Setenv("PATH", dir+string(os.PathListSeparator)+"/usr/bin")

	got := pathVar(headlessChildEnv())
	if strings.Count(got, dir) != 1 {
		t.Errorf("expected exactly one %q in PATH, got %q", dir, got)
	}
}
