package sandbox

import (
	"errors"
	"strings"
	"testing"
)

func TestEnsureCodexCLIInstallsMissingCodeModeHost(t *testing.T) {
	original := sshExec
	t.Cleanup(func() { sshExec = original })

	var calls []string
	sshExec = func(args []string, stdin string) ([]byte, error) {
		script := sentScript(args, stdin)
		calls = append(calls, script)
		if strings.Contains(script, "codex-code-mode-host.tar.gz") {
			return []byte("codex-cli 0.163.0\n"), nil
		}
		return nil, errors.New("code mode host is missing")
	}

	if err := ensureCodexCLI("127.0.0.1", "", "sandbox", "root"); err != nil {
		t.Fatal(err)
	}
	if len(calls) != 2 {
		t.Fatalf("got %d remote calls, want health check and install", len(calls))
	}
	if !strings.Contains(calls[0], "test -x /usr/local/bin/codex-code-mode-host") {
		t.Fatal("health check did not check the code mode host")
	}
	if !strings.Contains(calls[1], "install -m 0755 \"$tmp/codex-code-mode-host-") {
		t.Fatal("install did not install the code mode host")
	}
}
